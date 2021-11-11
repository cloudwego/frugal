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

func emu_gcall_mapassign(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) == 0 ||
       (p.Ar[2] & atm.ArgPointer) == 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 {
        panic("invalid mapassign call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    a2 = p.Ar[2] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask

    /* call the function */
    e.Pr[r0] = mapassign(
        (*rt.GoMapType) (e.Pr[a0]),
        (*rt.GoMap)     (e.Pr[a1]),
        e.Pr[a2],
    )
}

func emu_gcall_mapassign_fast32(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) == 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 {
        panic("invalid mapassign_fast32 call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    a2 = p.Ar[2] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask

    /* call the function */
    e.Pr[r0] = mapassign_fast32(
        (*rt.GoMapType) (e.Pr[a0]),
        (*rt.GoMap)     (e.Pr[a1]),
        uint32          (e.Gr[a2]),
    )
}

func emu_gcall_mapassign_fast64(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) == 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 {
        panic("invalid mapassign_fast64 call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    a2 = p.Ar[2] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask

    /* call the function */
    e.Pr[r0] = mapassign_fast64(
        (*rt.GoMapType) (e.Pr[a0]),
        (*rt.GoMap)     (e.Pr[a1]),
        e.Gr[a2],
    )
}

func emu_gcall_mapassign_faststr(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var a3 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 4 || p.Rn != 1) ||
        (p.Ar[0] & atm.ArgPointer) == 0 ||
        (p.Ar[1] & atm.ArgPointer) == 0 ||
        (p.Ar[2] & atm.ArgPointer) == 0 ||
        (p.Ar[3] & atm.ArgPointer) != 0 ||
        (p.Rr[0] & atm.ArgPointer) == 0 {
        panic("invalid mapassign_faststr call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    a2 = p.Ar[2] & atm.ArgMask
    a3 = p.Ar[3] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask

    /* construct the key */
    key := rt.GoString {
        Ptr: e.Pr[a2],
        Len: int(e.Gr[a3]),
    }

    /* call the function */
    e.Pr[r0] = mapassign_faststr(
        (*rt.GoMapType) (e.Pr[a0]),
        (*rt.GoMap)     (e.Pr[a1]),
        *(*string)      (unsafe.Pointer(&key)),
    )
}

func emu_gcall_mapassign_fast64ptr(e *atm.Emulator, p *atm.Instr) {
    var a0 uint8
    var a1 uint8
    var a2 uint8
    var r0 uint8

    /* check for arguments and return values */
    if (p.An != 3 || p.Rn != 1) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) == 0 ||
       (p.Ar[2] & atm.ArgPointer) == 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 {
        panic("invalid mapassign_fast64ptr call")
    }

    /* extract the arguments and return value index */
    a0 = p.Ar[0] & atm.ArgMask
    a1 = p.Ar[1] & atm.ArgMask
    a2 = p.Ar[2] & atm.ArgMask
    r0 = p.Rr[0] & atm.ArgMask

    /* call the function */
    e.Pr[r0] = mapassign_fast64ptr(
        (*rt.GoMapType) (e.Pr[a0]),
        (*rt.GoMap)     (e.Pr[a1]),
        e.Pr[a2],
    )
}
