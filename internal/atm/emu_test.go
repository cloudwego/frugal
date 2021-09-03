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

    `github.com/cloudwego/frugal/internal/rt`
    `github.com/davecgh/go-spew/spew`
)

func init() {
    gcallTab[rt.FuncAddr(testemu_pfunc)] = func(e *Emulator, p *Instr) {
        var v0 struct {P unsafe.Pointer; L uint64}
        var v1 struct {P unsafe.Pointer; L uint64}
        var v2 struct {P unsafe.Pointer; L uint64}
        if (p.An != 6 || p.Rn != 4) ||
           (p.Av[0] & ArgPointer) == 0 ||
           (p.Av[1] & ArgPointer) != 0 ||
           (p.Av[2] & ArgPointer) == 0 ||
           (p.Av[3] & ArgPointer) != 0 ||
           (p.Av[4] & ArgPointer) == 0 ||
           (p.Av[5] & ArgPointer) != 0 ||
           (p.Rv[0] & ArgPointer) == 0 ||
           (p.Rv[1] & ArgPointer) != 0 ||
           (p.Rv[2] & ArgPointer) == 0 ||
           (p.Rv[3] & ArgPointer) != 0 {
            panic("invalid testemu_pfunc call")
        }
        v0.P = e.Pr[p.Av[0] & ArgMask]
        v0.L = e.Gr[p.Av[1] & ArgMask]
        v1.P = e.Pr[p.Av[2] & ArgMask]
        v1.L = e.Gr[p.Av[3] & ArgMask]
        v2.P = e.Pr[p.Av[4] & ArgMask]
        v2.L = e.Gr[p.Av[5] & ArgMask]
        r0, r1 := testemu_pfunc(
            *(*string)(unsafe.Pointer(&v0)),
            *(*string)(unsafe.Pointer(&v1)),
            *(*string)(unsafe.Pointer(&v2)),
        )
        e.Gr[p.Rv[1] & ArgMask] = uint64(len(r0))
        e.Gr[p.Rv[3] & ArgMask] = uint64(len(r1))
        e.Pr[p.Rv[0] & ArgMask] = *(*unsafe.Pointer)(unsafe.Pointer(&r0))
        e.Pr[p.Rv[2] & ArgMask] = *(*unsafe.Pointer)(unsafe.Pointer(&r1))
    }
}

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
        p.GCALL(testemu_pfunc).A0(0, P0).A1(0, R0).A2(1, P1).A3(1, R1).A4(2, P2).A5(2, R2).R0(0, P0).R1(0, R0).R2(1, P1).R3(1, R1)
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
