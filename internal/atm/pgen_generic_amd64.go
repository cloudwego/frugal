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

func (self *CodeGen) abiPrologue(p *x86_64.Program) {
    for i, v := range self.ctxt.args.Args {
        if v.Tag == ByReg {
            p.MOVQ(v.Reg, self.ctxt.argv(i))
        }
    }
}

func (self *CodeGen) abiLoadInt(p *x86_64.Program, i int, d GenericRegister) {
    self.internalLoadArg(p, i, d)
}

func (self *CodeGen) abiLoadPtr(p *x86_64.Program, i int, d PointerRegister) {
    self.internalLoadArg(p, i, d)
}

func (self *CodeGen) abiStoreInt(p *x86_64.Program, s GenericRegister, i int) {
    self.internalStoreRet(p, s, i)
}

func (self *CodeGen) abiStorePtr(p *x86_64.Program, s PointerRegister, i int) {
    self.internalStoreRet(p, s, i)
}

func (self *CodeGen) internalLoadArg(p *x86_64.Program, i int, d Register) {
    p.MOVQ(self.ctxt.argv(i), self.r(d))
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
    if m = self.ctxt.args.Rets[i]; m.Tag == ByStack {
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
