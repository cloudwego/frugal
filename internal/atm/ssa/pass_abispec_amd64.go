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

import (
    `fmt`

    `github.com/cloudwego/frugal/internal/atm/abi`
)

// ABILowering lowers ABI-specific instructions to
type ABILowering struct{}

func (ABILowering) lower(cfg *CFG) {
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
                    if p.I < 0 || p.I >= len(cfg.Layout.Args) {
                        panic(fmt.Sprintf("abi: argument %d out of bound", p.I))
                    } else if arg := cfg.Layout.Args[p.I]; arg.InRegister {
                        bb.Ins = append(bb.Ins, &IrAMD64_MOV_reg { R: p.R, V: IrSetArch(Ra, arg.Reg) })
                    } else {
                        bb.Ins = append(bb.Ins, &IrAMD64_MOV_load_stack { R: p.R, S: arg.Mem, K: IrSlotArgs })
                    }
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
            var i int
            var r Reg
            var b []Reg

            /* copy return values */
            for i, r = range p.R {
                if i >= len(cfg.Layout.Args) {
                    panic(fmt.Sprintf("abi: return value %d out of bound", i))
                } else if ret := cfg.Layout.Rets[i]; ret.InRegister {
                    b = append(b, IrSetArch(r, ret.Reg))
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_reg { R: IrSetArch(r, ret.Reg), V: r })
                } else {
                    b = append(b, r)
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_store_stack { R: r, S: ret.Mem, K: IrSlotArgs })
                }
            }

            /* replace the terminator */
            bb.Term = &IrAMD64_RET {
                R: b,
            }
        }
    })
}

func (ABILowering) entry(cfg *CFG) {
    var r []Reg
    var p abi.Parameter

    /* extract all the args */
    for _, p = range cfg.Layout.Args {
        if p.InRegister {
            r = append(r, IrSetArch(Ra, p.Reg))
        }
    }

    /* insert an entry point node */
    cfg.Root.Ins = append(
        []IrNode { &IrEntry { r } },
        cfg.Root.Ins...
    )
}

func (self ABILowering) Apply(cfg *CFG) {
    self.lower(cfg)
    self.entry(cfg)
}
