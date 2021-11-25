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
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

type CallHandle struct {
    Id    int
    Slot  int
    Func  unsafe.Pointer
    Proxy func(e *Emulator, p *Instr)
}

var (
    invokeTab []CallHandle
)

func RegisterICall(mt rt.Method, proxy func(e *Emulator, p *Instr)) (h CallHandle) {
    h.Id      = len(invokeTab)
    h.Slot    = ABI.RegisterMethod(h.Id, mt)
    h.Proxy   = proxy
    invokeTab = append(invokeTab, h)
    return
}

func RegisterGCall(fn interface{}, proxy func(e *Emulator, p *Instr)) (h CallHandle) {
    h.Id      = len(invokeTab)
    h.Func    = ABI.RegisterFunction(h.Id, fn)
    h.Proxy   = proxy
    invokeTab = append(invokeTab, h)
    return
}

func RegisterCCall(fn unsafe.Pointer, proxy func(e *Emulator, p *Instr)) (h CallHandle) {
    h.Id      = len(invokeTab)
    h.Func    = fn
    h.Proxy   = proxy
    invokeTab = append(invokeTab, h)
    return
}
