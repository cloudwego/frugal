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

// PhiElim replaces Phi nodes that are essentially copies with actual copy operaions.
type PhiElim struct{}

func (PhiElim) Apply(cfg *CFG) {
    cfg.PostOrder(func(bb *BasicBlock) {
        phi := bb.Phi
        bb.Phi = bb.Phi[:0]

        /* scan every Phi node */
        for _, v := range phi {
            ok := true
            rr := Reg(0)

            /* get one of the registers */
            for _, r := range v.V {
                rr = *r
                break
            }

            /* check for Phi node */
            for _, r := range v.V {
                if rr != *r {
                    ok = false
                    break
                }
            }

            /* replace the Phi node with a copy operation if needed */
            if !ok {
                bb.Phi = append(bb.Phi, v)
            } else {
                bb.Ins = append([]IrNode { IrCopy(v.R, rr) }, bb.Ins...)
            }
        }
    })
}
