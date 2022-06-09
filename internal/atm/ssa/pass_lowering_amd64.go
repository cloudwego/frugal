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
    `sort`
    `sync/atomic`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/rt`
)

type _SplitPair struct {
    i  int
    bb *BasicBlock
}

// Lowering lowers generic SSA IR to arch-dependent SSA IR
type Lowering struct{}

func (Lowering) lower(cfg *CFG) {
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

                /* subroutine call */
                case *IrCallFunc: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* native subroutine call */
                case *IrCallNative: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* interface method call */
                case *IrCallMethod: {
                    // TODO: this
                    bb.Ins = append(bb.Ins, p)
                }

                /* write barrier, handled in later stage */
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
                    case 1  : bb.Term = &IrAMD64_Jcc_ri { X: p.V, Y: t[0].i, To: t[0].b, Ln: p.Ln, Op: IrAMD64_CmpEq }
                    default : break
                }
            }

            /* return terminator */
            case *IrReturn: {
                // TODO: this
            }
        }
    })
}

func (Lowering) barrier(cfg *CFG) {
    more := true
    next := uint64(cfg.MaxBlock())
    ptrs := make(map[Reg]unsafe.Pointer)
    mbir := make(map[*BasicBlock]int)

    /* find all constant pointers */
    cfg.PostOrder(func(bb *BasicBlock) {
        for _, v := range bb.Ins {
            if p, ok := v.(*IrAMD64_MOV_ptr); ok {
                ptrs[p.R] = p.P
            }
        }
    })

    /* loop until no more write barriers */
    for more {
        more = false
        rt.MapClear(mbir)

        /* Phase 1: Find all the memory barriers and pointer constants */
        cfg.PostOrder(func(bb *BasicBlock) {
            for i, v := range bb.Ins {
                if _, ok := v.(*IrWriteBarrier); ok {
                    if _, ok = mbir[bb]; ok {
                        more = true
                    } else {
                        mbir[bb] = i
                    }
                }
            }
        })

        /* split pair buffer */
        nb := len(mbir)
        mb := make([]_SplitPair, 0, nb)

        /* extract from the map */
        for p, i := range mbir {
            mb = append(mb, _SplitPair {
                i  : i,
                bb : p,
            })
        }

        /* sort by block ID */
        sort.Slice(mb, func(i int, j int) bool {
            return mb[i].bb.Id < mb[i].bb.Id
        })

        /* Phase 2: Split basic block at write barrier */
        for _, p := range mb {
            bb := new(BasicBlock)
            ds := new(BasicBlock)
            wb := new(BasicBlock)
            ir := p.bb.Ins[p.i].(*IrWriteBarrier)

            /* move instructions after the write barrier into a new block */
            bb.Id   = int(atomic.AddUint64(&next, 1))
            bb.Ins  = p.bb.Ins[p.i + 1:]
            bb.Term = p.bb.Term
            bb.Pred = []*BasicBlock { ds, wb }

            /* update all the predecessors & Phi nodes */
            for it := p.bb.Term.Successors(); it.Next(); {
                succ := it.Block()
                pred := succ.Pred

                /* update predecessors */
                for x, v := range pred {
                    if v == p.bb {
                        pred[x] = bb
                        break
                    }
                }

                /* update Phi nodes */
                for _, phi := range succ.Phi {
                    phi.V[bb] = phi.V[p.bb]
                    delete(phi.V, p.bb)
                }
            }

            /* rewrite the direct store instruction */
            st := &IrAMD64_MOV_store_r {
                R: ir.R,
                M: Mem { M: ir.M, I: Rz, S: 1, D: 0 },
                N: abi.PtrSize,
            }

            /* construct the direct store block */
            ds.Id   = int(atomic.AddUint64(&next, 1))
            ds.Ins  = []IrNode { st }
            ds.Term = &IrSwitch { Ln: bb }
            ds.Pred = []*BasicBlock { p.bb }

            /* rewrite the write barrier instruction */
            fn := &IrAMD64_MOV_wb {
                R  : ir.R,
                M  : ir.M,
                Fn : ptrs[ir.Fn],
            }

            /* function address must exist */
            if fn.Fn == nil {
                panic("missing write barrier function address")
            }

            /* construct the write barrier block */
            wb.Id   = int(atomic.AddUint64(&next, 1))
            wb.Ins  = []IrNode { fn }
            wb.Term = &IrSwitch { Ln: bb }
            wb.Pred = []*BasicBlock { p.bb }

            /* rewrite the terminator to check for write barrier */
            p.bb.Ins  = p.bb.Ins[:p.i]
            p.bb.Term = &IrAMD64_Jcc_mi {
                X  : Mem { M: ir.Var, I: Rz, S: 1, D: 0 },
                Y  : 0,
                N  : 1,
                To : wb,
                Ln : ds,
                Op : IrAMD64_CmpNe,
            }
        }

        /* Phase 3: Rebuild the CFG */
        if len(mbir) != 0 {
            cfg.Rebuild()
        }
    }
}

func (self Lowering) Apply(cfg *CFG) {
    self.lower(cfg)
    self.barrier(cfg)
}
