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
    `github.com/cloudwego/frugal/internal/atm/hir`
)

func memload(buf *[]IrNode, p *hir.Ir, rx hir.Register, size uint8) {
    *buf = append(
        *buf,
        &IrConstInt {
            R: Tr,
            V: p.Iv,
        },
        &IrLEA {
            R   : Pr,
            Off : Tr,
            Mem : Rv(p.Ps),
        },
        &IrLoad {
            R    : Rv(rx),
            Mem  : Pr,
            Size : size,
        },
    )
}

func memstore(buf *[]IrNode, p *hir.Ir, rx hir.Register, size uint8) {
    *buf = append(
        *buf,
        &IrConstInt {
            R: Tr,
            V: p.Iv,
        },
        &IrLEA {
            R   : Pr,
            Off : Tr,
            Mem : Rv(p.Pd),
        },
        &IrStore {
            R    : Rv(rx),
            Mem  : Pr,
            Size : size,
        },
    )
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
    hir.OP_muli : IrOpMul,
    hir.OP_andi : IrOpAnd,
    hir.OP_xori : IrOpXor,
    hir.OP_shri : IrOpShr,
    hir.OP_bsi  : IrOpBitSet,
}

type BasicBlock struct {
    Id   int
    Phi  []*IrPhi
    Ins  []IrNode
    Pred []*BasicBlock
    Term IrTerminator
}

func (self *BasicBlock) addInstr(p *hir.Ir) {
    switch p.Op {
        default: {
            panic("invalid instruction: " + p.Disassemble(nil))
        }

        /* no operation */
        case hir.OP_nop: {
            break
        }

        /* ptr(Pr) -> Pd */
        case hir.OP_ip: {
            self.Ins = append(
                self.Ins,
                &IrConstPtr {
                    P: p.Pr,
                    R: Rv(p.Pd),
                },
            )
        }

        /* *(Ps + Iv) -> Rx / Pd */
        case hir.OP_lb, hir.OP_lw, hir.OP_ll, hir.OP_lq : memload(&self.Ins, p, p.Rx, _MemSize[p.Op])
        case hir.OP_lp                                  : memload(&self.Ins, p, p.Pd, abi.PtrSize)

        /* Rx / Ps -> *(Pd + Iv) */
        case hir.OP_sb, hir.OP_sw, hir.OP_sl, hir.OP_sq : memstore(&self.Ins, p, p.Rx, _MemSize[p.Op])
        case hir.OP_sp                                  : memstore(&self.Ins, p, p.Ps, abi.PtrSize)

        /* arg[Iv] -> Rx */
        case hir.OP_ldaq: {
            self.Ins = append(
                self.Ins,
                &IrLoadArg {
                    R  : Rv(p.Rx),
                    Id : uint64(p.Iv),
                },
            )
        }

        /* arg[Iv] -> Pd */
        case hir.OP_ldap: {
            self.Ins = append(
                self.Ins,
                &IrLoadArg {
                    R  : Rv(p.Pd),
                    Id : uint64(p.Iv),
                },
            )
        }

        /* Ps + Rx -> Pd */
        case hir.OP_addp: {
            self.Ins = append(
                self.Ins,
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : Rv(p.Rx),
                },
            )
        }

        /* Ps - Rx -> Pd */
        case hir.OP_subp: {
            self.Ins = append(
                self.Ins,
                &IrUnaryExpr {
                    R  : Tr,
                    V  : Rv(p.Rx),
                    Op : IrOpNegate,
                },
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : Tr,
                },
            )
        }

        /* Ps + Iv -> Pd */
        case hir.OP_addpi: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr,
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : Tr,
                },
            )
        }

        /* Rx ± Ry -> Rz */
        case hir.OP_add, hir.OP_sub: {
            self.Ins = append(
                self.Ins,
                &IrBinaryExpr {
                    R  : Rv(p.Rz),
                    X  : Rv(p.Rx),
                    Y  : Rv(p.Ry),
                    Op : _BinaryOps[p.Op],
                },
            )
        }

        /* Ry & (1 << (Rx % PTR_BITS)) != 0 -> Rz, Ry |= 1 << (Rx % PTR_BITS) */
        case hir.OP_bts: {
            self.Ins = append(
                self.Ins,
                &IrBitTestSet {
                    T: Rv(p.Rz),
                    S: Rv(p.Ry),
                    X: Rv(p.Rx),
                    Y: Rv(p.Ry),
                },
            )
        }

        /* Rx {+,*,&,^,>>,bitset} Iv -> Ry */
        case hir.OP_addi, hir.OP_muli, hir.OP_andi, hir.OP_xori, hir.OP_shri, hir.OP_bsi: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr,
                    V: p.Iv,
                },
                &IrBinaryExpr {
                    R  : Rv(p.Ry),
                    X  : Rv(p.Rx),
                    Y  : Tr,
                    Op : _BinaryOps[p.Op],
                },
            )
        }

        /* {bswap{16/32/64}/sign_extend_32_to_64}(Rx) -> Ry */
        case hir.OP_swapw, hir.OP_swapl, hir.OP_swapq, hir.OP_sxlq: {
            self.Ins = append(
                self.Ins,
                &IrUnaryExpr {
                    R: Rv(p.Ry),
                    V: Rv(p.Rx),
                    Op: _UnaryOps[p.Op],
                },
            )
        }

        /* memset(Pd, 0, Iv) */
        case hir.OP_bzero: {
            self.Ins = append(
                self.Ins,
                &IrBlockZero {
                    Mem: Rv(p.Pd),
                    Len: uintptr(p.Iv),
                },
            )
        }

        /* memcpy(Pd, Ps, Rx) */
        case hir.OP_bcopy: {
            self.Ins = append(
                self.Ins,
                &IrBlockCopy {
                    Mem: Rv(p.Pd),
                    Src: Rv(p.Ps),
                    Len: Rv(p.Rx),
                },
            )
        }

        /* call external functions */
        case hir.OP_ccall, hir.OP_gcall, hir.OP_icall: {
            var in []Reg
            var out []Reg

            /* convert args and rets */
            for _, rr := range p.Ar[:p.An] { in = append(in, Rv(ri2reg(rr))) }
            for _, rr := range p.Rr[:p.Rn] { out = append(out, Rv(ri2reg(rr))) }

            /* build the IR */
            self.Ins = append(
                self.Ins,
                &IrCall {
                    Fn  : hir.LookupCall(p.Iv),
                    In  : in,
                    Out : out,
                },
            )
        }

        /* trigger a debugger breakpoint */
        case hir.OP_break: {
            self.Ins = append(
                self.Ins,
                new(IrBreakpoint),
            )
        }
    }
}

