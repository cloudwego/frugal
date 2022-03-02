/*
 * Copyright 2022 ByteDance Inc.
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

package emu

import (
    `testing`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/ir`
    `github.com/davecgh/go-spew/spew`
)

var (
    testfn = ir.RegisterGCall(testemu_pfunc, func(ctx ir.CallContext) {
        var v0 struct {P unsafe.Pointer; L uint64}
        var v1 struct {P unsafe.Pointer; L uint64}
        var v2 struct {P unsafe.Pointer; L uint64}
        if !ctx.Verify("*i*i*i", "*i*i") {
            panic("invalid testemu_pfunc call")
        }
        v0.P = ctx.Ap(0)
        v0.L = ctx.Au(1)
        v1.P = ctx.Ap(2)
        v1.L = ctx.Au(3)
        v2.P = ctx.Ap(4)
        v2.L = ctx.Au(5)
        r0, r1 := testemu_pfunc(
            *(*string)(unsafe.Pointer(&v0)),
            *(*string)(unsafe.Pointer(&v1)),
            *(*string)(unsafe.Pointer(&v2)),
        )
        ctx.Ru(1, uint64(len(r0)))
        ctx.Ru(3, uint64(len(r1)))
        ctx.Rp(0, *(*unsafe.Pointer)(unsafe.Pointer(&r0)))
        ctx.Rp(2, *(*unsafe.Pointer)(unsafe.Pointer(&r1)))
    })
)

func testemu_pfunc(a string, b string, c string) (d string, e string) {
    d = a + b
    e = b + c
    return
}

func runEmulator(init func(emu *Emulator), prog func(p *ir.Builder)) *Emulator {
    pb := ir.CreateBuilder()
    prog(pb)
    emu := LoadProgram(pb.Build())
    if init != nil {
        init(emu)
    }
    emu.Run()
    return emu
}

func TestEmu_OpCode_GCALL(t *testing.T) {
    a := "aaa"
    b := "bbb"
    c := "ccc"
    emu := runEmulator(nil, func(p *ir.Builder) {
        p.IP(&a, ir.P0)
        p.IP(&b, ir.P1)
        p.IP(&c, ir.P2)
        p.LQ(ir.P0, 8, ir.R0)
        p.LP(ir.P0, 0, ir.P0)
        p.LQ(ir.P1, 8, ir.R1)
        p.LP(ir.P1, 0, ir.P1)
        p.LQ(ir.P2, 8, ir.R2)
        p.LP(ir.P2, 0, ir.P2)
        p.GCALL(testfn).A0(ir.P0).A1(ir.R0).A2(ir.P1).A3(ir.R1).A4(ir.P2).A5(ir.R2).R0(ir.P0).R1(ir.R0).R2(ir.P1).R3(ir.R1)
        p.STRP(ir.P0, 0)
        p.STRQ(ir.R0, 1)
        p.STRP(ir.P1, 2)
        p.STRQ(ir.R1, 3)
    })
    val := [2]struct{P unsafe.Pointer; L uint64} {
        {P: emu.Rp(0), L: emu.Ru(1)},
        {P: emu.Rp(2), L: emu.Ru(3)},
    }
    spew.Dump(*(*[2]string)(unsafe.Pointer(&val)))
}
