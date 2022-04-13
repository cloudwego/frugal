/*
 * Copyright 2022 ByteDance Inc.
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

package ssa

// CopyElim removes unnessecery register copies.
type CopyElim struct{}

func (self CopyElim) Apply(cfg *CFG) {
    regs := make(map[Reg]Reg)
    consts := make(map[Reg]_ConstData)

    /* constant zero registers */
    consts[Rz] = constint(0)
    consts[Pn] = constptr(nil)

    /* register replacement func */
    replacereg := func(rr *Reg) {
        for {
            if r, ok := regs[*rr]; ok {
                *rr = r
            } else {
                break
            }
        }
    }

    /* Phase 1: Find all the constants */
    cfg.ReversePostOrder(func(bb *BasicBlock) {
        for _, v := range bb.Ins {
            switch p := v.(type) {
                case *IrConstInt: consts[p.R] = constint(p.V)
                case *IrConstPtr: consts[p.R] = constptr(p.P)
            }
        }
    })

    /* Phase 2: Identify all the identity operations */
    cfg.ReversePostOrder(func(bb *BasicBlock) {
        for _, v := range bb.Ins {
            if p, ok := v.(*IrLEA); ok {
                if cc, ok := consts[p.Off]; ok && cc.i && cc.v == 0 {
                    regs[p.R] = p.Mem
                }
            } else if p, ok := v.(*IrBinaryExpr); ok {
                if x, ok := consts[p.X]; ok && x.i && x.v == 0 {
                    regs[p.R] = p.Y
                } else if y, ok := consts[p.Y]; ok && y.i && y.v == 0 {
                    regs[p.R] = p.X
                }
            }
        }
    })

    /* Phase 3: Replace all the register references */
    cfg.ReversePostOrder(func(bb *BasicBlock) {
        var ok bool
        var use IrUsages

        /* replace in Phi nodes */
        for _, v := range bb.Phi {
            for _, u := range v.Usages() {
                replacereg(u)
            }
        }

        /* replace in instructions */
        for _, v := range bb.Ins {
            if use, ok = v.(IrUsages); ok {
                for _, u := range use.Usages() {
                    replacereg(u)
                }
            }
        }

        /* replace in terminators */
        if use, ok = bb.Term.(IrUsages); ok {
            for _, u := range use.Usages() {
                replacereg(u)
            }
        }
    })
}
