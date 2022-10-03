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

package loader

import (
    `fmt`
    `os`
    `syscall`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

const (
    MEM_COMMIT  = 0x00001000
    MEM_RESERVE = 0x00002000
)

var (
    libKernel32                = syscall.NewLazyDLL("KERNEL32.DLL")
    libKernel32_VirtualAlloc   = libKernel32.NewProc("VirtualAlloc")
    libKernel32_VirtualProtect = libKernel32.NewProc("VirtualProtect")
)

type Loader []byte
type Function unsafe.Pointer

func mkptr(m uintptr) unsafe.Pointer {
    return *(*unsafe.Pointer)(unsafe.Pointer(&m))
}

func alignUp(n uintptr, a int) uintptr {
    return (n + uintptr(a) - 1) &^ (uintptr(a) - 1)
}

func (self Loader) Load(fn string, frame rt.Frame) (f Function) {
    var mm uintptr
    var er error
    var r1 uintptr

    /* align the size to pages */
    nf := uintptr(len(self))
    nb := alignUp(nf, os.Getpagesize())

    /* allocate a block of memory */
    if mm, _, er = libKernel32_VirtualAlloc.Call(0, nb, MEM_COMMIT|MEM_RESERVE, syscall.PAGE_READWRITE); mm == 0 {
        panic(er)
    }

    /* copy code into the memory, and register the function */
    copy(rt.BytesFrom(mkptr(mm), len(self), int(nb)), self)
    registerFunction(fmt.Sprintf("(frugal).%s_%x", fn, mm), mm, nf, frame)

    /* make it executable */
    var oldPf uintptr
    if r1, _, er = libKernel32_VirtualProtect.Call(mm, nb, syscall.PAGE_EXECUTE_READ, uintptr(unsafe.Pointer(&oldPf))); r1 == 0 {
        panic(er)
    } else {
        return Function(&mm)
    }
}
