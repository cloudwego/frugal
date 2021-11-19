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
    i := new(*int)
    j := 12345
    b := CreateBuilder()
    b.IQ(0x1234, R0)
    // b.ADDI(R0, 1, R1)
    // b.ADDI(R0, 2, R2)
    // b.ADDI(R0, 3, R3)
    // b.ADDI(R0, 4, R4)
    // b.ADDI(R0, 5, R5)
    // b.ADDI(R0, 6, R6)
    b.IP(i, P0)
    b.IP(&j, P1)
    b.SP(P0, P1)
    b.HALT()
    g := CreateCodeGen()
    p := g.Generate(b.Build())
    c := p.Assemble(0)
    disasm(0, c)
}
