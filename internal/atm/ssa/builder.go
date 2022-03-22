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
    `github.com/cloudwego/frugal/internal/atm/hir`
)

func ri2reg(ri uint8) hir.Register {
    if ri & hir.ArgPointer == 0 {
        return hir.GenericRegister(ri & hir.ArgMask)
    } else {
        return hir.PointerRegister(ri & hir.ArgMask)
    }
}

func memload(p *hir.Ir, rx hir.Register, size uint8) []IrNode {
    tp := Pr()
    tr := Tr()
    return []IrNode {
        &IrConstInt {
            R: tr,
            V: p.Iv,
        },
        &IrLEA {
            R   : tp,
            Off : tr,
            Mem : Rv(p.Ps),
        },
        &IrLoad {
            R    : Rv(rx),
            Mem  : tp,
            Size : size,
        },
    }
}

func memstore(p *hir.Ir, rx hir.Register, size uint8) []IrNode {
    tp := Pr()
    tr := Tr()
    return []IrNode {
        &IrConstInt {
            R: tr,
            V: p.Iv,
        },
        &IrLEA {
            R   : tp,
            Off : tr,
            Mem : Rv(p.Pd),
        },
        &IrStore {
            R    : Rv(rx),
            Mem  : tp,
            Size : size,
        },
    }
}

var _MemSize = [...]uint8 {
    hir.OP_lb: 1,
    hir.OP_lw: 2,
    hir.OP_ll: 4,
    hir.OP_lq: 8,
    hir.OP_sb: 1,
    hir.OP_sw: 2,
    hir.OP_sl: 4,
    hir.OP_sq: 8,
}

var _UnaryOps = [...]IrUnaryOp {
    hir.OP_swapw : IrOpSwap16,
    hir.OP_swapl : IrOpSwap32,
    hir.OP_swapq : IrOpSwap64,
    hir.OP_sxlq  : IrOpSx32to64,
}

var _BinaryOps = [...]IrBinaryOp {
    hir.OP_add  : IrOpAdd,
    hir.OP_sub  : IrOpSub,
    hir.OP_addi : IrOpAdd,
    hir.OP_muli : IrOpSub,
    hir.OP_andi : IrOpAnd,
    hir.OP_xori : IrOpXor,
    hir.OP_shri : IrOpShr,
    hir.OP_bsi  : IrOpBitSet,
}

