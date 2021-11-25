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

func (self *CodeGen) internalStoreRet(p *x86_64.Program, s Register, i int) {
    // FIXME: This may cause register confliction, but it's only possible when
    //  storing return values, which is ususally located at the end of function.
    //  If such thing happens, adjust the order of STRP / STRQ instructions to
    //  remove confliction between stores.
    //  This issue should be resolved after we implemented a better register
    //  allocation algorithm.
    switch m := self.ctxt.args.Rets[i]; m.Tag {
        case ByReg   : p.MOVQ(self.r(s), m.Reg)
        case ByStack : p.MOVQ(self.r(s), self.ctxt.retv(i))
        default      : panic("internalStoreRet: invalid stack frame")
    }
}
