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
    return func(buf []byte, i int, p unsafe.Pointer, rs *RuntimeState, st int) (pos int, err error) {
        emu := atm.LoadProgram(prog)
        ret := (*rt.GoIface)(unsafe.Pointer(&err))
        src := (*rt.GoSlice)(unsafe.Pointer(&buf))
        emu.Ap(0, src.Ptr)
        emu.Au(1, uint64(src.Len))
        emu.Au(2, uint64(src.Cap))
        emu.Au(3, uint64(i))
        emu.Ap(4, p)
        emu.Ap(5, unsafe.Pointer(rs))
        emu.Au(6, uint64(st))
        emu.Run()
        pos = int(emu.Ru(0))
        ret.Itab = (*rt.GoItab)(emu.Rp(1))
        ret.Value = emu.Rp(2)
        emu.Free()
        return
    }
}

func emu_bytes(ctx atm.CallContext, i int) (v []byte) {
    (*rt.GoSlice)(unsafe.Pointer(&v)).Ptr = ctx.Ap(i)
    (*rt.GoSlice)(unsafe.Pointer(&v)).Len = int(ctx.Au(i + 1))
    (*rt.GoSlice)(unsafe.Pointer(&v)).Cap = int(ctx.Au(i + 2))
    return
}

func emu_gcall_decode(ctx atm.CallContext) {
    var ret int
    var err error

    /* check for arguments and return values */
    if !ctx.Verify("**iii**i", "i**") {
        panic("invalid decode call")
    }

    /* call the decoder */
    ret, err = decode(
        (*rt.GoType)(ctx.Ap(0)),
        emu_bytes(ctx, 1),
        int(ctx.Au(4)),
        ctx.Ap(5),
        (*RuntimeState)(ctx.Ap(6)),
        int(ctx.Au(7)),
    )

    /* pack the result */
    ctx.Ru(0, uint64(ret))
    emu_seterr(ctx, 1, err)
}