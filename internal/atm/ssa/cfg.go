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
    `github.com/cloudwego/frugal/internal/atm/hir`
)

type BasicBlock struct {
    Id   int
    Phi  []*IrPhi
    Ins  []IrNode
    Term IrTerminator
}

func (self *BasicBlock) addInstr(p *hir.Ir) {
    self.Ins = append(self.Ins, buildInstr(p)...)
}

func (self *BasicBlock) termBranch(to *BasicBlock) {
    self.Term = &IrSwitch{Ln: to}
}

func (self *BasicBlock) termCondition(p *hir.Ir, t *BasicBlock, f *BasicBlock) {
    v, c := buildBranchOp(p)
    self.Ins = append(self.Ins, c)
    self.Term = &IrSwitch{V: v, Ln: f, Br: map[int64]*BasicBlock{1: t}}
}
