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
    `github.com/cloudwego/frugal/internal/cpu`
    `github.com/cloudwego/frugal/internal/rt`
)

// Fusion fuses simple instructions into more complex one, to reduce the instruction count.
type Fusion struct{}

func (Fusion) flagsafe(bb *BasicBlock, ins IrNode) bool {
    i := -1
    p := IrNode(nil)

    /* find the instruction */
    for i, p = range bb.Ins {
        if p == ins {
            break
        }
    }

    /* not found, the instruction is in another basic
     * block, which cannot guarantee it's flags preserving */
    if p != ins {
        return false
    }

    /* check for instructions after it, only some instructions that are
     * known to preserve flags, all other instructions are assumed to clobber */
    for _, p = range bb.Ins[i + 1:] {
        switch p.(type) {
            case *IrAMD64_INT          : break
            case *IrAMD64_LEA          : break
            case *IrAMD64_BSWAP        : break
            case *IrAMD64_MOVSLQ       : break
            case *IrAMD64_MOV_abs      : break
            case *IrAMD64_MOV_ptr      : break
            case *IrAMD64_MOV_reg      : break
            case *IrAMD64_MOV_load     : break
            case *IrAMD64_MOV_store_r  : break
            case *IrAMD64_MOV_store_i  : break
            case *IrAMD64_MOV_load_be  : break
            case *IrAMD64_MOV_store_be : break
            case *IrAMD64_CALL_gcwb    : break
            default                    : return false
        }
    }

    /* everything checked fine */
    return true
}

