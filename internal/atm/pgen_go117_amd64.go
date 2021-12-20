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

/** Efficient Block Copy Algorithm **/

var (
    memcpyClobberSet  = resolveClobberSet(memmove)
    memcpyRegisterSet = map[x86_64.Register64]bool{RAX: true, RBX: true, RCX: true}
)

func (self *CodeGen) abiBlockZero(p *x86_64.Program, pd PointerRegister, nb int64) {
    dp := int32(0)
    p.MOVQ(self.r(pd), RDI)

    /* use XMM for larger blocks */
    if nb >= 16 {
        p.PXOR(XMM15, XMM15)
    }

    /* clear every 16-byte block */
    for nb >= 16 {
        p.MOVDQU(XMM15, Ptr(RDI, dp))
        dp += 16
        nb -= 16
    }

    /* only 1 byte left */
    if nb == 1 {
        p.MOVB(0, Ptr(RDI, dp))
        return
    }

    /* still bytes to be zeroed */
    if nb != 0 {
        p.XORL(EAX, EAX)
    }

    /* clear every 8-byte block */
    if nb >= 8 {
        p.MOVQ(RAX, Ptr(RDI, dp))
        dp += 8
        nb -= 8
    }

    /* clear every 4-byte block */
    if nb >= 8 {
        p.MOVL(EAX, Ptr(RDI, dp))
        dp += 4
        nb -= 4
    }

    /* clear every 2-byte block */
    if nb >= 2 {
        p.MOVW(AX, Ptr(RDI, dp))
        dp += 2
        nb -= 2
    }

    /* last byte */
    if nb > 0 {
        p.MOVB(AL, Ptr(RDI, dp))
    }
}

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
