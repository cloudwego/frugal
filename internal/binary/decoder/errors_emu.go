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

func emu_gcall_error_eof(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var r0 uint8
    var r1 uint8

    /* check for arguments and return values */
    if (p.An != 1 || p.Rn != 2) ||
       (p.Ar[0] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 ||
       (p.Rr[1] & atm.ArgPointer) == 0 {
        panic("invalid error_eof call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask
    r1 = p.Rr[1] & atm.ArgMask

    /* call the function */
    ex := error_eof(int(e.Gr[a0]))
    vv := (*rt.GoIface)(unsafe.Pointer(&ex))

    /* update the result register */
    e.Pr[r1] = vv.Value
    e.Pr[r0] = unsafe.Pointer(vv.Itab)
}

func emu_gcall_error_skip(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var r0 uint8
    var r1 uint8

    /* check for arguments and return values */
    if (p.An != 1 || p.Rn != 2) ||
       (p.Ar[0] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 ||
       (p.Rr[1] & atm.ArgPointer) == 0 {
        panic("invalid error_skip call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask
    r1 = p.Rr[1] & atm.ArgMask

    /* call the function */
    ex := error_skip(int(e.Gr[a0]))
    vv := (*rt.GoIface)(unsafe.Pointer(&ex))

    /* update the result register */
    e.Pr[r1] = vv.Value
    e.Pr[r0] = unsafe.Pointer(vv.Itab)
}

func emu_gcall_error_type(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var r0 uint8
    var r1 uint8

    /* check for arguments and return values */
    if (p.An != 2 || p.Rn != 2) ||
       (p.Ar[0] & atm.ArgPointer) != 0 ||
       (p.Ar[1] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 ||
       (p.Rr[1] & atm.ArgPointer) == 0 {
        panic("invalid error_type call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask
    r1 = p.Rr[1] & atm.ArgMask

    /* call the function */
    ret := error_type(
        uint8(e.Gr[a0]),
        uint8(e.Gr[a1]),
    )

    /* update the result register */
    e.Pr[r0] = (*[2]unsafe.Pointer)(unsafe.Pointer(&ret))[0]
    e.Pr[r1] = (*[2]unsafe.Pointer)(unsafe.Pointer(&ret))[1]
}

func emu_gcall_error_missing(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8
    var r1 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 2) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) != 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 ||
       (p.Rr[1] & atm.ArgPointer) == 0 {
        panic("invalid error_type call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    a2 = p.Ar[2] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask
    r1 = p.Rr[1] & atm.ArgMask

    /* call the function */
    ret := error_missing(
        (*rt.GoType)(e.Pr[a0]),
        int(e.Gr[a1]),
        e.Gr[a2],
    )

    /* update the result register */
    e.Pr[r0] = (*[2]unsafe.Pointer)(unsafe.Pointer(&ret))[0]
    e.Pr[r1] = (*[2]unsafe.Pointer)(unsafe.Pointer(&ret))[1]
}

func init() {
    atm.RegisterGCall(error_eof, emu_gcall_error_eof)
    atm.RegisterGCall(error_skip, emu_gcall_error_skip)
    atm.RegisterGCall(error_type, emu_gcall_error_type)
    atm.RegisterGCall(error_missing, emu_gcall_error_missing)
}
