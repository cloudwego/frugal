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
    `runtime`
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
)

var ArchRegs = [...]x86_64.Register64 {
    x86_64.RAX,
    x86_64.RCX,
    x86_64.RDX,
    x86_64.RBX,
    x86_64.RSP,
    x86_64.RBP,
    x86_64.RSI,
    x86_64.RDI,
    x86_64.R8,
    x86_64.R9,
    x86_64.R10,
    x86_64.R11,
    x86_64.R12,
    x86_64.R13,
    x86_64.R14,
    x86_64.R15,
}

var ArchRegNames = map[x86_64.Register64]string {
    x86_64.RAX : "rax",
    x86_64.RCX : "rcx",
    x86_64.RDX : "rdx",
    x86_64.RBX : "rbx",
    x86_64.RSP : "rsp",
    x86_64.RBP : "rbp",
    x86_64.RSI : "rsi",
    x86_64.RDI : "rdi",
    x86_64.R8  : "r8",
    x86_64.R9  : "r9",
    x86_64.R10 : "r10",
    x86_64.R11 : "r11",
    x86_64.R12 : "r12",
    x86_64.R13 : "r13",
    x86_64.R14 : "r14",
    x86_64.R15 : "r15",
}

type Mem struct {
    M Reg
    I Reg
    S uint8
    D int32
}

func (self Mem) String() string {
    if self.I.Kind() == K_zero {
        if self.D == 0 {
            return fmt.Sprintf("(%s)", self.M)
        } else {
            return fmt.Sprintf("%d(%s)", self.D, self.M)
        }
    } else if self.S == 1 {
        if self.D == 0 {
            return fmt.Sprintf("(%s,%s)", self.M, self.I)
        } else {
            return fmt.Sprintf("%d(%s,%s)", self.D, self.M, self.I)
        }
    } else {
        if self.D == 0 {
            return fmt.Sprintf("(%s,%s,%d)", self.M, self.I, self.S)
        } else {
            return fmt.Sprintf("%d(%s,%s,%d)", self.D, self.M, self.I, self.S)
        }
    }
}

func (*IrAMD64_INT)    irnode() {}
func (*IrAMD64_LEA)    irnode() {}
func (*IrAMD64_NEG)    irnode() {}
func (*IrAMD64_BSWAP)  irnode() {}
func (*IrAMD64_MOVSLQ) irnode() {}

func (*IrAMD64_MOV_abs)     irnode() {}
func (*IrAMD64_MOV_ptr)     irnode() {}
func (*IrAMD64_MOV_reg)     irnode() {}
func (*IrAMD64_MOV_load)    irnode() {}
func (*IrAMD64_MOV_store_r) irnode() {}
func (*IrAMD64_MOV_store_i) irnode() {}
func (*IrAMD64_MOV_wb)      irnode() {}

func (*IrAMD64_MOVBE_load)  irnode() {}
func (*IrAMD64_MOVBE_store) irnode() {}

func (*IrAMD64_BinOp_rr) irnode() {}
func (*IrAMD64_BinOp_ri) irnode() {}
func (*IrAMD64_BTSQ_rr)  irnode() {}
func (*IrAMD64_BTSQ_ri)  irnode() {}

func (*IrAMD64_CMPQ_rr) irnode() {}
func (*IrAMD64_CMPQ_ri) irnode() {}
func (*IrAMD64_CMPQ_rp) irnode() {}
func (*IrAMD64_CMPQ_ir) irnode() {}
func (*IrAMD64_CMPQ_pr) irnode() {}
func (*IrAMD64_CMPQ_rm) irnode() {}
func (*IrAMD64_CMPQ_mr) irnode() {}
func (*IrAMD64_CMPQ_mi) irnode() {}
func (*IrAMD64_CMPQ_mp) irnode() {}
func (*IrAMD64_CMPQ_im) irnode() {}
func (*IrAMD64_CMPQ_pm) irnode() {}

func (*IrAMD64_JMP)     irnode() {}
func (*IrAMD64_JNC)     irnode() {}

