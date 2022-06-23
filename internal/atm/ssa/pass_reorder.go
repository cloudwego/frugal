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

    `github.com/cloudwego/frugal/internal/rt`
)

type _ValuePos struct {
    i  int
    bb *BasicBlock
}

type _BlockRef struct {
    bb *BasicBlock
}

func (self *_BlockRef) update(cfg *CFG, bb *BasicBlock) {
    u := bb
    v := self.bb

    /* move them to the same depth */
    for cfg.Depth[u.Id] != cfg.Depth[v.Id] {
        if cfg.Depth[u.Id] > cfg.Depth[v.Id] {
            u = cfg.DominatedBy[u.Id]
        } else {
            v = cfg.DominatedBy[v.Id]
        }
    }

    /* move both nodes until they meet */
    for u != v {
        u = cfg.DominatedBy[u.Id]
        v = cfg.DominatedBy[v.Id]
    }

    /* sanity check */
    if u != nil {
        self.bb = u
    } else {
        panic("invalid CFG dominator tree")
    }
}

// Reorder moves value closer to it's usage, which reduces register pressure.
type Reorder struct{}

func (Reorder) moveLoadArgs(cfg *CFG) {
    var ok bool
    var ir []IrNode
    var vv *IrLoadArg

    /* extract all the argument loads */
    cfg.PostOrder(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = bb.Ins[:0]

        /* scan instructions */
        for _, v := range ins {
            if vv, ok = v.(*IrLoadArg); ok {
                ir = append(ir, vv)
            } else {
                bb.Ins = append(bb.Ins, v)
            }
        }
    })

    /* sort by argument ID */
    sort.Slice(ir, func(i int, j int) bool {
        return ir[i].(*IrLoadArg).I < ir[j].(*IrLoadArg).I
    })

    /* prepend to the root node */
    ins := cfg.Root.Ins
    cfg.Root.Ins = append(ir, ins...)
}

func (Reorder) moveInterblock(cfg *CFG) {
    defs := make(map[Reg]*_BlockRef)
    move := make(map[*BasicBlock]int)
    uses := make(map[_ValuePos]*_BlockRef)

    /* usage update routine */
    updateUsage := func(r Reg, bb *BasicBlock) {
        if m, ok := defs[r]; ok {
            if m.bb == nil {
                m.bb = bb
            } else {
                m.update(cfg, bb)
            }
        }
    }

    /* retry until no modifications */
    for move[nil] = 0; len(move) != 0; {
        rt.MapClear(defs)
        rt.MapClear(move)
        rt.MapClear(uses)

        /* Phase 1: Find all movable value definitions */
        cfg.PostOrder(func(bb *BasicBlock) {
            for i, v := range bb.Ins {
                var f bool
                var p *_BlockRef
                var d IrDefinitions

                /* value must be movable, and have definitions */
                if _, f = v.(IrImmovable)  ;  f { continue }
                if d, f = v.(IrDefinitions); !f { continue }

                /* initialize the lookup key */
                k := _ValuePos {
                    i  : i,
                    bb : bb,
                }

                /* create a new value movement if needed */
                if p, f = uses[k]; !f {
                    p = new(_BlockRef)
                    uses[k] = p
                }

                /* mark all the non-definition sites */
                for _, r := range d.Definitions() {
                    if r.Kind() != K_zero {
                        defs[*r] = p
                    }
                }
            }
        })

        /* Phase 2: Identify the earliest usage locations */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            var ok bool
            var use IrUsages

            /* search in Phi nodes */
            for _, v := range bb.Phi {
                for b, r := range v.V {
                    updateUsage(*r, b)
                }
            }

            /* search in instructions */
            for _, v := range bb.Ins {
                if use, ok = v.(IrUsages); ok {
                    for _, r := range use.Usages() {
                        updateUsage(*r, bb)
                    }
                }
            }

            /* search the terminator */
            if use, ok = bb.Term.(IrUsages); ok {
                for _, r := range use.Usages() {
                    updateUsage(*r, bb)
                }
            }
        })

        /* Phase 3: Move value definitions to their usage block */
        for p, m := range uses {
            if m.bb != nil && m.bb != p.bb {
                m.bb.Ins = append(m.bb.Ins, p.bb.Ins[p.i])
                move[m.bb] = move[m.bb] + 1
                p.bb.Ins[p.i] = new(IrNop)
            }
        }

        /* Phase 4: Move values to place */
        for bb, i := range move {
            v := bb.Ins
            n := len(bb.Ins)
            bb.Ins = make([]IrNode, n)
            copy(bb.Ins[i:], v[:n - i])
            copy(bb.Ins[:i], v[n - i:])
        }
    }

    /* Phase 5: Remove all the placeholder NOP instructions */
    cfg.PostOrder(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = bb.Ins[:0]

        /* filter out the NOP instructions */
        for _, v := range ins {
            if _, ok := v.(*IrNop); !ok {
                bb.Ins = append(bb.Ins, v)
            }
        }
    })
}

func (self Reorder) Apply(cfg *CFG) {
    self.moveLoadArgs(cfg)
    self.moveInterblock(cfg)
}

