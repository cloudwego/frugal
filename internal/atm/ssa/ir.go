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
    K_sys  = 7
    K_zero = 8
    K_tmp0 = 9
    K_tmp1 = 10
    K_tmp2 = 11
    K_tmp3 = 12
    K_tmp4 = 13
    K_arch = 14
    K_norm = 15
)

const (
    Rz Reg = (0 << _B_ptr) | (K_zero << _B_kind)
    Pn Reg = (1 << _B_ptr) | (K_zero << _B_kind)
)

func mkreg(ptr uint64, kind uint64) Reg {
    return Reg(((ptr & _M_ptr) << _B_ptr) | ((kind & _M_kind) << _B_kind))
}

func mksys(ptr uint64, kind uint64) Reg {
    if kind > K_sys {
        panic(fmt.Sprintf("invalid register kind: %d", kind))
    } else {
        return mkreg(ptr, kind)
    }
}

func Tr(i uint64) Reg {
    if i > K_tmp4 - K_tmp0 {
        panic("invalid generic temporary register index")
    } else {
        return mkreg(0, K_tmp0 + i)
    }
}

func Pr(i uint64) Reg {
    if i > K_tmp4 - K_tmp0 {
        panic("invalid generic temporary register index")
    } else {
        return mkreg(1, K_tmp0 + i)
    }
}

func Rv(reg hir.Register) Reg {
    switch r := reg.(type) {
        case hir.GenericRegister : if r == hir.Rz { return Rz } else { return mksys(0, uint64(r)) }
        case hir.PointerRegister : if r == hir.Pn { return Pn } else { return mksys(1, uint64(r)) }
        default                  : panic("unreachable")
    }
}

func (self Reg) Ptr() bool {
    return self & _R_ptr != 0
}

func (self Reg) Zero() Reg {
    return (self & _R_ptr) | (K_zero << _B_kind)
}

func (self Reg) Kind() uint8 {
    return uint8((self & _R_kind) >> _B_kind)
}

func (self Reg) Index() int {
    return int(self & _R_index)
}

func (self Reg) String() string {
    switch self.Kind() {
        default: {
            if self.Ptr() {
                return fmt.Sprintf("%%p%d.%d", self.Kind(), self.Index())
            } else {
                return fmt.Sprintf("%%r%d.%d", self.Kind(), self.Index())
            }
        }

        /* arch-specific registers */
        case K_arch: {
            if i := self.Index(); i >= len(ArchRegs) {
                panic(fmt.Sprintf("invalid arch-specific register index: %d", i))
            } else {
                return "%" + ArchRegNames[ArchRegs[i]]
            }
        }

        /* zero registers */
        case K_zero: {
            if self.Ptr() {
                return "nil"
            } else {
                return "$0"
            }
        }

        /* temp registers */
        case K_tmp0, K_tmp1, K_tmp2, K_tmp3, K_tmp4: {
            if self.Ptr() {
                return fmt.Sprintf("%%tp%d.%d", self.Kind() - K_tmp0, self.Index())
            } else {
                return fmt.Sprintf("%%tr%d.%d", self.Kind() - K_tmp0, self.Index())
            }
        }

        /* SSA normalized registers */
        case K_norm: {
            if self.Ptr() {
                return fmt.Sprintf("%%p%d", self.Index())
            } else {
                return fmt.Sprintf("%%r%d", self.Index())
            }
        }
    }
}

func (self Reg) Derive(i int) Reg {
    if self.Kind() == K_zero {
        return self
    } else {
        return (self & (_R_ptr | _R_kind)) | Reg(i & _R_index)
    }
}

func (self Reg) Normalize(i int) Reg {
    if self.Kind() == K_zero {
        return self
    } else {
        return (self & _R_ptr) | (K_norm << _B_kind) | Reg(i & _R_index)
    }
}

type IrNode interface {
    fmt.Stringer
    irnode()
}

func (*IrPhi)          irnode() {}
func (*IrSwitch)       irnode() {}
func (*IrReturn)       irnode() {}
func (*IrLoad)         irnode() {}
func (*IrStore)        irnode() {}
func (*IrLoadArg)      irnode() {}
func (*IrConstInt)     irnode() {}
func (*IrConstPtr)     irnode() {}
func (*IrLEA)          irnode() {}
func (*IrUnaryExpr)    irnode() {}
func (*IrBinaryExpr)   irnode() {}
func (*IrBitTestSet)   irnode() {}
func (*IrCallFunc)     irnode() {}
func (*IrCallNative)   irnode() {}
func (*IrCallMethod)   irnode() {}
func (*IrWriteBarrier) irnode() {}
func (*IrBreakpoint)   irnode() {}

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
    phi := make([]struct { int; Reg }, 0, nb)

    /* add each path */
    for bb, reg := range self.V {
        phi = append(phi, struct { int; Reg }{ bb.Id, *reg })
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

type _SwitchTarget struct {
    i int64
    b *BasicBlock
}

type _SwitchSuccessors struct {
    i int
    t []_SwitchTarget
}

func (self *_SwitchSuccessors) Next() bool {
    self.i++
    return self.i < len(self.t)
}

func (self *_SwitchSuccessors) Block() *BasicBlock {
    if self.i >= len(self.t) {
        return nil
    } else {
        return self.t[self.i].b
    }
}

func (self *_SwitchSuccessors) Value() (int64, bool) {
    if self.i >= len(self.t) - 1 {
        return 0, false
    } else {
        return self.t[self.i].i, true
    }
}

type IrSwitch struct {
    V  Reg
    Ln *BasicBlock
    Br map[int64]*BasicBlock
}

func (self *IrSwitch) iter() *_SwitchSuccessors {
    n := len(self.Br)
    t := make([]_SwitchTarget, 0, n + 1)

    /* add the key and values */
    for i, b := range self.Br {
        t = append(t, _SwitchTarget {
            i: i,
            b: b,
        })
    }

    /* add the default branch */
    t = append(t, _SwitchTarget {
        i: 0,
        b: self.Ln,
    })

    /* sort by switch value */
    sort.Slice(t[:n], func(i int, j int) bool {
        return t[i].i < t[j].i
    })

    /* construct the iterator */
    return &_SwitchSuccessors {
        t: t,
        i: -1,
    }
}

func (self *IrSwitch) String() string {
    n := len(self.Br)
    r := make([]string, 0, n)

    /* no branches */
    if n == 0 {
        return fmt.Sprintf("goto bb_%d", self.Ln.Id)
    }

    /* add each case */
    for _, v := range self.iter().t[:n] {
        r = append(r, fmt.Sprintf("  %d => bb_%d,", v.i, v.b.Id))
    }

    /* default branch */
    r = append(r, fmt.Sprintf(
        "  _ => bb_%d,",
        self.Ln.Id,
    ))

    /* join them together */
    return fmt.Sprintf(
        "switch %s {\n%s\n}",
        self.V,
        strings.Join(r, "\n"),
    )
}

func (self *IrSwitch) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrSwitch) Successors() IrSuccessors {
    return self.iter()
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
    if self.R.Ptr() {
        return fmt.Sprintf("%s = load.ptr %s", self.R, self.Mem)
    } else {
        return fmt.Sprintf("%s = load.u%d %s", self.R, self.Size * 8, self.Mem)
    }
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
    return fmt.Sprintf("store.u%d %s -> *%s", self.Size * 8, self.R, self.Mem)
}