func buildInstr(p *hir.Ir) []IrNode {
    switch p.Op {
        default: {
            panic("invalid instruction: " + p.Disassemble(nil))
        }

        /* no operation */
        case hir.OP_nop: {
            return nil
        }

        /* ptr(Pr) -> Pd */
        case hir.OP_ip: {
            return []IrNode {
                &IrConstPtr {
                    P: p.Pr,
                    R: Rv(p.Pd),
                },
            }
        }

        /* *(Ps + Iv) -> Rx / Pd */
        case hir.OP_lb, hir.OP_lw, hir.OP_ll, hir.OP_lq : return memload(p, p.Rx, _MemSize[p.Op])
        case hir.OP_lp                                  : return memload(p, p.Pd, abi.PtrSize)

        /* Rx / Ps -> *(Pd + Iv) */
        case hir.OP_sb, hir.OP_sw, hir.OP_sl, hir.OP_sq : return memstore(p, p.Rx, _MemSize[p.Op])
        case hir.OP_sp                                  : return memstore(p, p.Ps, abi.PtrSize)

        /* arg[Iv] -> Rx */
        case hir.OP_ldaq: {
            return []IrNode {
                &IrLoadArg {
                    R  : Rv(p.Rx),
                    Id : uint64(p.Iv),
                },
            }
        }

        /* arg[Iv] -> Pd */
        case hir.OP_ldap: {
            return []IrNode {
                &IrLoadArg {
                    R  : Rv(p.Pd),
                    Id : uint64(p.Iv),
                },
            }
        }

        /* Ps + Rx -> Pd */
        case hir.OP_addp: {
            return []IrNode {
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : Rv(p.Rx),
                },
            }
        }

        /* Ps - Rx -> Pd */
        case hir.OP_subp: {
            tr := Tr()
            return []IrNode {
                &IrUnaryExpr {
                    R  : tr,
                    V  : Rv(p.Rx),
                    Op : IrOpNegate,
                },
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : tr,
                },
            }
        }

        /* Ps + Iv -> Pd */
        case hir.OP_addpi: {
            tr := Tr()
            return []IrNode {
                &IrConstInt {
                    R: tr,
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : tr,
                },
            }
        }

        /* Rx ± Ry -> Rz */
        case hir.OP_add, hir.OP_sub: {
            return []IrNode {
                &IrBinaryExpr {
                    R  : Rv(p.Rz),
                    X  : Rv(p.Rx),
                    Y  : Rv(p.Ry),
                    Op : _BinaryOps[p.Op],
                },
            }
        }

        /* Ry & (1 << (Rx % PTR_BITS)) != 0 -> Rz, Ry |= 1 << (Rx % PTR_BITS) */
        case hir.OP_bts: {
            return []IrNode {
                &IrBitTestSet {
                    T: Rv(p.Rz),
                    S: Rv(p.Ry),
                    X: Rv(p.Rx),
                    Y: Rv(p.Ry),
                },
            }
        }

        /* Rx {+,*,&,^,>>,bitset} Iv -> Ry */
        case hir.OP_addi, hir.OP_muli, hir.OP_andi, hir.OP_xori, hir.OP_shri, hir.OP_bsi: {
            tr := Tr()
            return []IrNode {
                &IrConstInt {
                    R: tr,
                    V: p.Iv,
                },
                &IrBinaryExpr {
                    R  : Rv(p.Ry),
                    X  : Rv(p.Rx),
                    Y  : tr,
                    Op : _BinaryOps[p.Op],
                },
            }
        }

        /* {bswap{16/32/64}/sign_extend_32_to_64}(Rx) -> Ry */
        case hir.OP_swapw, hir.OP_swapl, hir.OP_swapq, hir.OP_sxlq: {
            return []IrNode {
                &IrUnaryExpr {
                    R: Rv(p.Ry),
                    V: Rv(p.Rx),
                    Op: _UnaryOps[p.Op],
                },
            }
        }

        /* memset(Pd, 0, Iv) */
        case hir.OP_bzero: {
            return []IrNode {
                &IrBlockZero {
                    Mem: Rv(p.Pd),
                    Len: uintptr(p.Iv),
                },
            }
        }

        /* memcpy(Pd, Ps, Rx) */
        case hir.OP_bcopy: {
            return []IrNode {
                &IrBlockCopy {
                    Mem: Rv(p.Pd),
                    Src: Rv(p.Ps),
                    Len: Rv(p.Rx),
                },
            }
        }

        /* call external functions */
        case hir.OP_ccall, hir.OP_gcall, hir.OP_icall: {
            var in []Reg
            var out []Reg

            /* convert args and rets */
            for _, rr := range p.Ar[:p.An] { in = append(in, Rv(ri2reg(rr))) }
            for _, rr := range p.Rr[:p.Rn] { out = append(out, Rv(ri2reg(rr))) }

            /* build the IR */
            return []IrNode {
                &IrCall {
                    Fn  : hir.LookupCall(p.Iv),
                    In  : in,
                    Out : out,
                },
            }
        }

        /* trigger a debugger breakpoint */
        case hir.OP_break: {
            return []IrNode {
                new(IrBreakpoint),
            }
        }
    }
}

