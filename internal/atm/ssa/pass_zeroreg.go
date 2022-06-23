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
    `runtime`
)

// ZeroReg replaces read to %z or %nil to a register that was
// initialized to zero for architectures that does not have constant
// zero registers, such as `x86_64`.
type ZeroReg struct{}

func (ZeroReg) replace(cfg *CFG) {
    cfg.PostOrder(func(bb *BasicBlock) {
        var ok bool
        var use IrUsages

        /* create the instruction buffer */
        ins := bb.Ins
        bb.Ins = make([]IrNode, 0, len(ins))

        /* zero register replacer */
        replacez := func(v IrUsages, ins *[]IrNode) {
            var z Reg
            var n Reg

            /* check for every usage of zero registers */
            for _, r := range use.Usages() {
                if r.Kind() == K_zero {
                    if r.Ptr() {
                        if n == 0 {
                            n = cfg.CreateRegister(true)
                            *ins = append(*ins, &IrConstPtr { R: n })
                        }
                    } else {
                        if z == 0 {
                            z = cfg.CreateRegister(false)
                            *ins = append(*ins, &IrConstInt { R: z })
                        }
                    }
                }
            }

            /* substitute all the zero register usages */
            for _, r := range use.Usages() {
                if r.Kind() == K_zero {
                    if r.Ptr() {
                        *r = n
                    } else {
                        *r = z
                    }
                }
            }
        }

        /* scan all the instructions */
        for _, v := range ins {
            if use, ok = v.(IrUsages); ok {
                replacez(use, &bb.Ins)
            }
            bb.Ins = append(bb.Ins, v)
        }

        /* scan the terminator */
        if use, ok = bb.Term.(IrUsages); ok {
            replacez(use, &bb.Ins)
        }
    })
}

//goland:noinspection GoBoolExpressions
func (self ZeroReg) Apply(cfg *CFG) {
    if runtime.GOARCH == "amd64" {
        self.replace(cfg)
    }
}
