// +build go1.17

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

// ABILowering lowers ABI-specific instructions to
type ABILowering struct{}

func (ABILowering) Apply(cfg *CFG) {
    cfg.PostOrder(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = make([]IrNode, 0, len(ins))

        /* scan every instruction */
        for _, v := range ins {
            switch p := v.(type) {
                default: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* load argument by index */
                case *IrLoadArg: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* subroutine call */
                case *IrCallFunc: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* native subroutine call */
                case *IrCallNative: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* interface method call */
                case *IrCallMethod: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* write-barrier special call */
                case *IrWriteBarrier: {
                    bb.Ins = append(bb.Ins, p)
                }
            }
        }

        /* scan the terminator */
        if p, ok := bb.Term.(*IrReturn); ok {
            _ = p
        }
    })
}