func buildBranchOp(p *hir.Ir) (Reg, IrNode) {
    var reg Reg
    var cmp IrBinaryOp
    var rhs hir.Register

    /* check for OpCode */
    switch p.Op {
        case hir.OP_beq  : reg, cmp, rhs = Tr(), IrCmpEq , p.Ry
        case hir.OP_bne  : reg, cmp, rhs = Tr(), IrCmpNe , p.Ry
        case hir.OP_blt  : reg, cmp, rhs = Tr(), IrCmpLt , p.Ry
        case hir.OP_bltu : reg, cmp, rhs = Tr(), IrCmpLtu, p.Ry
        case hir.OP_bgeu : reg, cmp, rhs = Tr(), IrCmpGeu, p.Ry
        case hir.OP_beqn : reg, cmp, rhs = Pr(), IrCmpEq , hir.Pn
        case hir.OP_bnen : reg, cmp, rhs = Pr(), IrCmpNe , hir.Pn
        default          : panic("invalid branch: " + p.Disassemble(nil))
    }

    /* construct the instruction */
    return reg, &IrBinaryExpr {
        R  : reg,
        X  : Rv(p.Rx),
        Y  : Rv(rhs),
        Op : cmp,
    }
}

type GraphBuilder struct {
    Pin   map[*hir.Ir]bool
    Graph map[*hir.Ir]*BasicBlock
}

func CreateGraphBuilder() *GraphBuilder {
    return &GraphBuilder {
        Pin   : make(map[*hir.Ir]bool),
        Graph : make(map[*hir.Ir]*BasicBlock),
    }
}

func (self *GraphBuilder) scan(p hir.Program) {
    for v := p.Head; v != nil; v = v.Ln {
        if v.IsBranch() {
            if v.Op != hir.OP_bsw {
                self.Pin[v.Br] = true
            } else {
                for _, lb := range v.Sw() {
                    self.Pin[lb] = true
                }
            }
        }
    }
}

func (self *GraphBuilder) block(p *hir.Ir, bb *BasicBlock) {
    bb.Phi = nil
    bb.Ins = make([]IrNode, 0, 16)

    /* traverse down until it hits a branch instruction */
    for p != nil && !p.IsBranch() && p.Op != hir.OP_ret {
        bb.addInstr(p)
        p = p.Ln

        /* hit a merge point, merge with existing block */
        if self.Pin[p] {
            bb.termBranch(self.branch(p))
            return
        }
    }

    /* basic block must terminate */
    if p == nil {
        panic(fmt.Sprintf("basic block %d does not terminate", bb.Id))
    }

    /* add terminators */
    switch p.Op {
        case hir.OP_bsw : self.termbsw(p, bb)
        case hir.OP_ret : self.termret(p, bb)
        case hir.OP_jmp : bb.termBranch(self.branch(p.Ln))
        default         : bb.termCondition(p, self.branch(p.Br), self.branch(p.Ln))
    }
}

func (self *GraphBuilder) branch(p *hir.Ir) *BasicBlock {
    var ok bool
    var bb *BasicBlock

    /* check for existing basic blocks */
    if bb, ok = self.Graph[p]; ok {
        return bb
    }

    /* create a new block */
    bb = new(BasicBlock)
    bb.Id = len(self.Graph) + 1

    /* process the new block */
    self.Graph[p] = bb
    self.block(p, bb)
    return bb
}

func (self *GraphBuilder) termbsw(p *hir.Ir, bb *BasicBlock) {
    sw := new(IrSwitch)
    sw.Br = make(map[int64]*BasicBlock, p.Iv)

    /* add every branch of the switch instruction */
    for i, br := range p.Sw() {
        if br != nil {
            to := self.branch(br)
            sw.Br[int64(i)] = to
        }
    }

    /* add the default branch */
    sw.Ln = self.branch(p.Ln)
    bb.Term = sw
}

func (self *GraphBuilder) termret(p *hir.Ir, bb *BasicBlock) {
    var i uint8
    var ret []Reg

    /* convert each register */
    for i = 0; i < p.Rn; i++ {
        ret = append(ret, Rv(ri2reg(p.Rr[i])))
    }

    /* build the "return" IR */
    bb.Term = &IrReturn {
        R: ret,
    }
}

func (self *GraphBuilder) Build(p hir.Program) *BasicBlock {
    self.scan(p)
    return self.branch(p.Head)
}
