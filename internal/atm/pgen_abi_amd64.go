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
    `fmt`

    `github.com/chenzhuoyu/iasm/x86_64`
)

/** Prologue & Epilogue **/

func (self *CodeGen) abiPrologue(p *x86_64.Program) {
    for i, v := range self.ctxt.args.Args {
        if v.InRegister {
            p.MOVQ(v.Reg, self.ctxt.argv(i))
        }
    }
}

func (self *CodeGen) abiEpilogue(_ *x86_64.Program) {
    /* do nothing */
}

/** Reserved Register Management **/

func (self *CodeGen) abiSaveReserved(p *x86_64.Program) {
    for rr := range self.ctxt.regr {
        p.MOVQ(rr, self.ctxt.rslot(rr))
    }
}

func (self *CodeGen) abiLoadReserved(p *x86_64.Program) {
    for rr := range self.ctxt.regr {
        p.MOVQ(self.ctxt.rslot(rr), rr)
    }
}

/** Argument & Return Value Management **/

func (self *CodeGen) abiLoadInt(p *x86_64.Program, i int, d GenericRegister) {
    p.MOVQ(self.ctxt.argv(i), self.r(d))
}

func (self *CodeGen) abiLoadPtr(p *x86_64.Program, i int, d PointerRegister) {
    p.MOVQ(self.ctxt.argv(i), self.r(d))
}

func (self *CodeGen) abiStoreInt(p *x86_64.Program, s GenericRegister, i int) {
    self.internalStoreRet(p, s, i)
}

func (self *CodeGen) abiStorePtr(p *x86_64.Program, s PointerRegister, i int) {
    self.internalStoreRet(p, s, i)
}

// internalStoreRet stores return value s into return value slot i.
//
// FIXME: This implementation messes with the register allocation, but currently
//        all the STRP / STRQ instructions appear at the end of the generated code
//        (guaranteed by `{encoder,decoder}/translator.go`), everything generated
//        after this is under our control, so it should be fine. This should be
//        fixed once SSA backend is ready.
func (self *CodeGen) internalStoreRet(p *x86_64.Program, s Register, i int) {
    var m Parameter
    var r *x86_64.Register64

    /* if return with stack, store directly */
    if m = self.ctxt.args.Rets[i]; !m.InRegister {
        p.MOVQ(self.r(s), self.ctxt.retv(i))
        return
    }

    /* check if the value is the very register required for return */
    if self.r(s) == m.Reg {
        return
    }

    /* search for register allocation */
    for n, v := range self.regs {
        if v == m.Reg {
            r = &self.regs[n]
            break
        }
    }

    /* if return with free registers, simply overwrite with new value */
    if r == nil {
        p.MOVQ(self.r(s), m.Reg)
        return
    }

    /* if not, swap the register allocation to meet the requirement */
    p.XCHGQ(self.r(s), m.Reg)
    self.regs[s.P()], *r = *r, self.regs[s.P()]
}

/** Function & Method Call **/

var argumentOrder = [6]x86_64.Register64 {
    RDI,
    RSI,
    RDX,
    RCX,
    R8,
    R9,
}

var argumentRegisters = map[x86_64.Register64]bool {
    RDI : true,
    RSI : true,
    RDX : true,
    RCX : true,
    R8  : true,
    R9  : true,
}

var reservedRegisters = map[x86_64.Register64]bool {
    RBX: true,
    R12: true,
    R13: true,
    R14: true,
    R15: true,
}

func ri2reg(ri uint8) Register {
    if ri & ArgPointer == 0 {
        return GenericRegister(ri & ArgMask)
    } else {
        return PointerRegister(ri & ArgMask)
    }
}

