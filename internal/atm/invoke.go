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
    CallType uint8
)

const (
    CCall CallType = iota
    GCall
    ICall
)

type CallHandle struct {
    Id    int
    Slot  int
    Func  uintptr
    Type  CallType
    proxy func(CallContext)
}

func (self CallHandle) Call(e *Emulator, p *Instr) {
    self.proxy(CallContext {
        Emu  : e,
        Type : self.Type,
        argc : p.An,
        retc : p.Rn,
        argv : p.Ar,
        retv : p.Rr,
        itab : p.Ps,
        data : p.Pd,
    })
}

type CallContext struct {
    Emu  *Emulator
    Type CallType
    itab PointerRegister
    data PointerRegister
    argc uint8
    retc uint8
    argv [8]uint8
    retv [8]uint8
}

func (self CallContext) Au(i int) uint64 {
    if p := self.argv[i]; p & ArgPointer != 0 {
        panic("invoke: invalid int argument")
    } else {
        return self.Emu.Gr[p & ArgMask]
    }
}

func (self CallContext) Ap(i int) unsafe.Pointer {
    if p := self.argv[i]; p & ArgPointer == 0 {
        panic("invoke: invalid pointer argument")
    } else {
        return self.Emu.Pr[p & ArgMask]
    }
}

func (self CallContext) Ru(i int, v uint64) {
    if p := self.retv[i]; p & ArgPointer != 0 {
        panic("invoke: invalid int return value")
    } else {
        self.Emu.Gr[p & ArgMask] = v
    }
}

func (self CallContext) Rp(i int, v unsafe.Pointer) {
    if p := self.retv[i]; p & ArgPointer == 0 {
        panic("invoke: invalid pointer return value")
    } else {
        self.Emu.Pr[p & ArgMask] = v
    }
}

func (self CallContext) Itab() *rt.GoItab {
    if self.Type != ICall {
        panic("invoke: itab is not available")
    } else {
        return (*rt.GoItab)(self.Emu.Pr[self.itab])
    }
}

func (self CallContext) Data() unsafe.Pointer {
    if self.Type != ICall {
        panic("invoke: data is not available")
    } else {
        return self.Emu.Pr[self.data]
    }
}

func (self CallContext) Verify(args string, rets string) bool {
    return self.verifySeq(args, self.argc, self.argv) && self.verifySeq(rets, self.retc, self.retv)
}

func (self CallContext) verifySeq(s string, n uint8, v [8]uint8) bool {
    nb := int(n)
    ne := len(s)

    /* sanity check */
    if ne > len(v) {
        panic("invoke: invalid descriptor")
    }

    /* check for value count */
    if nb != ne {
        return false
    }

    /* check for every argument */
    for i := 0; i < nb; i++ {
        switch s[i] {
            case 'i' : if v[i] & ArgPointer != 0 { return false }
            case '*' : if v[i] & ArgPointer == 0 { return false }
            default  : panic("invoke: invalid descriptor char: " + s[i:i + 1])
        }
    }

    /* all checked ok */
    return true
}

var (
    invokeTab []CallHandle
)

func RegisterCCall(fn uintptr, proxy func(CallContext)) (h CallHandle) {
    h.Id      = len(invokeTab)
    h.Type    = CCall
    h.Func    = fn
    h.proxy   = proxy
    invokeTab = append(invokeTab, h)
    return
}

func RegisterICall(mt rt.Method, proxy func(CallContext)) (h CallHandle) {
    h.Id      = len(invokeTab)
    h.Type    = ICall
    h.Slot    = ABI.RegisterMethod(h.Id, mt)
    h.proxy   = proxy
    invokeTab = append(invokeTab, h)
    return
}

func RegisterGCall(fn interface{}, proxy func(CallContext)) (h CallHandle) {
    h.Id      = len(invokeTab)
    h.Type    = GCall
    h.Func    = ABI.RegisterFunction(h.Id, fn)
    h.proxy   = proxy
    invokeTab = append(invokeTab, h)
    return
}
