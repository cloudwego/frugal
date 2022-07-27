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
    `reflect`

    `github.com/cloudwego/frugal/internal/atm/abi`
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
    { Name: "Early Constant Propagation" , Pass: new(ConstProp) },
    { Name: "Early Reduction"            , Pass: new(Reduce) },
    { Name: "Branch Elimination"         , Pass: new(BranchElim) },
    { Name: "Block Merging"              , Pass: new(BlockMerge) },
    { Name: "Return Spreading"           , Pass: new(ReturnSpread) },
    { Name: "Value Reordering"           , Pass: new(Reorder) },
    { Name: "Late Constant Propagation"  , Pass: new(ConstProp) },
    { Name: "Late Reduction"             , Pass: new(Reduce) },
    { Name: "Machine Dependent Lowering" , Pass: new(Lowering) },
    { Name: "Zero Register Substitution" , Pass: new(ZeroReg) },
    { Name: "Write Barrier Insertion"    , Pass: new(WriteBarrier) },
    { Name: "ABI-Specific Lowering"      , Pass: new(ABILowering) },
    { Name: "Instruction Fusion"         , Pass: new(Fusion) },
    { Name: "Instruction Compaction"     , Pass: new(Compaction) },
    { Name: "Operand Allocation"         , Pass: new(OperandAlloc) },
    { Name: "Phi Propagation"            , Pass: new(PhiProp) },
    { Name: "Constant Rematerialize"     , Pass: new(Rematerialize) },
    { Name: "Pre-allocation TDCE"        , Pass: new(TDCE) },
    // { Name: "Register Allocation"        , Pass: new(RegAlloc) },
}

func toFuncType(fn interface{}) reflect.Type {
    if vt := reflect.TypeOf(fn); vt.Kind() != reflect.Func {
        panic("ssa: fn must be a function prototype")
    } else {
        return vt
    }
}

func executeSSAPasses(cfg *CFG) {
    for _, p := range Passes {
        p.Pass.Apply(cfg)
    }
}

func Compile(p hir.Program, fn interface{}) (cfg *CFG) {
    cfg = newGraphBuilder().build(p)
    cfg.Layout = abi.ABI.LayoutFunc(-1, toFuncType(fn))
    insertPhiNodes(cfg)
    renameRegisters(cfg)
    executeSSAPasses(cfg)
    return
}