func (self *CodeGen) abiCallGo(p *x86_64.Program, v *Instr) {
    fp := invokeTab[v.Iv]
    fn := ABI.FnTab[fp.Id]
    rm := make(map[Register]int32)

    /* find the function */
    if fn == nil {
        panic(fmt.Sprintf("abiCallGo: invalid function ID: %d", v.Iv))
    }

    /* check for argument and return value count */
    if int(v.An) != len(fn.Args) || int(v.Rn) != len(fn.Rets) {
        panic("abiCallGo: argument or return value count mismatch")
    }

    /* save all the allocated registers before function call */
    for _, lr := range self.ctxt.regs {
        p.MOVQ(self.r(lr), self.ctxt.slot(lr))
    }

    /* load all the args */
    for i, argv := range fn.Args {
        if rr := ri2reg(v.Ar[i]); !argv.InRegister {
            if rr.P() == -1 {
                p.MOVQ(0, Ptr(RSP, int32(argv.Mem)))
            } else {
                p.MOVQ(self.r(rr), Ptr(RSP, int32(argv.Mem)))
            }
        } else {
            if rr.P() == -1 {
                p.XORL(x86_64.Register32(argv.Reg), x86_64.Register32(argv.Reg))
            } else if self.isRegUsed(argv.Reg) {
                p.MOVQ(self.ctxt.slot(rr), argv.Reg)
            } else {
                p.MOVQ(self.r(rr), argv.Reg)
            }
        }
    }

    /* restore reserved registers, and call the function, since
     * all the registers are been saved, and R12 is not one of
     * the argument registers in Go 1.17, we can clobber R12 safely */
    self.abiLoadReserved(p)
    p.MOVQ(uintptr(fp.faddr), R12)
    p.CALLQ(R12)
    self.abiSaveReserved(p)

    /* if the function returns a value with a used register, spill it on stack */
    for i, retv := range fn.Rets {
        if rr := ri2reg(v.Rr[i]); rr.P() != -1 {
            if !retv.InRegister {
                rm[rr] = int32(retv.Mem)
            } else if self.isRegUsed(retv.Reg) {
                p.MOVQ(retv.Reg, self.ctxt.slot(rr))
            }
        }
    }

    /* save all the non-spilled arguments */
    for i, retv := range fn.Rets {
        if rr := ri2reg(v.Rr[i]); rr.P() != -1 {
            if retv.InRegister && !self.isRegUsed(retv.Reg) {
                p.MOVQ(retv.Reg, self.r(rr))
            }
        }
    }

    /* restore all the allocated registers (except return values) after function call */
    for _, lr := range self.ctxt.regs {
        if _, ok := rm[lr]; !ok {
            p.MOVQ(self.ctxt.slot(lr), self.r(lr))
        }
    }

    /* store all the stack-based return values */
    for rr, mem := range rm {
        if mem != -1 {
            p.MOVQ(Ptr(RSP, mem), self.r(rr))
        }
    }
}

func (self *CodeGen) abiCallNative(p *x86_64.Program, v *Instr) {
    rv := Register(nil)
    fp := invokeTab[v.Iv]

    /* native function can have at most 1 return value */
    if v.Rn > 1 {
        panic("abiCallNative: native function can only have at most 1 return value")
    }

    /* passing arguments on stack is currently not implemented */
    if int(v.An) > len(argumentOrder) {
        panic("abiCallNative: not implemented: passing arguments on stack for native functions")
    }

    /* save all the allocated registers (except reserved registers) before function call */
    for _, lr := range self.ctxt.regs {
        if rr := self.r(lr); !reservedRegisters[rr] {
            p.MOVQ(rr, self.ctxt.slot(lr))
        }
    }

    /* load all the parameters */
    for i := 0; i < int(v.An); i++ {
        rr := ri2reg(v.Ar[i])
        rd := argumentOrder[i]

        /* check for zero source and spilled arguments */
        if rr.P() == -1 {
            p.XORL(x86_64.Register32(rd), x86_64.Register32(rd))
        } else if rs := self.r(rr); argumentRegisters[rs] {
            p.MOVQ(self.ctxt.slot(rr), rd)
        } else {
            p.MOVQ(rs, rd)
        }
    }

    /* call the function */
    p.MOVQ(uintptr(fp.faddr), RAX)
    p.CALLQ(RAX)

    /* store the result */
    if v.Rn != 0 {
        if rv = ri2reg(v.Rr[0]); rv.P() != -1 {
            p.MOVQ(RAX, self.r(rv))
        }
    }

    /* restore all the allocated registers (except reserved registers and result) after function call */
    for _, lr := range self.ctxt.regs {
        if rr := self.r(lr); (lr != rv) && !reservedRegisters[rr] {
            p.MOVQ(self.ctxt.slot(lr), rr)
        }
    }
}

func (self *CodeGen) abiCallMethod(p *x86_64.Program, v *Instr) {

}
