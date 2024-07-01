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

// CopyElim removes unnessecery register copies.
type CopyElim struct{}

func (CopyElim) Apply(cfg *CFG) {
	regs := make(map[Reg]Reg)
	consts := make(map[Reg]_ConstData)

	/* constant zero registers */
	consts[Rz] = constint(0)
	consts[Pn] = constptr(nil, Const)

	/* register replacement func */
	replacereg := func(rr *Reg) {
		var rv Reg
		var ok bool
		var cc _ConstData

		/* substitute registers */
		for {
			if rv, ok = regs[*rr]; ok {
				*rr = rv
			} else {
				break
			}
		}

		/* substitute zero registers */
		if cc, ok = consts[*rr]; ok {
			if cc.i && cc.v == 0 {
				*rr = Rz
			} else if !cc.i && cc.p == nil {
				*rr = Pn
			}
		}
	}

	/* Phase 1: Find all the constants */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		for _, v := range bb.Ins {
			switch p := v.(type) {
			case *IrConstInt:
				consts[p.R] = constint(p.V)
			case *IrConstPtr:
				consts[p.R] = constptr(p.P, p.M)
			}
		}
	})

	/* Phase 2: Identify all the identity operations */
	for _, bb := range cfg.PostOrder().Reversed() {
		for _, v := range bb.Ins {
			switch p := v.(type) {
			default:
				{
					continue
				}

			/* pointer arithmetic */
			case *IrLEA:
				{
					if cc, ok := consts[p.Off]; ok && cc.i && cc.v == 0 {
						regs[p.R] = p.Mem
					}
				}

			/* integer arithmetic */
			case *IrBinaryExpr:
				{
					var c bool
					var i int64
					var x _ConstData
					var y _ConstData

					/* calculate mathematical identities */
					switch p.Op {
					case IrOpAdd:
						i = 0
					case IrOpSub:
						i = 0
					case IrOpMul:
						i = 1
					case IrOpAnd:
						i = -1
					case IrOpOr:
						i = 0
					case IrOpXor:
						i = 0
					case IrOpShr:
						i = 0
					default:
						continue
					}

					/* check for identity operations */
					if x, c = consts[p.X]; c && x.i && x.v == i {
						regs[p.R] = p.Y
					} else if y, c = consts[p.Y]; c && y.i && y.v == i {
						regs[p.R] = p.X
					}
				}
			}
		}
	}

	/* Phase 3: Replace all the register references */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var ok bool
		var use IrUsages

		/* replace in Phi nodes */
		for _, v := range bb.Phi {
			for _, u := range v.Usages() {
				replacereg(u)
			}
		}

		/* replace in instructions */
		for _, v := range bb.Ins {
			if use, ok = v.(IrUsages); ok {
				for _, u := range use.Usages() {
					replacereg(u)
				}
			}
		}

		/* replace in terminators */
		if use, ok = bb.Term.(IrUsages); ok {
			for _, u := range use.Usages() {
				replacereg(u)
			}
		}
	})
}
