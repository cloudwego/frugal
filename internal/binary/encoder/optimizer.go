/*
 * Copyright 2021 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package encoder

import (
    `encoding/binary`
    `sync`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

func Optimize(p Program) Program {
    for _, f := range _PassTab { p = f(p) }
    return p
}

var (
    adjustPool sync.Pool
)

var _PassTab = [...]func(p Program) Program {
    _PASS_StaticSizeMerging,
    _PASS_Compacting,
    _PASS_SeekMerging,
    _PASS_ZeroSeekElimination,
    _PASS_Compacting,
    _PASS_SizeCheckMerging,
    _PASS_Compacting,
    _PASS_LiteralMerging,
    _PASS_Compacting,
}

const (
    _OP_adjpc OpCode = 0xff
)

func init() {
    _OpNames[_OP_adjpc] = "(PC-adjustment)"
}

func checksl(s *[]byte, n int) *rt.GoSlice {
    sl := (*rt.GoSlice)(unsafe.Pointer(s))
    sn := sl.Len

    /* check for length */
    if sn > 16 - n {
        panic("slice overflow")
    } else {
        return sl
    }
}

func append1(s *[]byte, v byte) {
    sl := checksl(s, 1)
    sl.Set(sl.Len, v)
    sl.Len++
}

func append2(s *[]byte, v uint16) {
    sl := checksl(s, 2)
    sl.Set(sl.Len + 0, byte(v >> 8))
    sl.Set(sl.Len + 1, byte(v))
    sl.Len += 2
}

func append4(s *[]byte, v uint32) {
    sl := checksl(s, 4)
    sl.Set(sl.Len + 0, byte(v >> 24))
    sl.Set(sl.Len + 1, byte(v >> 16))
    sl.Set(sl.Len + 2, byte(v >> 8))
    sl.Set(sl.Len + 3, byte(v))
    sl.Len += 4
}

func append8(s *[]byte, v uint64) {
    sl := checksl(s, 8)
    sl.Set(sl.Len + 0, byte(v >> 56))
    sl.Set(sl.Len + 1, byte(v >> 48))
    sl.Set(sl.Len + 2, byte(v >> 40))
    sl.Set(sl.Len + 3, byte(v >> 32))
    sl.Set(sl.Len + 4, byte(v >> 24))
    sl.Set(sl.Len + 5, byte(v >> 16))
    sl.Set(sl.Len + 6, byte(v >> 8))
    sl.Set(sl.Len + 7, byte(v))
    sl.Len += 8
}

func makeadj(v Instr, n *int) {
    if v.Op == _OP_adjpc {
        *n += int(v.Iv)
    }
}

func adjustpc(v *Instr, adj []int) {
    if _OpBranches[v.Op] {
        v.To += adj[v.To]
    }
}

func removeadj(p Program, v *Instr) {
    for p[v.To].Op == _OP_adjpc {
        v.To++
    }
}

func newAdjustBuffer(n int) []int {
    if v := adjustPool.Get(); v != nil {
        return v.([]int)[:0]
    } else {
        return make([]int, 0, n)
    }
}

// Compacting Pass: remove all the PC-adjustment instructions inserted in the previous pass.
func _PASS_Compacting(p Program) Program {
    j := 0
    n := 0
    a := newAdjustBuffer(len(p))

    /* move all the jumps so that it does not point to any PC adjustment pseudo-instructions */
    for i := 0; i < len(p); i++ {
        if _OpBranches[p[i].Op] {
            removeadj(p, &p[i])
        }
    }

    /* scan for PC adjustments */
    for _, v := range p {
        makeadj(v, &n)
        a = append(a, n)
    }

    /* adjust branch offsets */
    for _, v := range p {
        if v.Op != _OP_adjpc {
            adjustpc(&v, a)
            p[j] = v
            j++
        }
    }

    /* all done */
    adjustPool.Put(a)
    return p[:j]
}

// Size Check Merging Pass: merges size-checking instructions as much as possible.
func _PASS_SizeCheckMerging(p Program) Program {
    i := 0
    n := len(p)

    /* scan every instruction */
    for i < n {
        iv := p[i]
        op := iv.Op

        /* only interested in size instructions */
        if op != OP_size_check {
            i++
            continue
        }

        /* size accumulator */
        ip := i
        nb := int64(0)

        /* scan for mergable size instructions */
        loop: for i < n {
            iv = p[i]
            op = iv.Op

            /* check for mergable instructions */
            switch op {
                default            : break loop
                case OP_byte       : break
                case OP_word       : break
                case OP_long       : break
                case OP_quad       : break
                case OP_sint       : break
                case OP_seek       : break
                case OP_size_check : nb += iv.Iv
            }

            /* adjust the program counter */
            if i++; op == OP_size_check {
                p[i - 1] = Instr {
                    Iv: -1,
                    Op: _OP_adjpc,
                }
            }
        }

        /* replace the size instruction */
        if i != ip {
            p[ip] = Instr {
                Iv: nb,
                Op: OP_size_check,
            }
        }
    }

    /* all done */
    return p
}

