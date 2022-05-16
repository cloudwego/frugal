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

package rtx

import (
    `fmt`
    `math/rand`
    `testing`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
    `github.com/davecgh/go-spew/spew`
    `golang.org/x/arch/x86/x86asm`
)

//go:nosplit
//go:noescape
//goland:noinspection GoUnusedParameter
func callnative1(fn unsafe.Pointer, a0 uintptr)

func disasmfunc(p unsafe.Pointer) {
    pc := uintptr(0)
    for {
        pp := unsafe.Pointer(uintptr(p) + pc)
        ins, err := x86asm.Decode(rt.BytesFrom(pp, 15, 15), 64)
        if err != nil {
            panic(err)
        }
        fmt.Printf("%#x(%d) %s\n", uintptr(pp), ins.Len, x86asm.GNUSyntax(ins, uint64(uintptr(pp)), nil))
        pc += uintptr(ins.Len)
        if ins.Op == x86asm.RET {
            break
        }
    }
}

func zeromemsized(p unsafe.Pointer, n uintptr) {
    callnative1(unsafe.Pointer(uintptr(MemZero.Fn) + MemZero.Sz[n / ZeroStep]), uintptr(p))
}

func TestMemZero_Clear(t *testing.T) {
    mm := make([]byte, 256)
    rand.Read(mm)
    zeromemsized(unsafe.Pointer(&mm[0]), 48)
    spew.Dump(mm)
    disasmfunc(MemZero.Fn)
}
