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

    `github.com/cloudwego/frugal/internal/atm/abi`
)

type _CFGPrivate struct {
    reg   uint64
    block uint64
}

func (self *_CFGPrivate) allocreg() int {
    return int(atomic.AddUint64(&self.reg, 1)) - 1
}

func (self *_CFGPrivate) allocblock() int {
    return int(atomic.AddUint64(&self.block, 1)) - 1
}

type CFG struct {
    _CFGPrivate
    Func              FuncData
    Root              *BasicBlock
    Depth             map[int]int
    Layout            *abi.FunctionLayout
    DominatedBy       map[int]*BasicBlock
    DominatorOf       map[int][]*BasicBlock
    DominanceFrontier map[int][]*BasicBlock
}

func (self *CFG) Rebuild() {
    updateDominatorTree(self)
    updateDominatorDepth(self)
    updateDominatorFrontier(self)
}

func (self *CFG) MaxBlock() int {
    return int(self.block)
}

func (self *CFG) PostOrder() *BasicBlockIter {
    return newBasicBlockIter(self)
}

func (self *CFG) CreateBlock() (r *BasicBlock) {
    r = new(BasicBlock)
    r.Id = self.allocblock()
    return
}

func (self *CFG) CreateRegister(ptr bool) Reg {
    if i := self.allocreg(); ptr {
        return mkreg(1, K_norm, 0).Derive(i)
    } else {
        return mkreg(0, K_norm, 0).Derive(i)
    }
}

func (self *CFG) CreateUnreachable(bb *BasicBlock) (ret *BasicBlock) {
    ret      = self.CreateBlock()
    ret.Ins  = []IrNode { new(IrBreakpoint) }
    ret.Term = &IrSwitch { Ln: IrLikely(ret) }
    ret.Pred = []*BasicBlock { bb, ret }
    return
}
