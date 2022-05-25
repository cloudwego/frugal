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
    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/rt`
)

// Fusion fuses simple instructions into more complex one, to reduce the instruction count.
type Fusion struct{}

func (Fusion) Apply(cfg *CFG) {
    done := false
    defs := make(map[Reg]IrNode)

    /* retry until no more modifications */
    for !done {
        done = true
        rt.MapClear(defs)

        /* check every block */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            var r *Reg
            var ok bool

            /* mark all the definitions in Phi nodes */
            for _, v := range bb.Phi {
                for _, r = range v.Definitions() {
                    if _, ok = defs[*r]; !ok {
                        defs[*r] = v
                    } else {
                        panic("register redefined: " + r.String())
                    }
                }
            }

            /* scan all the instructions */
            for i, v := range bb.Ins {
                var x int64
                var d IrDefinitions

                /* fuse instructions if possible */
                switch p := v.(type) {
                    default: {
                        break
                    }

                    /* movabsq $imm, %r1     ; leaq (%r0,%r1), %r2       --> leaq $imm(%r0), %r2
                     * leaq {mem}, %r0       ; leaq {off}(%r0), %r1      --> leaq {mem+off}, %r1
                     * leaq {off1}(%r0), %r1 ; leaq {off2}(%r1,%r2), %r3 --> leaq {off1+off2}(%r0,%r2), %r3 */
                    case *IrAMD64_LEA: {
                        if p.M.I == Rz {
                            if ins, ok := defs[p.M.M].(*IrAMD64_LEA); ok {
                                if x = int64(p.M.D) + int64(ins.M.D); isi32(x) {
                                    p.M = ins.M
                                    done = false
                                    p.M.D = int32(x)
                                }
                            }
                        } else {
                            if ins, ok := defs[p.M.M].(*IrAMD64_LEA); ok && ins.M.I == Rz {
                                if x = int64(p.M.D) + int64(ins.M.D); isi32(x) {
                                    done = false
                                    p.M.M = ins.M.M
                                    p.M.D = int32(x)
                                }
                            } else if ins, ok := defs[p.M.I].(*IrAMD64_MOV_abs); ok {
                                if x = int64(p.M.D) + ins.V; isi32(x) {
                                    done = false
                                    p.M.I = Rz
                                    p.M.D = int32(x)
                                }
                            }
                        }
                    }

                    /* leaq {mem}, %r0; movx (%r0), %r1 --> movx {mem}, %r1 */
                    case *IrAMD64_MOV_load: {
                        if p.M.I == Rz {
                            if ins, ok := defs[p.M.M].(*IrAMD64_LEA); ok {
                                if x = int64(p.M.D) + int64(ins.M.D); isi32(x) {
                                    p.M = ins.M
                                    done = false
                                    p.M.D = int32(x)
                                }
                            }
                        }
                    }

                    /* leaq {mem}, %r0; movx (%r0), %r1 --> movx %r1, {mem} */
                    case *IrAMD64_MOV_store: {
                        if p.M.I == Rz {
                            if ins, ok := defs[p.M.M].(*IrAMD64_LEA); ok {
                                if x = int64(p.M.D) + int64(ins.M.D); isi32(x) {
                                    p.M = ins.M
                                    done = false
                                    p.M.D = int32(x)
                                }
                            }
                        }
                    }

                    /* movq {imm}, %r0; binop %r0, %r1 --> binop {imm}, %r1 */
                    case *IrAMD64_ADDQ_rr: {
                        if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_ADDQ_ri { R: p.R, X: p.X, Y: ins.V }
                        }
                    }

                    /* movq {imm}, %r0; cmpq %r0, %r1 --> cmpq {imm}, %r1
                     * movx {ptr}, %p0; cmpq %p0, %p1 --> cmpq {ptr}, %p1
                     * movq {mem}, %r0; cmpq %r0, %r1 --> cmpx {mem}, %r1
                     * movq {imm}, %r1; cmpq %r0, %r1 --> cmpq %r0, {imm}
                     * movq {ptr}, %p1; cmpq %p0, %p1 --> cmpq %p0, {ptr}
                     * movx {mem}, %r1; cmpq %r0, %r1 --> cmpx %r0, {mem} */
                    case *IrAMD64_CMPQ_rr: {
                        if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_ir { R: p.R, X: ins.V, Y: p.Y, Op: p.Op }
                        } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_pr { R: p.R, X: ins.P, Y: p.Y, Op: p.Op }
                        } else if ins, ok := defs[p.X].(*IrAMD64_MOV_load); ok && ins.N != 16 {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_mr { R: p.R, X: ins.M, Y: p.Y, Op: p.Op, N: ins.N }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_ri { R: p.R, X: p.X, Y: ins.V, Op: p.Op }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_rp { R: p.R, X: p.X, Y: ins.P, Op: p.Op }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_load); ok && ins.N != 16 {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_rm { R: p.R, X: p.X, Y: ins.M, Op: p.Op, N: ins.N }
                        }
                    }

                    /* movq {imm}, %r0; cmpx %r0, {mem} --> cmpx {imm}, {mem}
                     * movq {ptr}, %p0; cmpq %p0, {mem} --> cmpq {ptr}, {mem} */
                    case *IrAMD64_CMPQ_rm: {
                        if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_im { R: p.R, X: ins.V, Y: p.Y, Op: p.Op, N: p.N }
                        } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_pm { R: p.R, X: ins.P, Y: p.Y, Op: p.Op }
                        }
                    }

                    /* movq {imm}, %r0; cmpx {mem}, %r0 --> cmpx {mem}, {imm}
                     * movq {ptr}, %p0; cmpq {mem}, %p0 --> cmpq {mem}, {ptr} */
                    case *IrAMD64_CMPQ_mr: {
                        if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_mi { R: p.R, X: p.X, Y: ins.V, Op: p.Op, N: p.N }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok && p.N == abi.PtrSize {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_mp { R: p.R, X: p.X, Y: ins.P, Op: p.Op }
                        }
                    }
                }

                /* mark all the definitions in instructions */
                if d, ok = v.(IrDefinitions); ok {
                    for _, r = range d.Definitions() {
                        if _, ok = defs[*r]; !ok {
                            defs[*r] = v
                        } else {
                            panic("register redefined: " + r.String())
                        }
                    }
                }
            }

            /* fuse terminators if possible */
            switch p := bb.Term.(type) {
                default: {
                    break
                }

                /* movq {imm}, %r0; cmpq %r0, %r1; jcc {label} --> cmpq {imm}, %r1; jcc {label}
                 * movq {ptr}, %p0; cmpq %p0, %p1; jcc {label} --> cmpq {ptr}, %p1; jcc {label}
                 * movx {mem}, %r0; cmpq %r0, %r1; jcc {label} --> cmpx {mem}, %r1; jcc {label}
                 * movq {imm}, %r1; cmpq %r0, %r1; jcc {label} --> cmpq %r0, {imm}; jcc {label}
                 * movq {ptr}, %p1; cmpq %p0, %p1; jcc {label} --> cmpq %p0, {ptr}; jcc {label}
                 * movx {mem}, %r1; cmpq %r0, %r1; jcc {label} --> cmpx %r0, {mem}; jcc {label} */
                case *IrAMD64_Jcc_rr: {
                    if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_ir { X: ins.V, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }
                    } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_pr { X: ins.P, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }
                    } else if ins, ok := defs[p.X].(*IrAMD64_MOV_load); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_mr { X: ins.M, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op, N: ins.N }
                    } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_ri { X: p.X, Y: ins.V, To: p.To, Ln: p.Ln, Op: p.Op }
                    } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_rp { X: p.X, Y: ins.P, To: p.To, Ln: p.Ln, Op: p.Op }
                    } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_load); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_rm { X: p.X, Y: ins.M, To: p.To, Ln: p.Ln, Op: p.Op, N: ins.N }
                    }
                }

                /* setcc %r0; cmpq %r0, $0; je {label} --> jncc {label} */
                case *IrAMD64_Jcc_ri: {
                    if p.Y == 0 && p.Op == IrAMD64_CmpEq {
                        if ins, ok := defs[p.X].(*IrAMD64_CMPQ_rr); ok {
                            done = false
                            bb.Term = &IrAMD64_Jcc_rr { X: ins.X, Y: ins.Y, To: p.To, Ln: p.Ln, Op: ins.Op.Negated() }
                        } else if ins, ok := defs[p.X].(*IrAMD64_CMPQ_rm); ok {
                            done = false
                            bb.Term = &IrAMD64_Jcc_rm { X: ins.X, Y: ins.Y, To: p.To, Ln: p.Ln, Op: ins.Op.Negated(), N: ins.N }
                        } else if ins, ok := defs[p.X].(*IrAMD64_CMPQ_mr); ok {
                            done = false
                            bb.Term = &IrAMD64_Jcc_mr { X: ins.X, Y: ins.Y, To: p.To, Ln: p.Ln, Op: ins.Op.Negated(), N: ins.N }
                        }
                    }
                }

                /* setcc %r0; cmpq $0, %r0; je {label} --> jncc {label} */
                case *IrAMD64_Jcc_ir: {
                    if p.Y == 0 && p.Op == IrAMD64_CmpEq {
                        if ins, ok := defs[p.Y].(*IrAMD64_CMPQ_rr); ok {
                            done = false
                            bb.Term = &IrAMD64_Jcc_rr { X: ins.X, Y: ins.Y, To: p.To, Ln: p.Ln, Op: ins.Op.Negated() }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_CMPQ_rm); ok {
                            done = false
                            bb.Term = &IrAMD64_Jcc_rm { X: ins.X, Y: ins.Y, To: p.To, Ln: p.Ln, Op: ins.Op.Negated(), N: ins.N }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_CMPQ_mr); ok {
                            done = false
                            bb.Term = &IrAMD64_Jcc_mr { X: ins.X, Y: ins.Y, To: p.To, Ln: p.Ln, Op: ins.Op.Negated(), N: ins.N }
                        }
                    }
                }

                /* movq {imm}, %r0; cmpq %r0, {mem}; jcc {label} --> cmpq {imm}, {mem}; jcc {label}
                 * movq {ptr}, %p0; cmpq %p0, {mem}; jcc {label} --> cmpq {ptr}, {mem}; jcc {label} */
                case *IrAMD64_Jcc_rm: {
                    if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_im { X: ins.V, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op, N: p.N }
                    } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok && p.N == abi.PtrSize {
                        done = false
                        bb.Term = &IrAMD64_Jcc_pm { X: ins.P, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }
                    }
                }

                /* movq {imm}, %r0; cmpq {mem}, %r0; jcc {label} --> cmpq {mem}, {imm}; jcc {label}
                 * movq {ptr}, %p0; cmpq {mem}, %p0; jcc {label} --> cmpq {mem}, {ptr}; jcc {label} */
                case *IrAMD64_Jcc_mr: {
                    if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_mi { X: p.X, Y: ins.V, To: p.To, Ln: p.Ln, Op: p.Op, N: p.N }
                    } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok && p.N == abi.PtrSize {
                        done = false
                        bb.Term = &IrAMD64_Jcc_mp { X: p.X, Y: ins.P, To: p.To, Ln: p.Ln, Op: p.Op }
                    }
                }
            }
        })
    }
}
