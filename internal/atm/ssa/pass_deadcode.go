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

// DCE removes deadcode (unused registers, unreachable blocks, etc.) from CFG.
type DCE struct{}

func (DCE) unused(cfg *CFG) {
    for {
        done := true
        decl := make(map[Reg]struct{})

        /* Phase 1: Mark all the definations */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            var ok bool
            var defs IrDefinations

            /* mark all definations in Phi nodes */
            for _, v := range bb.Phi {
                for _, r := range v.Definations() {
                    decl[*r] = struct{}{}
                }
            }

            /* mark all definations in instructions if any */
            for _, v := range bb.Ins {
                if defs, ok = v.(IrDefinations); ok {
                    for _, r := range defs.Definations() {
                        decl[*r] = struct{}{}
                    }
                }
            }
        })

        /* Phase 2: Find all register usages */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            var ok bool
            var use IrUsages

            /* mark all usages in Phi nodes */
            for _, v := range bb.Phi {
                for _, r := range v.Usages() {
                    delete(decl, *r)
                }
            }

            /* mark all usages in instructions if any */
            for _, v := range bb.Ins {
                if use, ok = v.(IrUsages); ok {
                    for _, r := range use.Usages() {
                        delete(decl, *r)
                    }
                }
            }

            /* mark usages in the terminator if any */
            if use, ok = bb.Term.(IrUsages); ok {
                for _, r := range use.Usages() {
                    delete(decl, *r)
                }
            }
        })

        /* Phase 3: Remove all unused declarations */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            var ok bool
            var defs IrDefinations

            /* replace unused Phi assigments with zero registers */
            for _, v := range bb.Phi {
                for _, r := range v.Definations() {
                    if _, ok = decl[*r]; ok && r.kind() != _K_zero {
                        *r, done = r.zero(), false
                    }
                }
            }

            /* replace unused instruction assigments with zero registers */
            for _, v := range bb.Ins {
                if defs, ok = v.(IrDefinations); ok {
                    for _, r := range defs.Definations() {
                        if _, ok = decl[*r]; ok && r.kind() != _K_zero {
                            *r, done = r.zero(), false
                        }
                    }
                }
            }
        })

        /* Phase 4: Remove the entire defination if it's all zeros */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            phi := bb.Phi[:0]
            ins := bb.Ins[:0]

            /* scan Phi nodes */
            for _, v := range bb.Phi {
                for _, r := range v.Definations() {
                    if r.kind() != _K_zero {
                        phi = append(phi, v)
                        break
                    }
                }
            }

            /* scan instructions */
            for _, v := range bb.Ins {
                if defs, ok := v.(IrDefinations); ok {
                    for _, r := range defs.Definations() {
                        if r.kind() != _K_zero {
                            ins = append(ins, v)
                            break
                        }
                    }
                }
            }

            /* rebuild the basic block */
            bb.Phi = phi
            bb.Ins = ins
        })

        /* no more modifications */
        if done {
            normalizeRegisters(&cfg.DominatorTree)
            break
        }
    }
}

func (DCE) unreachable(cfg *CFG) {
    // TODO: remove unreachable blocks
}

func (self DCE) Apply(cfg *CFG) {
    self.unused(cfg)
    self.unreachable(cfg)
}
