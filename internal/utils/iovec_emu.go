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

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/cloudwego/frugal/iovec`
)

func emu_icall_IoVecPut(e *atm.Emulator, p *atm.Instr) {
    var v0 rt.GoIface
    var v1 rt.GoSlice

    /* check for arguments */
    if (p.An != 3 || p.Rn != 0) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) != 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 {
        panic("invalid IoVecPut call")
    }

    /* extract the argument */
    v1.Ptr =     e.Pr[p.Ar[0] & atm.ArgMask]
    v1.Len = int(e.Gr[p.Ar[1] & atm.ArgMask])
    v1.Cap = int(e.Gr[p.Ar[2] & atm.ArgMask])

    /* call the function */
    v0.Itab, v0.Value = (*rt.GoItab)(e.Pr[p.Ps]), e.Pr[p.Pd]
    (*(*iovec.IoVec)(unsafe.Pointer(&v0))).Put(*(*[]byte)(unsafe.Pointer(&v1)))
}

func emu_icall_IoVecCat(e *atm.Emulator, p *atm.Instr) {
    var v0 rt.GoIface
    var v1 rt.GoSlice
    var v2 rt.GoSlice

    /* check for arguments */
    if (p.An != 6 || p.Rn != 0) ||
       (p.Ar[0] & atm.ArgPointer) == 0 ||
       (p.Ar[1] & atm.ArgPointer) != 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 ||
       (p.Ar[3] & atm.ArgPointer) == 0 ||
       (p.Ar[4] & atm.ArgPointer) != 0 ||
       (p.Ar[5] & atm.ArgPointer) != 0 {
        panic("invalid IoVecCat call")
    }

    /* extract the arguments */
    v1.Ptr =     e.Pr[p.Ar[0] & atm.ArgMask]
    v1.Len = int(e.Gr[p.Ar[1] & atm.ArgMask])
    v1.Cap = int(e.Gr[p.Ar[2] & atm.ArgMask])
    v2.Ptr =     e.Pr[p.Ar[3] & atm.ArgMask]
    v2.Len = int(e.Gr[p.Ar[4] & atm.ArgMask])
    v2.Cap = int(e.Gr[p.Ar[5] & atm.ArgMask])

    /* call the function */
    v0.Itab, v0.Value = (*rt.GoItab)(e.Pr[p.Ps]), e.Pr[p.Pd]
    (*(*iovec.IoVec)(unsafe.Pointer(&v0))).Cat(*(*[]byte)(unsafe.Pointer(&v1)), *(*[]byte)(unsafe.Pointer(&v2)))
}

func emu_icall_IoVecAdd(e *atm.Emulator, p *atm.Instr) {
    var v1 int
    var v0 rt.GoIface
    var v2 rt.GoSlice

    /* check for arguments */
    if (p.An != 4 || p.Rn != 3) ||
       (p.Ar[0] & atm.ArgPointer) != 0 ||
       (p.Ar[1] & atm.ArgPointer) == 0 ||
       (p.Ar[2] & atm.ArgPointer) != 0 ||
       (p.Ar[3] & atm.ArgPointer) != 0 ||
       (p.Rr[0] & atm.ArgPointer) == 0 ||
       (p.Rr[1] & atm.ArgPointer) != 0 ||
       (p.Rr[2] & atm.ArgPointer) != 0 {
        panic("invalid IoVecAdd call")
    }

    /* extract the arguments */
    v1     = int(e.Gr[p.Ar[0] & atm.ArgMask])
    v2.Ptr =     e.Pr[p.Ar[1] & atm.ArgMask]
    v2.Len = int(e.Gr[p.Ar[2] & atm.ArgMask])
    v2.Cap = int(e.Gr[p.Ar[3] & atm.ArgMask])

    /* call the function */
    v0.Itab, v0.Value = (*rt.GoItab)(e.Pr[p.Ps]), e.Pr[p.Pd]
    ret := (*(*iovec.IoVec)(unsafe.Pointer(&v0))).Add(v1, *(*[]byte)(unsafe.Pointer(&v2)))

    /* set the return value */
    e.Gr[p.Rr[2] & atm.ArgMask] = uint64(cap(ret))
    e.Gr[p.Rr[1] & atm.ArgMask] = uint64(len(ret))
    e.Pr[p.Rr[0] & atm.ArgMask] = *(*unsafe.Pointer)(unsafe.Pointer(&ret))
}

func init() {
    atm.RegisterICall(IoVecPut, emu_icall_IoVecPut)
    atm.RegisterICall(IoVecCat, emu_icall_IoVecCat)
    atm.RegisterICall(IoVecAdd, emu_icall_IoVecAdd)
}
