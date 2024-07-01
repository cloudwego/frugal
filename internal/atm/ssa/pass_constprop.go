/*
 * Copyright 2022 CloudWeGo Authors
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
	"fmt"
	"math/bits"
	"unsafe"

	"github.com/cloudwego/frugal/internal/atm/abi"
)

// ConstProp propagates constant through the expression tree.
type ConstProp struct{}

func (ConstProp) unary(v int64, op IrUnaryOp) int64 {
	switch op {
	case IrOpNegate:
		return -v
	case IrOpSwap16:
		return int64(bits.ReverseBytes16(uint16(v)))
	case IrOpSwap32:
		return int64(bits.ReverseBytes32(uint32(v)))
	case IrOpSwap64:
		return int64(bits.ReverseBytes64(uint64(v)))
	case IrOpSx32to64:
		return int64(int32(v))
	default:
		panic(fmt.Sprintf("constprop: invalid unary operator: %d", op))
	}
}

func (ConstProp) binary(x int64, y int64, op IrBinaryOp) int64 {
	switch op {
	case IrOpAdd:
		return x + y
	case IrOpSub:
		return x - y
	case IrOpMul:
		return x * y
	case IrOpAnd:
		return x & y
	case IrOpOr:
		return x | y
	case IrOpXor:
		return x ^ y
	case IrOpShr:
		return x >> y
	case IrCmpEq:
		if x == y {
			return 1
		} else {
			return 0
		}
	case IrCmpNe:
		if x != y {
			return 1
		} else {
			return 0
		}
	case IrCmpLt:
		if x < y {
			return 1
		} else {
			return 0
		}
	case IrCmpLtu:
		if uint64(x) < uint64(y) {
			return 1
		} else {
			return 0
		}
	case IrCmpGeu:
		if uint64(x) >= uint64(y) {
			return 1
		} else {
			return 0
		}
	default:
		panic(fmt.Sprintf("constprop: invalid binary operator: %d", op))
	}
}

func (ConstProp) testandset(x int64, y int64) (int64, int64) {
	if bv := int64(1 << y); x&bv == 0 {
		return 0, x | bv
	} else {
		return 1, x | bv
	}
}

func (self ConstProp) Apply(cfg *CFG) {
	done := false
	consts := make(map[Reg]_ConstData)

	/* constant zero registers */
	consts[Rz] = constint(0)
	consts[Pn] = constptr(nil, Const)

	/* const adder */
	addconst := func(r Reg, v _ConstData) {
		if _, ok := consts[r]; !ok {
			done = false
			consts[r] = v
		}
	}

	/* evaluate const expression until no modifications were made */
	for !done {
		done = true
		for _, bb := range cfg.PostOrder().Reversed() {
			phi := make([]*IrPhi, 0, len(bb.Phi))
			ins := make([]IrNode, 0, len(bb.Ins))
			isconst := false

			/* check every Phi node */
			for _, p := range bb.Phi {
				var first bool
				var cdata _ConstData

				/* assume it's a const */
				first = true
				isconst = true

				/* a Phi node is a const iff all it's arguments are the same const */
				for _, r := range p.V {
					if *r != p.R {
						if cc, ok := consts[*r]; !ok {
							isconst = false
							break
						} else if first {
							cdata = cc
							first = false
						} else if cdata != cc {
							isconst = false
							break
						}
					}
				}

				/* not constant, keep it as is */
				if !isconst {
					phi = append(phi, p)
					continue
				}

				/* registers declared by Phi nodes can never be zero registers */
				if p.R.Kind() == K_zero {
					panic("constprop: assignment to zero registers in Phi node: " + p.String())
				}

				/* replace the Phi node with a Const node */
				if cdata.i {
					ins = append(ins, &IrConstInt{R: p.R, V: cdata.v})
				} else {
					ins = append(ins, &IrConstPtr{R: p.R, P: cdata.p, M: cdata.c})
				}

				/* mark as constant */
				done = false
				consts[p.R] = cdata
			}

			/* check every instructions */
			for _, v := range bb.Ins {
				switch p := v.(type) {
				default:
					{
						ins = append(ins, p)
					}

				/* value loads */
				case *IrLoad:
					{
						var ok bool
						var iv int64
						var cc _ConstData
						var pv unsafe.Pointer

						/* must be constant memory address */
						if cc, ok = consts[p.Mem]; !ok {
							ins = append(ins, p)
							break
						}

						/* address must be a pointer */
						if cc.i {
							panic(fmt.Sprintf("constprop: value load on integer value %#x: %s", cc.v, p))
						}

						/* do not load a volatile pointer */
						if cc.c != Const {
							ins = append(ins, p)
							break
						}

						/* constant pointer */
						if p.R.Ptr() {
							if p.Size != abi.PtrSize {
								panic("constprop: pointer load of invalid size: " + p.String())
							} else {
								pv = *(*unsafe.Pointer)(cc.p)
								ins = append(ins, &IrConstPtr{R: p.R, P: pv, M: Volatile})
								addconst(p.R, constptr(pv, Volatile))
								break
							}
						}

						/* read the integer */
						switch p.Size {
						case 1:
							iv = int64(*(*uint8)(cc.p))
						case 2:
							iv = int64(*(*uint16)(cc.p))
						case 4:
							iv = int64(*(*uint32)(cc.p))
						case 8:
							iv = int64(*(*uint64)(cc.p))
						default:
							panic("constprop: integer load of invalid size: " + p.String())
						}

						/* replace the instruction */
						ins = append(ins, &IrConstInt{R: p.R, V: iv})
						addconst(p.R, constint(iv))
					}

				/* integer constant */
				case *IrConstInt:
					{
						ins = append(ins, p)
						addconst(p.R, constint(p.V))
					}

				/* pointer constant */
				case *IrConstPtr:
					{
						ins = append(ins, p)
						addconst(p.R, constptr(p.P, p.M))
					}

				/* pointer arithmetics */
				case *IrLEA:
					{
						if mem, ok := consts[p.Mem]; !ok {
							ins = append(ins, p)
						} else if off, ok := consts[p.Off]; !ok {
							ins = append(ins, p)
						} else if mem.i {
							panic(fmt.Sprintf("constprop: pointer operation on integer value %#x: %s", mem.v, p))
						} else if !off.i {
							panic(fmt.Sprintf("constprop: pointer operation with pointer offset %p: %s", off.p, p))
						} else {
							r := addptr(mem.p, off.v)
							ins = append(ins, &IrConstPtr{R: p.R, P: r, M: Volatile})
							addconst(p.R, constptr(r, mem.c))
						}
					}

				/* unary expressions */
				case *IrUnaryExpr:
					{
						if cc, ok := consts[p.V]; !ok {
							ins = append(ins, p)
						} else if !cc.i {
							panic(fmt.Sprintf("constprop: integer operation on pointer value %p: %s", cc.p, p))
						} else {
							r := self.unary(cc.v, p.Op)
							ins = append(ins, &IrConstInt{R: p.R, V: r})
							addconst(p.R, constint(r))
						}
					}

				/* binary expressions */
				case *IrBinaryExpr:
					{
						if x, ok := consts[p.X]; !ok {
							ins = append(ins, p)
						} else if y, ok := consts[p.Y]; !ok {
							ins = append(ins, p)
						} else if !x.i {
							panic(fmt.Sprintf("constprop: integer operation on pointer value %p: %s", x.p, p))
						} else if !y.i {
							panic(fmt.Sprintf("constprop: integer operation on pointer value %p: %s", y.p, p))
						} else {
							r := self.binary(x.v, y.v, p.Op)
							ins = append(ins, &IrConstInt{R: p.R, V: r})
							addconst(p.R, constint(r))
						}
					}

				/* bit test and set operation */
				case *IrBitTestSet:
					{
						if x, ok := consts[p.X]; !ok {
							ins = append(ins, p)
						} else if y, ok := consts[p.Y]; !ok {
							ins = append(ins, p)
						} else if !x.i {
							panic(fmt.Sprintf("constprop: integer operation on pointer value %p: %s", x.p, p))
						} else if !y.i {
							panic(fmt.Sprintf("constprop: integer operation on pointer value %p: %s", y.p, p))
						} else {
							t, s := self.testandset(x.v, y.v)
							ins = append(ins, &IrConstInt{R: p.T, V: t}, &IrConstInt{R: p.S, V: s})
							addconst(p.T, constint(t))
							addconst(p.S, constint(s))
						}
					}
				}
			}

			/* rebuild the basic block */
			bb.Phi = phi
			bb.Ins = ins
		}
	}
}
