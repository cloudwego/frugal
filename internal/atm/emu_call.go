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

type (
    CFunction = unsafe.Pointer
    CallProxy = func(e *Emulator, p *Instr)
)

var (
    icallTab = map[rt.Method]CallProxy{}
    ccallTab = map[unsafe.Pointer]CallProxy{}
    gcallTab = map[unsafe.Pointer]CallProxy{}
)

func RegisterICall(mt rt.Method, proxy CallProxy) {
    icallTab[mt] = proxy
}

func RegisterCCall(fn CFunction, proxy CallProxy) {
    ccallTab[fn] = proxy
}

func RegisterGCall(fn interface{}, proxy CallProxy) {
    gcallTab[rt.FuncAddr(fn)] = proxy
}
