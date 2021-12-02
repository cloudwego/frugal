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
    `runtime`
    `testing`
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/stretchr/testify/assert`
    `github.com/stretchr/testify/require`
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
    var asm x86_64.Assembler
    if runtime.Version() >= "go1.17" {
        require.NoError(t, asm.Assemble("movq $1234, (%rax)\nret"))
    } else {
        require.NoError(t, asm.Assemble("movq 8(%rsp), %rax\nmovq $1234, (%rax)\nret"))
    }
    v0 := 0
    cc := asm.Code()
    fp := Loader(cc).Load("test", 0, 0, nil, nil)
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
