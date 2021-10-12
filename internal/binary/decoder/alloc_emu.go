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

func emu_gcall_makemap(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ai[0] & atm.ArgPointer) == 0 ||
       (p.Ai[1] & atm.ArgPointer) != 0 ||
       (p.Ai[2] & atm.ArgPointer) == 0 ||
       (p.Rv[0] & atm.ArgPointer) == 0 {
        panic("invalid makemap call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ai[0] & atm.ArgMask
    a1 = p.Ai[1] & atm.ArgMask
    a2 = p.Ai[2] & atm.ArgMask
    r0 = p.Rv[0] & atm.ArgMask

    /* call the function */
    e.Pr[r0] = unsafe.Pointer(makemap(
        (*rt.GoMapType) (e.Pr[a0]),
        int             (e.Gr[a1]),
        (*rt.GoMap)     (e.Pr[a2]),
    ))
}

func emu_gcall_mallocgc(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ai[0] & atm.ArgPointer) != 0 ||
       (p.Ai[1] & atm.ArgPointer) == 0 ||
       (p.Ai[2] & atm.ArgPointer) != 0 ||
       (p.Rv[0] & atm.ArgPointer) == 0 {
        panic("invalid mallocgc call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ai[0] & atm.ArgMask
    a1 = p.Ai[1] & atm.ArgMask
    a2 = p.Ai[2] & atm.ArgMask
    r0 = p.Rv[0] & atm.ArgMask

    /* call the function */
    e.Pr[r0] = mallocgc(
        uintptr      (e.Gr[a0]),
        (*rt.GoType) (e.Pr[a1]),
        e.Gr[a2] != 0,
    )
}

func init() {
    atm.RegisterGCall(makemap, emu_gcall_makemap)
    atm.RegisterGCall(mallocgc, emu_gcall_mallocgc)
}
