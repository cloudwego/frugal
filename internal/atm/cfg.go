/*
 * Copyright 2021 ByteDance Inc.
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

package atm

import (
    `github.com/oleiade/lane`
)

type BasicBlock struct {
    Id   int
    Len  int
    Src  *Instr
    Link []*BasicBlock
}

func (self *BasicBlock) Free() {
    q := lane.NewQueue()
    m := make(map[*BasicBlock]struct{})

    /* traverse the graph with BFS */
    for q.Enqueue(self); !q.Empty(); {
        v := q.Dequeue()
        p := v.(*BasicBlock)

        /* add all links into queue */
        for _, r := range p.Link {
            if _, ok := m[r]; !ok {
                q.Enqueue(r)
            }
        }

        /* clear links, and add to free list */
        m[p] = struct{}{}
        p.Link = p.Link[:0]
    }

    /* reset and free all the nodes */
    for p := range m {
        p.Id  = 0
        p.Len = 0
        p.Src = nil
        freeBasicBlock(p)
    }
}

type GraphBuilder struct {
    Pin   map[*Instr]bool
    Graph map[*Instr]*BasicBlock
}

func CreateGraphBuilder() *GraphBuilder {
    return &GraphBuilder {
        Pin   : make(map[*Instr]bool),
        Graph : make(map[*Instr]*BasicBlock),
    }
}

func (self *GraphBuilder) scan(p Program) {
    for v := p.Head; v != nil; v = v.Ln {
        if v.isBranch() {
            if v.Op != OP_bsw {
                self.Pin[v.Br] = true
            } else {
                for _, lb := range v.Sw() {
                    self.Pin[lb] = true
                }
            }
        }
    }
}

func (self *GraphBuilder) block(p *Instr, bb *BasicBlock) {
    bb.Len = 0
    bb.Src = p

    /* traverse down until it hits a branch instruction */
    for p != nil && !p.isBranch() {
        p = p.Ln
        bb.Len++

        /* hit a merge point, merge with existing block */
        if self.Pin[p] {
            bb.Link = append(bb.Link, self.branch(p))
            return
        }
    }

    /* end of basic block */
    if p == nil {
        return
    }

    /* JAL instruction doesn't technically "branch" */
    if bb.Len++; p.Op != OP_jal {
        bb.Link = append(bb.Link, self.branch(p.Ln))
    }

    /* check for switch instruction (it has multiple branches) */
    if p.Op != OP_bsw {
        bb.Link = append(bb.Link, self.branch(p.Br))
        return
    }

    /* add every branch of the switch instruction */
    for _, br := range p.Sw() {
        bb.Link = append(bb.Link, self.branch(br))
    }
}

func (self *GraphBuilder) branch(p *Instr) *BasicBlock {
    var ok bool
    var bb *BasicBlock

    /* check for existing basic blocks */
    if bb, ok = self.Graph[p]; ok {
        return bb
    }

    /* create a new block */
    bb = newBasicBlock()
    bb.Id = len(self.Graph) + 1

    /* process the new block */
    self.Graph[p] = bb
    self.block(p, bb)
    return bb
}

func (self *GraphBuilder) Build(p Program) *BasicBlock {
    self.scan(p)
    return self.branch(p.Head)
}
