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

    `github.com/cloudwego/frugal/internal/atm/hir`
)

var (
    _GenericRegs []hir.GenericRegister
    _PointerRegs []hir.PointerRegister
)

func init() {
    _GenericRegs = make([]hir.GenericRegister, 0, len(hir.GenericRegisters))
    _PointerRegs= make([]hir.PointerRegister, 0, len(hir.PointerRegisters))

    /* extract all generic registers */
    for i := range hir.GenericRegisters {
        if i != hir.Rz {
            _GenericRegs = append(_GenericRegs, i)
        }
    }

    /* extract all pointer registers */
    for m := range hir.PointerRegisters {
        if m != hir.Pn {
            _PointerRegs = append(_PointerRegs, m)
        }
    }

    /* sort by register ID */
    sort.Slice(_GenericRegs, func(i int, j int) bool { return _GenericRegs[i] < _GenericRegs[j] })
    sort.Slice(_PointerRegs, func(i int, j int) bool { return _PointerRegs[i] < _PointerRegs[j] })
}

type _GraphBuilder struct {
    p map[*hir.Ir]bool
    g map[*hir.Ir]*BasicBlock
}

func newGraphBuilder() *_GraphBuilder {
    return &_GraphBuilder {
        p: make(map[*hir.Ir]bool),
        g: make(map[*hir.Ir]*BasicBlock),
    }
}

func (self *_GraphBuilder) build(p hir.Program) *CFG {
    ret := &CFG {
        Depth             : make(map[int]int),
        DominatedBy       : make(map[int]*BasicBlock),
        DominatorOf       : make(map[int][]*BasicBlock),
        DominanceFrontier : make(map[int][]*BasicBlock),
    }

    /* create the root block */
    ret.Root = ret.CreateBlock()
    ret.Root.Ins = make([]IrNode, 0, len(_GenericRegs) + len(_PointerRegs) + (K_tmp4 - K_tmp0) * 2)

    /* implicit defination of all generic registers */
    for _, v := range _GenericRegs {
        ret.Root.Ins = append(ret.Root.Ins, &IrConstInt { R: Rv(v), V: 0 })
    }

    /* implicit defination of all pointer registers */
    for _, v := range _PointerRegs {
        ret.Root.Ins = append(ret.Root.Ins, &IrConstPtr { R: Rv(v), P: nil })
    }

    /* implicitly define all the temporary registers */
    for i := uint64(K_tmp0); i <= K_tmp4; i++ {
        ret.Root.Ins = append(
            ret.Root.Ins,
            &IrConstInt { R: Tr(i - K_tmp0), V: 0 },
            &IrConstPtr { R: Pr(i - K_tmp0), P: nil },
        )
    }

    /* mark all the branch targets */
    for v := p.Head; v != nil; v = v.Ln {
        if v.IsBranch() {
            if v.Op != hir.OP_bsw {
                self.p[v.Br] = true
            } else {
                for _, lb := range v.Switch() {
                    self.p[lb] = true
                }
            }
        }
    }

    /* process the root block */
    self.g[p.Head] = ret.Root
    self.block(ret, p.Head, ret.Root)

    /* build the CFG */
    ret.Rebuild()
    return ret
}

func (self *_GraphBuilder) block(cfg *CFG, p *hir.Ir, bb *BasicBlock) {
    bb.Phi = nil
    bb.Term = nil

    /* traverse down until it hits a branch instruction */
    for p != nil && !p.IsBranch() && p.Op != hir.OP_ret {
        bb.addInstr(p)
        p = p.Ln

        /* hit a merge point, merge with existing block */
        if self.p[p] {
            bb.termBranch(self.branch(cfg, p))
            return
        }
    }

    /* basic block must terminate */
    if p == nil {
        panic(fmt.Sprintf("basic block %d does not terminate", bb.Id))
    }

    /* add terminators */
    switch p.Op {
        case hir.OP_bsw : self.termbsw(cfg, p, bb)
        case hir.OP_ret : bb.termReturn(p)
        case hir.OP_jmp : bb.termBranch(self.branch(cfg, p.Br))
        default         : bb.termCondition(p, self.branch(cfg, p.Br), self.branch(cfg, p.Ln))
    }
}

func (self *_GraphBuilder) branch(cfg *CFG, p *hir.Ir) *BasicBlock {
    var ok bool
    var bb *BasicBlock

    /* check for existing basic blocks */
    if bb, ok = self.g[p]; ok {
        return bb
    }

    /* create and process the new block */
    bb = cfg.CreateBlock()
    self.g[p] = bb
    self.block(cfg, p, bb)
    return bb
}

func (self *_GraphBuilder) termbsw(cfg *CFG, p *hir.Ir, bb *BasicBlock) {
    sw := new(IrSwitch)
    sw.Br = make(map[int32]*BasicBlock, p.Iv)
    bb.Term = sw

    /* add every branch of the switch instruction */
    for i, br := range p.Switch() {
        if br != nil {
            to := self.branch(cfg, br)
            to.addPred(bb)
            sw.Br[int32(i)] = to
        }
    }

    /* add the default branch */
    sw.Ln = self.branch(cfg, p.Ln)
    sw.Ln.addPred(bb)
}
