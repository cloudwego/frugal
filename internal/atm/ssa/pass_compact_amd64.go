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
)

type _MemAddr struct {
    m Mem
    n uint8
}

func memaddr(m Mem, n uint8) _MemAddr {
    return _MemAddr { m, n }
}

type _MemTree struct {
    next *_MemTree
    mems map[Mem]Reg
}

func (self *_MemTree) add(mm Mem, rr Reg) {
    if _, ok := self.mems[mm]; ok {
        panic("memtree: memory operand conflict: " + mm.String())
    } else {
        self.mems[mm] = rr
    }
}

func (self *_MemTree) find(mm Mem) (Reg, bool) {
    if rr, ok := self.mems[mm]; ok {
        return rr, true
    } else if self.next == nil {
        return 0, false
    } else {
        return self.next.find(mm)
    }
}

func (self *_MemTree) derive() *_MemTree {
    return &_MemTree {
        next: self,
        mems: make(map[Mem]Reg),
    }
}

// Compaction is like Fusion, but it performs the reverse action to reduce redundant operations.
type Compaction struct{}

func (self Compaction) dfs(cfg *CFG, bb *BasicBlock, mems *_MemTree, next *bool) {
    id := bb.Id
    vv := make(map[_MemAddr]Reg)

    /* check every instructions, reuse memory addresses as much as possible */
    for i, v := range bb.Ins {
        switch p := v.(type) {
            default: {
                break
            }

            /* load effective address */
            case *IrAMD64_LEA: {
                mems.add(p.M, p.R)
            }

            /* lea {mem}, %r0; movx {mem}, %r1 --> lea {mem}, %r0; movx (%r0), %r1 */
            case *IrAMD64_MOV_load: {
                if m, ok := mems.find(p.M); ok {
                    p.M, *next = Ptr(m, 0), true
                } else if _, ok = vv[memaddr(p.M, p.N)]; !ok {
                    vv[memaddr(p.M, p.N)] = p.R
                }
            }

            /* movx {mem}, %r0; movbex {mem}, %r1 --> movx {mem}, %r0; bswapx %r0, %r1
             * leax {mem}, %r0; movbex {mem}, %r1 --> leax {mem}, %r0; movbex (%r0), %r1 */
            case *IrAMD64_MOV_load_be: {
                if m, ok := mems.find(p.M); ok {
                    p.M, *next = Ptr(m, 0), true
                } else if r, ok := vv[memaddr(p.M, p.N)]; ok {
                    bb.Ins[i], *next = &IrAMD64_BSWAP { R: p.R, V: r, N: p.N }, true
                }
            }

            /* lea {mem}, %r0; movx %r1, {mem} --> lea {mem}, %r0; movx %r1, (%r0) */
            case *IrAMD64_MOV_store_r: {
                if m, ok := mems.find(p.M); ok {
                    p.M, *next = Ptr(m, 0), true
                    delete(vv, memaddr(p.M, p.N))
                }
            }

            /* lea {mem}, %r0; movx {imm}, {mem} --> lea {mem}, %r0; movx {imm}, (%r0) */
            case *IrAMD64_MOV_store_i: {
                if m, ok := mems.find(p.M); ok {
                    p.M, *next = Ptr(m, 0), true
                    delete(vv, memaddr(p.M, p.N))
                }
            }

            /* lea {mem}, %r0; movx {ptr}, {mem} --> lea {mem}, %r0; movx {ptr}, (%r0) */
            case *IrAMD64_MOV_store_p: {
                if m, ok := mems.find(p.M); ok {
                    p.M, *next = Ptr(m, 0), true
                    delete(vv, memaddr(p.M, abi.PtrSize))
                }
            }

            /* lea {mem}, %r0; movbex %r1, {mem} --> lea {mem}, %r0; movbex %r1, (%r0) */
            case *IrAMD64_MOV_store_be: {
                if m, ok := mems.find(p.M); ok {
                    p.M, *next = Ptr(m, 0), true
                    delete(vv, memaddr(p.M, p.N))
                }
            }

            /* movx {mem}, %r0; cmpx %r1, {mem} --> movx {mem}, %r0; cmpx %r1, %r0
             * leaq {mem}, %r0; cmpx %r1, {mem} --> leaq {mem}, %r0; cmpx %r1, (%r0) */
            case *IrAMD64_CMPQ_rm: {
                if m, ok := mems.find(p.Y); ok {
                    p.Y, *next = Ptr(m, 0), true
                } else if v, ok := vv[memaddr(p.Y, p.N)]; ok {
                    bb.Ins[i], *next = &IrAMD64_CMPQ_rr { X: p.X, Y: v, Op: p.Op }, true
                }
            }

            /* movx {mem}, %r0; cmpx {mem}, %r1 --> movx {mem}, %r0; cmpx %r0, %r1
             * leaq {mem}, %r0; cmpx {mem}, %r1 --> leaq {mem}, %r0; cmpx (%r0), %r1 */
            case *IrAMD64_CMPQ_mr: {
                if m, ok := mems.find(p.X); ok {
                    p.X, *next = Ptr(m, 0), true
                } else if r, ok := vv[memaddr(p.X, p.N)]; ok {
                    bb.Ins[i], *next = &IrAMD64_CMPQ_rr { X: r, Y: p.Y, Op: p.Op }, true
                }
            }

            /* movx {mem}, %r0; cmpx {mem}, {imm} --> movx {mem}, %r0; cmpx %r0, {imm}
             * leaq {mem}, %r0; cmpx {mem}, {imm} --> leaq {mem}, %r0; cmpx (%r0), {imm} */
            case *IrAMD64_CMPQ_mi: {
                if m, ok := mems.find(p.X); ok {
                    p.X, *next = Ptr(m, 0), true
                } else if r, ok := vv[memaddr(p.X, p.N)]; ok {
                    bb.Ins[i], *next = &IrAMD64_CMPQ_ri { X: r, Y: p.Y, Op: p.Op }, true
                }
            }

            /* movx {mem}, %r0; cmpx {mem}, {ptr} --> movx {mem}, %r0; cmpx %r0, {ptr}
             * leaq {mem}, %r0; cmpx {mem}, {ptr} --> leaq {mem}, %r0; cmpx (%r0), {ptr} */
            case *IrAMD64_CMPQ_mp: {
                if m, ok := mems.find(p.X); ok {
                    p.X, *next = Ptr(m, 0), true
                } else if r, ok := vv[memaddr(p.X, abi.PtrSize)]; ok {
                    bb.Ins[i], *next = &IrAMD64_CMPQ_rp { X: r, Y: p.Y, Op: p.Op }, true
                }
            }

            /* movx {mem}, %r0; cmpx {imm}, {mem} --> movx {mem}, %r0; cmpx {imm}, %r0
             * leaq {mem}, %r0; cmpx {imm}, {mem} --> leaq {mem}, %r0; cmpx {imm}, (%r0) */
            case *IrAMD64_CMPQ_im: {
                if m, ok := mems.find(p.Y); ok {
                    p.Y, *next = Ptr(m, 0), true
                } else if r, ok := vv[memaddr(p.Y, p.N)]; ok {
                    bb.Ins[i], *next = &IrAMD64_CMPQ_ir { X: p.X, Y: r, Op: p.Op }, true
                }
            }

            /* movx {mem}, %r0; cmpx {ptr}, {mem} --> movx {mem}, %r0; cmpx %r0, {ptr}
             * leaq {mem}, %r0; cmpx {ptr}, {mem} --> leaq {mem}, %r0; cmpx (%r0), {ptr} */
            case *IrAMD64_CMPQ_pm: {
                if m, ok := mems.find(p.Y); ok {
                    p.Y, *next = Ptr(m, 0), true
                } else if r, ok := vv[memaddr(p.Y, abi.PtrSize)]; ok {
                    bb.Ins[i], *next = &IrAMD64_CMPQ_pr { X: p.X, Y: r, Op: p.Op }, true
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
            if m, ok := mems.find(p.Y); ok {
                p.Y, *next = Ptr(m, 0), true
            } else if r, ok := vv[memaddr(p.Y, p.N)]; ok {
                bb.Term, *next = &IrAMD64_Jcc_rr { X: p.X, Y: r, To: p.To, Ln: p.Ln, Op: p.Op }, true
            }
        }

        /* movx {mem}, %r0; cmpx {mem}, %r1; jcc --> movx {mem}, %r0; cmpx %r0, %r1; jcc
         * leaq {mem}, %r0; cmpx {mem}, %r1; jcc --> leaq {mem}, %r0; cmpx (%r0), %r1; jcc */
        case *IrAMD64_Jcc_mr: {
            if m, ok := mems.find(p.X); ok {
                p.X, *next = Ptr(m, 0), true
            } else if r, ok := vv[memaddr(p.X, p.N)]; ok {
                bb.Term, *next = &IrAMD64_Jcc_rr { X: r, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }, true
            }
        }

        /* movx {mem}, %r0; cmpx {mem}, {imm}; jcc --> movx {mem}, %r0; cmpx %r0, {imm}; jcc
         * leaq {mem}, %r0; cmpx {mem}, {imm}; jcc --> leaq {mem}, %r0; cmpx (%r0), {imm}; jcc */
        case *IrAMD64_Jcc_mi: {
            if m, ok := mems.find(p.X); ok {
                p.X, *next = Ptr(m, 0), true
            } else if r, ok := vv[memaddr(p.X, p.N)]; ok {
                bb.Term, *next = &IrAMD64_Jcc_ri { X: r, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }, true
            }
        }

        /* movx {mem}, %r0; cmpx {mem}, {ptr}; jcc --> movx {mem}, %r0; cmpx %r0, {ptr}; jcc
         * leaq {mem}, %r0; cmpx {mem}, {ptr}; jcc --> leaq {mem}, %r0; cmpx (%r0), {ptr}; jcc */
        case *IrAMD64_Jcc_mp: {
            if m, ok := mems.find(p.X); ok {
                p.X, *next = Ptr(m, 0), true
            } else if r, ok := vv[memaddr(p.X, abi.PtrSize)]; ok {
                bb.Term, *next = &IrAMD64_Jcc_rp { X: r, Y: p.Y, To: p.To, Ln: p.Ln, Op: p.Op }, true
            }
        }

        /* movx {mem}, %r0; cmpx {imm}, {mem}; jcc --> movx {mem}, %r0; cmpx {imm}, %r0; jcc
         * leaq {mem}, %r0; cmpx {imm}, {mem}; jcc --> leaq {mem}, %r0; cmpx {imm}, (%r0); jcc */
        case *IrAMD64_Jcc_im: {
            if m, ok := mems.find(p.Y); ok {
                p.Y, *next = Ptr(m, 0), true
            } else if r, ok := vv[memaddr(p.Y, p.N)]; ok {
                bb.Term, *next = &IrAMD64_Jcc_ir { X: p.X, Y: r, To: p.To, Ln: p.Ln, Op: p.Op }, true
            }
        }

        /* movx {mem}, %r0; cmpx {ptr}, {mem}; jcc --> movx {mem}, %r0; cmpx %r0, {ptr}; jcc
         * leaq {mem}, %r0; cmpx {ptr}, {mem}; jcc --> leaq {mem}, %r0; cmpx (%r0), {ptr}; jcc */
        case *IrAMD64_Jcc_pm: {
            if m, ok := mems.find(p.Y); ok {
                p.Y, *next = Ptr(m, 0), true
            } else if r, ok := vv[memaddr(p.Y, abi.PtrSize)]; ok {
                bb.Term, *next = &IrAMD64_Jcc_pr { X: p.X, Y: r, To: p.To, Ln: p.Ln, Op: p.Op }, true
            }
        }
    }

    /* DFS the dominator tree */
    for _, p := range cfg.DominatorOf[id] {
        self.dfs(cfg, p, mems.derive(), next)
    }
}

func (self Compaction) Apply(cfg *CFG) {
    for next := true; next; {
        next = false
        self.dfs(cfg, cfg.Root, (*_MemTree).derive(nil), &next)
    }
}
