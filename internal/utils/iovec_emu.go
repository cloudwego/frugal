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

func emu_iovec(ctx atm.CallContext) (v iovec.IoVec) {
    (*rt.GoIface)(unsafe.Pointer(&v)).Itab = ctx.Itab()
    (*rt.GoIface)(unsafe.Pointer(&v)).Value = ctx.Data()
    return
}

func emu_bytes(ctx atm.CallContext, i int) (v []byte) {
    (*rt.GoSlice)(unsafe.Pointer(&v)).Ptr = ctx.Ap(i)
    (*rt.GoSlice)(unsafe.Pointer(&v)).Len = int(ctx.Au(i + 1))
    (*rt.GoSlice)(unsafe.Pointer(&v)).Cap = int(ctx.Au(i + 2))
    return
}

func emu_setbytes(ctx atm.CallContext, v []byte) {
    ctx.Rp(0, (*rt.GoSlice)(unsafe.Pointer(&v)).Ptr)
    ctx.Ru(1, uint64((*rt.GoSlice)(unsafe.Pointer(&v)).Len))
    ctx.Ru(2, uint64((*rt.GoSlice)(unsafe.Pointer(&v)).Cap))
}

func emu_icall_IoVecPut(ctx atm.CallContext) {
    if !ctx.Verify("*ii", "") {
        panic("invalid IoVecPut call")
    } else {
        emu_iovec(ctx).Put(emu_bytes(ctx, 0))
    }
}

func emu_icall_IoVecCat(ctx atm.CallContext) {
    if !ctx.Verify("*ii*ii", "") {
        panic("invalid IoVecCat call")
    } else {
        emu_iovec(ctx).Cat(emu_bytes(ctx, 0), emu_bytes(ctx, 3))
    }
}

func emu_icall_IoVecAdd(ctx atm.CallContext) {
    if !ctx.Verify("i*ii", "*ii") {
        panic("invalid IoVecAdd call")
    } else {
        emu_setbytes(ctx, emu_iovec(ctx).Add(int(ctx.Au(0)), emu_bytes(ctx, 1)))
    }
}