func (self Fusion) Apply(cfg *CFG) {
    done := false
    defs := make(map[Reg]IrNode)

    /* leaq {mem}, %r1       ; op {disp}(%r1), %r2             --> op {disp}{mem}, %r2
     * leaq {off1}(%r0), %r2 ; op {off2}(%r1,%r2), %r3         --> op {off1+off2}(%r1,%r0), %r3
     * leaq {off1}(%r0), %r1 ; op {off2}(%r1,%r2,{scale}), %r3 --> op {off1+off2}(%r0,%r2,{scale}), %r3
     * addsub $imm, %r1, %r2 ; op {disp}(%r3,%r2), %r4         --> op {disp+imm}(%r3,%r1), %r4
     * movabsq $imm, %r1     ; op {disp}(%r0,%r1,{scale}), %r2 --> op {disp+imm*scale}(%r0), %r2 */
    fusemem := func(m *Mem) {
        if m.I == Rz {
            if ins, ok := defs[m.M].(*IrAMD64_LEA); ok {
                if x := int64(m.D) + int64(ins.M.D); isi32(x) {
                    m.M = ins.M.M
                    m.I = ins.M.I
                    m.S = ins.M.S
                    m.D = int32(x)
                    done = false
                }
            }
        } else {
            if ins, ok := defs[m.M].(*IrAMD64_LEA); ok && ins.M.I == Rz {
                if x := int64(m.D) + int64(ins.M.D); isi32(x) {
                    m.M = ins.M.M
                    m.D = int32(x)
                    done = false
                }
            } else if ins, ok := defs[m.I].(*IrAMD64_LEA); ok && m.S == 1 && ins.M.I == Rz {
                if x := int64(m.D) + int64(ins.M.D); isi32(x) {
                    m.I = ins.M.M
                    m.D = int32(x)
                    done = false
                }
            } else if ins, ok := defs[m.I].(*IrAMD64_BinOp_ri); ok && m.S == 1 && ins.Op.IsAdditive() {
                if x := int64(m.D) + int64(ins.Y * ins.Op.ScaleFactor()); isi32(x) {
                    m.I = ins.X
                    m.D = int32(x)
                    done = false
                }
            } else if ins, ok := defs[m.I].(*IrAMD64_MOV_abs); ok {
                if x := int64(m.D) + ins.V; isi32(x) {
                    m.I = Rz
                    m.D = int32(x)
                    done = false
                }
            }
        }
    }

    /* retry until no more modifications */
    for !done {
        done = true
        rt.MapClear(defs)

        /* pseudo-definition for zero registers */
        defs[Rz] = &IrAMD64_MOV_abs { R: Rz, V: 0 }
        defs[Pn] = &IrAMD64_MOV_ptr { R: Rz, P: nil }

        /* check every block */
        for _, bb := range cfg.PostOrder().Reversed() {
            var r *Reg
            var ok bool

            /* mark all the definitions in Phi nodes */
            for _, v := range bb.Phi {
                for _, r = range v.Definitions() {
                    if _, ok = defs[*r]; !ok {
                        defs[*r] = v
                    } else if r.Kind() != K_zero {
                        panic("register redefined: " + r.String())
                    }
                }
            }

            /* scan all the instructions */
            for i, v := range bb.Ins {
                var m IrAMD64_MemOp
                var d IrDefinitions

                /* fuse memory addresses in instructions */
                if m, ok = v.(IrAMD64_MemOp); ok {
                    fusemem(m.MemOp())
                }

                /* fuse instructions if possible */
                switch p := v.(type) {
                    default: {
                        break
                    }

                    /* movx {mem}, %r0; bswapx %r0, %r1 --> movbex {mem}, %r1 */
                    case *IrAMD64_BSWAP: {
                        if ins, ok := defs[p.V].(*IrAMD64_MOV_load); ok && ins.N != 1 && cpu.HasMOVBE {
                            done = false
                            bb.Ins[i] = &IrAMD64_MOV_load_be { R: p.R, M: ins.M, N: ins.N }
                        }
                    }

                    /* movq {i32}, %r0; movx %r0, {mem} --> movx {i32}, {mem}
                     * movq {p32}, %r0; movx %r0, {mem} --> movx {p32}, {mem}
                     * bswapx %r0, %r1; movx %r1, {mm1} --> movbex %r0, {mm1} */
                    case *IrAMD64_MOV_store_r: {
                        if ins, ok := defs[p.R].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_MOV_store_i { V: int32(ins.V), M: p.M, N: p.N }
                        } else if ins, ok := defs[p.R].(*IrAMD64_MOV_ptr); ok && isp32(ins.P) && p.N == abi.PtrSize {
                            done = false
                            bb.Ins[i] = &IrAMD64_MOV_store_p { P: ins.P, M: p.M }
                        } else if ins, ok := defs[p.R].(*IrAMD64_BSWAP); ok && p.N != 1 && cpu.HasMOVBE {
                            done = false
                            bb.Ins[i] = &IrAMD64_MOV_store_be { R: ins.V, M: p.M, N: p.N }
                        }
                    }

                    /* movq {i32}, %r0; binop %r0, %r1 --> binop {i32}, %r1
                     * movq {mem}, %r0; binop %r0, %r1 --> binop {mem}, %r1 */
                    case *IrAMD64_BinOp_rr: {
                        if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_BinOp_ri { R: p.R, X: p.X, Y: int32(ins.V), Op: p.Op }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_load); ok && ins.N == abi.PtrSize {
                            done = false
                            bb.Ins[i] = &IrAMD64_BinOp_rm { R: p.R, X: p.X, Y: ins.M, Op: p.Op }
                        }
                    }

                    /* movq {u8}, %r0; btsq %r0, %r1; setc %r2 --> btsq {u8}, %r1; setc %r2 */
                    case *IrAMD64_BTSQ_rr: {
                        if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok && isu8(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_BTSQ_ri { T: p.T, S: p.S, X: p.X, Y: uint8(ins.V) }
                        }
                    }

                    /* movq {i32}, %r0; cmpq %r0, %r1 --> cmpq {i32}, %r1
                     * movx {ptr}, %p0; cmpq %p0, %p1 --> cmpq {ptr}, %p1
                     * movq {mem}, %r0; cmpq %r0, %r1 --> cmpx {mem}, %r1
                     * movq {i32}, %r1; cmpq %r0, %r1 --> cmpq %r0, {i32}
                     * movq {ptr}, %p1; cmpq %p0, %p1 --> cmpq %p0, {ptr}
                     * movx {mem}, %r1; cmpq %r0, %r1 --> cmpx %r0, {mem} */
                    case *IrAMD64_CMPQ_rr: {
                        if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_ir { R: p.R, X: int32(ins.V), Y: p.Y, Op: p.Op }
                        } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_pr { R: p.R, X: ins.P, Y: p.Y, Op: p.Op }
                        } else if ins, ok := defs[p.X].(*IrAMD64_MOV_load); ok && ins.N != 16 {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_mr { R: p.R, X: ins.M, Y: p.Y, Op: p.Op, N: ins.N }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_ri { R: p.R, X: p.X, Y: int32(ins.V), Op: p.Op }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_rp { R: p.R, X: p.X, Y: ins.P, Op: p.Op }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_load); ok && ins.N != 16 {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_rm { R: p.R, X: p.X, Y: ins.M, Op: p.Op, N: ins.N }
                        }
                    }

                    /* movq {i32}, %r0; cmpx %r0, {mem} --> cmpx {i32}, {mem}
                     * movq {p32}, %p0; cmpq %p0, {mem} --> cmpq {p32}, {mem} */
                    case *IrAMD64_CMPQ_rm: {
                        if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_im { R: p.R, X: int32(ins.V), Y: p.Y, Op: p.Op, N: p.N }
                        } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok && isp32(ins.P) && p.N == abi.PtrSize {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_pm { R: p.R, X: ins.P, Y: p.Y, Op: p.Op }
                        }
                    }

                    /* movq {i32}, %r0; cmpx {mem}, %r0 --> cmpx {mem}, {i32}
                     * movq {p32}, %p0; cmpq {mem}, %p0 --> cmpq {mem}, {p32} */
                    case *IrAMD64_CMPQ_mr: {
                        if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                            done = false
                            bb.Ins[i] = &IrAMD64_CMPQ_mi { R: p.R, X: p.X, Y: int32(ins.V), Op: p.Op, N: p.N }
                        } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok && isp32(ins.P) && p.N == abi.PtrSize {
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
                        } else if r.Kind() != K_zero {
                            panic("register redefined: " + r.String())
                        }
                    }
                }
            }

            /* fuse memory operation in terminators */
            if m, ok := bb.Term.(IrAMD64_MemOp); ok {
                fusemem(m.MemOp())
            }

            /* fuse terminators if possible */
            switch p := bb.Term.(type) {
                default: {
                    break
                }

                /* movq {i32}, %r0; cmpq %r0, %r1; jcc {label} --> cmpq {i32}, %r1; jcc {label}
                 * movq {ptr}, %p0; cmpq %p0, %p1; jcc {label} --> cmpq {ptr}, %p1; jcc {label}
                 * movx {mem}, %r0; cmpq %r0, %r1; jcc {label} --> cmpx {mem}, %r1; jcc {label}
                 * movq {i32}, %r1; cmpq %r0, %r1; jcc {label} --> cmpq %r0, {i32}; jcc {label}
                 * movq {ptr}, %p1; cmpq %p0, %p1; jcc {label} --> cmpq %p0, {ptr}; jcc {label}
                 * movx {mem}, %r1; cmpq %r0, %r1; jcc {label} --> cmpx %r0, {mem}; jcc {label} */
                case *IrAMD64_Jcc_rr: {
                    if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                        done = false
                        bb.Term = &IrAMD64_Jcc_ir { X: int32(ins.V), Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }
                    } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_pr { X: ins.P, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }
                    } else if ins, ok := defs[p.X].(*IrAMD64_MOV_load); ok {
                        done = false
                        bb.Term = &IrAMD64_Jcc_mr { X: ins.M, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op, N: ins.N }
                    } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                        done = false
                        bb.Term = &IrAMD64_Jcc_ri { X: p.X, Y: int32(ins.V), To: p.To, Ln: p.Ln, Op: p.Op }
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
                        } else if ins, ok := defs[p.X].(*IrAMD64_BTSQ_rr); ok && p.X == ins.T && self.flagsafe(bb, ins) {
                            done = false
                            bb.Term, ins.T = &IrAMD64_JNC { To: p.To, Ln: p.Ln }, Rz
                        } else if ins, ok := defs[p.X].(*IrAMD64_BTSQ_ri); ok && p.X == ins.T && self.flagsafe(bb, ins) {
                            done = false
                            bb.Term, ins.T = &IrAMD64_JNC { To: p.To, Ln: p.Ln }, Rz
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
                        } else if ins, ok := defs[p.Y].(*IrAMD64_BTSQ_rr); ok && p.Y == ins.T && self.flagsafe(bb, ins) {
                            done = false
                            bb.Term, ins.T = &IrAMD64_JNC { To: p.To, Ln: p.Ln }, Rz
                        } else if ins, ok := defs[p.Y].(*IrAMD64_BTSQ_ri); ok && p.Y == ins.T && self.flagsafe(bb, ins) {
                            done = false
                            bb.Term, ins.T = &IrAMD64_JNC { To: p.To, Ln: p.Ln }, Rz
                        }
                    }
                }

                /* movq {i32}, %r0; cmpq %r0, {mem}; jcc {label} --> cmpq {i32}, {mem}; jcc {label}
                 * movq {p32}, %p0; cmpq %p0, {mem}; jcc {label} --> cmpq {p32}, {mem}; jcc {label} */
                case *IrAMD64_Jcc_rm: {
                    if ins, ok := defs[p.X].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                        done = false
                        bb.Term = &IrAMD64_Jcc_im { X: int32(ins.V), Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op, N: p.N }
                    } else if ins, ok := defs[p.X].(*IrAMD64_MOV_ptr); ok && isp32(ins.P) && p.N == abi.PtrSize {
                        done = false
                        bb.Term = &IrAMD64_Jcc_pm { X: ins.P, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }
                    }
                }

                /* movq {i32}, %r0; cmpq {mem}, %r0; jcc {label} --> cmpq {mem}, {i32}; jcc {label}
                 * movq {p32}, %p0; cmpq {mem}, %p0; jcc {label} --> cmpq {mem}, {p32}; jcc {label} */
                case *IrAMD64_Jcc_mr: {
                    if ins, ok := defs[p.Y].(*IrAMD64_MOV_abs); ok && isi32(ins.V) {
                        done = false
                        bb.Term = &IrAMD64_Jcc_mi { X: p.X, Y: int32(ins.V), To: p.To, Ln: p.Ln, Op: p.Op, N: p.N }
                    } else if ins, ok := defs[p.Y].(*IrAMD64_MOV_ptr); ok && isp32(ins.P) && p.N == abi.PtrSize {
                        done = false
                        bb.Term = &IrAMD64_Jcc_mp { X: p.X, Y: ins.P, To: p.To, Ln: p.Ln, Op: p.Op }
                    }
                }
            }
        }

        /* perform TDCE & reorder after each round */
        new(TDCE).Apply(cfg)
        new(Reorder).Apply(cfg)
    }
}