func (*IrAMD64_Jcc_rr)  irnode() {}
func (*IrAMD64_Jcc_ri)  irnode() {}
func (*IrAMD64_Jcc_rp)  irnode() {}
func (*IrAMD64_Jcc_ir)  irnode() {}
func (*IrAMD64_Jcc_pr)  irnode() {}
func (*IrAMD64_Jcc_rm)  irnode() {}
func (*IrAMD64_Jcc_mr)  irnode() {}
func (*IrAMD64_Jcc_mi)  irnode() {}
func (*IrAMD64_Jcc_mp)  irnode() {}
func (*IrAMD64_Jcc_im)  irnode() {}
func (*IrAMD64_Jcc_pm)  irnode() {}

func (*IrAMD64_JMP)    irterminator() {}
func (*IrAMD64_JNC)    irterminator() {}

func (*IrAMD64_Jcc_rr) irterminator() {}
func (*IrAMD64_Jcc_ri) irterminator() {}
func (*IrAMD64_Jcc_rp) irterminator() {}
func (*IrAMD64_Jcc_ir) irterminator() {}
func (*IrAMD64_Jcc_pr) irterminator() {}
func (*IrAMD64_Jcc_rm) irterminator() {}
func (*IrAMD64_Jcc_mr) irterminator() {}
func (*IrAMD64_Jcc_mi) irterminator() {}
func (*IrAMD64_Jcc_mp) irterminator() {}
func (*IrAMD64_Jcc_im) irterminator() {}
func (*IrAMD64_Jcc_pm) irterminator() {}

type IrAMD64_INT struct {
    I uint8
}

func (self *IrAMD64_INT) String() string {
    switch self.I {
        case 1  : return "int1"
        case 3  : return "int3"
        default : return fmt.Sprintf("int $%d  # %#x", self.I, self.I)
    }
}

type IrAMD64_LEA struct {
    R Reg
    M Mem
}

func (self *IrAMD64_LEA) String() string {
    return fmt.Sprintf("leaq %s, %s", self.M, self.R)
}

func (self *IrAMD64_LEA) Usages() (r []*Reg) {
    if self.M.M.Kind() != K_zero { r = append(r, &self.M.M) }
    if self.M.I.Kind() != K_zero { r = append(r, &self.M.I) }
    return
}

func (self *IrAMD64_LEA) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_NEG struct {
    R Reg
    V Reg
}

func (self *IrAMD64_NEG) String() string {
    if self.R == self.V {
        return fmt.Sprintf("negq %s", self.R)
    } else {
        return fmt.Sprintf("movq %s, %s; negq %s", self.V, self.R, self.R)
    }
}

func (self *IrAMD64_NEG) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrAMD64_NEG) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_BSWAP struct {
    R Reg
    V Reg
    S uint8
}

func (self *IrAMD64_BSWAP) String() string {
    if self.R == self.V {
        switch self.S {
            case 2  : return fmt.Sprintf("rolw $8, %s", self.R)
            case 4  : return fmt.Sprintf("bswapl %s", self.R)
            case 8  : return fmt.Sprintf("bswapq %s", self.R)
            default : panic("invalid bswap size")
        }
    } else {
        switch self.S {
            case 2  : return fmt.Sprintf("movq %s, %s; rolw $8, %s", self.V, self.R, self.R)
            case 4  : return fmt.Sprintf("movq %s, %s; bswapl %s", self.V, self.R, self.R)
            case 8  : return fmt.Sprintf("movq %s, %s; bswapq %s", self.V, self.R, self.R)
            default : panic("invalid bswap size")
        }
    }
}

func (self *IrAMD64_BSWAP) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrAMD64_BSWAP) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOVSLQ struct {
    R Reg
    V Reg
}

func (self *IrAMD64_MOVSLQ) String() string {
    return fmt.Sprintf("movslq %s, %s", self.V, self.R)
}

func (self *IrAMD64_MOVSLQ) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrAMD64_MOVSLQ) Definitions() []*Reg {
    return []*Reg { &self.R }
}


type IrAMD64_MOV_abs struct {
    R Reg
    V int64
}

func (self *IrAMD64_MOV_abs) String() string {
    return fmt.Sprintf("movabsq $%d, %s  # %#x", self.V, self.R, self.V)
}

func (self *IrAMD64_MOV_abs) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_ptr struct {
    R Reg
    P unsafe.Pointer
}

