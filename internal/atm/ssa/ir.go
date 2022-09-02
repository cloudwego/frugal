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

    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/cloudwego/frugal/internal/rt`
)

type (
    Reg        uint64
    Constness  uint8
	Likeliness uint8
)

const (
    _B_ptr  = 63
    _B_kind = 60
    _B_name = 52
)

const (
    _M_ptr  = 1
    _M_kind = 0x07
    _M_name = 0xff
)

const (
    _R_ptr   = _M_ptr << _B_ptr
    _R_kind  = _M_kind << _B_kind
    _R_name  = _M_name << _B_name
    _R_index = (1 << _B_name) - 1
)

const (
    K_sys  = 0
    K_zero = 1
    K_temp = 2
    K_arch = 3
    K_norm = 4
)

const (
    N_size = _M_name + 1
)

const (
    Rz Reg = (0 << _B_ptr) | (K_zero << _B_kind)
    Pn Reg = (1 << _B_ptr) | (K_zero << _B_kind)
)

const (
    Const Constness = iota
    Volatile
)

const (
    Likely Likeliness = iota
    Unlikely
)

func (self Constness) String() string {
    switch self {
        case Const    : return "const"
        case Volatile : return "volatile"
        default       : return "???"
    }
}

func (self Likeliness) String() string {
    switch self {
        case Likely   : return "likely"
        case Unlikely : return "unlikely"
        default       : return "???"
    }
}

func mksys(ptr uint64, kind uint64) Reg {
    if kind > N_size {
        panic(fmt.Sprintf("invalid register kind: %d", kind))
    } else {
        return mkreg(ptr, K_sys, kind)
    }
}

func mkreg(ptr uint64, kind uint64, name uint64) Reg {
    return Reg(((ptr & _M_ptr) << _B_ptr) | ((kind & _M_kind) << _B_kind) | ((name & _M_name) << _B_name))
}

func Tr(i int) Reg {
    if i < 0 || i > N_size {
        panic("invalid generic temporary register index")
    } else {
        return mkreg(0, K_temp, uint64(i))
    }
}

func Pr(i int) Reg {
    if i < 0 || i > N_size {
        panic("invalid generic temporary register index")
    } else {
        return mkreg(1, K_temp, uint64(i))
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

func (self Reg) Kind() int {
    return int((self & _R_kind) >> _B_kind)
}

func (self Reg) Name() int {
    return int((self & _R_name) >> _B_name)
}

func (self Reg) Index() int {
    return int(self & _R_index)
}

func (self Reg) String() string {
    switch self.Kind() {
        default: {
            if self.Ptr() {
                return fmt.Sprintf("p%d.%d", self.Kind(), self.Index())
            } else {
                return fmt.Sprintf("r%d.%d", self.Kind(), self.Index())
            }
        }

        /* arch-specific registers */
        case K_arch: {
            if i := self.Name(); i >= len(ArchRegs) {
                panic(fmt.Sprintf("invalid arch-specific register index: %d", i))
            } else if self.Index() == 0 {
                return fmt.Sprintf("%%%s", ArchRegNames[ArchRegs[i]])
            } else if self.Ptr() {
                return fmt.Sprintf("p%d{%%%s}", self.Index(), ArchRegNames[ArchRegs[i]])
            } else {
                return fmt.Sprintf("r%d{%%%s}", self.Index(), ArchRegNames[ArchRegs[i]])
            }
        }

        /* zero registers */
        case K_zero: {
            if self.Ptr() {
                return "nil"
            } else {
                return "zero"
            }
        }

        /* temp registers */
        case K_temp: {
            if self.Ptr() {
                return fmt.Sprintf("tp%d.%d", self.Name(), self.Index())
            } else {
                return fmt.Sprintf("tr%d.%d", self.Name(), self.Index())
            }
        }

        /* SSA normalized registers */
        case K_norm: {
            if self.Ptr() {
                return fmt.Sprintf("p%d", self.Index())
            } else {
                return fmt.Sprintf("r%d", self.Index())
            }
        }
    }
}

func (self Reg) Derive(i int) Reg {
    if self.Kind() == K_zero {
        return self
    } else {
        return (self & (_R_ptr | _R_kind | _R_name)) | Reg(i & _R_index)
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
    Clone() IrNode
    irnode()
}

type IrImpure interface {
    IrNode
    irimpure()
}

type IrImmovable interface {
    IrNode
    irimmovable()
}

func (*IrPhi)          irnode() {}
func (*IrSwitch)       irnode() {}
func (*IrReturn)       irnode() {}
func (*IrNop)          irnode() {}
func (*IrBreakpoint)   irnode() {}
func (*IrAlias)        irnode() {}
func (*IrEntry)        irnode() {}
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
func (*IrClobberList)  irnode() {}
func (*IrWriteBarrier) irnode() {}

func (*IrStore)        irimpure() {}
func (*IrCallFunc)     irimpure() {}
func (*IrCallNative)   irimpure() {}
func (*IrCallMethod)   irimpure() {}
func (*IrClobberList)  irimpure() {}
func (*IrWriteBarrier) irimpure() {}

func (*IrLoad)         irimmovable() {}
func (*IrStore)        irimmovable() {}
func (*IrAlias)        irimmovable() {}
func (*IrEntry)        irimmovable() {}
func (*IrLoadArg)      irimmovable() {}
func (*IrClobberList)  irimmovable() {}
func (*IrWriteBarrier) irimmovable() {}

type IrUsages interface {
    IrNode
    Usages() []*Reg
}

type IrDefinitions interface {
    IrNode
    Definitions() []*Reg
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

func (self *IrPhi) Clone() IrNode {
    ret := new(IrPhi)
    ret.V = make(map[*BasicBlock]*Reg, len(self.V))

    /* clone the Phi mappings */
    for b, r := range self.V {
        p := *r
        ret.V[b] = &p
    }

    /* set the dest register */
    ret.R = self.R
    return ret
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

func (self *IrPhi) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrBranch struct {
    To         *BasicBlock
    Likeliness Likeliness
}

func IrLikely(bb *BasicBlock) IrBranch {
    return IrBranch {
        To         : bb,
        Likeliness : Likely,
    }
}

func IrUnlikely(bb *BasicBlock) IrBranch {
    return IrBranch {
        To         : bb,
        Likeliness : Unlikely,
    }
}

func (self IrBranch) String() string {
    return fmt.Sprintf("bb_%d (%s)", self.To.Id, self.Likeliness)
}

type IrSuccessors interface {
    Next() bool
    Block() *BasicBlock
    Value() (int32, bool)
    Likeliness() Likeliness
}

type IrTerminator interface {
    IrNode
    Successors() IrSuccessors
    irterminator()
}

func (*IrSwitch) irterminator() {}
func (*IrReturn) irterminator() {}

type _SwitchTarget struct {
    i int32
    b IrBranch
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
        return self.t[self.i].b.To
    }
}

func (self *_SwitchSuccessors) Value() (int32, bool) {
    if self.i >= len(self.t) - 1 {
        return 0, false
    } else {
        return self.t[self.i].i, true
    }
}

func (self *_SwitchSuccessors) Likeliness() Likeliness {
    if self.i >= len(self.t) {
        return Unlikely
    } else {
        return self.t[self.i].b.Likeliness
    }
}

type IrSwitch struct {
    V  Reg
    Ln IrBranch
    Br map[int32]IrBranch
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

func (self *IrSwitch) Clone() IrNode {
    ret := new(IrSwitch)
    ret.Br = make(map[int32]IrBranch, len(ret.Br))

    /* clone the switch branches */
    for v, b := range self.Br {
        ret.Br[v] = b
    }

    /* set the switch register and default branch */
    ret.V = self.V
    ret.Ln = self.Ln
    return ret
}

func (self *IrSwitch) String() string {
    n := len(self.Br)
    r := make([]string, 0, n)

    /* no branches */
    if n == 0 {
        return "goto " + self.Ln.String()
    }

    /* add each case */
    for _, v := range self.iter().t[:n] {
        r = append(r, fmt.Sprintf("  %d => %s,", v.i, v.b))
    }

    /* default branch */
    r = append(r, fmt.Sprintf(
        "  _ => %s,",
        self.Ln,
    ))

    /* join them together */
    return fmt.Sprintf(
        "switch %s {\n%s\n}",
        self.V,
        strings.Join(r, "\n"),
    )
}

func (self *IrSwitch) Usages() []*Reg {
    if len(self.Br) == 0 {
        return nil
    } else {
        return []*Reg { &self.V }
    }
}

func (self *IrSwitch) Successors() IrSuccessors {
    return self.iter()
}

type _EmptySuccessor struct{}
func (_EmptySuccessor) Next()       bool          { return false }
func (_EmptySuccessor) Block()      *BasicBlock   { return nil }
func (_EmptySuccessor) Value()      (int32, bool) { return 0, false }
func (_EmptySuccessor) Likeliness() Likeliness    { return Unlikely }

type IrReturn struct {
    R []Reg
}

func (self *IrReturn) Clone() IrNode {
    r := new(IrReturn)
    r.R = make([]Reg, len(self.R))
    copy(r.R, self.R)
    return r
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

type (
    IrNop        struct{}
    IrBreakpoint struct{}
)

func (*IrNop)        Clone() IrNode { return new(IrNop) }
func (*IrBreakpoint) Clone() IrNode { return new(IrBreakpoint) }

func (*IrNop)        String() string { return "nop" }
func (*IrBreakpoint) String() string { return "breakpoint" }

type IrAlias struct {
    R Reg
    V Reg
}

func (self *IrAlias) Clone() IrNode {
    panic(`alias node "` + self.String() + `" is not cloneable`)
}

func (self *IrAlias) String() string {
    return fmt.Sprintf("alias %s = %s", self.R, self.V)
}

func (self *IrAlias) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrAlias) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrEntry struct {
    R []Reg
}

func (self *IrEntry) Clone() IrNode {
    panic(`entry node "` + self.String() + `" is not cloneable`)
}

func (self *IrEntry) String() string {
    return "entry_point " + regslicerepr(self.R)
}

func (self *IrEntry) Definitions() []*Reg {
    return regsliceref(self.R)
}

type IrLoad struct {
    R    Reg
    Mem  Reg
    Size uint8
}

func (self *IrLoad) Clone() IrNode {
    r := *self
    return &r
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

func (self *IrLoad) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrStore struct {
    R    Reg
    Mem  Reg
    Size uint8
}

func (self *IrStore) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrStore) String() string {
    return fmt.Sprintf("store.u%d %s -> *%s", self.Size * 8, self.R, self.Mem)
}

func (self *IrStore) Usages() []*Reg {
    return []*Reg { &self.R, &self.Mem }
}

type IrLoadArg struct {
    R Reg
    I int
}

func (self *IrLoadArg) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrLoadArg) String() string {
    if self.R.Ptr() {
        return fmt.Sprintf("%s = loadarg.ptr #%d", self.R, self.I)
    } else {
        return fmt.Sprintf("%s = loadarg.i64 #%d", self.R, self.I)
    }
}

func (self *IrLoadArg) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrConstInt struct {
    R Reg
    V int64
}

func (self *IrConstInt) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrConstInt) String() string {
    return fmt.Sprintf("%s = const.i64 %d (%#x)", self.R, self.V, self.V)
}

func (self *IrConstInt) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrConstPtr struct {
    R Reg
    P unsafe.Pointer
    M Constness
}

func (self *IrConstPtr) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrConstPtr) String() string {
    return fmt.Sprintf("%s = const.ptr (%s)%p [%s]", self.R, self.M, self.P, rt.FuncName(self.P))
}

func (self *IrConstPtr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrLEA struct {
    R   Reg
    Mem Reg
    Off Reg
}

func (self *IrLEA) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrLEA) String() string {
    return fmt.Sprintf("%s = &(%s)[%s]", self.R, self.Mem, self.Off)
}

func (self *IrLEA) Usages() []*Reg {
    return []*Reg { &self.Mem, &self.Off }
}

func (self *IrLEA) Definitions() []*Reg {
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

func (self *IrUnaryExpr) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrUnaryExpr) String() string {
    return fmt.Sprintf("%s = %s %s", self.R, self.Op, self.V)
}

func (self *IrUnaryExpr) Usages() []*Reg {
    return []*Reg { &self.V }
}

func (self *IrUnaryExpr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrBinaryExpr struct {
    R  Reg
    X  Reg
    Y  Reg
    Op IrBinaryOp
}

func IrCopy(r Reg, v Reg) IrNode {
    switch {
        case  r.Ptr() &&  v.Ptr() : return &IrLEA { R: r, Mem: v, Off: Rz }
        case !r.Ptr() && !v.Ptr() : return &IrBinaryExpr { R: r, X: v, Y: Rz, Op: IrOpAdd }
        default                   : panic("copy between different kind of registers")
    }
}

func (self *IrBinaryExpr) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrBinaryExpr) String() string {
    return fmt.Sprintf("%s = %s %s %s", self.R, self.X, self.Op, self.Y)
}

func (self *IrBinaryExpr) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrBinaryExpr) Definitions() []*Reg {
    return []*Reg { &self.R }
}

type IrBitTestSet struct {
    T Reg
    S Reg
    X Reg
    Y Reg
}

func (self *IrBitTestSet) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrBitTestSet) String() string {
    return fmt.Sprintf("t.%s, s.%s = bts %s, %s", self.T, self.S, self.X, self.Y)
}

func (self *IrBitTestSet) Usages() []*Reg {
    return []*Reg { &self.X, &self.Y }
}

func (self *IrBitTestSet) Definitions() []*Reg {
    return []*Reg { &self.T, &self.S }
}

type IrCallFunc struct {
    R    Reg
    In   []Reg
    Out  []Reg
    Func *abi.FunctionLayout
}

func (self *IrCallFunc) Clone() IrNode {
    r := new(IrCallFunc)
    r.R = self.R
    r.In = make([]Reg, len(self.In))
    r.Out = make([]Reg, len(self.Out))
    r.Func = self.Func
    copy(r.In, self.In)
    copy(r.Out, self.Out)
    return r
}

func (self *IrCallFunc) String() string {
    if in := regslicerepr(self.In); len(self.Out) == 0 {
        return fmt.Sprintf("gcall *%s, {%s}", self.R, in)
    } else {
        return fmt.Sprintf("%s = gcall *%s, {%s}", regslicerepr(self.Out), self.R, in)
    }
}

func (self *IrCallFunc) Usages() []*Reg {
    return append(regsliceref(self.In), &self.R)
}

func (self *IrCallFunc) Definitions() []*Reg {
    return regsliceref(self.Out)
}

type IrCallNative struct {
    R   Reg
    In  []Reg
    Out Reg
}

func (self *IrCallNative) Clone() IrNode {
    r := new(IrCallNative)
    r.R = self.R
    r.In = make([]Reg, len(self.In))
    r.Out = self.Out
    copy(r.In, self.In)
    return r
}

func (self *IrCallNative) String() string {
    if in := regslicerepr(self.In); self.Out.Kind() == K_zero {
        return fmt.Sprintf("ccall *%s, {%s}", self.R, in)
    } else {
        return fmt.Sprintf("%s = ccall *%s, {%s}", self.Out, self.R, in)
    }
}

func (self *IrCallNative) Usages() []*Reg {
    return append(regsliceref(self.In), &self.R)
}

func (self *IrCallNative) Definitions() []*Reg {
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
    Func *abi.FunctionLayout
}

func (self *IrCallMethod) Clone() IrNode {
    r := new(IrCallMethod)
    r.T = self.T
    r.V = self.V
    r.In = make([]Reg, len(self.In))
    r.Out = make([]Reg, len(self.Out))
    r.Slot = self.Slot
    r.Func = self.Func
    copy(r.In, self.In)
    copy(r.Out, self.Out)
    return r
}

func (self *IrCallMethod) String() string {
    if in := regslicerepr(self.In); len(self.Out) == 0 {
        return fmt.Sprintf("icall #%d, (%s:%s), {%s}", self.Slot, self.T, self.V, in)
    } else {
        return fmt.Sprintf("%s = icall #%d, (%s:%s), {%s}", regslicerepr(self.Out), self.Slot, self.T, self.V, in)
    }
}

func (self *IrCallMethod) Usages() []*Reg {
    return append(regsliceref(self.In), &self.T, &self.V)
}

func (self *IrCallMethod) Definitions() []*Reg {
    return regsliceref(self.Out)
}

type IrClobberList struct {
    R []Reg
}

func (self *IrClobberList) Clone() IrNode {
    r := new(IrClobberList)
    r.R = make([]Reg, len(self.R))
    copy(r.R, self.R)
    return r
}

func (self *IrClobberList) String() string {
    return "drop " + regslicerepr(self.R)
}

func (self *IrClobberList) Usages() []*Reg {
    return regsliceref(self.R)
}

type IrWriteBarrier struct {
    R   Reg
    M   Reg
    Fn  Reg
    Var Reg
}

func (self *IrWriteBarrier) Clone() IrNode {
    r := *self
    return &r
}

func (self *IrWriteBarrier) String() string {
    return fmt.Sprintf("write_barrier (%s:%s), %s -> *%s", self.Var, self.Fn, self.R, self.M)
}

func (self *IrWriteBarrier) Usages() []*Reg {
    return []*Reg { &self.R, &self.M, &self.Var, &self.Fn }
}
