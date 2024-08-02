//go:build !go1.17
// +build !go1.17

/*
 * Copyright 2022 CloudWeGo Authors
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

package pgen

import (
	"runtime"

	"github.com/cloudwego/frugal/internal/jit/atm/hir"
	"github.com/cloudwego/frugal/internal/jit/atm/rtx"
	"github.com/cloudwego/iasm/x86_64"
)

/** Stack Checking **/

const (
	_M_memcpyargs  = 24
	_G_stackguard0 = 0x10
)

func (self *CodeGen) abiStackCheck(p *x86_64.Program, to *x86_64.Label, sp uintptr) {
	ctxt := self.ctxt
	size := ctxt.size() + int32(sp)

	/* get the current goroutine */
	switch runtime.GOOS {
	case "linux":
		p.MOVQ(Abs(-8), RCX).FS()
	case "darwin":
		p.MOVQ(Abs(0x30), RCX).GS()
	default:
		panic("unsupported operating system")
	}

	/* check the stack guard */
	p.LEAQ(Ptr(RSP, -size), RAX)
	p.CMPQ(Ptr(RCX, _G_stackguard0), RAX)
	p.JBE(to)
}

/** Efficient Block Copy Algorithm **/

func (self *CodeGen) abiBlockCopy(p *x86_64.Program, pd hir.PointerRegister, ps hir.PointerRegister, nb hir.GenericRegister) {
	rd := self.r(pd)
	rs := self.r(ps)
	rl := self.r(nb)

	/* save all the registers, if they will be clobbered */
	for _, lr := range self.ctxt.regs {
		if rr := self.r(lr); rtx.R_memmove[rr] {
			p.MOVQ(rr, self.ctxt.slot(lr))
		}
	}

	/* load the args and call the function */
	p.MOVQ(rd, Ptr(RSP, 0))
	p.MOVQ(rs, Ptr(RSP, 8))
	p.MOVQ(rl, Ptr(RSP, 16))
	p.MOVQ(uintptr(rtx.F_memmove), RDI)
	p.CALLQ(RDI)

	/* restore all the registers, if they were clobbered */
	for _, lr := range self.ctxt.regs {
		if rr := self.r(lr); rtx.R_memmove[rr] {
			p.MOVQ(self.ctxt.slot(lr), rr)
		}
	}
}
