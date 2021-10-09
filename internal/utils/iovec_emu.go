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

package utils

import (
    `unsafe`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
)

func emu_gcall_IoVecPut(e *atm.Emulator, p *atm.Instr) {
    var v0 rt.GoIface
    var v1 rt.GoSlice

    /* check for arguments */
    if (p.An != 5 || p.Rn != 0) ||
        (p.Ai[0] & atm.ArgPointer) == 0 ||
        (p.Ai[1] & atm.ArgPointer) == 0 ||
        (p.Ai[2] & atm.ArgPointer) == 0 ||
        (p.Ai[3] & atm.ArgPointer) != 0 ||
        (p.Ai[4] & atm.ArgPointer) != 0 {
        panic("invalid IoVecPut call")
    }

    /* extract the arguments */
    v0.Itab  = (*rt.GoItab)(e.Pr[p.Ai[0] & atm.ArgMask])
    v0.Value =              e.Pr[p.Ai[1] & atm.ArgMask]
    v1.Ptr   =              e.Pr[p.Ai[2] & atm.ArgMask]
    v1.Len   =          int(e.Gr[p.Ai[3] & atm.ArgMask])
    v1.Cap   =          int(e.Gr[p.Ai[4] & atm.ArgMask])

    /* call the function */
    IoVecPut(
        *(*frugal.IoVec)(unsafe.Pointer(&v0)),
        *(*[]byte)      (unsafe.Pointer(&v1)),
    )
}

func emu_gcall_IoVecCat(e *atm.Emulator, p *atm.Instr) {
    var v0 rt.GoIface
    var v1 rt.GoSlice
    var v2 rt.GoSlice

    /* check for arguments */
    if (p.An != 8 || p.Rn != 0) ||
        (p.Ai[0] & atm.ArgPointer) == 0 ||
        (p.Ai[1] & atm.ArgPointer) == 0 ||
        (p.Ai[2] & atm.ArgPointer) == 0 ||
        (p.Ai[3] & atm.ArgPointer) != 0 ||
        (p.Ai[4] & atm.ArgPointer) != 0 ||
        (p.Ai[5] & atm.ArgPointer) == 0 ||
        (p.Ai[6] & atm.ArgPointer) != 0 ||
        (p.Ai[7] & atm.ArgPointer) != 0 {
        panic("invalid IoVecCat call")
    }

    /* extract the arguments */
    v0.Itab  = (*rt.GoItab)(e.Pr[p.Ai[0] & atm.ArgMask])
    v0.Value =              e.Pr[p.Ai[1] & atm.ArgMask]
    v1.Ptr   =              e.Pr[p.Ai[2] & atm.ArgMask]
    v1.Len   =          int(e.Gr[p.Ai[3] & atm.ArgMask])
    v1.Cap   =          int(e.Gr[p.Ai[4] & atm.ArgMask])
    v2.Ptr   =              e.Pr[p.Ai[5] & atm.ArgMask]
    v2.Len   =          int(e.Gr[p.Ai[6] & atm.ArgMask])
    v2.Cap   =          int(e.Gr[p.Ai[7] & atm.ArgMask])

    /* call the function */
    IoVecCat(
        *(*frugal.IoVec)(unsafe.Pointer(&v0)),
        *(*[]byte)      (unsafe.Pointer(&v1)),
        *(*[]byte)      (unsafe.Pointer(&v2)),
    )
}

func emu_gcall_IoVecAdd(e *atm.Emulator, p *atm.Instr) {
    var v1 int
    var v0 rt.GoIface
    var v2 rt.GoSlice

    /* check for arguments */
    if (p.An != 6 || p.Rn != 3) ||
        (p.Ai[0] & atm.ArgPointer) == 0 ||
        (p.Ai[1] & atm.ArgPointer) == 0 ||
        (p.Ai[2] & atm.ArgPointer) != 0 ||
        (p.Ai[3] & atm.ArgPointer) == 0 ||
        (p.Ai[4] & atm.ArgPointer) != 0 ||
        (p.Ai[5] & atm.ArgPointer) != 0 ||
        (p.Rv[0] & atm.ArgPointer) == 0 ||
        (p.Rv[1] & atm.ArgPointer) != 0 ||
        (p.Rv[2] & atm.ArgPointer) != 0 {
        panic("invalid IoVecAdd call")
    }

    /* extract the arguments */
    v0.Itab  = (*rt.GoItab)(e.Pr[p.Ai[0] & atm.ArgMask])
    v0.Value =              e.Pr[p.Ai[1] & atm.ArgMask]
    v1       =          int(e.Gr[p.Ai[2] & atm.ArgMask])
    v2.Ptr   =              e.Pr[p.Ai[3] & atm.ArgMask]
    v2.Len   =          int(e.Gr[p.Ai[4] & atm.ArgMask])
    v2.Cap   =          int(e.Gr[p.Ai[5] & atm.ArgMask])

    /* call the function */
    ret := IoVecAdd(
        *(*frugal.IoVec)(unsafe.Pointer(&v0)),
        v1,
        *(*[]byte)(unsafe.Pointer(&v2)),
    )

    /* set the return value */
    e.Gr[p.Rv[2] & atm.ArgMask] = uint64(cap(ret))
    e.Gr[p.Rv[1] & atm.ArgMask] = uint64(len(ret))
    e.Pr[p.Rv[0] & atm.ArgMask] = *(*unsafe.Pointer)(unsafe.Pointer(&ret))
}

func init() {
    atm.RegisterGCall(IoVecPut, emu_gcall_IoVecPut)
    atm.RegisterGCall(IoVecCat, emu_gcall_IoVecCat)
    atm.RegisterGCall(IoVecAdd, emu_gcall_IoVecAdd)
}
