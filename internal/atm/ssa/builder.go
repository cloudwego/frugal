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
    self.anchor(p)
    self.define(&p)
    return self.begin(p)
}

func (self *_GraphBuilder) begin(p hir.Program) (r *CFG) {
    r = new(CFG)
    r.Root = self.branch(p.Head)
    r.Rebuild()
    return
}

func (self *_GraphBuilder) block(p *hir.Ir, bb *BasicBlock) {
    bb.Phi = nil
    bb.Ins = make([]IrNode, 0, 16)

    /* traverse down until it hits a branch instruction */
    for p != nil && !p.IsBranch() && p.Op != hir.OP_ret {
        bb.addInstr(p)
        p = p.Ln

        /* hit a merge point, merge with existing block */
        if self.p[p] {
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
        case hir.OP_jmp : bb.termBranch(self.branch(p.Br))
        default         : bb.termCondition(p, self.branch(p.Br), self.branch(p.Ln))
    }
}

func (self *_GraphBuilder) anchor(p hir.Program) {
    for v := p.Head; v != nil; v = v.Ln {
        if v.IsBranch() {
            if v.Op != hir.OP_bsw {
                self.p[v.Br] = true
            } else {
                for _, lb := range v.Sw() {
                    self.p[lb] = true
                }
            }
        }
    }
}

func (self *_GraphBuilder) define(p *hir.Program) {
    i := hir.Rz
    m := hir.Pn
    b := hir.CreateBuilder()

    /* implicit defination of all generic registers */
    for i = range hir.GenericRegisters {
        if i != hir.Rz {
            b.MOV(hir.Rz, i)
        }
    }

    /* implicit defination of all pointer registers */
    for m = range hir.PointerRegisters {
        if m != hir.Pn {
            b.MOVP(hir.Pn, m)
        }
    }

    /* prepend to the program */
    r := b.Append(p.Head)
    p.Head = r
}

func (self *_GraphBuilder) branch(p *hir.Ir) *BasicBlock {
    var ok bool
    var bb *BasicBlock

    /* check for existing basic blocks */
    if bb, ok = self.g[p]; ok {
        return bb
    }

    /* create a new block */
    bb = new(BasicBlock)
    bb.Id = len(self.g) + 1

    /* process the new block */
    self.g[p] = bb
    self.block(p, bb)
    return bb
}

func (self *_GraphBuilder) termbsw(p *hir.Ir, bb *BasicBlock) {
    sw := new(IrSwitch)
    sw.Br = make(map[int64]*BasicBlock, p.Iv)
    bb.Term = sw

    /* add every branch of the switch instruction */
    for i, br := range p.Sw() {
        if br != nil {
            to := self.branch(br)
            to.Pred = append(to.Pred, bb)
            sw.Br[int64(i)] = to
        }
    }

    /* add the default branch */
    sw.Ln = self.branch(p.Ln)
    sw.Ln.Pred = append(sw.Ln.Pred, bb)
}

func (self *_GraphBuilder) termret(p *hir.Ir, bb *BasicBlock) {
    bb.Term = &IrReturn { R: ri2regs(p.Rr[:p.Rn]) }
}
