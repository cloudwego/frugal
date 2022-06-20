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

// PhiProp propagates Phi nodes into it's source blocks,
// essentially get rid of them.
// The CFG is no longer in SSA form after this pass.
type PhiProp struct{}

func (PhiProp) Apply(cfg *CFG) {
    cfg.PostOrder(func(bb *BasicBlock) {
        vv := bb.Phi
        bb.Phi = nil

        /* process each Phi node */
        for _, p := range vv {
            for b, r := range p.V {
                b.Ins = append(b.Ins, IrCopyArch(p.R, *r))
            }
        }
    })
}
