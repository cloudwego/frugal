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

package loader

import (
    `fmt`
    `reflect`
    `runtime`
    `testing`
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/stretchr/testify/assert`
    `github.com/stretchr/testify/require`
    `golang.org/x/arch/x86/x86asm`
)

type funcInfo struct {
    *_Func
    datap *_ModuleData
}

//go:linkname findfunc runtime.findfunc
//goland:noinspection GoUnusedParameter
func findfunc(pc uintptr) funcInfo

//go:linkname pcdatavalue2 runtime.pcdatavalue2
//goland:noinspection GoUnusedParameter
func pcdatavalue2(f funcInfo, table uint32, targetpc uintptr) (int32, uintptr)

func TestLoader_Load(t *testing.T) {
    var src string
    var asm x86_64.Assembler
    if runtime.Version() < "go1.17" { src += `
        movq 8(%rsp), %rax`
    }
    src += `
        movq $1234, (%rax)
        ret`
    require.NoError(t, asm.Assemble(src))
    v0 := 0
    cc := asm.Code()
    fp := Loader(cc).Load("test", rt.Frame{})
    (*(*func(*int))(unsafe.Pointer(&fp)))(&v0)
    pc := *(*uintptr)(fp)
    assert.Equal(t, 1234, v0)
    assert.Equal(t, fmt.Sprintf("(frugal).test_%x", pc), runtime.FuncForPC(pc).Name())
    file, line := runtime.FuncForPC(pc).FileLine(pc + 1)
    assert.Equal(t, "(jit-generated)", file)
    assert.Equal(t, 1, line)
    smi, startpc := pcdatavalue2(findfunc(pc), _PCDATA_StackMapIndex, pc + uintptr(len(cc)) - 1)
    assert.Equal(t, int32(0), smi)
    assert.Equal(t, pc, startpc)
    aup, startpc2 := pcdatavalue2(findfunc(pc), _PCDATA_UnsafePoint, pc + uintptr(len(cc)) - 1)
    assert.Equal(t, int32(_PCDATA_UnsafePointUnsafe), aup)
    assert.Equal(t, pc, startpc2)
}

func mkpointer() *int {
    ret := new(int)
    *ret = 1234
    runtime.SetFinalizer(ret, func(_ *int) {
        println("ret has been recycled")
    })
    println("ret is allocated")
    return ret
}

func collect() {
    println("start collecting")
    for i := 1; i < 1000; i++ {
        runtime.GC()
    }
    println("done collecting")
}

func TestLoader_StackMap(t *testing.T) {
    var asm x86_64.Assembler
    var smb rt.StackMapBuilder
    src := `
        subq    $24, %rsp
        movq    %rbp, 16(%rsp)
        leaq    16(%rsp), %rbp
        
        movq    $` + fmt.Sprintf("%p", mkpointer) + `, %r12
        callq   %r12`
    if runtime.Version() < "go1.17" { src += `
        movq    (%rsp), %rax`
    }
    src += `
        movq    %rax, 8(%rsp)
        movq    $0x123, (%rsp)
        movq    $` + fmt.Sprintf("%p", collect) + `, %r12
        callq   %r12
        movq    16(%rsp), %rbp
        addq    $24, %rsp
        ret`
    require.NoError(t, asm.Assemble(src))
    smb.AddField(true)
    fp := Loader(asm.Code()).Load("test_with_stackmap", rt.Frame {
        Size      : 24,
        ArgSize   : 0,
        ArgPtrs   : new(rt.StackMapBuilder).Build(),
        LocalPtrs : smb.Build(),
    })
    println("enter function")
    (*(*func())(unsafe.Pointer(&fp)))()
    println("leave function")
    collect()
}

//go:linkname step runtime.step
//goland:noinspection GoUnusedParameter
func step(p []byte, pc *uintptr, val *int32, first bool) (newp []byte, ok bool)

func dumpfunction(f interface{}) {
    fp := rt.FuncAddr(f)
    fn := findfunc(uintptr(fp))
    var name string
    if runtime.Version() >= "go1.16" {
        name = "pctab"
    } else {
        name = "pclntable"
    }
    datap := reflect.ValueOf(fn.datap)
    ff, ok := datap.Type().Elem().FieldByName(name)
    if !ok {
        panic("no such field: pctab")
    }
    p := (*(*[]byte)(unsafe.Pointer(uintptr(unsafe.Pointer(fn.datap)) + ff.Offset)))[fn.pcsp:]
    pc := fn.entry
    val := int32(-1)
    for {
        var ok bool
        lastpc := pc
        p, ok = step(p, &pc, &val, pc == fn.entry)
        if !ok {
            break
        }
        fmt.Printf("%#x = %d\n", lastpc, val)
    }
    pc = 0
    for i := 0; i < 16; i++ {
        ins, err := x86asm.Decode(rt.BytesFrom(unsafe.Pointer(uintptr(fp) + pc), 15, 15), 64)
        if err != nil {
            panic(err)
        }
        fmt.Printf("%#x %s\n", uintptr(fp) + pc, x86asm.GNUSyntax(ins, uint64(uintptr(fp) + pc), nil))
        pc += uintptr(ins.Len)
        if ins.Op == x86asm.RET {
            break
        }
    }
}

func TestLoader_PCSPDelta(t *testing.T) {
    dumpfunction(moduledataverify1)
}
