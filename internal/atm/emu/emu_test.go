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

    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/davecgh/go-spew/spew`
)

var (
    testfn = hir.RegisterGCall(testemu_pfunc, func(ctx hir.CallContext) {
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

func runEmulator(init func(emu *Emulator), prog func(p *hir.Builder)) *Emulator {
    pb := hir.CreateBuilder()
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
    emu := runEmulator(nil, func(p *hir.Builder) {
        p.IP(&a, hir.P0)
        p.IP(&b, hir.P1)
        p.IP(&c, hir.P2)
        p.LQ(hir.P0, 8, hir.R0)
        p.LP(hir.P0, 0, hir.P0)
        p.LQ(hir.P1, 8, hir.R1)
        p.LP(hir.P1, 0, hir.P1)
        p.LQ(hir.P2, 8, hir.R2)
        p.LP(hir.P2, 0, hir.P2)
        p.GCALL(testfn).A0(hir.P0).A1(hir.R0).A2(hir.P1).A3(hir.R1).A4(hir.P2).A5(hir.R2).R0(hir.P0).R1(hir.R0).R2(hir.P1).R3(hir.R1)
        p.STRP(hir.P0, 0)
        p.STRQ(hir.R0, 1)
        p.STRP(hir.P1, 2)
        p.STRQ(hir.R1, 3)
    })
    val := [2]struct{P unsafe.Pointer; L uint64} {
        {P: emu.Rp(0), L: emu.Ru(1)},
        {P: emu.Rp(2), L: emu.Ru(3)},
    }
    spew.Dump(*(*[2]string)(unsafe.Pointer(&val)))
}
