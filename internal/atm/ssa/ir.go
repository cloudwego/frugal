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
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/cloudwego/frugal/internal/rt`
)

type Reg uint64

const (
    _R_ptr   = 1 << 63
    _R_kind  = 15 << 59
    _R_index = (1 << 59) - 1
)

const (
    _K_temp = 14
    _K_zero = 15
)

var (
    _tr = mkreg(0, _K_temp)
    _tp = mkreg(1, _K_temp)
)

const (
    Rz Reg = (0 << 63) | (_K_zero << 59)
    Pn Reg = (1 << 63) | (_K_zero << 59)
)

func mkreg(ptr uint64, kind uint64) Reg {
    return Reg(((ptr & 1) << 63) | ((kind & 15) << 59))
}

func Tr() (r Reg) {
    r, _tr = _tr, _tr.Derive()
    return
}

func Pr() (r Reg) {
    r, _tp = _tp, _tp.Derive()
    return
}

func Rv(reg hir.Register) Reg {
    switch r := reg.(type) {
        case hir.GenericRegister : if r == hir.Rz { return Rz } else { return mkreg(0, uint64(r)) }
        case hir.PointerRegister : if r == hir.Pn { return Pn } else { return mkreg(1, uint64(r)) }
        default                  : panic("unreachable")
    }
}

func (self Reg) Ptr() bool {
    return self & _R_ptr != 0
}

func (self Reg) Kind() uint8 {
    return uint8((self & _R_kind) >> 59)
}

func (self Reg) Index() uint64 {
    return uint64(self & _R_index)
}

func (self Reg) Derive() Reg {
    return self.WithIndex(self.Index() + 1)
}

func (self Reg) String() string {
    if self.Kind() == _K_zero {
        if self.Ptr() {
            return "nil"
        } else {
            return "zero"
        }
    } else if self.Kind() == _K_temp {
        if self.Ptr() {
            return fmt.Sprintf("%%tp%d", self.Index())
        } else {
            return fmt.Sprintf("%%tr%d", self.Index())
        }
    } else {
        if self.Ptr() {
            return fmt.Sprintf("%%p%d.%d", self.Kind(), self.Index())
        } else {
            return fmt.Sprintf("%%r%d.%d", self.Kind(), self.Index())
        }
    }
}

func (self Reg) WithIndex(i uint64) Reg {
    return (self & (_R_ptr | _R_kind)) | Reg(i & _R_index)
}

type IrPhi struct {
    R  Reg
    Pr map[*BasicBlock]Reg
}

func (self *IrPhi) String() string {
    nb := len(self.Pr)
    ret := make([]string, 0, nb)

    /* add each path */
    for bb, reg := range self.Pr {
        ret = append(ret, fmt.Sprintf("%s @ bb_%d", reg, bb.Id))
    }

    /* join them together */
    return fmt.Sprintf(
        "%s = φ(%s)",
        self.R,
        strings.Join(ret, ", "),
    )
}

type IrSuccessors interface {
    Next() bool
    Block() *BasicBlock
    Value() (int64, bool)
}

type IrTerminator interface {
    fmt.Stringer
    Successors() IrSuccessors
    irterminator()
}

func (*IrSwitch) irterminator() {}
func (*IrReturn) irterminator() {}

type _SwitchSuccessors struct {
    k *int64
    v *BasicBlock
    r *BasicBlock
    p *rt.GoMapIterator
}

func (self *_SwitchSuccessors) Next() bool {
    if self.p.K != nil {
        self.k = (*int64)(self.p.K)
        self.v = *(**BasicBlock)(self.p.V)
        self.p.Next()
        return true
    } else if self.r != nil {
        self.k = nil
        self.v = self.r
        self.r = nil
        return true
    } else {
        return false
    }
}

func (self *_SwitchSuccessors) Block() *BasicBlock {
    return self.v
}

func (self *_SwitchSuccessors) Value() (int64, bool) {
    if self.k == nil {
        return 0, false
    } else {
        return *self.k, true
    }
}

type IrSwitch struct {
    V  Reg
    Ln *BasicBlock
    Br map[int64]*BasicBlock
}

func (self *IrSwitch) String() string {
    nb := len(self.Br)
    ret := make([]string, 0, nb)

    /* no branches */
    if nb == 0 {
        return fmt.Sprintf("goto bb_%d", self.Ln.Id)
    }

    /* add each case */
    for id, bb := range self.Br {
        ret = append(ret, fmt.Sprintf("  %d => bb_%d,", id, bb.Id))
    }

    /* default branch */
    ret = append(ret, fmt.Sprintf(
        "  _ => bb_%d,",
        self.Ln.Id,
    ))

    /* join them together */
    return fmt.Sprintf(
        "switch %s {\n%s\n}",
        self.V,
        strings.Join(ret, "\n"),
    )
}

func (self *IrSwitch) Successors() IrSuccessors {
    return &_SwitchSuccessors {
        v: nil,
        r: self.Ln,
        p: rt.MapIter(self.Br),
    }
}

type _EmptySuccessor struct{}
func (_EmptySuccessor) Next()  bool          { return false }
func (_EmptySuccessor) Block() *BasicBlock   { return nil }
func (_EmptySuccessor) Value() (int64, bool) { return 0, false }

type IrReturn struct {
    R []Reg
}

func (self *IrReturn) String() string {
    nb := len(self.R)
    ret := make([]string, 0, nb)

    /* dump registers */
    for _, r := range self.R {
        ret = append(ret, r.String())
    }

    /* join them together */
    return fmt.Sprintf(
        "ret {%s}",
        strings.Join(ret, ", "),
    )
}

func (self *IrReturn) Successors() IrSuccessors {
    return _EmptySuccessor{}
}

type IrNode interface {
    fmt.Stringer
    irnode()
}

func (*IrLoad)       irnode() {}
func (*IrStore)      irnode() {}
func (*IrLoadArg)    irnode() {}
func (*IrConstInt)   irnode() {}
func (*IrConstPtr)   irnode() {}
func (*IrLEA)        irnode() {}
func (*IrUnaryExpr)  irnode() {}
func (*IrBinaryExpr) irnode() {}
func (*IrBitTestSet) irnode() {}
func (*IrCall)       irnode() {}
func (*IrBlockZero)  irnode() {}
func (*IrBlockCopy)  irnode() {}
func (*IrBreakpoint) irnode() {}

type IrLoad struct {
    R    Reg
    Mem  Reg
    Size uint8
}

func (self *IrLoad) String() string {
    return fmt.Sprintf("%s = load.u%d(%s)", self.R, self.Size * 8, self.Mem)
}

type IrStore struct {
    R    Reg
    Mem  Reg
    Size uint8
}

func (self *IrStore) String() string {
    return fmt.Sprintf("store.u%d(%s -> *%s)", self.Size * 8, self.R, self.Mem)
}

type IrLoadArg struct {
    R  Reg
    Id uint64
}

func (self *IrLoadArg) String() string {
    return fmt.Sprintf("%s = load.arg(#%d)", self.R, self.Id)
}

type IrConstInt struct {
    R Reg
    V int64
}

func (self *IrConstInt) String() string {
    return fmt.Sprintf("%s = const.i64 %d", self.R, self.V)
}

type IrConstPtr struct {
    R Reg
    P unsafe.Pointer
}

func (self *IrConstPtr) String() string {
    return fmt.Sprintf("%s = const.ptr %p", self.R, self.P)
}

type IrLEA struct {
    R   Reg
    Mem Reg
    Off Reg
}

func (self *IrLEA) String() string {
    return fmt.Sprintf("%s = &(%s)[%s]", self.R, self.Mem, self.Off)
}

type (
    IrUnaryOp  uint8
    IrBinaryOp uint8
)

const (
    IrOpNegate IrUnaryOp = iota
    IrOpSwap16
    IrOpSwap32
    IrOpSwap64
)

const (
    IrOpAdd IrBinaryOp = iota
    IrOpSub
    IrOpMul
    IrOpAnd
    IrOpXor
    IrOpShr
    IrOpBitSet
    IrCmpEq
    IrCmpNe
    IrCmpLt
    IrCmpLtu
    IrCmpGeu
)

func (self IrUnaryOp) String() string {
    switch self {
        case IrOpSwap16 : return "bswap16"
        case IrOpSwap32 : return "bswap32"
        case IrOpSwap64 : return "bswap64"
        default         : panic("unreachable")
    }
}

func (self IrBinaryOp) String() string {
    switch self {
        case IrOpAdd  : return "+"
        case IrOpSub  : return "-"
        case IrOpMul  : return "*"
        case IrOpAnd  : return "&"
        case IrOpXor  : return "^"
        case IrOpShr  : return ">>"
        case IrCmpEq  : return "=="
        case IrCmpNe  : return "!="
        case IrCmpLt  : return "<"
        case IrCmpLtu : return "<#"
        case IrCmpGeu : return ">=#"
        default       : panic("unreachable")
    }
}

type IrUnaryExpr struct {
    R  Reg
    V  Reg
    Op IrUnaryOp
}

func (self *IrUnaryExpr) String() string {
    return fmt.Sprintf("%s = %s %s", self.R, self.Op, self.V)
}

type IrBinaryExpr struct {
    R  Reg
    X  Reg
    Y  Reg
    Op IrBinaryOp
}

func (self *IrBinaryExpr) String() string {
    return fmt.Sprintf("%s = %s %s %s", self.R, self.X, self.Op, self.Y)
}

type IrBitTestSet struct {
    T Reg
    S Reg
    X Reg
    Y Reg
}

func (self *IrBitTestSet) String() string {
    return fmt.Sprintf("t.%s, s.%s = bts %s, %s", self.T, self.S, self.X, self.Y)
}

type IrCall struct {
    Fn  hir.CallHandle
    In  []Reg
    Out []Reg
}

func (self *IrCall) String() string {
    in := make([]string, 0, len(self.In))
    out := make([]string, 0, len(self.Out))

    /* dump args and rets */
    for _, r := range self.In  { in = append(in, r.String()) }
    for _, r := range self.Out { out = append(out, r.String()) }

    /* join them together */
    return fmt.Sprintf(
        "%s = call %s, {%s}",
        strings.Join(out, ", "),
        self.Fn,
        strings.Join(in, ", "),
    )
}

type IrBlockZero struct {
    Mem Reg
    Len uintptr
}

func (self *IrBlockZero) String() string {
    return fmt.Sprintf("memset(%s, 0, %d)", self.Mem, self.Len)
}

type IrBlockCopy struct {
    Mem Reg
    Src Reg
    Len Reg
}

func (self *IrBlockCopy) String() string {
    return fmt.Sprintf("memmove(%s, %s, %s)", self.Mem, self.Src, self.Len)
}

type (
	IrBreakpoint struct{}
)

func (IrBreakpoint) String() string {
    return "breakpoint"
}
