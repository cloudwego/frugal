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

import (
    `sync/atomic`
)

// ReturnSpread spreads the return block to all it's
// successors, in order to shorten register live ranges.
type ReturnSpread struct{}

func (ReturnSpread) Apply(cfg *CFG) {
    more := true
    nrid := uint64(0)
    nbid := uint64(cfg.MaxBlock())
    rets := make([]*BasicBlock, 0, 1)

    /* register index updater */
    updateregs := func(rr []*Reg) {
        for _, r := range rr {
            if i := uint64(r.Index()); i > nrid {
                nrid = i
            }
        }
    }

    /* register replacer */
    replaceregs := func(rr map[Reg]Reg, ins IrNode) {
        var v Reg
        var ok bool
        var use IrUsages
        var def IrDefinitions

        /* replace register usages */
        if use, ok = ins.(IrUsages); ok {
            for _, r := range use.Usages() {
                if v, ok = rr[*r]; ok {
                    *r = v
                }
            }
        }

        /* replace register definitions */
        if def, ok = ins.(IrDefinitions); ok {
            for _, r := range def.Definitions() {
                if v, ok = rr[*r]; ok {
                    *r = v
                }
            }
        }
    }

    /* find the maximum register index */
    cfg.PostOrder(func(bb *BasicBlock) {
        var ok bool
        var use IrUsages
        var def IrDefinitions

        /* scan Phi nodes */
        for _, p := range bb.Phi {
            updateregs(p.Usages())
            updateregs(p.Definitions())
        }

        /* scan instructions */
        for _, p := range bb.Ins {
            if use, ok = p.(IrUsages)      ; ok { updateregs(use.Usages()) }
            if def, ok = p.(IrDefinitions) ; ok { updateregs(def.Definitions()) }
        }

        /* scan terminator */
        if use, ok = bb.Term.(IrUsages); ok {
            updateregs(use.Usages())
        }
    })

    /* loop until no more modifications */
    for more {
        more = false
        rets = rets[:0]

        /* Phase 1: Find the return blocks that has more than one predecessors */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            if _, ok := bb.Term.(*IrReturn); ok && len(bb.Pred) > 1 {
                more = true
                rets = append(rets, bb)
            }
        })

        /* Phase 2: Spread the blocks to it's predecessors */
        for _, bb := range rets {
            for _, pred := range bb.Pred {
                var ok bool
                var sw *IrSwitch

                /* register mappings */
                rr := make(map[Reg]Reg)
                nb := len(bb.Phi) + len(bb.Ins)

                /* allocate registers for Phi definitions */
                for _, phi := range bb.Phi {
                    rr[phi.R] = phi.R.Derive(int(atomic.AddUint64(&nrid, 1)))
                }

                /* allocate registers for instruction definitions */
                for _, ins := range bb.Ins {
                    if def, ok := ins.(IrDefinitions); ok {
                        for _, r := range def.Definitions() {
                            rr[*r] = r.Derive(int(atomic.AddUint64(&nrid, 1)))
                        }
                    }
                }

                /* create a new basic block */
                ret := &BasicBlock {
                    Id   : int(atomic.AddUint64(&nbid, 1)),
                    Ins  : make([]IrNode, 0, nb),
                    Pred : []*BasicBlock { pred },
                }

                /* add copy instruction for Phi nodes */
                for _, phi := range bb.Phi {
                    ret.Ins = append(ret.Ins, IrCopy(rr[phi.R], *phi.V[pred]))
                }

                /* copy all instructions */
                for _, ins := range bb.Ins {
                    ins = ins.Clone()
                    ret.Ins = append(ret.Ins, ins)
                    replaceregs(rr, ins)
                }

                /* copy the terminator */
                ret.Term = bb.Term.Clone().(IrTerminator)
                replaceregs(rr, ret.Term)

                /* link to the predecessor */
                if sw, ok = pred.Term.(*IrSwitch); !ok {
                    panic("invalid block terminator: " + pred.Term.String())
                }

                /* check for default branch */
                if sw.Ln == bb {
                    sw.Ln = ret
                    continue
                }

                /* replace the switch targets */
                for v, b := range sw.Br {
                    if b == bb {
                        sw.Br[v] = ret
                    }
                }
            }
        }

        /* rebuild & cleanup the graph if needed */
        if more {
            cfg.Rebuild()
            new(BlockMerge).Apply(cfg)
        }
    }
}