func (self *IrAMD64_MOV_ptr) String() string {
    if fn := runtime.FuncForPC(uintptr(self.P)); fn == nil {
        return fmt.Sprintf("movabsq $%p, %s", self.P, self.R)
    } else if fp := fn.Entry(); fp == uintptr(self.P) {
        return fmt.Sprintf("movabsq $%p, %s  # %s", self.P, self.R, fn.Name())
    } else {
        return fmt.Sprintf("movabsq $%p, %s  # %s+%#x", self.P, self.R, fn.Name(), uintptr(self.P) - fp)
    }
}

func (self *IrAMD64_MOV_ptr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_reg struct {
    R Reg
    V Reg
}

func (self *IrAMD64_MOV_reg) String() string {
    return fmt.Sprintf("movq %s, %s", self.V, self.R)
}

func (self *IrAMD64_MOV_reg) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrAMD64_MOV_reg) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_load struct {
    R Reg
    M Mem
    N uint8
}

func (self *IrAMD64_MOV_load) String() string {
    switch self.N {
        case 1  : return fmt.Sprintf("movzbq %s, %s", self.M, self.R)
        case 2  : return fmt.Sprintf("movzwq %s, %s", self.M, self.R)
        case 4  : return fmt.Sprintf("movl %s, %s", self.M, self.R)
        case 8  : return fmt.Sprintf("movq %s, %s", self.M, self.R)
        case 16 : return fmt.Sprintf("movdqu %s, %s", self.M, self.R)
        default : panic("invalid load size")
    }
}

func (self *IrAMD64_MOV_load) Usages() (r []*Reg) {
    if self.M.M.Kind() != K_zero { r = append(r, &self.M.M) }
    if self.M.I.Kind() != K_zero { r = append(r, &self.M.I) }
    return
}

func (self *IrAMD64_MOV_load) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_store_r struct {
    R Reg
    M Mem
    N uint8
}

func (self *IrAMD64_MOV_store_r) String() string {
    return fmt.Sprintf("mov%c %s, %s", memsizec(self.N), self.R, self.M)
}

func (self *IrAMD64_MOV_store_r) Usages() (r []*Reg) {
    r = []*Reg { &self.R }
    if self.M.M.Kind() != K_zero { r = append(r, &self.M.M) }
    if self.M.I.Kind() != K_zero { r = append(r, &self.M.I) }
    return
}

type IrAMD64_MOV_store_i struct {
    V int32
    M Mem
    N uint8
}

func (self *IrAMD64_MOV_store_i) String() string {
    return fmt.Sprintf("mov%c $%d, %s  # %#x", memsizec(self.N), self.V, self.M, self.V)
}

func (self *IrAMD64_MOV_store_i) Usages() (r []*Reg) {
    if self.M.M.Kind() != K_zero { r = append(r, &self.M.M) }
    if self.M.I.Kind() != K_zero { r = append(r, &self.M.I) }
    return
}

type IrAMD64_MOV_wb struct {
    M  Reg
    R  Reg
    V  Reg
    Fn unsafe.Pointer
}

func (self *IrAMD64_MOV_wb) String() string {
    return fmt.Sprintf("cmpb $0, (%s); jne *%p; movq %s, (%s)", self.V, self.Fn, self.R, self.M)
}

func (self *IrAMD64_MOV_wb) Usages() (r []*Reg) {
    return []*Reg { &self.V, &self.R, &self.M }
}

type IrAMD64_MOVBE_load struct {
    R Reg
    M Mem
    S uint8
}

func (self *IrAMD64_MOVBE_load) String() string {
    switch self.S {
        case 2  : return fmt.Sprintf("movbew %s, %s", self.M, self.R)
        case 4  : return fmt.Sprintf("movbel %s, %s", self.M, self.R)
        case 8  : return fmt.Sprintf("movbeq %s, %s", self.M, self.R)
        default : panic("invalid load size")
    }
}

func (self *IrAMD64_MOVBE_load) Usages() (r []*Reg) {
    if self.M.M.Kind() != K_zero { r = append(r, &self.M.M) }
    if self.M.I.Kind() != K_zero { r = append(r, &self.M.I) }
    return
}

