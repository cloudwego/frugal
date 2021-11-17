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

func symlookup(pc uint64) (string, uint64) {
    fn := runtime.FuncForPC(uintptr(pc))
    if fn == nil {
        return "", 0
    }
    return fn.Name(), uint64(fn.Entry())
}

func disasm(c []byte) {
    var pc int
    for pc < len(c) {
        i, err := x86asm.Decode(c[pc:], 64)
        if err != nil {
            panic(err)
        }
        dis := x86asm.GNUSyntax(i, uint64(pc), symlookup)
        fmt.Printf("0x%08x : ", pc)
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
    b := CreateBuilder()
    f := RegisterGCall(newBuilder, nil)
    b.IQ(0x1234, R0)
    b.ADDI(R0, 0x5678abcde, R1)
    b.GCALL(f).A0(R0).A1(R1).R0(R2).R1(R3)
    b.ADDI(R2, 0xaa, R3)
    b.HALT()
    g := CreateCodeGen()
    p := g.Generate(b.Build())
    c := p.Assemble(0)
    disasm(c)
}
