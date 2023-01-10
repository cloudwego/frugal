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
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/loader`
    `github.com/cloudwego/frugal/internal/rt`
)

const (
    ZeroStep    = 16
    MaxZeroSize = 65536
)

func toaddr(p *x86_64.Label) uintptr {
    if v, err := p.Evaluate(); err != nil {
        panic(err)
    } else {
        return uintptr(v)
    }
}

func asmmemzero() MemZeroFn {
    p := x86_64.DefaultArch.CreateProgram()
    x := make([]*x86_64.Label, MaxZeroSize / ZeroStep + 1)

    /* create all the labels */
    for i := range x {
        x[i] = x86_64.CreateLabel(fmt.Sprintf("zero_%d", i * ZeroStep))
    }

    /* fill backwards */
    for n := MaxZeroSize; n >= ZeroStep; n -= ZeroStep {
        p.Link(x[n / ZeroStep])
        p.MOVDQU(x86_64.XMM15, x86_64.Ptr(x86_64.RDI, int32(n - ZeroStep)))
    }

    /* finish the function */
    p.Link(x[0])
    p.RET()

    /* assemble the function */
    c := p.Assemble(0)
    r := make([]uintptr, len(x))

    /* resolve all the labels */
    for i, v := range x {
        r[i] = toaddr(v)
    }

    /* load the function */
    defer p.Free()
    return MemZeroFn {
        Sz: r,
        Fn: *(*unsafe.Pointer)(loader.Loader(c).Load("_frugal_memzero", rt.Frame{})),
    }
}
