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

func link_emu(prog atm.Program) Decoder {
    return func(buf unsafe.Pointer, nb int, i int, p unsafe.Pointer, rs *RuntimeState, st int) (pos int, err error) {
        emu := atm.LoadProgram(prog)
        ret := (*rt.GoIface)(unsafe.Pointer(&err))
        emu.Ap(0, buf)
        emu.Au(1, uint64(nb))
        emu.Au(2, uint64(i))
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

func emu_decode(ctx atm.CallContext) (int, error) {
    return decode(
        (*rt.GoType)(ctx.Ap(0)),
        ctx.Ap(1),
        int(ctx.Au(2)),
        int(ctx.Au(3)),
        ctx.Ap(4),
        (*RuntimeState)(ctx.Ap(5)),
        int(ctx.Au(6)),
    )
}

func emu_mkreturn(ctx atm.CallContext) func(int, error) {
    return func(ret int, err error) {
        ctx.Ru(0, uint64(ret))
        emu_seterr(ctx, 1, err)
    }
}

func emu_gcall_decode(ctx atm.CallContext) {
    if !ctx.Verify("**ii**i", "i**") {
        panic("invalid decode call")
    } else {
        emu_mkreturn(ctx)(emu_decode(ctx))
    }
}