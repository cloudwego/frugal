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
    `sort`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/rt`
)

type _SplitPair struct {
    i  int
    bb *BasicBlock
}

func (self _SplitPair) isPriorTo(other _SplitPair) bool {
    return self.bb.Id < other.bb.Id || (self.i < other.i && self.bb.Id == other.bb.Id)
}

// WriteBarrier inserts write barriers for pointer stores.
type WriteBarrier struct{}

func (WriteBarrier) Apply(cfg *CFG) {
    more := true
    mbir := make(map[*BasicBlock]int)
    ptrs := make(map[Reg]unsafe.Pointer)

    /* find all constant pointers */
    cfg.PostOrder(func(bb *BasicBlock) {
        for _, v := range bb.Ins {
            if p, ok := v.(*IrAMD64_MOV_ptr); ok {
                ptrs[p.R] = p.P
            }
        }
    })

    /* loop until no more write barriers */
    for more {
        more = false
        rt.MapClear(mbir)

        /* Phase 1: Find all the memory barriers and pointer constants */
        cfg.PostOrder(func(bb *BasicBlock) {
            for i, v := range bb.Ins {
                if _, ok := v.(*IrWriteBarrier); ok {
                    if _, ok = mbir[bb]; ok {
                        more = true
                    } else {
                        mbir[bb] = i
                    }
                }
            }
        })

        /* split pair buffer */
        nb := len(mbir)
        mb := make([]_SplitPair, 0, nb)

        /* extract from the map */
        for p, i := range mbir {
            mb = append(mb, _SplitPair {
                i  : i,
                bb : p,
            })
        }

        /* sort by block ID */
        sort.Slice(mb, func(i int, j int) bool {
            return mb[i].isPriorTo(mb[j])
        })

        /* Phase 2: Split basic block at write barrier */
        for _, p := range mb {
            bb := cfg.CreateBlock()
            ds := cfg.CreateBlock()
            wb := cfg.CreateBlock()
            ir := p.bb.Ins[p.i].(*IrWriteBarrier)

            /* move instructions after the write barrier into a new block */
            bb.Ins  = p.bb.Ins[p.i + 1:]
            bb.Term = p.bb.Term
            bb.Pred = []*BasicBlock { ds, wb }

            /* update all the predecessors & Phi nodes */
            for it := p.bb.Term.Successors(); it.Next(); {
                succ := it.Block()
                pred := succ.Pred

                /* update predecessors */
                for x, v := range pred {
                    if v == p.bb {
                        pred[x] = bb
                        break
                    }
                }

                /* update Phi nodes */
                for _, phi := range succ.Phi {
                    phi.V[bb] = phi.V[p.bb]
                    delete(phi.V, p.bb)
                }
            }

            /* rewrite the direct store instruction */
            st := &IrAMD64_MOV_store_r {
                R: ir.R,
                M: Mem { M: ir.M, I: Rz, S: 1, D: 0 },
                N: abi.PtrSize,
            }

            /* construct the direct store block */
            ds.Ins  = []IrNode { st }
            ds.Term = &IrSwitch { Ln: IrLikely(bb) }
            ds.Pred = []*BasicBlock { p.bb }

            /* rewrite the write barrier instruction */
            fn := &IrAMD64_CALL_gcwb {
                R  : ir.R,
                M  : ir.M,
                Fn : ptrs[ir.Fn],
            }

            /* function address must exist */
            if fn.Fn == nil {
                panic("missing write barrier function address")
            }

            /* construct the write barrier block */
            wb.Ins  = []IrNode { fn }
            wb.Term = &IrSwitch { Ln: IrLikely(bb) }
            wb.Pred = []*BasicBlock { p.bb }

            /* rewrite the terminator to check for write barrier */
            p.bb.Ins  = p.bb.Ins[:p.i]
            p.bb.Term = &IrAMD64_Jcc_mi {
                X  : Mem { M: ir.Var, I: Rz, S: 1, D: 0 },
                Y  : 0,
                N  : 1,
                To : IrUnlikely(wb),
                Ln : IrLikely(ds),
                Op : IrAMD64_CmpNe,
            }
        }

        /* Phase 3: Rebuild the CFG */
        if len(mbir) != 0 {
            cfg.Rebuild()
        }
    }
}
