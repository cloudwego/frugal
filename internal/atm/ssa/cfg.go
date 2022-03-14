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

    `github.com/cloudwego/frugal/internal/atm/hir`
)

type BasicBlock struct {
    Id   int
    Phi  []*IrPhi
    Ins  []IrNode
    Term IrTerminator
    Pred []*BasicBlock
}

func (self *BasicBlock) addInstr(p *hir.Ir) {
    self.Ins = append(self.Ins, buildInstr(p)...)
}

func (self *BasicBlock) termBranch(to *BasicBlock) {
    to.Pred = append(to.Pred, self)
    self.Term = &IrSwitch{Ln: to}
}

func (self *BasicBlock) termCondition(p *hir.Ir, t *BasicBlock, f *BasicBlock) {
    v, c := buildBranchOp(p)
    t.Pred = append(t.Pred, self)
    f.Pred = append(f.Pred, self)
    self.Ins = append(self.Ins, c)
    self.Term = &IrSwitch{V: v, Ln: f, Br: map[int64]*BasicBlock{1: t}}
}

type GraphBuilder struct {
    Pin   map[*hir.Ir]bool
    Graph map[*hir.Ir]*BasicBlock
}

func CreateGraphBuilder() *GraphBuilder {
    return &GraphBuilder {
        Pin   : make(map[*hir.Ir]bool),
        Graph : make(map[*hir.Ir]*BasicBlock),
    }
}

func (self *GraphBuilder) scan(p hir.Program) {
    for v := p.Head; v != nil; v = v.Ln {
        if v.IsBranch() {
            if v.Op != hir.OP_bsw {
                self.Pin[v.Br] = true
            } else {
                for _, lb := range v.Sw() {
                    self.Pin[lb] = true
                }
            }
        }
    }
}

func (self *GraphBuilder) block(p *hir.Ir, bb *BasicBlock) {
    bb.Phi = nil
    bb.Ins = make([]IrNode, 0, 16)

    /* traverse down until it hits a branch instruction */
    for p != nil && !p.IsBranch() && p.Op != hir.OP_ret {
        bb.addInstr(p)
        p = p.Ln

        /* hit a merge point, merge with existing block */
        if self.Pin[p] {
            bb.termBranch(self.branch(p))
            return
        }
    }

    /* basic block must terminate */
    if p == nil {
        panic(fmt.Sprintf("basic block %d does not terminate", bb.Id))
    }

    /* add terminators */
    switch p.Op {
        case hir.OP_bsw : self.termbsw(p, bb)
        case hir.OP_ret : self.termret(p, bb)
        case hir.OP_jmp : bb.termBranch(self.branch(p.Ln))
        default         : bb.termCondition(p, self.branch(p.Br), self.branch(p.Ln))
    }
}

func (self *GraphBuilder) branch(p *hir.Ir) *BasicBlock {
    var ok bool
    var bb *BasicBlock

    /* check for existing basic blocks */
    if bb, ok = self.Graph[p]; ok {
        return bb
    }

    /* create a new block */
    bb = new(BasicBlock)
    bb.Id = len(self.Graph) + 1

    /* process the new block */
    self.Graph[p] = bb
    self.block(p, bb)
    return bb
}

func (self *GraphBuilder) termbsw(p *hir.Ir, bb *BasicBlock) {
    sw := new(IrSwitch)
    sw.Br, bb.Term = make(map[int64]*BasicBlock, p.Iv), sw

    /* add every branch of the switch instruction */
    for i, br := range p.Sw() {
        if br != nil {
            to := self.branch(br)
            to.Pred, sw.Br[int64(i)] = append(to.Pred, bb), to
        }
    }

    /* add the default branch */
    sw.Ln = self.branch(p.Ln)
    sw.Ln.Pred = append(sw.Ln.Pred, bb)
}

func (self *GraphBuilder) termret(p *hir.Ir, bb *BasicBlock) {
    var i uint8
    var ret []Reg

    /* convert each register */
    for i = 0; i < p.Rn; i++ {
        ret = append(ret, Rv(ri2reg(p.Rr[i])))
    }

    /* build the "return" IR */
    bb.Term = &IrReturn {
        R: ret,
    }
}

func (self *GraphBuilder) Build(p hir.Program) *BasicBlock {
    self.scan(p)
    return self.branch(p.Head)
}
