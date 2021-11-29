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

package atm

import (
    `testing`
    `unsafe`

    `github.com/davecgh/go-spew/spew`
)

var (
    testfn = RegisterGCall(testemu_pfunc, func(ctx CallContext) {
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

func runEmulator(init func(emu *Emulator), prog func(p *Builder)) *Emulator {
    pb := CreateBuilder()
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
    emu := runEmulator(nil, func(p *Builder) {
        p.IP(&a, P0)
        p.IP(&b, P1)
        p.IP(&c, P2)
        p.ADDPI(P0, 8, P3)
        p.LQ(P3, R0)
        p.LP(P0, P0)
        p.ADDPI(P1, 8, P3)
        p.LQ(P3, R1)
        p.LP(P1, P1)
        p.ADDPI(P2, 8, P3)
        p.LQ(P3, R2)
        p.LP(P2, P2)
        p.GCALL(testfn).A0(P0).A1(R0).A2(P1).A3(R1).A4(P2).A5(R2).R0(P0).R1(R0).R2(P1).R3(R1)
        p.STRP(P0, 0)
        p.STRQ(R0, 1)
        p.STRP(P1, 2)
        p.STRQ(R1, 3)
    })
    val := [2]struct{P unsafe.Pointer; L uint64} {
        {P: emu.Rp(0), L: emu.Ru(1)},
        {P: emu.Rp(2), L: emu.Ru(3)},
    }
    spew.Dump(*(*[2]string)(unsafe.Pointer(&val)))
}
