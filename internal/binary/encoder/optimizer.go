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
    _PASS_SizeMerging,
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

func newAdjustBuffer(n int) []int {
    if v := adjustPool.Get(); v != nil {
        return v.([]int)[:0]
    } else {
        return make([]int, 0, n)
    }
}

// Compacting Pass: remove all the PC-adjustment instructions inserted in the previous pass.
func _PASS_Compacting(p Program) Program {
    i := 0
    j := 0
    n := 0
    a := newAdjustBuffer(len(p))

    /* scan for PC adjustments */
    for _, v := range p {
        if v.Op() == _OP_adjpc { n += v.To() }
        a = append(a, n)
    }

    /* adjust branch offsets */
    for i < len(p) {
        iv := p[i]
        op := iv.Op()

        /* skip the PC adjustment */
        if op == _OP_adjpc {
            i++
            continue
        }

        /* copy instructions and adjust branch targets */
        if p[j] = p[i]; _OpBranches[op] {
            p[j] = mkins(op, iv.To() + a[iv.To()], iv.Iv(), nil)
        }

        /* move forward */
        i++
        j++
    }

    /* all done */
    adjustPool.Put(a)
    return p[:j]
}

// Size Merging Pass: merges size-checking instructions as much as possible.
func _PASS_SizeMerging(p Program) Program {
    i := 0
    n := len(p)

    /* scan every instruction */
    for i < n {
        iv := p[i]
        op := iv.Op()

        /* only interested in size instructions */
        if op != OP_size {
            i++
            continue
        }

        /* size accumulator */
        ip := i
        nb := int64(0)

        /* scan for mergable size instructions */
        loop: for i < n {
            iv = p[i]
            op = iv.Op()

            /* check for mergable instructions */
            switch op {
                default      : break loop
                case OP_byte : break
                case OP_word : break
                case OP_long : break
                case OP_quad : break
                case OP_sint : break
                case OP_seek : break
                case OP_size : nb += iv.Iv()
            }

            /* adjust the program counter */
            if i++; op == OP_size {
                p[i - 1] = mkins(_OP_adjpc, -1, 0, nil)
            }
        }

        /* replace the size instruction */
        if i != ip {
            p[ip] = mkins(OP_size, 0, nb, nil)
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
        op := iv.Op()

        /* only interested in literal instructions */
        if op > OP_quad {
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
            op = iv.Op()

            /* check for OpCode */
            switch op {
                default      : break loop
                case OP_byte : append1(&sl, byte(iv.Iv()))
                case OP_word : append2(&sl, uint16(iv.Iv()))
                case OP_long : append4(&sl, uint32(iv.Iv()))
                case OP_quad : append8(&sl, uint64(iv.Iv()))
            }

            /* adjust the program counter */
            p[i] = mkins(_OP_adjpc, -1, 0, nil)
            i++

            /* commit the buffer if needed */
            for len(sl) >= 8 {
                p[ip] = mkins(OP_quad, 0, int64(binary.BigEndian.Uint64(sl)), nil)
                sl = sl[8:]
                ip++
            }

            /* move the remaining bytes to the front */
            copy(mm[:], sl)
            sl = mm[:len(sl):cap(mm)]
        }

        /* add the remaining bytes */
        if len(sl) >= 4 { p[ip] = mkins(OP_long, 0, int64(binary.BigEndian.Uint32(sl)), nil) ; sl = sl[4:]; ip++ }
        if len(sl) >= 2 { p[ip] = mkins(OP_word, 0, int64(binary.BigEndian.Uint16(sl)), nil) ; sl = sl[2:]; ip++ }
        if len(sl) >= 1 { p[ip] = mkins(OP_byte, 0, int64(sl[0]), nil)                       ; sl = sl[1:]; ip++ }
    }

    /* all done */
    return p
}
