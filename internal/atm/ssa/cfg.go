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

    `github.com/oleiade/lane`
)

type _CFGPrivate struct {
    nextreg   uint64
    nextblock uint64
}

func (self *_CFGPrivate) allocreg() int {
    return int(atomic.AddUint64(&self.nextreg, 1)) - 1
}

func (self *_CFGPrivate) allocblock() int {
    return int(atomic.AddUint64(&self.nextblock, 1)) - 1
}

type CFG struct {
    _CFGPrivate
    Root              *BasicBlock
    Depth             map[int]int
    DominatedBy       map[int]*BasicBlock
    DominatorOf       map[int][]*BasicBlock
    DominanceFrontier map[int][]*BasicBlock
}

func (self *CFG) Rebuild() {
    updateDominatorTree(self)
    updateDominatorDepth(self)
    updateDominatorFrontier(self)
}

func (self *CFG) DeriveFrom(r Reg) Reg {
    return r.Derive(self.allocreg())
}

func (self *CFG) CreateBlock() (r *BasicBlock) {
    r = new(BasicBlock)
    r.Id = self.allocblock()
    return
}

func (self *CFG) CreateUnreachable(bb *BasicBlock) (ret *BasicBlock) {
    ret      = self.CreateBlock()
    ret.Ins  = []IrNode { new(IrBreakpoint) }
    ret.Term = &IrSwitch { Ln: IrLikely(ret) }
    ret.Pred = []*BasicBlock { bb, ret }
    return
}

func (self *CFG) PostOrder(action func(bb *BasicBlock)) {
    stack := lane.NewStack()
    visited := make(map[int]bool)

    /* add root node */
    visited[self.Root.Id] = true
    stack.Push(self.Root)

    /* traverse the graph */
    for !stack.Empty() {
        tail := true
        this := stack.Head().(*BasicBlock)

        /* add all the successors */
        for _, p := range self.DominatorOf[this.Id] {
            if !visited[p.Id] {
                tail = false
                visited[p.Id] = true
                stack.Push(p)
                break
            }
        }

        /* all the successors are visited, pop the current node */
        if tail {
            action(stack.Pop().(*BasicBlock))
        }
    }
}

func (self *CFG) ReversePostOrder(action func(bb *BasicBlock)) {
    var i int
    var bb []*BasicBlock

    /* traverse as post-order */
    self.PostOrder(func(p *BasicBlock) {
        bb = append(bb, p)
    })

    /* reverse post-order */
    for i = len(bb) - 1; i >= 0; i-- {
        action(bb[i])
    }
}
