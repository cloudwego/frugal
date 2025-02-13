//go:build go1.21

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
	"github.com/cloudwego/iasm/x86_64"

	"github.com/cloudwego/frugal/internal/jit/atm/abi"
	"github.com/cloudwego/frugal/internal/jit/atm/hir"
	"github.com/cloudwego/frugal/internal/jit/atm/rtx"
)

/*
 * wbStorePointer is the underlying logic for hir.OP_sp (sp: Store Pointer)
 * It checks rtx.V_pWriteBarrier first.
 * IF the flag equals to zero, we only need to update the pointer; go GC is not involved at all.
 * ELSE we jump to the label `_wb_store`, which (following the go1.21 way):
 *   1. Calls runtime.gcWriteBarrier2
 *   2. Stores new/old pointers to the allocated slots, with R11 as the beginning address
 *   3. Updates the pointer
 *   4. Jumps back to label `_wb_return`
 * Since it's unlikely to go the `ELSE` path, we postpone the translation, to make generated binary
 * more compact, which can make most use of the program cache of x86_64 architecture.
 */
func (self *CodeGen) wbStorePointer(p *x86_64.Program, s hir.PointerRegister, d *x86_64.MemoryOperand) {
	wb := x86_64.CreateLabel("_wb_store")
	rt := x86_64.CreateLabel("_wb_return")

	/* check for write barrier */
	p.MOVQ(uintptr(rtx.V_pWriteBarrier), RAX)
	p.CMPB(0, Ptr(RAX, 0))
	p.JNE(wb) /* jump to wbStoreFn (write barrier store pointer) */

	/* replace with the new pointer */
	wbUpdatePointer := func() {
		if s == hir.Pn {
			p.MOVQ(0, d)
		} else {
			p.MOVQ(self.r(s), d)
		}
	}

	/* Save new pointer to 0[R11] as required by gcWriteBarrier2 */
	wbStoreNewPointerForGC := func(r11 x86_64.Register64) {
		if s == hir.Pn { /* Pointer to Nil */
			p.MOVQ(0, Ptr(r11, 0))
		} else {
			p.MOVQ(self.r(s), Ptr(r11, 0))
		}
	}

	/* Save old pointer to 8[R11] as required by gcWriteBarrier2 */
	wbStoreOldPointerForGC := func(r11 x86_64.Register64) {
		p.MOVQ(d, RDI)
		p.MOVQ(RDI, Ptr(r11, abi.PtrSize))
	}

	/* write barrier wrapper */
	wbStoreFn := func(p *x86_64.Program) {
		self.abiSpillReserved(p)
		self.abiLoadReserved(p)
		p.MOVQ(R11, RAX) /* Save R11 -> RAX since R11 will be clobbered by gcWriteBarrier2 */
		p.MOVQ(uintptr(rtx.F_gcWriteBarrier2), RSI)
		p.CALLQ(RSI)      /* apply 2 slots and save the beginning address in R11 */
		p.XCHGQ(RAX, R11) /* Restore R11 <- RAX, and save R11(slotAddr) -> RAX */
		self.abiSaveReserved(p)
		self.abiRestoreReserved(p)
		wbStoreNewPointerForGC(RAX) /* MOV r(s), 0[R11] */
		wbStoreOldPointerForGC(RAX) /* MOV RDI, 8[R11] */
		p.JMP(rt)
	}

	/* defer the call to the end of generated code */
	p.Link(rt)
	wbUpdatePointer() /* Need to do it in go1.21+ both for direct_store or wb_store */
	self.later(wb, wbStoreFn)
}