func (self *BasicBlock) termBranch(to *BasicBlock) {
    to.Pred = append(to.Pred, self)
    self.Term = &IrSwitch{Ln: to}
}

func (self *BasicBlock) termCondition(p *hir.Ir, t *BasicBlock, f *BasicBlock) {
    var reg Reg
    var cmp IrBinaryOp
    var rhs hir.Register

    /* check for OpCode */
    switch p.Op {
        case hir.OP_beq  : reg, cmp, rhs = Tr, IrCmpEq , p.Ry
        case hir.OP_bne  : reg, cmp, rhs = Tr, IrCmpNe , p.Ry
        case hir.OP_blt  : reg, cmp, rhs = Tr, IrCmpLt , p.Ry
        case hir.OP_bltu : reg, cmp, rhs = Tr, IrCmpLtu, p.Ry
        case hir.OP_bgeu : reg, cmp, rhs = Tr, IrCmpGeu, p.Ry
        case hir.OP_beqn : reg, cmp, rhs = Pr, IrCmpEq , hir.Pn
        case hir.OP_bnen : reg, cmp, rhs = Pr, IrCmpNe , hir.Pn
        default          : panic("invalid branch: " + p.Disassemble(nil))
    }

    /* construct the instruction */
    ins := &IrBinaryExpr {
        R  : reg,
        X  : Rv(p.Rx),
        Y  : Rv(rhs),
        Op : cmp,
    }

    /* attach to the block */
    t.Pred = append(t.Pred, self)
    f.Pred = append(f.Pred, self)
    self.Ins = append(self.Ins, ins)
    self.Term = &IrSwitch{V: reg, Ln: f, Br: map[int64]*BasicBlock{1: t}}
}