func (self *IrAMD64_MOVBE_load) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOVBE_store struct {
    R Reg
    M Mem
    S uint8
}

func (self *IrAMD64_MOVBE_store) String() string {
    switch self.S {
        case 2  : return fmt.Sprintf("movbew %s, %s", self.R, self.M)
        case 4  : return fmt.Sprintf("movbel %s, %s", self.R, self.M)
        case 8  : return fmt.Sprintf("movbeq %s, %s", self.R, self.M)
        default : panic("invalid store size")
    }
}

func (self *IrAMD64_MOVBE_store) Usages() (r []*Reg) {
    r = []*Reg { &self.R }
    if self.M.M.Kind() != K_zero { r = append(r, &self.M.M) }
    if self.M.I.Kind() != K_zero { r = append(r, &self.M.I) }
    return
}

type (
    IrAMD64_BinOp uint8
    IrAMD64_CmpOp uint8
)

const (
    IrAMD64_BinAdd IrAMD64_BinOp = iota
    IrAMD64_BinSub
    IrAMD64_BinMul
    IrAMD64_BinAnd
    IrAMD64_BinOr
    IrAMD64_BinXor
    IrAMD64_BinShr
)

const (
    IrAMD64_CmpEq IrAMD64_CmpOp = iota
    IrAMD64_CmpNe
    IrAMD64_CmpLt
    IrAMD64_CmpGe
    IrAMD64_CmpLtu
    IrAMD64_CmpGeu
)

func (self IrAMD64_BinOp) String() string {
    switch self {
        case IrAMD64_BinAdd : return "addq"
        case IrAMD64_BinSub : return "subq"
        case IrAMD64_BinMul : return "imulq"
        case IrAMD64_BinAnd : return "andq"
        case IrAMD64_BinOr  : return "orq"
        case IrAMD64_BinXor : return "xorq"
        case IrAMD64_BinShr : return "shrq"
        default             : panic("unreachable")
    }
}

func (self IrAMD64_CmpOp) String() string {
    switch self {
        case IrAMD64_CmpEq  : return "e"
        case IrAMD64_CmpNe  : return "ne"
        case IrAMD64_CmpLt  : return "l"
        case IrAMD64_CmpGe  : return "ge"
        case IrAMD64_CmpLtu : return "b"
        case IrAMD64_CmpGeu : return "ae"
        default             : panic("unreachable")
    }
}

func (self IrAMD64_CmpOp) Negated() IrAMD64_CmpOp {
    switch self {
        case IrAMD64_CmpEq  : return IrAMD64_CmpNe
        case IrAMD64_CmpNe  : return IrAMD64_CmpEq
        case IrAMD64_CmpLt  : return IrAMD64_CmpGe
        case IrAMD64_CmpGe  : return IrAMD64_CmpLt
        case IrAMD64_CmpLtu : return IrAMD64_CmpGeu
        case IrAMD64_CmpGeu : return IrAMD64_CmpLtu
        default             : panic("unreachable")
    }
}

type IrAMD64_BinOp_rr struct {
    R  Reg
    X  Reg
    Y  Reg
    Op IrAMD64_BinOp
}

func (self *IrAMD64_BinOp_rr) String() string {
    if self.R == self.X {
        return fmt.Sprintf("%s %s, %s", self.Op, self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; %s %s, %s", self.X, self.R, self.Op, self.Y, self.R)
    }
}

func (self *IrAMD64_BinOp_rr) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_BinOp_rr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_BinOp_ri struct {
    R  Reg
    X  Reg
    Y  int32
    Op IrAMD64_BinOp
}

func (self *IrAMD64_BinOp_ri) String() string {
    if self.Op == IrAMD64_BinMul {
        return fmt.Sprintf("imulq $%d, %s, %s  # %#x", self.Y, self.X, self.R, self.Y)
    } else if self.R == self.X {
        return fmt.Sprintf("%s $%d, %s  # %#x", self.Op, self.Y, self.X, self.Y)
    } else {
        return fmt.Sprintf("movq %s, %s; %s $%d, %s  # %#x", self.X, self.R, self.Op, self.Y, self.R, self.Y)
    }
}

func (self *IrAMD64_BinOp_ri) Usages() []*Reg {
    return []*Reg { &self.X }
}

func (self *IrAMD64_BinOp_ri) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_BTSQ_rr struct {
    T Reg
    S Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_BTSQ_rr) String() string {
    if self.T.Kind() == K_zero {
        if self.S == self.Y {
            return fmt.Sprintf("btsq %s, %s", self.Y, self.X)
        } else {
            return fmt.Sprintf("movq %s, %s; btsq %s, %s", self.X, self.S, self.Y, self.S)
        }
    } else {
        if self.S == self.Y {
            return fmt.Sprintf("btsq %s, %s; setc %s", self.Y, self.X, self.T)
        } else {
            return fmt.Sprintf("movq %s, %s; btsq %s, %s; setc %s", self.X, self.S, self.Y, self.S, self.T)
        }
    }
}

func (self *IrAMD64_BTSQ_rr) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_BTSQ_rr) Definitions() []*Reg {
    return []*Reg { &self.T, &self.S }
}

