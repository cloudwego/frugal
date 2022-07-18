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
    `github.com/oleiade/lane`
)

type BasicBlockIter struct {
    g *CFG
    b *BasicBlock
    s *lane.Stack
    v map[int]struct{}
}

func newBasicBlockIter(cfg *CFG) *BasicBlockIter {
    return &BasicBlockIter {
        g: cfg,
        s: stacknew(cfg.Root),
        v: map[int]struct{}{ cfg.Root.Id: {} },
    }
}

func (self *BasicBlockIter) Next() bool {
    var tail bool
    var this *BasicBlock

    /* scan until the stack is empty */
    for !self.s.Empty() {
        tail = true
        this = self.s.Head().(*BasicBlock)

        /* add all the successors */
        for _, p := range self.g.DominatorOf[this.Id] {
            if _, ok := self.v[p.Id]; !ok {
                tail = false
                self.v[p.Id] = struct{}{}
                self.s.Push(p)
                break
            }
        }

        /* all the successors are visited, pop the current node */
        if tail {
            self.b = self.s.Pop().(*BasicBlock)
            return true
        }
    }

    /* clear the basic block pointer to indicate no more blocks */
    self.b = nil
    return false
}

func (self *BasicBlockIter) Block() *BasicBlock {
    return self.b
}

func (self *BasicBlockIter) ForEach(action func(bb *BasicBlock)) {
    for self.Next() {
        action(self.b)
    }
}

func (self *BasicBlockIter) Reversed() []*BasicBlock {
    nb := len(self.g.Depth)
    ret := make([]*BasicBlock, 0, nb)

    /* dump all the blocks */
    for self.Next() {
        ret = append(ret, self.b)
    }

    /* reverse the order */
    blockreverse(ret)
    return ret
}
