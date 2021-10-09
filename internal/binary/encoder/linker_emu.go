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
    `unsafe`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
)

func init() {
    atm.RegisterGCall(encode, emu_gcall_encode)
}

func link(prog atm.Program) Encoder {
    return func(iov frugal.IoVec, p unsafe.Pointer, rs *RuntimeState, st int) (err error) {
        emu := atm.LoadProgram(prog)
        ret := *(*rt.GoIface)(unsafe.Pointer(&err))
        iop := *(*rt.GoIface)(unsafe.Pointer(&iov))
        emu.Ap(0, unsafe.Pointer(iop.Itab))
        emu.Ap(1, iop.Value)
        emu.Ap(2, p)
        emu.Ap(3, unsafe.Pointer(rs))
        emu.Au(4, uint64(st))
        emu.Run()
        ret.Itab = (*rt.GoItab)(emu.Rp(0))
        ret.Value = emu.Rp(1)
        emu.Free()
        return
    }
}

func emu_gcall_encode(e *atm.Emulator, p *atm.Instr) {
    var v4 int
    var v1 rt.GoIface
    var v0 *rt.GoType
    var v3 *RuntimeState
    var v2 unsafe.Pointer

    /* check for arguments and return values */
    if (p.An != 6 || p.Rn != 2) ||
       (p.Ai[0] & atm.ArgPointer) == 0 ||
       (p.Ai[1] & atm.ArgPointer) == 0 ||
       (p.Ai[2] & atm.ArgPointer) == 0 ||
       (p.Ai[3] & atm.ArgPointer) == 0 ||
       (p.Ai[4] & atm.ArgPointer) == 0 ||
       (p.Ai[5] & atm.ArgPointer) != 0 ||
       (p.Rv[0] & atm.ArgPointer) == 0 ||
       (p.Rv[1] & atm.ArgPointer) == 0 {
        panic("invalid encode call")
    }

    /* extract the arguments and return value index */
    v0       =    (*rt.GoType)(e.Pr[p.Ai[0] & atm.ArgMask])
    v1.Itab  =    (*rt.GoItab)(e.Pr[p.Ai[1] & atm.ArgMask])
    v1.Value =                 e.Pr[p.Ai[2] & atm.ArgMask]
    v2       =                 e.Pr[p.Ai[3] & atm.ArgMask]
    v3       = (*RuntimeState)(e.Pr[p.Ai[4] & atm.ArgMask])
    v4       =             int(e.Gr[p.Ai[5] & atm.ArgMask])

    /* call the function */
    iov := *(*frugal.IoVec)(unsafe.Pointer(&v1))
    ret := encode(v0, iov, v2, v3, v4)

    /* pack the result */
    e.Pr[p.Rv[0] & atm.ArgMask] = *(*unsafe.Pointer)(unsafe.Pointer(&ret))
    e.Pr[p.Rv[1] & atm.ArgMask] = (*rt.GoIface)(unsafe.Pointer(&ret)).Value
}
