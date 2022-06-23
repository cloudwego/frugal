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

// Lowering lowers generic SSA IR to arch-dependent SSA IR.
type Lowering struct{}

func (Lowering) Apply(cfg *CFG) {
    cfg.PostOrder(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = make([]IrNode, 0, len(ins))

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
                        N: p.Size,
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
                    bb.Ins = append(bb.Ins, &IrAMD64_MOV_store_r {
                        R: p.R,
                        N: p.Size,
                        M: Mem {
                            M: p.Mem,
                            I: Rz,
                            S: 1,
                            D: 0,
                        },
                    })
                }

                /* load argument by index, ABI specific, will be lowered in later pass */
                case *IrLoadArg: {
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
                            M: p.Mem,
                            I: p.Off,
                            S: 1,
                            D: 0,
                        },
                    })
                }

                /* unary operators */
                case *IrUnaryExpr: {
                    switch p.Op {
                        case IrOpNegate   : bb.Ins = append(bb.Ins, &IrAMD64_NEG    { R: p.R, V: p.V })
                        case IrOpSwap16   : bb.Ins = append(bb.Ins, &IrAMD64_BSWAP  { R: p.R, V: p.V, N: 2 })
                        case IrOpSwap32   : bb.Ins = append(bb.Ins, &IrAMD64_BSWAP  { R: p.R, V: p.V, N: 4 })
                        case IrOpSwap64   : bb.Ins = append(bb.Ins, &IrAMD64_BSWAP  { R: p.R, V: p.V, N: 8 })
                        case IrOpSx32to64 : bb.Ins = append(bb.Ins, &IrAMD64_MOVSLQ { R: p.R, V: p.V })
                        default           : panic("unreachable")
                    }
                }

                /* binary operators */
                case *IrBinaryExpr: {
                    switch p.Op {
                        case IrOpAdd  : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinAdd })
                        case IrOpSub  : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinSub })
                        case IrOpMul  : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinMul })
                        case IrOpAnd  : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinAnd })
                        case IrOpOr   : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinOr  })
                        case IrOpXor  : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinXor })
                        case IrOpShr  : bb.Ins = append(bb.Ins, &IrAMD64_BinOp_rr { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_BinShr })
                        case IrCmpEq  : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_rr  { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_CmpEq  })
                        case IrCmpNe  : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_rr  { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_CmpNe  })
                        case IrCmpLt  : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_rr  { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_CmpLt  })
                        case IrCmpLtu : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_rr  { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_CmpLtu })
                        case IrCmpGeu : bb.Ins = append(bb.Ins, &IrAMD64_CMPQ_rr  { R: p.R, X: p.X, Y: p.Y, Op: IrAMD64_CmpGeu })
                        default       : panic("unreachable")
                    }
                }

                /* bit test and set */
                case *IrBitTestSet: {
                    bb.Ins = append(bb.Ins, &IrAMD64_BTSQ_rr {
                        T: p.T,
                        S: p.S,
                        X: p.X,
                        Y: p.Y,
                    })
                }

                /* subroutine call, ABI specific, will be lowered in later pass */
                case *IrCallFunc: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* native subroutine call, ABI specific, will be lowered in later pass */
                case *IrCallNative: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* interface method call, ABI specific, will be lowered in later pass */
                case *IrCallMethod: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* write barrier, handled in later pass */
                case *IrWriteBarrier: {
                    bb.Ins = append(bb.Ins, p)
                }

                /* breakpoint */
                case *IrBreakpoint: {
                    bb.Ins = append(bb.Ins, &IrAMD64_INT { 3 })
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
                    case 0  : bb.Term = &IrAMD64_JMP    { To: p.Ln }
                    case 1  : bb.Term = &IrAMD64_Jcc_ri { X: p.V, Y: t[0].i, To: p.Br[t[0].i], Ln: p.Ln, Op: IrAMD64_CmpEq }
                    default : break
                }
            }

            /* return terminator, ABI specific, will be lowered in later pass */
            case *IrReturn: {
                break
            }
        }
    })
}
