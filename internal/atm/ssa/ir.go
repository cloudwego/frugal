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
    `sort`
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/hir`
)

type Reg uint64

const (
    _B_ptr  = 63
    _B_kind = 59
)

const (
    _M_ptr  = 1
    _M_kind = 0x0f
)

const (
    _R_ptr   = _M_ptr << _B_ptr
    _R_kind  = _M_kind << _B_kind
    _R_index = (1 << _B_kind) - 1
)

const (
    _K_max  = 7
    _K_arch = 12
    _K_zero = 13
    _K_temp = 14
    _K_norm = 15
)

const (
    Rz Reg = (0 << _B_ptr) | (_K_zero << _B_kind)
    Pn Reg = (1 << _B_ptr) | (_K_zero << _B_kind)
)

const (
    Tr Reg = (0 << _B_ptr) | (_K_temp << _B_kind)
    Pr Reg = (1 << _B_ptr) | (_K_temp << _B_kind)
)

func mkreg(ptr uint64, kind uint64) Reg {
    if kind > _K_max {
        panic(fmt.Sprintf("invalid register kind: %d", kind))
    } else {
        return Reg(((ptr & _M_ptr) << _B_ptr) | ((kind & _M_kind) << _B_kind))
    }
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

func (self Reg) Index() int {
    return int(self & _R_index)
}

func (self Reg) String() string {
    switch self.kind() {
        default: {
            if self.Ptr() {
                return fmt.Sprintf("%%p%d.%d", self.kind(), self.Index())
            } else {
                return fmt.Sprintf("%%r%d.%d", self.kind(), self.Index())
            }
        }

        /* arch-specific registers */
        case _K_arch: {
            if i := self.Index(); i >= len(ArchRegs) {
                panic(fmt.Sprintf("invalid arch-specific register index: %d", i))
            } else {
                return "%" + ArchRegNames[ArchRegs[i]]
            }
        }

        /* zero registers */
        case _K_zero: {
            if self.Ptr() {
                return "nil"
            } else {
                return "$0"
            }
        }

        /* temp registers */
        case _K_temp: {
            if self.Ptr() {
                return fmt.Sprintf("%%tp%d", self.Index())
            } else {
                return fmt.Sprintf("%%tr%d", self.Index())
            }
        }

        /* SSA normalized registers */
        case _K_norm: {
            if self.Ptr() {
                return fmt.Sprintf("%%p%d", self.Index())
            } else {
                return fmt.Sprintf("%%r%d", self.Index())
            }
        }
    }
}

func (self Reg) zero() Reg {
    if self.Ptr() {
        return Pn
    } else {
        return Rz
    }
}

func (self Reg) kind() uint8 {
    return uint8((self & _R_kind) >> _B_kind)
}

func (self Reg) rename(i int) Reg {
    return (self & (_R_ptr | _R_kind)) | Reg(i & _R_index)
}

func (self Reg) normalize(i int) Reg {
    return (self & _R_ptr) | (_K_norm << _B_kind) | Reg(i & _R_index)
}

type IrNode interface {
    fmt.Stringer
    irnode()
}

func (*IrPhi)        irnode() {}
func (*IrSwitch)     irnode() {}
func (*IrReturn)     irnode() {}
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

type IrUsages interface {
    IrNode
    Usages() []*Reg
}

type IrDefinations interface {
    IrNode
    Definations() []*Reg
}

type _PhiSorter struct {
    k []int
    v []*Reg
}

func (self _PhiSorter) Len() int {
    return len(self.k)
}

func (self _PhiSorter) Swap(i int, j int) {
    self.k[i], self.k[j] = self.k[j], self.k[i]
    self.v[i], self.v[j] = self.v[j], self.v[i]
}

func (self _PhiSorter) Less(i int, j int) bool {
    return self.k[i] < self.k[j]
}

type IrPhi struct {
    R Reg
    V map[*BasicBlock]*Reg
}

func (self *IrPhi) String() string {
    nb := len(self.V)
    ret := make([]string, 0, nb)
    phi := make([]struct{int; Reg}, 0, nb)

    /* add each path */
    for bb, reg := range self.V {
        phi = append(phi, struct{int; Reg}{bb.Id, *reg})
    }

    /* sort by basic block ID */
    sort.Slice(phi, func(i int, j int) bool {
        return phi[i].int < phi[j].int
    })

    /* dump as string */
    for _, p := range phi {
        ret = append(ret, fmt.Sprintf("bb_%d: %s", p.int, p.Reg))
    }

    /* join them together */
    return fmt.Sprintf(
        "%s = φ(%s)",
        self.R,
        strings.Join(ret, ", "),
    )
}

func (self *IrPhi) Usages() []*Reg {
    k := make([]int, 0, len(self.V))
    v := make([]*Reg, 0, len(self.V))

    /* dump the registers */
    for b, r := range self.V {
        v = append(v, r)
        k = append(k, b.Id)
    }

    /* sort by basic block ID */
    sort.Sort(_PhiSorter { k, v })
    return v
}

func (self *IrPhi) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrSuccessors interface {
    Next() bool
    Block() *BasicBlock
    Value() (int64, bool)
}

type IrTerminator interface {
    IrNode
    Successors() IrSuccessors
    irterminator()
}

func (*IrSwitch) irterminator() {}
func (*IrReturn) irterminator() {}

type _SwitchSorter struct {
    k []int64
    v []*BasicBlock
}

func newSwitchSorter(k []int64, v []*BasicBlock) _SwitchSorter {
    return _SwitchSorter {
        k: k,
        v: v,
    }
}

func (self _SwitchSorter) Len() int {
    return len(self.k)
}

func (self _SwitchSorter) Swap(i int, j int) {
    self.k[i], self.k[j] = self.k[j], self.k[i]
    self.v[i], self.v[j] = self.v[j], self.v[i]
}

func (self _SwitchSorter) Less(i int, j int) bool {
    return self.k[i] < self.k[j]
}

type _SwitchSuccessors struct {
    i int
    k []int64
    v []*BasicBlock
}

func (self *_SwitchSuccessors) Next() bool {
    self.i++
    return self.i < len(self.v)
}

func (self *_SwitchSuccessors) Block() *BasicBlock {
    if self.i >= len(self.v) {
        return nil
    } else {
        return self.v[self.i]
    }
}

func (self *_SwitchSuccessors) Value() (int64, bool) {
    if self.i >= len(self.k) {
        return 0, false
    } else {
        return self.k[self.i], true
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

func (self *IrSwitch) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrSwitch) Successors() IrSuccessors {
    n := len(self.Br)
    k := make([]int64, 0, n)
    v := make([]*BasicBlock, 0, n + 1)

    /* add the key and values */
    for i, b := range self.Br {
        k = append(k, i)
        v = append(v, b)
    }

    /* add the default branch, and sort by value */
    v = append(v, self.Ln)
    sort.Sort(newSwitchSorter(k, v))

    /* construct the iterator */
    return &_SwitchSuccessors {
        k: k,
        v: v,
        i: -1,
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

func (self *IrReturn) Usages() []*Reg {
    return regsliceref(self.R)
}

func (self *IrReturn) Successors() IrSuccessors {
    return _EmptySuccessor{}
}

type IrLoad struct {
    R    Reg
    Mem  Reg
    Size uint8
}

func (self *IrLoad) String() string {
    return fmt.Sprintf("%s = load.u%d %s", self.R, self.Size * 8, self.Mem)
}

func (self *IrLoad) Usages() []*Reg {
    return []*Reg { &self.Mem }
}

func (self *IrLoad) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrStore struct {
    R    Reg
    Mem  Reg
    Size uint8
}

func (self *IrStore) String() string {
    return fmt.Sprintf("store.u%d(%s -> *%s)", self.Size * 8, self.R, self.Mem)
}

func (self *IrStore) Usages() []*Reg {
    return []*Reg { &self.R, &self.Mem }
}

type IrLoadArg struct {
    R  Reg
    Id uint64
}

func (self *IrLoadArg) String() string {
    return fmt.Sprintf("%s = load.arg(#%d)", self.R, self.Id)
}

func (self *IrLoadArg) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrConstInt struct {
    R Reg
    V int64
}

func (self *IrConstInt) String() string {
    return fmt.Sprintf("%s = const.i64 %d", self.R, self.V)
}

func (self *IrConstInt) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrConstPtr struct {
    R Reg
    P unsafe.Pointer
}

func (self *IrConstPtr) String() string {
    return fmt.Sprintf("%s = const.ptr %p", self.R, self.P)
}

func (self *IrConstPtr) Definations() []*Reg {
    return []*Reg { &self.R }
}

type IrLEA struct {
    R   Reg
    Mem Reg
    Off Reg
}

func (self *IrLEA) String() string {
    return fmt.Sprintf("%s = &(%s)[%s]", self.R, self.Mem, self.Off)
}

func (self *IrLEA) Usages() []*Reg {
    return []*Reg { &self.Mem, &self.Off }
}

func (self *IrLEA) Definations() []*Reg {
    return []*Reg { &self.R }
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
    IrOpSx32to64
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
        case IrOpNegate   : return "negate"
        case IrOpSwap16   : return "bswap16"
        case IrOpSwap32   : return "bswap32"
        case IrOpSwap64   : return "bswap64"
        case IrOpSx32to64 : return "sign_extend_32_to_64"
        default           : panic("unreachable")
    }
}

func (self IrBinaryOp) String() string {
    switch self {
        case IrOpAdd    : return "+"
        case IrOpSub    : return "-"
        case IrOpMul    : return "*"
        case IrOpAnd    : return "&"
        case IrOpXor    : return "^"
        case IrOpShr    : return ">>"
        case IrOpBitSet : return "|=#"
        case IrCmpEq    : return "=="
        case IrCmpNe    : return "!="
        case IrCmpLt    : return "<"
        case IrCmpLtu   : return "<#"
        case IrCmpGeu   : return ">=#"
        default         : panic("unreachable")
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

func (self *IrUnaryExpr) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrUnaryExpr) Definations() []*Reg {
    return []*Reg { &self.R }
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

func (self *IrBinaryExpr) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrBinaryExpr) Definations() []*Reg {
    return []*Reg { &self.R }
}

func IrCopy(r Reg, v Reg) *IrBinaryExpr {
    return &IrBinaryExpr {
        R  : r,
        X  : v,
        Y  : Rz,
        Op : IrOpAdd,
    }
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

func (self *IrBitTestSet) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrBitTestSet) Definations() []*Reg {
    return []*Reg { &self.T, &self.S }
}

type IrReceiver struct {
    T Reg
    V Reg
}

type IrCall struct {
    Fn  *hir.CallHandle
    Rx  *IrReceiver
    In  []Reg
    Out []Reg
}

func (self *IrCall) String() string {
    var desc string
    var kind string
    var recv string

    /* check for receivers */
    if (self.Rx == nil) == (self.Fn.Type == hir.ICall) {
        panic("invalid receiver value")
    }

    /* argument buffer */
    in := make([]string, 0, len(self.In))
    out := make([]string, 0, len(self.Out))

    /* convert call type */
    switch self.Fn.Type {
        case hir.CCall : kind = "ccall"
        case hir.GCall : kind = "gcall"
        case hir.ICall : kind = "icall"
        default        : panic("invalid call type")
    }

    /* convert function descriptor */
    if self.Fn.Type != hir.ICall {
        desc = self.Fn.String()
    } else {
        desc = fmt.Sprintf("#%d", self.Fn.Slot)
    }

    /* add receiver type if any */
    if self.Rx != nil {
        recv = fmt.Sprintf(", {%s, %s}", self.Rx.T, self.Rx.V)
    }

    /* dump args and rets */
    for _, r := range self.In  { in = append(in, r.String()) }
    for _, r := range self.Out { out = append(out, r.String()) }

    /* join them together */
    return fmt.Sprintf(
        "%s = %s %s%s, {%s}",
        strings.Join(out, ", "),
        kind,
        desc,
        recv,
        strings.Join(in, ", "),
    )
}

func (self *IrCall) Usages() []*Reg {
    if in := regsliceref(self.In); self.Rx == nil {
        return in
    } else {
        return append([]*Reg { &self.Rx.T, &self.Rx.V }, in...)
    }
}

func (self *IrCall) Definations() []*Reg {
    return regsliceref(self.Out)
}

type IrBlockZero struct {
    Mem Reg
    Len uintptr
}

func (self *IrBlockZero) String() string {
    return fmt.Sprintf("memset(%s, 0, %d)", self.Mem, self.Len)
}

func (self *IrBlockZero) Usages() []*Reg {
    return []*Reg { &self.Mem }
}

type IrBlockCopy struct {
    Mem Reg
    Src Reg
    Len Reg
}

func (self *IrBlockCopy) String() string {
    return fmt.Sprintf("memmove(%s, %s, %s)", self.Mem, self.Src, self.Len)
}

func (self *IrBlockCopy) Usages() []*Reg {
    return []*Reg { &self.Mem, &self.Src, &self.Len }
}

type (
	IrBreakpoint struct{}
)

func (IrBreakpoint) String() string {
    return "breakpoint"
}
