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
    `github.com/cloudwego/frugal/internal/atm/rtx`
)

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
}

type BasicBlock struct {
    Id   int
    Phi  []*IrPhi
    Ins  []IrNode
    Pred []*BasicBlock
    Term IrTerminator
}

func Unreachable(bb *BasicBlock, id int) (ret *BasicBlock) {
    ret      = new(BasicBlock)
    ret.Id   = id
    ret.Ins  = append(ret.Ins, new(IrBreakpoint))
    ret.Pred = append(ret.Pred, bb, ret)
    ret.Term = &IrSwitch { Ln: ret }
    return
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

        /* *(Ps + Iv) -> Rx */
        case hir.OP_lb, hir.OP_lw, hir.OP_ll, hir.OP_lq: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Pr(0),
                    Mem : Rv(p.Ps),
                    Off : Tr(0),
                },
                &IrLoad {
                    R    : Rv(p.Rx),
                    Mem  : Pr(0),
                    Size : _MemSize[p.Op],
                },
            )
        }

        /* *(Ps + Iv) -> Pd */
        case hir.OP_lp: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Pr(0),
                    Mem : Rv(p.Ps),
                    Off : Tr(0),
                },
                &IrLoad {
                    R    : Rv(p.Pd),
                    Mem  : Pr(0),
                    Size : abi.PtrSize,
                },
            )
        }

        /* Rx -> *(Pd + Iv) */
        case hir.OP_sb, hir.OP_sw, hir.OP_sl, hir.OP_sq: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Pr(0),
                    Mem : Rv(p.Pd),
                    Off : Tr(0),
                },
                &IrStore {
                    R    : Rv(p.Rx),
                    Mem  : Pr(0),
                    Size : _MemSize[p.Op],
                },
            )
        }

        /* Ps -> *(Pd + Iv) */
        case hir.OP_sp: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Pr(0),
                    Mem : Rv(p.Pd),
                    Off : Tr(0),
                },
                &IrConstPtr {
                    R: Pr(1),
                    P: rtx.V_pWriteBarrier,
                },
                &IrConstPtr {
                    R: Pr(2),
                    P: rtx.F_gcWriteBarrier,
                },
                &IrWriteBarrier {
                    R   : Pr(0),
                    V   : Rv(p.Ps),
                    Fn  : Pr(2),
                    Var : Pr(1),
                },
            )
        }

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
                    R  : Tr(0),
                    V  : Rv(p.Rx),
                    Op : IrOpNegate,
                },
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : Tr(0),
                },
            )
        }

        /* Ps + Iv -> Pd */
        case hir.OP_addpi: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: p.Iv,
                },
                &IrLEA {
                    R   : Rv(p.Pd),
                    Mem : Rv(p.Ps),
                    Off : Tr(0),
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

        /* Ry & (1 << (Rx % PTR_BITS)) != 0 -> Rz
         * Ry |= 1 << (Rx % PTR_BITS) */
        case hir.OP_bts: {
            self.Ins = append(
                self.Ins,
                &IrBitTestSet {
                    T: Rv(p.Rz),
                    S: Rv(p.Ry),
                    X: Rv(p.Ry),
                    Y: Rv(p.Rx),
                },
            )
        }

        /* Rx {+,*,&,^,>>} Iv -> Ry */
        case hir.OP_addi, hir.OP_muli, hir.OP_andi, hir.OP_xori, hir.OP_shri: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: p.Iv,
                },
                &IrBinaryExpr {
                    R  : Rv(p.Ry),
                    X  : Rv(p.Rx),
                    Y  : Tr(0),
                    Op : _BinaryOps[p.Op],
                },
            )
        }

        /* Rx | (1 << Iv) -> Ry */
        case hir.OP_bsi: {
            self.Ins = append(
                self.Ins,
                &IrConstInt {
                    R: Tr(0),
                    V: 1 << p.Iv,
                },
                &IrBinaryExpr {
                    R  : Rv(p.Ry),
                    X  : Rv(p.Rx),
                    Y  : Tr(0),
                    Op : IrOpOr,
                },
            )
        }

        /* {bswap{16/32/64}/sign_extend_32_to_64}(Rx) -> Ry */
        case hir.OP_swapw, hir.OP_swapl, hir.OP_swapq, hir.OP_sxlq: {
            self.Ins = append(
                self.Ins,
                &IrUnaryExpr {
                    R  : Rv(p.Ry),
                    V  : Rv(p.Rx),
                    Op : _UnaryOps[p.Op],
                },
            )
        }

        /* memset(Pd, 0, Iv) */
        case hir.OP_bzero: {
            r := Rv(p.Pd)
            d := uintptr(0)
            n := uintptr(p.Iv)

            /* call memory zero for large blocks */
            for ; n >= rtx.MaxZeroSize; n -= rtx.MaxZeroSize {
                self.zeroBlock(r, d, rtx.MaxZeroSize)
                d += rtx.MaxZeroSize
            }

            /* call memory zero for smaller blocks */
            if n >= rtx.ZeroStep {
                self.zeroBlock(r, d, n)
                d += n / rtx.ZeroStep * rtx.ZeroStep
                n %= rtx.ZeroStep
            }

            /* use scalar code for remaining bytes */
            for _, v := range []uintptr { 8, 4, 2, 1 } {
                for n >= v {
                    self.zeroUnit(r, d, v)
                    d += v
                    n -= v
                }
            }
        }

        /* memcpy(Pd, Ps, Rx) */
        case hir.OP_bcopy: {
            self.Ins = append(
                self.Ins,
                &IrConstPtr {
                    R: Pr(0),
                    P: rtx.F_memmove,
                },
                &IrCallFunc {
                    R  : Pr(0),
                    In : []Reg { Rv(p.Pd), Rv(p.Ps), Rv(p.Rx) },
                },
            )
        }

        /* C subroutine calls */
        case hir.OP_ccall: {
            self.Ins = append(
                self.Ins,
                &IrConstPtr {
                    R: Pr(0),
                    P: hir.LookupCall(p.Iv).Func,
                },
                &IrCallNative {
                    R   : Pr(0),
                    In  : ri2regs(p.Ar[:p.An]),
                    Out : ri2regz(p.Rr[:p.Rn]),
                },
            )
        }

        /* Go subroutine calls */
        case hir.OP_gcall: {
            self.Ins = append(
                self.Ins,
                &IrConstPtr {
                    R: Pr(0),
                    P: hir.LookupCall(p.Iv).Func,
                },
                &IrCallFunc {
                    R   : Pr(0),
                    In  : ri2regs(p.Ar[:p.An]),
                    Out : ri2regs(p.Rr[:p.Rn]),
                },
            )
        }

        /* interface method calls */
        case hir.OP_icall: {
            self.Ins = append(
                self.Ins,
                &IrCallMethod {
                    T    : Rv(p.Ps),
                    V    : Rv(p.Pd),
                    In   : ri2regs(p.Ar[:p.An]),
                    Out  : ri2regs(p.Rr[:p.Rn]),
                    Slot : hir.LookupCall(p.Iv).Slot,
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

func (self *BasicBlock) zeroUnit(r Reg, d uintptr, n uintptr) {
    self.Ins = append(self.Ins,
        &IrConstInt {
            R: Tr(0),
            V: int64(d),
        },
        &IrLEA {
            R   : Pr(0),
            Mem : r,
            Off : Tr(0),
        },
        &IrStore {
            R    : Rz,
            Mem  : Pr(0),
            Size : uint8(n),
        },
    )
}

func (self *BasicBlock) zeroBlock(r Reg, d uintptr, n uintptr) {
    self.Ins = append(self.Ins,
        &IrConstInt {
            R: Tr(0),
            V: int64(d),
        },
        &IrLEA {
            R   : Pr(0),
            Mem : r,
            Off : Tr(0),
        },
        &IrConstPtr {
            R: Pr(1),
            P: rtx.MemZero.ForSize(n),
        },
        &IrCallNative {
            R  : Pr(1),
            In : []Reg { Pr(0) },
        },
    )
}

func (self *BasicBlock) termReturn(p *hir.Ir) {
    self.Term = &IrReturn {
        R: ri2regs(p.Rr[:p.Rn]),
    }
}

func (self *BasicBlock) termBranch(to *BasicBlock) {
    to.Pred = append(to.Pred, self)
    self.Term = &IrSwitch { Ln: to }
}

func (self *BasicBlock) termCondition(p *hir.Ir, t *BasicBlock, f *BasicBlock) {
    var cmp IrBinaryOp
    var lhs hir.Register
    var rhs hir.Register

    /* check for OpCode */
    switch p.Op {
        case hir.OP_beq  : cmp, lhs, rhs = IrCmpEq  , p.Rx, p.Ry
        case hir.OP_bne  : cmp, lhs, rhs = IrCmpNe  , p.Rx, p.Ry
        case hir.OP_blt  : cmp, lhs, rhs = IrCmpLt  , p.Rx, p.Ry
        case hir.OP_bltu : cmp, lhs, rhs = IrCmpLtu , p.Rx, p.Ry
        case hir.OP_bgeu : cmp, lhs, rhs = IrCmpGeu , p.Rx, p.Ry
        case hir.OP_beqn : cmp, lhs, rhs = IrCmpEq  , p.Ps, hir.Pn
        case hir.OP_bnen : cmp, lhs, rhs = IrCmpNe  , p.Ps, hir.Pn
        default          : panic("invalid branch: " + p.Disassemble(nil))
    }

    /* construct the instruction */
    ins := &IrBinaryExpr {
        R  : Tr(0),
        X  : Rv(lhs),
        Y  : Rv(rhs),
        Op : cmp,
    }

    /* attach to the block */
    t.Pred = append(t.Pred, self)
    f.Pred = append(f.Pred, self)
    self.Ins = append(self.Ins, ins)
    self.Term = &IrSwitch { V: Tr(0), Ln: t, Br: map[int32]*BasicBlock { 0: f } }
}
