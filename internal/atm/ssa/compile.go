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

type PassDescriptor struct {
    Pass Pass
    Desc string
}

var Passes = [...]PassDescriptor {
    { Desc: "Constant Propagation"                , Pass: new(ConstProp) },
    { Desc: "Common Sub-expression Elimination"   , Pass: new(CSE) },
    { Desc: "Early Phi Elimination"               , Pass: new(PhiElim) },
    { Desc: "Early Copy Elimination"              , Pass: new(CopyElim) },
    { Desc: "Early Trivial Dead Code Elimination" , Pass: new(TDCE) },
    { Desc: "Branch Elimination"                  , Pass: new(BranchElim) },
    { Desc: "Late Phi Elimination"                , Pass: new(PhiElim) },
    { Desc: "Late Copy Elimination"               , Pass: new(CopyElim) },
    { Desc: "Late Trivial Dead Code Elimination"  , Pass: new(TDCE) },
    { Desc: "Intermediate Block Merging"          , Pass: new(BlockMerge) },
    { Desc: "Machine Dependent Lowering"          , Pass: new(Lowering) },
}

func applySSAPasses(cfg *CFG) {
    for _, p := range Passes {
        p.Pass.Apply(cfg)
    }
}

func Compile(p hir.Program) (cfg *CFG) {
    cfg = newGraphBuilder().build(p)
    insertPhiNodes(&cfg.DominatorTree)
    renameRegisters(&cfg.DominatorTree)
    applySSAPasses(cfg)
    return
}
