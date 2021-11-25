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
    `fmt`
    `runtime`
    `strings`
    `testing`

    `golang.org/x/arch/x86/x86asm`
)

const (
    _MaxByte = 10
)

func symlookup(addr uint64) (string, uint64) {
    fn := runtime.FuncForPC(uintptr(addr))
    if fn != nil {
        ent := uint64(fn.Entry())
        if addr == ent {
            return fmt.Sprintf("%#x{%s}", addr, fn.Name()), ent
        }
        return fmt.Sprintf("%#x{%s+%#x}", addr, fn.Name(), addr - ent), ent
    }
    if addr == uint64(V_pWriteBarrier) {
        return fmt.Sprintf("%#x{runtime.writeBarrier}", addr), addr
    }
    return "", 0
}

func disasm(orig uintptr, c []byte) {
    var pc int
    for pc < len(c) {
        i, err := x86asm.Decode(c[pc:], 64)
        if err != nil {
            panic(err)
        }
        dis := x86asm.GNUSyntax(i, uint64(pc) + uint64(orig), symlookup)
        fmt.Printf("0x%08x : ", pc + int(orig))
        for x := 0; x < i.Len; x++ {
            if x != 0 && x % _MaxByte == 0 {
                fmt.Printf("\n           : ")
            }
            fmt.Printf(" %02x", c[pc + x])
            if x == _MaxByte - 1 {
                fmt.Printf("    %s", dis)
            }
        }
        if i.Len < _MaxByte {
            fmt.Printf("%s    %s", strings.Repeat(" ", (_MaxByte - i.Len) * 3), dis)
        }
        fmt.Printf("\n")
        pc += i.Len
    }
}

func TestPGen_Generate(t *testing.T) {
    p := CreateBuilder()
    p.LDAP(0, P0)
    p.LDAP(1, P1)
    p.LDAP(2, P2)
    p.LQ(P2, R0)
    p.LQ(P2, R1)
    // p.ADDPI(P0, 8, P3)
    // p.LQ(P3, R0)
    // p.LP(P0, P0)
    // p.ADDPI(P1, 8, P3)
    // p.LQ(P3, R1)
    // p.LP(P1, P1)
    // p.ADDPI(P2, 8, P3)
    // p.LQ(P3, R2)
    // p.LP(P2, P2)
    // p.GCALL(testfn).A0(P0).A1(R0).A2(P1).A3(R1).A4(P2).A5(R2).R0(P0).R1(R0).R2(P1).R3(R1)
    p.STRP(P0, 0)
    p.STRQ(R0, 1)
    p.STRP(P1, 2)
    p.STRQ(R1, 3)
    p.HALT()
    g := CreateCodeGen((func(*string, *string, *string) (string, string))(nil))
    m := g.Generate(p.Build())
    c := m.Assemble(0)
    disasm(0, c)
}
