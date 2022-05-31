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
    `fmt`
    `sort`

    `github.com/cloudwego/frugal/internal/rt`
)

type _ValuePos struct {
    i  int
    bb *BasicBlock
}

func (self *_ValuePos) lca(cfg *CFG, bb *BasicBlock) *BasicBlock {
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
        return u
    } else {
        panic("invalid CFG dominator tree")
    }
}

func (self *_ValuePos) update(cfg *CFG, bb *BasicBlock, p int) {
    switch lca := self.lca(cfg, bb); {
        case lca == self.bb : break
        case lca == bb      : self.i, self.bb = p, bb
        default             : self.i, self.bb = -1, lca
    }
}

func (self *_ValuePos) String() string {
    return fmt.Sprintf("bb_%d:%d", self.bb.Id, self.i)
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
        return ir[i].(*IrLoadArg).Id < ir[j].(*IrLoadArg).Id
    })

    /* prepend to the root node */
    ins := cfg.Root.Ins
    cfg.Root.Ins = append(ir, ins...)
}

func (Reorder) moveInterblock(cfg *CFG) {
    defs := make(map[Reg]*_ValuePos)
    move := make(map[*BasicBlock][]int)
    uses := make(map[_ValuePos]*_ValuePos)

    /* usage update routine */
    updateUsage := func(r Reg, bb *BasicBlock, i int) {
        if m, ok := defs[r]; ok {
            if m.bb == nil {
                m.i, m.bb = i, bb
            } else {
                m.update(cfg, bb, i)
            }
        }
    }

    /* retry until no modifications */
    for move[nil] = nil; len(move) != 0; {
        rt.MapClear(defs)
        rt.MapClear(move)
        rt.MapClear(uses)

        /* Phase 1: Find all movable value definitions */
        cfg.PostOrder(func(bb *BasicBlock) {
            for i, v := range bb.Ins {
                var f bool
                var k _ValuePos
                var p *_ValuePos
                var d IrDefinitions

                /* we can't move values which have a memory arg, it
                 * might make two memory values live across a block boundary */
                if _, f = v.(*IrLoad)    ; f { continue }
                if _, f = v.(*IrStore)   ; f { continue }
                if _, f = v.(*IrLoadArg) ; f { continue }

                /* add all the definitions */
                if d, f = v.(IrDefinitions); f {
                    k.i = i
                    k.bb = bb

                    /* create a new value movement if needed */
                    if p, f = uses[k]; !f {
                        p = new(_ValuePos)
                        uses[k] = p
                    }

                    /* mark all the non-definition sites */
                    for _, r := range d.Definitions() {
                        if r.Kind() != K_zero {
                            defs[*r] = p
                        }
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
                    updateUsage(*r, b, len(b.Ins))
                }
            }

            /* search in instructions */
            for i, v := range bb.Ins {
                if use, ok = v.(IrUsages); ok {
                    for _, r := range use.Usages() {
                        updateUsage(*r, bb, i)
                    }
                }
            }

            /* search the terminator */
            if use, ok = bb.Term.(IrUsages); ok {
                for _, r := range use.Usages() {
                    updateUsage(*r, bb, len(bb.Ins))
                }
            }
        })

        /* Phase 3: Move value definitions to their usage block */
        for p, m := range uses {
            if m.bb != nil && m.bb != p.bb {
                m.bb.Ins = append(m.bb.Ins, p.bb.Ins[p.i])
                move[m.bb] = append(move[m.bb], m.i)
                p.bb.Ins[p.i] = new(IrNop)
            }
        }

        /* Phase 4: Move values to place */
        for bb, m := range move {
            for i := len(m) - 1; i >= 0; i-- {
                n := len(bb.Ins)
                p := bb.Ins[n - 1]
                copy(bb.Ins[m[i] + 1:], bb.Ins[m[i]:n - 1])
                bb.Ins[m[i]] = p
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
}

func (self Reorder) Apply(cfg *CFG) {
    self.moveLoadArgs(cfg)
    self.moveInterblock(cfg)
}

