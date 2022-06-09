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
    Name string
}

var Passes = [...]PassDescriptor {
    { Name: "Constant Propagation"         , Pass: new(ConstProp) },
    { Name: "Early Reduction"              , Pass: new(Reduce) },
    { Name: "Branch Elimination"           , Pass: new(BranchElim) },
    { Name: "Intermediate Block Merging"   , Pass: new(BlockMerge) },
    { Name: "Value Reordering"             , Pass: new(Reorder) },
    { Name: "Late Reduction"               , Pass: new(Reduce) },
    { Name: "Machine Dependent Lowering"   , Pass: new(Lowering) },
    { Name: "Late Value Reordering"        , Pass: new(Reorder) },
    { Name: "Machine Dependent Fusion"     , Pass: new(Fusion) },
    { Name: "Final Value Reordering"       , Pass: new(Reorder) },
    { Name: "Final Reduction"              , Pass: new(Reduce) },
    { Name: "Machine Dependent Compaction" , Pass: new(Compaction) },
    { Name: "Liveness Analysis"            , Pass: new(Liveness) },
}

func executeSSAPasses(cfg *CFG) {
    for _, p := range Passes {
        p.Pass.Apply(cfg)
    }
}

func Compile(p hir.Program) (cfg *CFG) {
    cfg = newGraphBuilder().build(p)
    insertPhiNodes(cfg)
    renameRegisters(cfg)
    executeSSAPasses(cfg)
    return
}
