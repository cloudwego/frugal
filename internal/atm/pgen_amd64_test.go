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
    `reflect`
    `runtime`
    `strings`
    `testing`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
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
    cfunc unsafe.Pointer
)

func init() {
    cfunc = unsafe.Pointer(&cfunc)
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
    p.HALT()
    g := CreateCodeGen(stackmaptestfunc)
    m := g.Generate(p.Build())
    c := m.Assemble(0)
    disasm(0, c)
    args, locals := g.StackMap()
    println("--- args ---")
    dumpstackmap(args)
    println("--- stack ---")
    dumpstackmap(locals)
}

const (
    _FUNCDATA_ArgsPointerMaps = 0
    _FUNCDATA_LocalsPointerMaps = 1
)

type funcInfo struct {
    fn    unsafe.Pointer
    datap unsafe.Pointer
}

type bitvector struct {
    n        int32 // # of bits
    bytedata *uint8
}

//go:nosplit
func addb(p *byte, n uintptr) *byte {
    return (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + n))
}

func (bv *bitvector) ptrbit(i uintptr) uint8 {
    b := *(addb(bv.bytedata, i/8))
    return (b >> (i % 8)) & 1
}

//go:linkname findfunc runtime.findfunc
func findfunc(_ uintptr) funcInfo

//go:linkname funcdata runtime.funcdata
func funcdata(_ funcInfo, _ uint8) unsafe.Pointer

//go:linkname stackmapdata runtime.stackmapdata
func stackmapdata(_ *StackMap, _ int32) bitvector

func stackMap(f interface{}) (*StackMap, *StackMap) {
    fv := reflect.ValueOf(f)
    if fv.Kind() != reflect.Func {
        panic("f must be reflect.Func kind!")
    }
    fi := findfunc(fv.Pointer())
    args := funcdata(fi, uint8(_FUNCDATA_ArgsPointerMaps))
    locals := funcdata(fi, uint8(_FUNCDATA_LocalsPointerMaps))
    return (*StackMap)(args), (*StackMap)(locals)
}

func dumpstackmap(m *StackMap) {
    for i := int32(0); i < m.N; i++ {
        fmt.Printf("bitmap #%d/%d: ", i, m.L)
        bv := stackmapdata(m, i)
        for j := int32(0); j < bv.n; j++ {
            fmt.Printf("%d ", bv.ptrbit(uintptr(j)))
        }
        fmt.Printf("\n")
    }
}

var keepalive struct {
    s  string
    i  int
    vp unsafe.Pointer
    sb interface{}
    fv uint64
}

func stackmaptestfunc(s string, i int, vp unsafe.Pointer, sb interface{}, fv uint64) (x *uint64, y string, z *int) {
    z = new(int)
    x = new(uint64)
    y = s + "asdf"
    keepalive.s = s
    keepalive.i = i
    keepalive.vp = vp
    keepalive.sb = sb
    keepalive.fv = fv
    return
}

func TestPGen_StackMap(t *testing.T) {
    args, locals := stackMap(stackmaptestfunc)
    println("--- args ---")
    dumpstackmap(args)
    println("--- locals ---")
    dumpstackmap(locals)
}
