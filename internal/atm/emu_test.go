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

func testemu_pfunc(a string, b string, c string) (d string, e string) {
    d = a + b
    e = b + c
    return
}

func TestEmu_GCALL_Pointer(t *testing.T) {
    fn := testemu_pfunc
    buf := [5]string{"1", "2", "3"}
    rt.ReflectCall(nil, *(*unsafe.Pointer)(unsafe.Pointer(&fn)), unsafe.Pointer(&buf), 80, 48)
    spew.Dump(buf)
}

func runEmulator(init func(emu *Emulator), prog func(p *ProgramBuilder)) *Emulator {
    pb := CreateProgramBuilder()
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
    emu := runEmulator(nil, func(p *ProgramBuilder) {
        p.IB(8, R4)
        p.IP(&a, P0)
        p.IP(&b, P1)
        p.IP(&c, P2)
        p.ADDP(P0, R4, P3)
        p.LQ(P3, R0)
        p.LP(P0, P0)
        p.ADDP(P1, R4, P3)
        p.LQ(P3, R1)
        p.LP(P1, P1)
        p.ADDP(P2, R4, P3)
        p.LQ(P3, R2)
        p.LP(P2, P2)
        p.GCALL(testemu_pfunc).A0(P0).A1(R0).A2(P1).A3(R1).A4(P2).A5(R2).R0(P0).R1(R0).R2(P1).R3(R1)
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