type IrAMD64_BTSQ_ri struct {
    T Reg
    S Reg
    X Reg
    Y uint8
}

func (self *IrAMD64_BTSQ_ri) String() string {
    if self.S == self.X {
        return fmt.Sprintf("btsq $%d, %s; setc %s", self.Y, self.X, self.T)
    } else {
        return fmt.Sprintf("movq %s, %s; btsq $%d, %s; setc %s", self.X, self.S, self.Y, self.S, self.T)
    }
}

func (self *IrAMD64_BTSQ_ri) Usages() []*Reg {
    return []*Reg { &self.X }
}

func (self *IrAMD64_BTSQ_ri) Definitions() []*Reg {
    return []*Reg { &self.T, &self.S }
}

type IrAMD64_CMPQ_rr struct {
    R  Reg
    X  Reg
    Y  Reg
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_rr) String() string {
    return fmt.Sprintf("cmpq %s, %s; set%s %s", self.X, self.Y, self.Op, self.R)
}

func (self *IrAMD64_CMPQ_rr) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_CMPQ_rr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_ri struct {
    R  Reg
    X  Reg
    Y  int32
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_ri) String() string {
    return fmt.Sprintf("cmpq %s, $%d; set%s %s  # %#x", self.X, self.Y, self.Op, self.R, self.Y)
}

func (self *IrAMD64_CMPQ_ri) Usages() []*Reg {
    return []*Reg { &self.X }
}

func (self *IrAMD64_CMPQ_ri) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_rp struct {
    R  Reg
    X  Reg
    Y  unsafe.Pointer
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_rp) String() string {
    return fmt.Sprintf("cmpq %s, $%p; set%s %s", self.X, self.Y, self.Op, self.R)
}

func (self *IrAMD64_CMPQ_rp) Usages() []*Reg {
    return []*Reg { &self.X }
}

func (self *IrAMD64_CMPQ_rp) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_ir struct {
    R  Reg
    X  int32
    Y  Reg
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_ir) String() string {
    return fmt.Sprintf("cmpq $%d, %s; set%s %s  # %#x", self.X, self.Y, self.Op, self.R, self.X)
}

func (self *IrAMD64_CMPQ_ir) Usages() []*Reg {
    return []*Reg { &self.Y }
}

func (self *IrAMD64_CMPQ_ir) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_pr struct {
    R  Reg
    X  unsafe.Pointer
    Y  Reg
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_pr) String() string {
    return fmt.Sprintf("cmpq $%p, %s; set%s %s", self.X, self.Y, self.Op, self.R)
}

func (self *IrAMD64_CMPQ_pr) Usages() []*Reg {
    return []*Reg { &self.Y }
}

