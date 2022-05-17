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

package pgen

import (
    `fmt`
    `runtime`
    `strings`
    `testing`
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/cloudwego/frugal/internal/atm/rtx`
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
    fp := runtime.FuncForPC(uintptr(addr))
    if fp != nil {
        ent := uint64(fp.Entry())
        if addr == ent {
            return fmt.Sprintf("%#x{%s}", addr, fp.Name()), ent
        }
        return fmt.Sprintf("%#x{%s+%#x}", addr, fp.Name(), addr - ent), ent
    }
    if addr == uint64(uintptr(rtx.V_pWriteBarrier)) {
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
    hfunc *hir.CallHandle
    hmeth *hir.CallHandle
    cfunc uintptr
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

func init() {
    hfunc = hir.RegisterCCall(unsafe.Pointer(&cfunc), nil)
    hmeth = hir.RegisterICall(rt.GetMethod((*TestIface)(nil), "Foo"), nil)
}

func testemu_pfunc(a string, b string, c string) (d string, e string) {
    d = a + b
    e = b + c
    return
}

func TestPGen_Generate(t *testing.T) {
    p := hir.CreateBuilder()
    p.IQ(0, hir.R0)
    p.IQ(1, hir.R1)
    p.IQ(2, hir.R2)
    p.MOVP(hir.Pn, hir.P0)
    p.MOVP(hir.Pn, hir.P1)
    p.MOVP(hir.Pn, hir.P2)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.CCALL(hfunc).A0(hir.R0).A1(hir.R1).A2(hir.R2).R0(hir.R0)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.GCALL(testfn).A0(hir.P0).A1(hir.R0).A2(hir.P1).A3(hir.R1).A4(hir.P2).A5(hir.R2).R0(hir.P0).R1(hir.R0).R2(hir.P1).R3(hir.R1)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.ICALL(hir.P0, hir.P1, hmeth).A0(hir.R0).A1(hir.R1).R0(hir.R2)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BCOPY(hir.P1, hir.R1, hir.P0)
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.BREAK()
    p.RET()
    g := CreateCodeGen(func(){})
    c := g.Generate(p.Build(), 0)
    disasm(0, c.Code)
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

func mkccalltestfn() unsafe.Pointer {
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
    return *(*unsafe.Pointer)(p)
}

func TestPGen_FunctionCall(t *testing.T) {
    var s ifacetest
    var i ifacetesttype = 123456
    s = i
    m := hir.RegisterICall(rt.GetMethod((*ifacetest)(nil), "Foo"), nil)
    c := hir.RegisterCCall(mkccalltestfn(), nil)
    h := hir.RegisterGCall(gcalltestfn, nil)
    p := hir.CreateBuilder()
    e := *(*rt.GoIface)(unsafe.Pointer(&s))
    p.IP(e.Itab, hir.P0)
    p.IP(e.Value, hir.P1)
    p.LDAQ(0, hir.R0)
    p.GCALL(h).A0(hir.R0).R0(hir.R1).R1(hir.R2).R2(hir.R3)
    p.ADD(hir.R1, hir.R2, hir.R1)
    p.ADD(hir.R2, hir.R3, hir.R2)
    p.ICALL(hir.P0, hir.P1, m).A0(hir.R3).R0(hir.R4)
    p.ADDI(hir.R4, 10000000, hir.R3)
    p.CCALL(c).A0(hir.R3).R0(hir.R4)
    p.RET().R0(hir.R1).R1(hir.R2).R2(hir.R4)
    g := CreateCodeGen((func(int) (int, int, int))(nil))
    r := g.Generate(p.Build(), 0)
    spew.Dump(r.Frame)
    v := loader.Loader(r.Code).Load("_test_gcall", r.Frame)
    disasm(*(*uintptr)(v), r.Code)
    f := *(*func(int) (int, int, int))(unsafe.Pointer(&v))
    x, y, z := f(123)
    println("f(123) is", x, y, z)
    require.Equal(t, 546, x)
    require.Equal(t, 746, y)
    require.Equal(t, 20211206, z)
}
