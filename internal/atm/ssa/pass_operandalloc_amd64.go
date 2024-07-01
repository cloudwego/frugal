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

// OperandAlloc for AMD64 converts 3-operand or 2-operand pseudo-instructions
// to 2-operand or one-operand real instructions.
type OperandAlloc struct{}

func (OperandAlloc) Apply(cfg *CFG) {
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		ins := bb.Ins
		bb.Ins = make([]IrNode, 0, len(ins))

		/* check for every instruction */
		for _, v := range ins {
			switch p := v.(type) {
			default:
				{
					bb.Ins = append(bb.Ins, v)
				}

			/* negation */
			case *IrAMD64_NEG:
				{
					if p.R == p.V {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.V = append(bb.Ins, IrArchCopy(p.R, p.V), v), p.R
					}
				}

			/* byte swap */
			case *IrAMD64_BSWAP:
				{
					if p.R == p.V {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.V = append(bb.Ins, IrArchCopy(p.R, p.V), v), p.R
					}
				}

			/* binary operations, register to register */
			case *IrAMD64_BinOp_rr:
				{
					if p.R == p.X {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.X = append(bb.Ins, IrArchCopy(p.R, p.X), v), p.R
					}
				}

			/* binary operations, register to immediate */
			case *IrAMD64_BinOp_ri:
				{
					if p.R == p.X || p.Op == IrAMD64_BinMul {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.X = append(bb.Ins, IrArchCopy(p.R, p.X), v), p.R
					}
				}

			/* binary operations, register to memory */
			case *IrAMD64_BinOp_rm:
				{
					if p.R == p.X {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.X = append(bb.Ins, IrArchCopy(p.R, p.X), v), p.R
					}
				}

			/* bit test and set, register to register */
			case *IrAMD64_BTSQ_rr:
				{
					if p.S == p.X {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.X = append(bb.Ins, IrArchCopy(p.S, p.X), v), p.S
					}
				}

			/* bit test and set, register to immediate */
			case *IrAMD64_BTSQ_ri:
				{
					if p.S == p.X {
						bb.Ins = append(bb.Ins, v)
					} else {
						bb.Ins, p.X = append(bb.Ins, IrArchCopy(p.S, p.X), v), p.S
					}
				}
			}
		}
	})
}