func (self *IrAMD64_CMPQ_pr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_rm struct {
    R  Reg
    X  Reg
    Y  Mem
    N  uint8
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_rm) String() string {
    return fmt.Sprintf(
        "cmp%c %s, %s; set%s %s",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.R,
    )
}

func (self *IrAMD64_CMPQ_rm) Usages() []*Reg {
    if self.Y.I == Rz {
        return []*Reg { &self.X, &self.Y.M }
    } else {
        return []*Reg { &self.X, &self.Y.M, &self.Y.I }
    }
}

func (self *IrAMD64_CMPQ_rm) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_mr struct {
    R  Reg
    X  Mem
    Y  Reg
    N  uint8
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_mr) String() string {
    return fmt.Sprintf(
        "cmp%c %s, %s; set%s %s",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.R,
    )
}

func (self *IrAMD64_CMPQ_mr) Usages() []*Reg {
    if self.X.I == Rz {
        return []*Reg { &self.X.M, &self.Y }
    } else {
        return []*Reg { &self.X.M, &self.X.I, &self.Y }
    }
}

func (self *IrAMD64_CMPQ_mr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_mi struct {
    R  Reg
    X  Mem
    Y  int32
    N  uint8
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_mi) String() string {
    return fmt.Sprintf(
        "cmp%c %s, $%d; set%s %s  # %#x",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.R,
        self.Y,
    )
}

func (self *IrAMD64_CMPQ_mi) Usages() []*Reg {
    if self.X.I == Rz {
        return []*Reg { &self.X.M }
    } else {
        return []*Reg { &self.X.M, &self.X.I }
    }
}

func (self *IrAMD64_CMPQ_mi) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_mp struct {
    R  Reg
    X  Mem
    Y  unsafe.Pointer
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_mp) String() string {
    return fmt.Sprintf("cmpq %s, $%d; set%s %s  # %#x", self.X, self.Y, self.Op, self.R, self.Y)
}

func (self *IrAMD64_CMPQ_mp) Usages() []*Reg {
    if self.X.I == Rz {
        return []*Reg { &self.X.M }
    } else {
        return []*Reg { &self.X.M, &self.X.I }
    }
}

func (self *IrAMD64_CMPQ_mp) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_im struct {
    R  Reg
    X  int32
    Y  Mem
    N  uint8
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_im) String() string {
    return fmt.Sprintf(
        "cmp%c $%d, %s; set%s %s  # %#x",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.R,
        self.X,
    )
}

func (self *IrAMD64_CMPQ_im) Usages() []*Reg {
    if self.Y.I == Rz {
        return []*Reg { &self.Y.M }
    } else {
        return []*Reg { &self.Y.M, &self.Y.I }
    }
}

func (self *IrAMD64_CMPQ_im) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_pm struct {
    R  Reg
    X  unsafe.Pointer
    Y  Mem
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_CMPQ_pm) String() string {
    return fmt.Sprintf("cmpq $%p, %s; set%s %s", self.X, self.Y, self.Op, self.R)
}

func (self *IrAMD64_CMPQ_pm) Usages() []*Reg {
    if self.Y.I == Rz {
        return []*Reg { &self.Y.M }
    } else {
        return []*Reg { &self.Y.M, &self.Y.I }
    }
}

func (self *IrAMD64_CMPQ_pm) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_JMP struct {
    To *BasicBlock
}

func (self *IrAMD64_JMP) String() string {
    return fmt.Sprintf("jmp bb_%d", self.To.Id)
}

func (self *IrAMD64_JMP) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {{ b: self.To }},
    }
}

type IrAMD64_JNC struct {
    To *BasicBlock
    Ln *BasicBlock
}

func (self *IrAMD64_JNC) String() string {
    return fmt.Sprintf("jnc bb_%d; jmp bb_%d", self.To.Id, self.Ln.Id)
}

func (self *IrAMD64_JNC) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_rr struct {
    X  Reg
    Y  Reg
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_rr) String() string {
    return fmt.Sprintf(
        "cmpq %s, %s; j%s bb_%d; jmp bb_%d",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_Jcc_rr) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_Jcc_rr) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_ri struct {
    X  Reg
    Y  int32
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_ri) String() string {
    return fmt.Sprintf(
        "cmpq %s, $%d; j%s bb_%d; jmp bb_%d  # %#x",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
        self.Y,
    )
}

func (self *IrAMD64_Jcc_ri) Usages() []*Reg {
    return []*Reg { &self.X }
}

