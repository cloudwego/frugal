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

// Lowering lowers generic SSA IR to arch-dependent SSA IR
type Lowering struct{}

func (Lowering) Apply(cfg *CFG) {
    cfg.ReversePostOrder(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = bb.Ins[:0]

        /* lower every instruction */
        for _, v := range ins {
            switch p := v.(type) {
                default: {
                    panic("invalid instruction: " + v.String())
                }

                /* load from memory */
                case *IrLoad: {
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_load {
                        R: p.R,
                        S: p.Size,
                        M: Mem {
                            M: p.Mem,
                            I: Rz,
                            S: 1,
                            D: 0,
                        },
                    })
                }

                /* store into memory */
                case *IrStore: {
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_store {
                        R: p.R,
                        S: p.Size,
                        M: Mem {
                            M: p.Mem,
                            I: Rz,
                            S: 1,
                            D: 0,
                        },
                    })
                }

                /* load argument by index */
                case *IrLoadArg: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* load constant into register */
                case *IrConstInt: {
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_abs {
                        R: p.R,
                        V: p.V,
                    })
                }

                /* load pointer constant into register */
                case *IrConstPtr: {
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_ptr {
                        R: p.R,
                        P: p.P,
                    })
                }

                /* load effective address */
                case *IrLEA: {
                    bb.Ins = append(bb.Ins, &IrAMD64_LEA {
                        R: p.R,
                        M: Mem {
                            M: p.R,
                            I: p.Off,
                            S: 1,
                            D: 0,
                        },
                    })
                }

                /* unary operators */
                case *IrUnaryExpr: {
                    switch p.Op {
                        case IrOpNegate   : bb.Ins = append(bb.Ins, &IrAMD64_NEG { R: p.R, V: p.V })
                        case IrOpSwap16   : bb.Ins = append(bb.Ins, &IrAMD64_BSWAP { R: p.R, V: p.V, S: 2 })
                        case IrOpSwap32   : bb.Ins = append(bb.Ins, &IrAMD64_BSWAP { R: p.R, V: p.V, S: 4 })
                        case IrOpSwap64   : bb.Ins = append(bb.Ins, &IrAMD64_BSWAP { R: p.R, V: p.V, S: 8 })
                        case IrOpSx32to64 : bb.Ins = append(bb.Ins, &IrAMD64_MOVSLQ { R: p.R, V: p.V })
                        default           : panic("unreachable")
                    }
                }

                /* binary operators */
                case *IrBinaryExpr: {
                    switch p.Op {
                        case IrOpAdd  : bb.Ins = append(bb.Ins, &IrAMD64_ADDQ { R: p.R, X: p.X, Y: p.Y })
                        case IrOpSub  : bb.Ins = append(bb.Ins, &IrAMD64_SUBQ { R: p.R, X: p.X, Y: p.Y })
                        case IrOpMul  : bb.Ins = append(bb.Ins, &IrAMD64_IMULQ { R: p.R, X: p.X, Y: p.Y })
                        case IrOpAnd  : bb.Ins = append(bb.Ins, &IrAMD64_ANDQ { R: p.R, X: p.X, Y: p.Y })
                        case IrOpOr   : bb.Ins = append(bb.Ins, &IrAMD64_ORQ { R: p.R, X: p.X, Y: p.Y })
                        case IrOpXor  : bb.Ins = append(bb.Ins, &IrAMD64_XORQ { R: p.R, X: p.X, Y: p.Y })
                        case IrOpShr  : bb.Ins = append(bb.Ins, &IrAMD64_SHRQ { R: p.R, X: p.X, Y: p.Y })
                        case IrCmpEq  : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_eq { R: p.R, X: p.X, Y: p.Y })
                        case IrCmpNe  : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_ne { R: p.R, X: p.X, Y: p.Y })
                        case IrCmpLt  : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_lt { R: p.R, X: p.X, Y: p.Y })
                        case IrCmpLtu : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_ltu { R: p.R, X: p.X, Y: p.Y })
                        case IrCmpGeu : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_geu { R: p.R, X: p.X, Y: p.Y })
                        default       : panic("unreachable")
                    }
                }

                /* bit test and set */
                case *IrBitTestSet: {
                    bb.Ins = append(bb.Ins, &IrAMD64_BTSQ {
                        T: p.T,
                        S: p.S,
                        X: p.X,
                        Y: p.Y,
                    })
                }

                /* subroutine call */
                case *IrCall: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* fill block with zeros */
                case *IrBlockZero: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* memory copy */
                case *IrBlockCopy: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* breakpoint */
                case *IrBreakpoint: {
                    bb.Ins = append(bb.Ins, &IrAMD64_INT {
                        I: 3,
                    })
                }
            }
        }

        /* lower the terminator */
        switch p := bb.Term.(type) {
            default: {
                panic("invalid terminator: " + bb.Term.String())
            }

            /* branch terminator */
            case *IrSwitch: {
                switch t := p.iter().t; len(p.Br) {
                    case 0: bb.Term = &IrAMD64_JMP { To: p.Ln }
                    case 1: bb.Term = &IrAMD64_JE_imm { R: p.V, V: t[0].i, To: t[0].b, Ln: p.Ln }
                }
            }

            /* return terminator */
            case *IrReturn: {
                // TODO: this
            }
        }
    })
}
