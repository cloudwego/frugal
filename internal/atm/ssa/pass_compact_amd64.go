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

type _LoadMem struct {
    m Mem
    n uint8
}

// Compaction is like Fusion, but it performs the reverse action to reduce redundant operations.
type Compaction struct{}

func (Compaction) Apply(cfg *CFG) {
    next := true
    mems := make(map[Mem]Reg)
    vals := make(map[_LoadMem]Reg)

    /* at this point, all unused definations should have been removed */
    for next {
        next = false
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            rt.MapClear(mems)
            rt.MapClear(vals)

            /* check every instructions, reuse memory addresses as much as possible */
            for i, v := range bb.Ins {
                switch p := v.(type) {
                    default: {
                        break
                    }

                    /* load effective address */
                    case *IrAMD64_LEA: {
                        if _, ok := mems[p.M]; !ok {
                            mems[p.M] = p.R
                        }
                    }

                    /* lea {mem}, %r0; movx {mem}, %r1 --> lea {mem}, %r0; movx (%r0), %r1 */
                    case *IrAMD64_MOV_load: {
                        if m, ok := mems[p.M]; ok {
                            p.M, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if _, ok = vals[_LoadMem { p.M, p.N }]; !ok {
                            vals[_LoadMem { p.M, p.N }] = p.R
                        }
                    }

                    /* movx {mem}, %r0; movbex {mem}, %r1 --> movx {mem}, %r0; bswapx %r0, %r1
                     * leax {mem}, %r0; movbex {mem}, %r1 --> leax {mem}, %r0; movbex (%r0), %r1 */
                    case *IrAMD64_MOV_load_be: {
                        if m, ok := mems[p.M]; ok {
                            p.M, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if r, ok := vals[_LoadMem {p.M, p.N }]; ok {
                            bb.Ins[i], next = &IrAMD64_BSWAP { R: p.R, V: r, N: p.N }, true
                        }
                    }

                    /* lea {mem}, %r0; movx %r1, {mem} --> lea {mem}, %r0; movx %r1, (%r0) */
                    case *IrAMD64_MOV_store_r: {
                        if m, ok := mems[p.M]; ok {
                            p.M, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        }
                    }

                    /* lea {mem}, %r0; movx {imm}, {mem} --> lea {mem}, %r0; movx {imm}, (%r0) */
                    case *IrAMD64_MOV_store_i: {
                        if m, ok := mems[p.M]; ok {
                            p.M, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        }
                    }

                    /* lea {mem}, %r0; movx {ptr}, {mem} --> lea {mem}, %r0; movx {ptr}, (%r0) */
                    case *IrAMD64_MOV_store_p: {
                        if m, ok := mems[p.M]; ok {
                            p.M, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        }
                    }

                    /* lea {mem}, %r0; movbex %r1, {mem} --> lea {mem}, %r0; movbex %r1, (%r0) */
                    case *IrAMD64_MOV_store_be: {
                        if m, ok := mems[p.M]; ok {
                            p.M, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        }
                    }

                    /* movx {mem}, %r0; cmpx %r1, {mem} --> movx {mem}, %r0; cmpx %r1, %r0
                     * leaq {mem}, %r0; cmpx %r1, {mem} --> leaq {mem}, %r0; cmpx %r1, (%r0) */
                    case *IrAMD64_CMPQ_rm: {
                        if m, ok := mems[p.Y]; ok {
                            p.Y, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if v, ok := vals[_LoadMem { p.Y, p.N }]; ok {
                            bb.Ins[i], next = &IrAMD64_CMPQ_rr { X: p.X, Y: v, Op: p.Op }, true
                        }
                    }

                    /* movx {mem}, %r0; cmpx {mem}, %r1 --> movx {mem}, %r0; cmpx %r0, %r1
                     * leaq {mem}, %r0; cmpx {mem}, %r1 --> leaq {mem}, %r0; cmpx (%r0), %r1 */
                    case *IrAMD64_CMPQ_mr: {
                        if m, ok := mems[p.X]; ok {
                            p.X, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if r, ok := vals[_LoadMem { p.X, p.N }]; ok {
                            bb.Ins[i], next = &IrAMD64_CMPQ_rr { X: r, Y: p.Y, Op: p.Op }, true
                        }
                    }

                    /* movx {mem}, %r0; cmpx {mem}, {imm} --> movx {mem}, %r0; cmpx %r0, {imm}
                     * leaq {mem}, %r0; cmpx {mem}, {imm} --> leaq {mem}, %r0; cmpx (%r0), {imm} */
                    case *IrAMD64_CMPQ_mi: {
                        if m, ok := mems[p.X]; ok {
                            p.X, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if r, ok := vals[_LoadMem { p.X, p.N }]; ok {
                            bb.Ins[i], next = &IrAMD64_CMPQ_ri { X: r, Y: p.Y, Op: p.Op }, true
                        }
                    }

                    /* movx {mem}, %r0; cmpx {mem}, {ptr} --> movx {mem}, %r0; cmpx %r0, {ptr}
                     * leaq {mem}, %r0; cmpx {mem}, {ptr} --> leaq {mem}, %r0; cmpx (%r0), {ptr} */
                    case *IrAMD64_CMPQ_mp: {
                        if m, ok := mems[p.X]; ok {
                            p.X, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if r, ok := vals[_LoadMem { p.X, abi.PtrSize }]; ok {
                            bb.Ins[i], next = &IrAMD64_CMPQ_rp { X: r, Y: p.Y, Op: p.Op }, true
                        }
                    }

                    /* movx {mem}, %r0; cmpx {imm}, {mem} --> movx {mem}, %r0; cmpx {imm}, %r0
                     * leaq {mem}, %r0; cmpx {imm}, {mem} --> leaq {mem}, %r0; cmpx {imm}, (%r0) */
                    case *IrAMD64_CMPQ_im: {
                        if m, ok := mems[p.Y]; ok {
                            p.Y, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if r, ok := vals[_LoadMem { p.Y, p.N }]; ok {
                            bb.Ins[i], next = &IrAMD64_CMPQ_ir { X: p.X, Y: r, Op: p.Op }, true
                        }
                    }

                    /* movx {mem}, %r0; cmpx {ptr}, {mem} --> movx {mem}, %r0; cmpx %r0, {ptr}
                     * leaq {mem}, %r0; cmpx {ptr}, {mem} --> leaq {mem}, %r0; cmpx (%r0), {ptr} */
                    case *IrAMD64_CMPQ_pm: {
                        if m, ok := mems[p.Y]; ok {
                            p.Y, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                        } else if r, ok := vals[_LoadMem { p.Y, abi.PtrSize }]; ok {
                            bb.Ins[i], next = &IrAMD64_CMPQ_pr { X: p.X, Y: r, Op: p.Op }, true
                        }
                    }
                }
            }

            /* check for terminators */
            switch p := bb.Term.(type) {
                default: {
                    break
                }

                /* movx {mem}, %r0; cmpx %r1, {mem}; jcc --> movx {mem}, %r0; cmpx %r1, %r0; jcc
                 * leaq {mem}, %r0; cmpx %r1, {mem}; jcc --> leaq {mem}, %r0; cmpx %r1, (%r0); jcc */
                case *IrAMD64_Jcc_rm: {
                    if m, ok := mems[p.Y]; ok {
                        p.Y, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                    } else if r, ok := vals[_LoadMem { p.Y, p.N }]; ok {
                        bb.Term, next = &IrAMD64_Jcc_rr { X: p.X, Y: r, To: p.To, Ln: p.Ln, Op: p.Op }, true
                    }
                }

                /* movx {mem}, %r0; cmpx {mem}, %r1; jcc --> movx {mem}, %r0; cmpx %r0, %r1; jcc
                 * leaq {mem}, %r0; cmpx {mem}, %r1; jcc --> leaq {mem}, %r0; cmpx (%r0), %r1; jcc */
                case *IrAMD64_Jcc_mr: {
                    if m, ok := mems[p.X]; ok {
                        p.X, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                    } else if r, ok := vals[_LoadMem { p.X, p.N }]; ok {
                        bb.Term, next = &IrAMD64_Jcc_rr { X: r, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }, true
                    }
                }

                /* movx {mem}, %r0; cmpx {mem}, {imm}; jcc --> movx {mem}, %r0; cmpx %r0, {imm}; jcc
                 * leaq {mem}, %r0; cmpx {mem}, {imm}; jcc --> leaq {mem}, %r0; cmpx (%r0), {imm}; jcc */
                case *IrAMD64_Jcc_mi: {
                    if m, ok := mems[p.X]; ok {
                        p.X, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                    } else if r, ok := vals[_LoadMem { p.X, p.N }]; ok {
                        bb.Term, next = &IrAMD64_Jcc_ri { X: r, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }, true
                    }
                }

                /* movx {mem}, %r0; cmpx {mem}, {ptr}; jcc --> movx {mem}, %r0; cmpx %r0, {ptr}; jcc
                 * leaq {mem}, %r0; cmpx {mem}, {ptr}; jcc --> leaq {mem}, %r0; cmpx (%r0), {ptr}; jcc */
                case *IrAMD64_Jcc_mp: {
                    if m, ok := mems[p.X]; ok {
                        p.X, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                    } else if r, ok := vals[_LoadMem { p.X, abi.PtrSize }]; ok {
                        bb.Term, next = &IrAMD64_Jcc_rp { X: r, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }, true
                    }
                }

                /* movx {mem}, %r0; cmpx {imm}, {mem}; jcc --> movx {mem}, %r0; cmpx {imm}, %r0; jcc
                 * leaq {mem}, %r0; cmpx {imm}, {mem}; jcc --> leaq {mem}, %r0; cmpx {imm}, (%r0); jcc */
                case *IrAMD64_Jcc_im: {
                    if m, ok := mems[p.Y]; ok {
                        p.Y, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                    } else if r, ok := vals[_LoadMem { p.Y, p.N }]; ok {
                        bb.Term, next = &IrAMD64_Jcc_ir { X: p.X, Y: r, To: p.To, Ln: p.Ln, Op: p.Op }, true
                    }
                }

                /* movx {mem}, %r0; cmpx {ptr}, {mem}; jcc --> movx {mem}, %r0; cmpx %r0, {ptr}; jcc
                 * leaq {mem}, %r0; cmpx {ptr}, {mem}; jcc --> leaq {mem}, %r0; cmpx (%r0), {ptr}; jcc */
                case *IrAMD64_Jcc_pm: {
                    if m, ok := mems[p.Y]; ok {
                        p.Y, next = Mem { M: m, I: Rz, S: 1, D: 0 }, true
                    } else if r, ok := vals[_LoadMem { p.Y, abi.PtrSize }]; ok {
                        bb.Term, next = &IrAMD64_Jcc_pr { X: p.X, Y: r, To: p.To, Ln: p.Ln, Op: p.Op }, true
                    }
                }
            }
        })
    }
}