func (self *IrStore) Usages() []*Reg {
    return []*Reg { &self.R, &self.Mem }
}

type IrLoadArg struct {
    R  Reg
    Id uint64
}

func (self *IrLoadArg) String() string {
    if self.R.Ptr() {
        return fmt.Sprintf("%s = loadarg.ptr #%d", self.R, self.Id)
    } else {
        return fmt.Sprintf("%s = loadarg.i64 #%d", self.R, self.Id)
    }
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
    if fn := runtime.FuncForPC(uintptr(self.P)); fn == nil {
        return fmt.Sprintf("%s = const.ptr %p", self.R, self.P)
    } else if fp := fn.Entry(); fp == uintptr(self.P) {
        return fmt.Sprintf("%s = const.ptr %p [%s]", self.R, self.P, fn.Name())
    } else {
        return fmt.Sprintf("%s = const.ptr %p [%s+%#x]", self.R, self.P, fn.Name(), uintptr(self.P) - fp)
    }
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
    IrOpOr
    IrOpXor
    IrOpShr
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
        case IrOpOr     : return "|"
        case IrOpXor    : return "^"
        case IrOpShr    : return ">>"
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

func IrCopy(r Reg, v Reg) *IrBinaryExpr {
    return &IrBinaryExpr {
        R  : r,
        X  : v,
        Y  : Rz,
        Op : IrOpAdd,
    }
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

type IrCallFunc struct {
    R   Reg
    In  []Reg
    Out []Reg
}

func (self *IrCallFunc) String() string {
    if in := regslicerepr(self.In); len(self.Out) == 0 {
        return fmt.Sprintf("gcall *%s, {%s}", self.R, in)
    } else {
        return fmt.Sprintf("%s = gcall *%s, {%s}", regslicerepr(self.Out), self.R, in)
    }
}

func (self *IrCallFunc) Usages() []*Reg {
    return append([]*Reg { &self.R }, regsliceref(self.In)...)
}

func (self *IrCallFunc) Definations() []*Reg {
    return regsliceref(self.Out)
}

type IrCallNative struct {
    R   Reg
    In  []Reg
    Out Reg
}

func (self *IrCallNative) String() string {
    if in := regslicerepr(self.In); self.Out.Kind() == K_zero {
        return fmt.Sprintf("ccall *%s, {%s}", self.R, in)
    } else {
        return fmt.Sprintf("%s = ccall *%s, {%s}", self.Out, self.R, in)
    }
}

func (self *IrCallNative) Usages() []*Reg {
    return append([]*Reg { &self.R }, regsliceref(self.In)...)
}

func (self *IrCallNative) Definations() []*Reg {
    if self.Out.Kind() == K_zero {
        return nil
    } else {
        return []*Reg { &self.Out }
    }
}

type IrCallMethod struct {
    T    Reg
    V    Reg
    In   []Reg
    Out  []Reg
    Slot int
}

func (self *IrCallMethod) String() string {
    if in := regslicerepr(self.In); len(self.Out) == 0 {
        return fmt.Sprintf("icall #%d, (%s:%s), {%s}", self.Slot, self.T, self.V, in)
    } else {
        return fmt.Sprintf("%s = icall #%d, (%s:%s), {%s}", regslicerepr(self.Out), self.Slot, self.T, self.V, in)
    }
}

func (self *IrCallMethod) Usages() []*Reg {
    return regsliceref(self.In)
}

func (self *IrCallMethod) Definations() []*Reg {
    return append([]*Reg { &self.T, &self.V }, regsliceref(self.Out)...)
}

type IrWriteBarrier struct {
    R   Reg
    V   Reg
    Fn  Reg
    Var Reg
}

func (self *IrWriteBarrier) String() string {
    return fmt.Sprintf("write_barrier (%s:%s), %s -> *%s", self.Var, self.Fn, self.V, self.R)
}

func (self *IrWriteBarrier) Usages() []*Reg {
    return []*Reg { &self.R, &self.V, &self.Fn, &self.Var }
}

type (
	IrBreakpoint struct{}
)

func (IrBreakpoint) String() string {
    return "breakpoint"
}
