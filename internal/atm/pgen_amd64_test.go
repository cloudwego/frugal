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
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/loader`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
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

type TestIface interface {
    Bar(x int, y int) int
    Foo(x int, y int) int
}

var (
    hfunc CallHandle
    hmeth CallHandle
    cfunc uintptr
)

func init() {
    cfunc = uintptr(unsafe.Pointer(&cfunc))
    hfunc = RegisterCCall(cfunc, nil)
    hmeth = RegisterICall(rt.GetMethod((*TestIface)(nil), "Foo"), nil)
}

func TestPGen_Generate(t *testing.T) {
    p := CreateBuilder()
    p.IQ(0, R0)
    p.IQ(1, R1)
    p.IQ(2, R2)
    p.MOVP(Pn, P0)
    p.MOVP(Pn, P1)
    p.MOVP(Pn, P2)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.CCALL(hfunc).A0(R0).A1(R1).A2(R2).R0(R0)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.GCALL(testfn).A0(P0).A1(R0).A2(P1).A3(R1).A4(P2).A5(R2).R0(P0).R1(R0).R2(P1).R3(R1)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.ICALL(P0, P1, hmeth).A0(R0).A1(R1).R0(R2)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BCOPY(P1, R1, P0)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.HALT()
    g := CreateCodeGen(func(){})
    c := g.Generate(p.Build(), 0)
    disasm(0, c)
}

type ifacetest interface {
    Foo(int) int
}

type ifacetesttype int
func (self ifacetesttype) Foo(v int) int {
    runtime.GC()
    println("iface Foo(), self is", self, ", v is", v)
    return int(self) + v
}

func gcalltestfn(a int) (int, int, int) {
    runtime.GC()
    println("a is", a)
    return a + 100, a + 200, a + 300
}

func mkccalltestfn() uintptr {
    var asm x86_64.Assembler
    err := asm.Assemble(`
        movq    %rdi, %rax
        addq    $10087327, %rax
        ret
    `)
    if err != nil {
        panic(err)
    }
    p := loader.Loader(asm.Code()).Load("_ccalltestfn", rt.Frame{})
    return *(*uintptr)(p)
}

func TestPGen_FunctionCall(t *testing.T) {
    var s ifacetest
    var i ifacetesttype = 123456
    s = i
    m := RegisterICall(rt.GetMethod((*ifacetest)(nil), "Foo"), nil)
    c := RegisterCCall(mkccalltestfn(), nil)
    h := RegisterGCall(gcalltestfn, nil)
    p := CreateBuilder()
    e := *(*rt.GoIface)(unsafe.Pointer(&s))
    p.IP(e.Itab, P0)
    p.IP(e.Value, P1)
    p.LDAQ(0, R0)
    p.GCALL(h).A0(R0).R0(R1).R1(R2).R2(R3)
    p.ADD(R1, R2, R1)
    p.ADD(R2, R3, R2)
    p.ICALL(P0, P1, m).A0(R3).R0(R4)
    p.ADDI(R4, 10000000, R3)
    p.CCALL(c).A0(R3).R0(R4)
    p.STRQ(R1, 0)
    p.STRQ(R2, 1)
    p.STRQ(R4, 2)
    p.HALT()
    g := CreateCodeGen((func(int) (int, int, int))(nil))
    r := g.Generate(p.Build(), 0)
    spew.Dump(g.Frame())
    v := loader.Loader(r).Load("_test_gcall", g.Frame())
    disasm(*(*uintptr)(v), r)
    f := *(*func(int) (int, int, int))(unsafe.Pointer(&v))
    x, y, z := f(123)
    println("f(123) is", x, y, z)
    require.Equal(t, 546, x)
    require.Equal(t, 746, y)
    require.Equal(t, 20211206, z)
}