// Literal Merging Pass: merges all consectutive byte, word or long instructions.
func _PASS_LiteralMerging(p Program) Program {
    i := 0
    n := len(p)

    /* scan every instruction */
    for i < n {
        iv := p[i]
        op := iv.Op

        /* only interested in literal instructions */
        if op < OP_byte || op > OP_quad {
            i++
            continue
        }

        /* byte merging buffer */
        ip := i
        mm := [16]byte{}
        sl := mm[:0:cap(mm)]

        /* scan for consecutive bytes */
        loop: for i < n {
            iv = p[i]
            op = iv.Op

            /* check for OpCode */
            switch op {
                default      : break loop
                case OP_byte : append1(&sl, byte(iv.Iv))
                case OP_word : append2(&sl, uint16(iv.Iv))
                case OP_long : append4(&sl, uint32(iv.Iv))
                case OP_quad : append8(&sl, uint64(iv.Iv))
            }

            /* adjust the program counter */
            p[i] = Instr{Op: _OP_adjpc, Iv: -1}
            i++

            /* commit the buffer if needed */
            for len(sl) >= 8 {
                p[ip] = Instr{Op: OP_quad, Iv: int64(binary.BigEndian.Uint64(sl))}
                sl = sl[8:]
                ip++
            }

            /* move the remaining bytes to the front */
            copy(mm[:], sl)
            sl = mm[:len(sl):cap(mm)]
        }

        /* add the remaining bytes */
        if len(sl) >= 4 { p[ip] = Instr{Op: OP_long, Iv: int64(binary.BigEndian.Uint32(sl))} ; sl = sl[4:]; ip++ }
        if len(sl) >= 2 { p[ip] = Instr{Op: OP_word, Iv: int64(binary.BigEndian.Uint16(sl))} ; sl = sl[2:]; ip++ }
        if len(sl) >= 1 { p[ip] = Instr{Op: OP_byte, Iv: int64(sl[0])}                       ; sl = sl[1:]; ip++ }
    }

    /* all done */
    return p
}

// Static Size Merging Pass: merges constant size instructions as much as possible.
func _PASS_StaticSizeMerging(p Program) Program {
    i := 0
    n := len(p)

    /* scan every instruction */
    for i < n {
        iv := p[i]
        op := iv.Op

        /* only interested in size instructions */
        if op != OP_size_const {
            i++
            continue
        }

        /* size accumulator */
        ip := i
        nb := int64(0)

        /* scan for mergable size instructions */
        loop: for i < n {
            iv = p[i]
            op = iv.Op

            /* check for mergable instructions */
            switch op {
                default            : break loop
                case OP_seek       : break
                case OP_size_dyn   : break
                case OP_size_const : nb += iv.Iv
            }

            /* adjust the program counter */
            if i++; op == OP_size_const {
                p[i - 1] = Instr {
                    Iv: -1,
                    Op: _OP_adjpc,
                }
            }
        }

        /* replace the size instruction */
        if i != ip {
            p[ip] = Instr {
                Iv: nb,
                Op: OP_size_const,
            }
        }
    }

    /* all done */
    return p
}

// Seek Merging Pass: merges seeking instructions as much as possible.
func _PASS_SeekMerging(p Program) Program {
    i := 0
    n := len(p)

    /* scan every instruction */
    for i < n {
        iv := p[i]
        op := iv.Op

        /* only interested in size instructions */
        if op != OP_seek {
            i++
            continue
        }

        /* size accumulator */
        ip := i
        nb := int64(0)

        /* scan for mergable size instructions */
        for i < n {
            iv = p[i]
            op = iv.Op

            /* check for mergable instructions */
            if op != OP_seek {
                break
            }

            /* add up the offsets */
            i++
            nb += iv.Iv

            /* adjust the program counter */
            p[i - 1] = Instr {
                Iv: -1,
                Op: _OP_adjpc,
            }
        }

        /* replace the size instruction */
        if i != ip {
            p[ip] = Instr {
                Iv: nb,
                Op: OP_seek,
            }
        }
    }

    /* all done */
    return p
}

// Zero Seek Elimination Pass: remove seek instruction with zero offsets
func _PASS_ZeroSeekElimination(p Program) Program {
    var i int
    var v Instr

    /* replace every zero-offset seek with NOP */
    for i, v = range p {
        if v.Iv == 0 && v.Op == OP_seek {
            p[i] = Instr {
                Iv: -1,
                Op: _OP_adjpc,
            }
        }
    }

    /* all done */
    return p
}
