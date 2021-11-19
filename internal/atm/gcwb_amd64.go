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
    _ `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
)

func (self *CodeGen) wbStoreNull(p *x86_64.Program, d PointerRegister) {
    rd := self.r(d)
    wb := x86_64.CreateLabel("_wb_store")
    rt := x86_64.CreateLabel("_wb_return")

    /* check for write barrier */
    p.MOVQ  (uintptr(unsafe.Pointer(&writeBarrier)), RSI)
    p.CMPB  (0, Ptr(RSI, 0))
    p.JNE   (wb)
    p.MOVQ  (0, Ptr(rd, 0))
    p.Link  (rt)

    /* defer the call to the end of generated code */
    self.later(wb, func(p *x86_64.Program) {
        p.XORL  (EAX, EAX)
        p.MOVQ  (rd, RDI)
        p.MOVQ  (int64(F_gcWriteBarrier), RSI)
        p.CALLQ (RSI)
        p.JMP   (rt)
    })
}

func (self *CodeGen) wbStorePointer(p *x86_64.Program, s PointerRegister, d PointerRegister) {
    rs := self.r(s)
    rd := self.r(d)
    wb := x86_64.CreateLabel("_wb_store")
    rt := x86_64.CreateLabel("_wb_return")

    /* check for write barrier */
    p.MOVQ  (uintptr(unsafe.Pointer(&writeBarrier)), RSI)
    p.CMPB  (0, Ptr(RSI, 0))
    p.JNE   (wb)
    p.MOVQ  (rs, Ptr(rd, 0))
    p.Link  (rt)

    /* defer the call to the end of generated code */
    self.later(wb, func(p *x86_64.Program) {
        p.MOVQ  (rs, RAX)
        p.MOVQ  (rd, RDI)
        p.MOVQ  (int64(F_gcWriteBarrier), RSI)
        p.CALLQ (RSI)
        p.JMP   (rt)
    })
}
