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

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/cloudwego/frugal/iovec`
)

func link_emu(prog atm.Program) Encoder {
    return func(iov iovec.IoVec, p unsafe.Pointer, rs *RuntimeState, st int) (err error) {
        emu := atm.LoadProgram(prog)
        ret := (*rt.GoIface)(unsafe.Pointer(&err))
        iop := (*rt.GoIface)(unsafe.Pointer(&iov))
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

func emu_iovec(ctx atm.CallContext, i int) (v iovec.IoVec) {
    (*rt.GoIface)(unsafe.Pointer(&v)).Itab = (*rt.GoItab)(ctx.Ap(i))
    (*rt.GoIface)(unsafe.Pointer(&v)).Value = ctx.Ap(i + 1)
    return
}

func emu_seterr(ctx atm.CallContext, err error) {
    vv := (*rt.GoIface)(unsafe.Pointer(&err))
    ctx.Rp(0, unsafe.Pointer(vv.Itab))
    ctx.Rp(1, vv.Value)
}

func emu_gcall_encode(ctx atm.CallContext) {
    if !ctx.Verify("*****i", "**") {
        panic("invalid encode call")
    } else {
        emu_seterr(ctx, encode((*rt.GoType)(ctx.Ap(0)), emu_iovec(ctx, 1), ctx.Ap(3), (*RuntimeState)(ctx.Ap(4)), int(ctx.Au(5))))
    }
}
