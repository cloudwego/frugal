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

package decoder

import (
    `math/bits`
    `sync`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/binary/defs`
)

type _skipbuf_t [defs.MaxStack]struct {
    t defs.Tag
    k defs.Tag
    v defs.Tag
    n uint32
}

var (
	_skipbuf_p sync.Pool
)

var _SkipSizeFixed = [256]int {
    defs.T_bool   : 1,
    defs.T_i8     : 1,
    defs.T_double : 8,
    defs.T_i16    : 2,
    defs.T_i32    : 4,
    defs.T_i64    : 8,
}

const (
    _T_map_pair defs.Tag = 0xa0
)

func mksb() *_skipbuf_t {
    if v := _skipbuf_p.Get(); v != nil {
        return v.(*_skipbuf_t)
    } else {
        return new(_skipbuf_t)
    }
}

func u32be(s unsafe.Pointer) int {
    return int(bits.ReverseBytes32(*(*uint32)(s)))
}

func stpop(s *_skipbuf_t, p *int) bool {
    if s[*p].n == 0 {
        *p--
        return true
    } else {
        s[*p].n--
        return false
    }
}

func stadd(s *_skipbuf_t, p *int, t defs.Tag) {
    if *p++; *p < defs.MaxStack {
        s[*p].t, s[*p].n = t, 0
    } else {
        panic("skip stack overflow")
    }
}

func mvbuf(s *unsafe.Pointer, n *int, r *int, nb int) {
    *n = *n - nb
    *r = *r + nb
    *s = unsafe.Pointer(uintptr(*s) + uintptr(nb))
}

func do_skip(s unsafe.Pointer, n int, t defs.Tag) (rv int) {
    sp := 0
    st := mksb()
    st[0].t = t

    /* run until drain */
    for sp >= 0 {
        switch st[sp].t {
            default: {
                _skipbuf_p.Put(st)
                return ETAG
            }

            /* simple fixed types */
            case defs.T_bool   : fallthrough
            case defs.T_i8     : fallthrough
            case defs.T_double : fallthrough
            case defs.T_i16    : fallthrough
            case defs.T_i32    : fallthrough
            case defs.T_i64    : {
                if nb := _SkipSizeFixed[st[sp].t]; n < nb {
                    _skipbuf_p.Put(st)
                    return EEOF
                } else {
                    stpop(st, &sp)
                    mvbuf(&s, &n, &rv, nb)
                }
            }

            /* strings & binaries */
            case defs.T_string: {
                if n < 4 {
                    _skipbuf_p.Put(st)
                    return EEOF
                } else if nb := u32be(s) + 4; n < nb {
                    _skipbuf_p.Put(st)
                    return EEOF
                } else {
                    stpop(st, &sp)
                    mvbuf(&s, &n, &rv, nb)
                }
            }

            /* structs */
            case defs.T_struct: {
                var nb int
                var vt defs.Tag

                /* must have at least 1 byte */
                if n < 1 {
                    _skipbuf_p.Put(st)
                    return EEOF
                }

                /* check for end of tag */
                if vt = *(*defs.Tag)(s); vt == 0 {
                    stpop(st, &sp)
                    mvbuf(&s, &n, &rv, 1)
                    continue
                }

                /* check for tag value */
                if !vt.IsWireTag() {
                    _skipbuf_p.Put(st)
                    return ETAG
                }

                /* fast-path for primitive fields */
                if nb = _SkipSizeFixed[vt]; nb != 0 {
                    if n < nb + 3 {
                        _skipbuf_p.Put(st)
                        return EEOF
                    } else {
                        mvbuf(&s, &n, &rv, nb + 3)
                        continue
                    }
                }

                /* must have more than 3 bytes (fields cannot have a size of zero) */
                if n <= 3 {
                    _skipbuf_p.Put(st)
                    return EEOF
                }

                /* also skip the field ID cause we don't care */
                stadd(st, &sp, vt)
                mvbuf(&s, &n, &rv, 3)
            }

            /* maps */
            case defs.T_map: {
                var np int
                var kt defs.Tag
                var vt defs.Tag

                /* must have at least 6 bytes */
                if n < 6 {
                    _skipbuf_p.Put(st)
                    return EEOF
                }

                /* get the element type and count */
                kt = (*[2]defs.Tag)(s)[0]
                vt = (*[2]defs.Tag)(s)[1]
                np = u32be(unsafe.Pointer(uintptr(s) + 2))

                /* check for tag value */
                if !kt.IsWireTag() || !vt.IsWireTag() {
                    _skipbuf_p.Put(st)
                    return ETAG
                }

                /* empty map */
                if np == 0 {
                    stpop(st, &sp)
                    mvbuf(&s, &n, &rv, 6)
                    continue
                }

                /* fast path for fixed key and value */
                if nk, nv := _SkipSizeFixed[kt], _SkipSizeFixed[vt]; nk != 0 && nv != 0 {
                    if nb := np * (nk + nv) + 6; n < nb {
                        _skipbuf_p.Put(st)
                        return EEOF
                    } else {
                        stpop(st, &sp)
                        mvbuf(&s, &n, &rv, nb)
                        continue
                    }
                }

                /* set to parse the map pairs */
                st[sp].k = kt
                st[sp].v = vt
                st[sp].t = _T_map_pair
                st[sp].n = uint32(np) * 2 - 1
                mvbuf(&s, &n, &rv, 6)
            }

            /* map pairs */
            case _T_map_pair: {
                if vt := st[sp].v; stpop(st, &sp) || st[sp].n & 1 != 0 {
                    stadd(st, &sp, vt)
                } else {
                    stadd(st, &sp, st[sp].k)
                }
            }

            /* sets and lists */
            case defs.T_set  : fallthrough
            case defs.T_list : {
                var nv int
                var et defs.Tag

                /* must have at least 5 bytes */
                if n < 5 {
                    _skipbuf_p.Put(st)
                    return EEOF
                }

                /* get the element type and count */
                et = *(*defs.Tag)(s)
                nv = u32be(unsafe.Pointer(uintptr(s) + 1))

                /* check for tag value */
                if !et.IsWireTag() {
                    _skipbuf_p.Put(st)
                    return ETAG
                }

                /* empty sequence */
                if nv == 0 {
                    stpop(st, &sp)
                    mvbuf(&s, &n, &rv, 5)
                    continue
                }

                /* fast path for fixed types */
                if nt := _SkipSizeFixed[et]; nt != 0 {
                    if nb := nv * nt + 5; n < nb {
                        _skipbuf_p.Put(st)
                        return EEOF
                    } else {
                        stpop(st, &sp)
                        mvbuf(&s, &n, &rv, nb)
                        continue
                    }
                }

                /* set to parse the elements */
                st[sp].t = et
                st[sp].n = uint32(nv) - 1
                mvbuf(&s, &n, &rv, 5)
            }
        }
    }

    /* return the skipbuf to the pool */
    _skipbuf_p.Put(st)
    return
}

func emu_ccall_skip(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ai[0] & atm.ArgPointer) == 0 ||
       (p.Ai[1] & atm.ArgPointer) != 0 ||
       (p.Ai[2] & atm.ArgPointer) != 0 ||
       (p.Rv[0] & atm.ArgPointer) != 0 {
        panic("invalid skip call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ai[0] & atm.ArgMask
    a1 = p.Ai[1] & atm.ArgMask
    a2 = p.Ai[2] & atm.ArgMask
    r0 = p.Rv[0] & atm.ArgMask

    /* call the function */
    e.Gr[r0] = uint64(do_skip(
        e.Pr[a0],
        int(e.Gr[a1]),
        defs.Tag(e.Gr[a2]),
    ))
}

func init() {
    FnSkip = unsafe.Pointer(&FnSkip)
    atm.RegisterCCall(FnSkip, emu_ccall_skip)
}
