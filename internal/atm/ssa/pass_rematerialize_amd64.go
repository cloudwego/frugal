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

// Rematerialize recalculates simple values to reduce register pressure.
type Rematerialize struct{}

func (Rematerialize) Apply(cfg *CFG) {
    consts := make(map[Reg]_ConstData)
    consts[Rz] = constint(0)
    consts[Pn] = constptr(nil, Const)

    /* Phase 1: Scan all the constants */
    for _, bb := range cfg.PostOrder().Reversed() {
        for _, v := range bb.Ins {
            switch p := v.(type) {
                case *IrAMD64_MOV_abs: consts[p.R] = constint(p.V)
                case *IrAMD64_MOV_ptr: consts[p.R] = constptr(p.P, Volatile)
            }
        }
    }

    /* Phase 2: Replace register copies with consts if possible */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        for i, v := range bb.Ins {
            if p, ok := v.(*IrAMD64_MOV_reg); ok {
                if cc, ok := consts[p.V]; ok {
                    if cc.i {
                        bb.Ins[i] = &IrAMD64_MOV_abs { R: p.R, V: cc.v }
                    } else {
                        bb.Ins[i] = &IrAMD64_MOV_ptr { R: p.R, P: cc.p }
                    }
                }
            }
        }
    })
}
