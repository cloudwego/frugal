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
    if self.I == Rz {
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

func (*IrAMD64_INT)          irnode() {}
func (*IrAMD64_LEA)          irnode() {}
func (*IrAMD64_MOV_abs)      irnode() {}
func (*IrAMD64_MOV_ptr)      irnode() {}
func (*IrAMD64_MOV_reg)      irnode() {}
func (*IrAMD64_MOV_load)     irnode() {}
func (*IrAMD64_MOV_store)    irnode() {}
func (*IrAMD64_MOVBE_load)   irnode() {}
func (*IrAMD64_MOVBE_store)  irnode() {}
func (*IrAMD64_NEG)          irnode() {}
func (*IrAMD64_BSWAP)        irnode() {}
func (*IrAMD64_MOVSLQ)       irnode() {}
func (*IrAMD64_ADDQ)         irnode() {}
func (*IrAMD64_SUBQ)         irnode() {}
func (*IrAMD64_IMULQ)        irnode() {}
func (*IrAMD64_ANDQ)         irnode() {}
func (*IrAMD64_ORQ)          irnode() {}
func (*IrAMD64_XORQ)         irnode() {}
func (*IrAMD64_SHRQ)         irnode() {}
func (*IrAMD64_CMPQ_eq)      irnode() {}
func (*IrAMD64_CMPQ_ne)      irnode() {}
func (*IrAMD64_CMPQ_lt)      irnode() {}
func (*IrAMD64_CMPQ_ltu)     irnode() {}
func (*IrAMD64_CMPQ_geu)     irnode() {}
func (*IrAMD64_BTSQ)         irnode() {}
func (*IrAMD64_JE_imm)       irnode() {}
func (*IrAMD64_JMP)          irnode() {}

func (*IrAMD64_JE_imm) irterminator() {}
func (*IrAMD64_JMP)    irterminator() {}

type IrAMD64_INT struct {
    I uint8
}

func (self *IrAMD64_INT) String() string {
    switch self.I {
        case 1  : return "int1"
        case 3  : return "int3"
        default : return fmt.Sprintf("int $%d", self.I)
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
    if self.M.M != Pn { r = append(r, &self.M.M) }
    if self.M.I != Rz { r = append(r, &self.M.I) }
    return
}

func (self *IrAMD64_LEA) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_abs struct {
    R Reg
    V int64
}

func (self *IrAMD64_MOV_abs) String() string {
    return fmt.Sprintf("movabsq $%d, %s", self.V, self.R)
}

func (self *IrAMD64_MOV_abs) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_ptr struct {
    R Reg
    P unsafe.Pointer
}

func (self *IrAMD64_MOV_ptr) String() string {
    return fmt.Sprintf("movabsq $%p, %s", self.P, self.R)
}

func (self *IrAMD64_MOV_ptr) Definations() []*Reg {
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

func (self *IrAMD64_MOV_reg) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_load struct {
    R Reg
    M Mem
    S uint8
}

func (self *IrAMD64_MOV_load) String() string {
    switch self.S {
        case 1  : return fmt.Sprintf("movzbq %s, %s", self.M, self.R)
        case 2  : return fmt.Sprintf("movzwq %s, %s", self.M, self.R)
        case 4  : return fmt.Sprintf("movl %s, %s", self.M, self.R)
        case 8  : return fmt.Sprintf("movq %s, %s", self.M, self.R)
        case 16 : return fmt.Sprintf("movdqu %s, %s", self.M, self.R)
        default : panic("invalid load size")
    }
}

func (self *IrAMD64_MOV_load) Usages() (r []*Reg) {
    if self.M.M != Pn { r = append(r, &self.M.M) }
    if self.M.I != Rz { r = append(r, &self.M.I) }
    return
}

func (self *IrAMD64_MOV_load) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_MOV_store struct {
    R Reg
    M Mem
    S uint8
}

func (self *IrAMD64_MOV_store) String() string {
    switch self.S {
        case 1  : return fmt.Sprintf("movb %s, %s", self.R, self.M)
        case 2  : return fmt.Sprintf("movw %s, %s", self.R, self.M)
        case 4  : return fmt.Sprintf("movl %s, %s", self.R, self.M)
        case 8  : return fmt.Sprintf("movq %s, %s", self.R, self.M)
        default : panic("invalid store size")
    }
}

func (self *IrAMD64_MOV_store) Usages() (r []*Reg) {
    r = []*Reg { &self.R }
    if self.M.M != Pn { r = append(r, &self.M.M) }
    if self.M.I != Rz { r = append(r, &self.M.I) }
    return
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
    if self.M.M != Pn { r = append(r, &self.M.M) }
    if self.M.I != Rz { r = append(r, &self.M.I) }
    return
}

func (self *IrAMD64_MOVBE_load) Definations() []*Reg {
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
    if self.M.M != Pn { r = append(r, &self.M.M) }
    if self.M.I != Rz { r = append(r, &self.M.I) }
    return
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

func (self *IrAMD64_NEG) Definations() []*Reg {
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

func (self *IrAMD64_BSWAP) Definations() []*Reg {
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

func (self *IrAMD64_MOVSLQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_ADDQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_ADDQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("addq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; addq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_ADDQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_ADDQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_SUBQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_SUBQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("subq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; subq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_SUBQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_SUBQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_IMULQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_IMULQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("imulq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; imulq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_IMULQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_IMULQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_ANDQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_ANDQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("andq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; andq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_ANDQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_ANDQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_ORQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_ORQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("orq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; orq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_ORQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_ORQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_XORQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_XORQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("xorq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; xorq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_XORQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_XORQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_SHRQ struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_SHRQ) String() string {
    if self.R == self.Y {
        return fmt.Sprintf("shrq %s, %s", self.Y, self.X)
    } else {
        return fmt.Sprintf("movq %s, %s; shrq %s, %s", self.X, self.R, self.Y, self.R)
    }
}

func (self *IrAMD64_SHRQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_SHRQ) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_eq struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_CMPQ_eq) String() string {
    return fmt.Sprintf("cmpq %s, %s; sete %s", self.Y, self.X, self.R)
}

func (self *IrAMD64_CMPQ_eq) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_CMPQ_eq) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_ne struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_CMPQ_ne) String() string {
    return fmt.Sprintf("cmpq %s, %s; setne %s", self.Y, self.X, self.R)
}

func (self *IrAMD64_CMPQ_ne) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_CMPQ_ne) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_lt struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_CMPQ_lt) String() string {
    return fmt.Sprintf("cmpq %s, %s; setl %s", self.Y, self.X, self.R)
}

func (self *IrAMD64_CMPQ_lt) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_CMPQ_lt) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_ltu struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_CMPQ_ltu) String() string {
    return fmt.Sprintf("cmpq %s, %s; setb %s", self.Y, self.X, self.R)
}

func (self *IrAMD64_CMPQ_ltu) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_CMPQ_ltu) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_CMPQ_geu struct {
    R Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_CMPQ_geu) String() string {
    return fmt.Sprintf("cmpq %s, %s; setae %s", self.Y, self.X, self.R)
}

func (self *IrAMD64_CMPQ_geu) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_CMPQ_geu) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrAMD64_BTSQ struct {
    T Reg
    S Reg
    X Reg
    Y Reg
}

func (self *IrAMD64_BTSQ) String() string {
    if self.S == self.Y {
        return fmt.Sprintf("btsq %s, %s; setc %s", self.Y, self.X, self.T)
    } else {
        return fmt.Sprintf("movq %s, %s; btsq %s, %s; setc %s", self.X, self.S, self.Y, self.S, self.T)
    }
}

func (self *IrAMD64_BTSQ) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrAMD64_BTSQ) Definations() []*Reg {
    return []*Reg { &self.T, &self.S }
}

type IrAMD64_JE_imm struct {
    R  Reg
    V  int64
    To *BasicBlock
    Ln *BasicBlock
}

func (self *IrAMD64_JE_imm) String() string {
    return fmt.Sprintf(
        "cmpq $%d, %s; je bb_%d; jmp bb_%d",
        self.V,
        self.R,
        self.To.Id,
        self.Ln.Id,
    )
}

func (self *IrAMD64_JE_imm) Usages() []*Reg {
    return []*Reg { &self.R }
}

func (self *IrAMD64_JE_imm) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        i: -1,
        t: []_SwitchTarget {
            { b: self.To, i: 1 },
            { b: self.Ln },
        },
    }
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
