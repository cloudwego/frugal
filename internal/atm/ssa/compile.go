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

type Pass interface {
    Apply(*CFG)
}

type _PassDescriptor struct {
    pass Pass
    desc string
}

var _passes = [...]_PassDescriptor {
    { desc: "Constant Propagation"                , pass: new(ConstProp) },
    { desc: "Common Sub-expression Elimination"   , pass: new(CSE) },
    { desc: "Early Phi Elimination"               , pass: new(PhiElim) },
    { desc: "Early Copy Elimination"              , pass: new(CopyElim) },
    { desc: "Early Trivial Dead Code Elimination" , pass: new(TDCE) },
    { desc: "Branch Elimination"                  , pass: new(BranchElim) },
    { desc: "Late Phi Elimination"                , pass: new(PhiElim) },
    { desc: "Late Copy Elimination"               , pass: new(CopyElim) },
    { desc: "Late Trivial Dead Code Elimination"  , pass: new(TDCE) },
    { desc: "Intermediate Block Merging"          , pass: new(BlockMerge) },
    { desc: "Machine Dependent Lowering"          , pass: new(Lowering) },
}

func applySSAPasses(cfg *CFG) {
    for _, p := range _passes {
        p.pass.Apply(cfg)
    }
}

func Compile(p hir.Program) (cfg *CFG) {
    cfg = newGraphBuilder().build(p)
    insertPhiNodes(&cfg.DominatorTree)
    renameRegisters(&cfg.DominatorTree)
    applySSAPasses(cfg)
    return
}
