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
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
)

func init() {
    Link = link_emu
    atm.RegisterGCall(decode, emu_gcall_decode)
}

func link_emu(prog atm.Program) Decoder {
    return func(buf []byte, p unsafe.Pointer, rs *RuntimeState, st int) (pos int, err error) {
        emu := atm.LoadProgram(prog)
        ret := (*rt.GoIface)(unsafe.Pointer(&err))
        src := (*rt.GoSlice)(unsafe.Pointer(&buf))
        emu.Ap(0, src.Ptr)
        emu.Au(1, uint64(src.Len))
        emu.Au(2, uint64(src.Cap))
        emu.Ap(3, p)
        emu.Ap(4, unsafe.Pointer(rs))
        emu.Au(5, uint64(st))
        emu.Run()
        pos = int(emu.Ru(0))
        ret.Itab = (*rt.GoItab)(emu.Rp(1))
        ret.Value = emu.Rp(2)
        emu.Free()
        return
    }
}

func emu_gcall_decode(e *atm.Emulator, p *atm.Instr) {
    var v4 int
    var v1 rt.GoSlice
    var v0 *rt.GoType
    var v3 *RuntimeState
    var v2 unsafe.Pointer

    /* check for arguments and return values */
    if (p.An != 7 || p.Rn != 3) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) == 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 ||
       (p.Ar[3] & atm.ArgPointer) != 0 ||
       (p.Ar[4] & atm.ArgPointer) == 0 ||
       (p.Ar[5] & atm.ArgPointer) == 0 ||
       (p.Ar[6] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) != 0 ||
       (p.Rr[1] & atm.ArgPointer) == 0 ||
       (p.Rr[2] & atm.ArgPointer) == 0 {
        panic("invalid decode call")
    }

    /* extract the arguments */
    v0       =    (*rt.GoType)(e.Pr[p.Ar[0] & atm.ArgMask])
    v1.Ptr  =                  e.Pr[p.Ar[1] & atm.ArgMask]
    v1.Len =               int(e.Gr[p.Ar[2] & atm.ArgMask])
    v1.Cap =               int(e.Gr[p.Ar[3] & atm.ArgMask])
    v2       =                 e.Pr[p.Ar[4] & atm.ArgMask]
    v3       = (*RuntimeState)(e.Pr[p.Ar[5] & atm.ArgMask])
    v4       =             int(e.Gr[p.Ar[6] & atm.ArgMask])

    /* call the function */
    buf := *(*[]byte)(unsafe.Pointer(&v1))
    ret, err := decode(v0, buf, v2, v3, v4)

    /* pack the result */
    e.Gr[p.Rr[0] & atm.ArgMask] = uint64(ret)
    e.Pr[p.Rr[1] & atm.ArgMask] = (*[2]unsafe.Pointer)(unsafe.Pointer(&err))[0]
    e.Pr[p.Rr[2] & atm.ArgMask] = (*[2]unsafe.Pointer)(unsafe.Pointer(&err))[1]
}