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

type _MoveInfo struct {
    ins  IrNode
    dest *_ValuePos
}

// Reorder moves value closer to it's usage, which reduces register pressure.
type Reorder struct{}

func (Reorder) moveLoadArgs(cfg *CFG) {
    cfg.ReversePostOrder(func(bb *BasicBlock) {
        if bb != cfg.Root {
            ins := bb.Ins
            bb.Ins = bb.Ins[:0]

            /* move all argument loads into the entry block */
            for _, v := range ins {
                if _, ok := v.(*IrLoadArg); !ok {
                    bb.Ins = append(bb.Ins, v)
                } else {
                    cfg.Root.Ins = append(cfg.Root.Ins, v)
                }
            }
        }
    })
}

func (Reorder) sortLoadArgs(cfg *CFG) {
    var ok bool
    var vv *IrLoadArg
    var ir []*IrLoadArg

    /* extract all the argument loads */
    for _, v := range cfg.Root.Ins {
        if vv, ok = v.(*IrLoadArg); ok {
            ir = append(ir, vv)
        }
    }

    /* sort by argument ID */
    sort.Slice(ir, func(i int, j int) bool {
        return ir[i].Id < ir[j].Id
    })

    /* make a copy of all the instructions */
    ins := cfg.Root.Ins
    cfg.Root.Ins = make([]IrNode, 0, len(ins))

    /* add all the argument loads */
    for _, v := range ir {
        cfg.Root.Ins = append(cfg.Root.Ins, v)
    }

    /* add all the remaining instructions */
    for _, v := range ins {
        if _, ok = v.(*IrLoadArg); !ok {
            cfg.Root.Ins = append(cfg.Root.Ins, v)
        }
    }
}

func (Reorder) moveValueDefs(cfg *CFG) {
    move := make([]_MoveInfo, 16)
    defs := make(map[Reg]*_ValuePos)
    uses := make(map[Reg]*_ValuePos)
    dest := make(map[int][]*_ValuePos)

    /* usage update routine */
    updateUsage := func(r Reg, bb *BasicBlock, i int) {
        var ok bool
        var pos *_ValuePos

        /* immovable values, ignore */
        if _, ok = defs[r]; !ok {
            return
        }

        /* update values if already met */
        if pos, ok = uses[r]; ok {
            pos.update(cfg, bb, i)
            return
        }

        /* add the new value */
        pos = &_ValuePos { i: i, bb: bb }
        uses[r], dest[bb.Id] = pos, append(dest[bb.Id], pos)
    }

    /* retry until no modifications */
    for len(move) != 0 {
        move = move[:0]
        rt.MapClear(defs)
        rt.MapClear(uses)
        rt.MapClear(dest)

        /* Phase 1: Find all movable value definitions */
        cfg.PostOrder(func(bb *BasicBlock) {
            for i, v := range bb.Ins {
                var f bool
                var d IrDefinations

                /* we can't move values which have a memory arg, it
                 * might make two memory values live across a block boundary */
                if _, f = v.(*IrLoad)    ; f { continue }
                if _, f = v.(*IrStore)   ; f { continue }
                if _, f = v.(*IrLoadArg) ; f { continue }

                /* add all the definitions */
                if d, f = v.(IrDefinations); f {
                    for _, r := range d.Definations() {
                        defs[*r] = &_ValuePos { i: i, bb: bb }
                    }
                }
            }
        })

        /* Phase 2: Identify the nearest location to place the definition */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            var ok bool
            var use IrUsages

            /* search in Phi nodes */
            for _, v := range bb.Phi {
                for b, r := range v.V {
                    updateUsage(*r, b, -1)
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
                    updateUsage(*r, bb, -1)
                }
            }
        })

        /* Phase 3: Move value definitions to their usage block */
        for r, v := range defs {
            var ins IrNode
            var pos *_ValuePos

            /* in the same block, ignore */
            if pos = uses[r]; v.bb == pos.bb {
                continue
            }

            /* extract the instruction */
            ins = v.bb.Ins[v.i]
            v.bb.Ins = append(v.bb.Ins[:v.i], v.bb.Ins[v.i + 1:]...)

            /* adjust positions if needed */
            if pv, ok := dest[v.bb.Id]; ok {
                for _, p := range pv {
                    if p.i > v.i {
                        p.i--
                    }
                }
            }

            /* add the movement info */
            move = append(move, _MoveInfo {
                ins  : ins,
                dest : pos,
            })
        }

        /* insert instruction into new block */
        for _, p := range move {
            if p.dest.i == -1 {
                p.dest.bb.Ins = append(p.dest.bb.Ins, p.ins)
            } else {
                p.dest.bb.Ins = append(p.dest.bb.Ins[:p.dest.i], append([]IrNode { p.ins }, p.dest.bb.Ins[p.dest.i:]...)...)
            }
        }
    }
}

func (self Reorder) Apply(cfg *CFG) {
    self.moveLoadArgs(cfg)
    self.sortLoadArgs(cfg)
    self.moveValueDefs(cfg)
}