func (self *IrAMD64_Jcc_ri) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_rp struct {
    X  Reg
    Y  unsafe.Pointer
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_rp) String() string {
    return fmt.Sprintf(
        "cmpq %s, $%p; j%s bb_%d; jmp bb_%d",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_Jcc_rp) Usages() []*Reg {
    return []*Reg { &self.X }
}

func (self *IrAMD64_Jcc_rp) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_ir struct {
    X  int32
    Y  Reg
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_ir) String() string {
    return fmt.Sprintf(
        "cmpq $%d, %s; j%s bb_%d; jmp bb_%d  # %#x",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
        self.X,
    )
}

func (self *IrAMD64_Jcc_ir) Usages() []*Reg {
    return []*Reg { &self.Y }
}

func (self *IrAMD64_Jcc_ir) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_pr struct {
    X  unsafe.Pointer
    Y  Reg
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_pr) String() string {
    return fmt.Sprintf(
        "cmpq $%p, %s; j%s bb_%d; jmp bb_%d",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_Jcc_pr) Usages() []*Reg {
    return []*Reg { &self.Y }
}

func (self *IrAMD64_Jcc_pr) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_rm struct {
    X  Reg
    Y  Mem
    N  uint8
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_rm) String() string {
    return fmt.Sprintf(
        "cmp%c %s, %s; j%s bb_%d; jmp bb_%d",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_Jcc_rm) Usages() []*Reg {
    if self.Y.I == Rz {
        return []*Reg { &self.X, &self.Y.M }
    } else {
        return []*Reg { &self.X, &self.Y.M, &self.Y.I }
    }
}

func (self *IrAMD64_Jcc_rm) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_mr struct {
    X  Mem
    Y  Reg
    N  uint8
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_mr) String() string {
    return fmt.Sprintf(
        "cmp%c %s, %s; j%s bb_%d; jmp bb_%d",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_Jcc_mr) Usages() []*Reg {
    if self.X.I == Rz {
        return []*Reg { &self.X.M, &self.Y }
    } else {
        return []*Reg { &self.X.M, &self.X.I, &self.Y }
    }
}

func (self *IrAMD64_Jcc_mr) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_mi struct {
    X  Mem
    Y  int32
    N  uint8
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_mi) String() string {
    return fmt.Sprintf(
        "cmp%c %s, $%d; j%s bb_%d; jmp bb_%d  # %#x",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
        self.Y,
    )
}

func (self *IrAMD64_Jcc_mi) Usages() []*Reg {
    if self.X.I == Rz {
        return []*Reg { &self.X.M }
    } else {
        return []*Reg { &self.X.M, &self.X.I }
    }
}

func (self *IrAMD64_Jcc_mi) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_mp struct {
    X  Mem
    Y  unsafe.Pointer
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_mp) String() string {
    return fmt.Sprintf(
        "cmpq %s, $%d; j%s bb_%d; jmp bb_%d  # %#x",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
        self.Y,
    )
}

func (self *IrAMD64_Jcc_mp) Usages() []*Reg {
    if self.X.I == Rz {
        return []*Reg { &self.X.M }
    } else {
        return []*Reg { &self.X.M, &self.X.I }
    }
}

func (self *IrAMD64_Jcc_mp) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}

type IrAMD64_Jcc_im struct {
    X  int32
    Y  Mem
    N  uint8
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_im) String() string {
    return fmt.Sprintf(
        "cmp%c $%d, %s; j%s bb_%d; jmp bb_%d  # %#x",
        memsizec(self.N),
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
        self.X,
    )
}

func (self *IrAMD64_Jcc_im) Usages() []*Reg {
    if self.Y.I == Rz {
        return []*Reg { &self.Y.M }
    } else {
        return []*Reg { &self.Y.M, &self.Y.I }
    }
}

func (self *IrAMD64_Jcc_im) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}
type IrAMD64_Jcc_pm struct {
    X  unsafe.Pointer
    Y  Mem
    To *BasicBlock
    Ln *BasicBlock
    Op IrAMD64_CmpOp
}

func (self *IrAMD64_Jcc_pm) String() string {
    return fmt.Sprintf(
        "cmpq $%p, %s; j%s bb_%d; jmp bb_%d",
        self.X,
        self.Y,
        self.Op,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_Jcc_pm) Usages() []*Reg {
    if self.Y.I == Rz {
        return []*Reg { &self.Y.M }
    } else {
        return []*Reg { &self.Y.M, &self.Y.I }
    }
}

func (self *IrAMD64_Jcc_pm) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
}
