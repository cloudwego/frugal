// +build go1.17,!go1.18

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
    `github.com/chenzhuoyu/iasm/x86_64`
)

/** Stack Checking **/

const (
    _G_stackguard0 = 0x10
)

func (self *CodeGen) abiStackCheck(p *x86_64.Program, to *x86_64.Label, sp uintptr) {
    p.LEAQ (Ptr(RSP, -self.ctxt.size() - int32(sp)), R12)
    p.CMPQ (Ptr(R14, _G_stackguard0), R12)
    p.JBE  (to)
}

/** Efficient Block Copy Algorithm **/

var (
    memcpyClobberSet  = resolveClobberSet(memmove)
    memcpyRegisterSet = map[x86_64.Register64]bool{RAX: true, RBX: true, RCX: true}
)

func (self *CodeGen) abiBlockCopy(p *x86_64.Program, pd PointerRegister, ps PointerRegister, nb GenericRegister) {
    rs := self.r(ps)
    rl := self.r(nb)

    /* save all the registers, if they will be clobbered */
    for _, lr := range self.ctxt.regs {
        if rr := self.r(lr); memcpyClobberSet[rr] || memcpyRegisterSet[rr] {
            p.MOVQ(self.r(lr), self.ctxt.slot(lr))
        }
    }

    /* enumerate different register cases */
    switch {
        case rs == RBX && rl == RCX : p.MOVQ(self.r(pd), RAX)
        case rs == RBX && rl != RCX : p.MOVQ(self.r(pd), RAX); p.MOVQ  (rl, RCX)
        case rs != RBX && rl == RCX : p.MOVQ(self.r(pd), RAX); p.MOVQ  (rs, RBX)
        case rs == RCX && rl == RBX : p.MOVQ(self.r(pd), RAX); p.XCHGQ (RBX, RCX)
        case rs == RCX && rl != RBX : p.MOVQ(self.r(pd), RAX); p.MOVQ  (RCX, RBX); p.MOVQ(rl, RCX)
        case rs != RCX && rl == RBX : p.MOVQ(self.r(pd), RAX); p.MOVQ  (RBX, RCX); p.MOVQ(rs, RBX)
        default                     : p.MOVQ(self.r(pd), RAX); p.MOVQ  (rs, RBX);  p.MOVQ(rl, RCX)
    }

    /* call the function */
    p.MOVQ(F_memmove, RDI)
    p.CALLQ(RDI)

    /* restore all the registers, if they were clobbered */
    for _, lr := range self.ctxt.regs {
        if rr := self.r(lr); memcpyClobberSet[rr] || memcpyRegisterSet[rr] {
            p.MOVQ(self.ctxt.slot(lr), rr)
        }
    }
}
