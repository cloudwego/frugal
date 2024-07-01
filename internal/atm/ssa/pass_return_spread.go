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

// ReturnSpread spreads the return block to all it's
// successors, in order to shorten register live ranges.
type ReturnSpread struct{}

func (ReturnSpread) Apply(cfg *CFG) {
	more := true
	rets := make([]*BasicBlock, 0, 1)

	/* register replacer */
	replaceregs := func(rr map[Reg]Reg, ins IrNode) {
		var v Reg
		var ok bool
		var use IrUsages
		var def IrDefinitions

		/* replace register usages */
		if use, ok = ins.(IrUsages); ok {
			for _, r := range use.Usages() {
				if v, ok = rr[*r]; ok {
					*r = v
				}
			}
		}

		/* replace register definitions */
		if def, ok = ins.(IrDefinitions); ok {
			for _, r := range def.Definitions() {
				if v, ok = rr[*r]; ok {
					*r = v
				}
			}
		}
	}

	/* loop until no more modifications */
	for more {
		more = false
		rets = rets[:0]

		/* Phase 1: Find the return blocks that has more than one predecessors */
		for _, bb := range cfg.PostOrder().Reversed() {
			if _, ok := bb.Term.(*IrReturn); ok && len(bb.Pred) > 1 {
				more = true
				rets = append(rets, bb)
			}
		}

		/* Phase 2: Spread the blocks to it's predecessors */
		for _, bb := range rets {
			for _, pred := range bb.Pred {
				var ok bool
				var sw *IrSwitch

				/* register mappings */
				rr := make(map[Reg]Reg)
				nb := len(bb.Phi) + len(bb.Ins)

				/* allocate registers for Phi definitions */
				for _, phi := range bb.Phi {
					rr[phi.R] = cfg.CreateRegister(phi.R.Ptr())
				}

				/* allocate registers for instruction definitions */
				for _, ins := range bb.Ins {
					if def, ok := ins.(IrDefinitions); ok {
						for _, r := range def.Definitions() {
							rr[*r] = cfg.CreateRegister(r.Ptr())
						}
					}
				}

				/* create a new basic block */
				ret := cfg.CreateBlock()
				ret.Ins = make([]IrNode, 0, nb)
				ret.Pred = []*BasicBlock{pred}

				/* add copy instruction for Phi nodes */
				for _, phi := range bb.Phi {
					ret.Ins = append(ret.Ins, IrCopy(rr[phi.R], *phi.V[pred]))
				}

				/* copy all instructions */
				for _, ins := range bb.Ins {
					ins = ins.Clone()
					ret.Ins = append(ret.Ins, ins)
					replaceregs(rr, ins)
				}

				/* copy the terminator */
				ret.Term = bb.Term.Clone().(IrTerminator)
				replaceregs(rr, ret.Term)

				/* link to the predecessor */
				if sw, ok = pred.Term.(*IrSwitch); !ok {
					panic("invalid block terminator: " + pred.Term.String())
				}

				/* check for default branch */
				if sw.Ln.To == bb {
					sw.Ln.To = ret
					continue
				}

				/* replace the switch targets */
				for v, b := range sw.Br {
					if b.To == bb {
						sw.Br[v] = &IrBranch{
							To:         ret,
							Likeliness: b.Likeliness,
						}
					}
				}
			}
		}

		/* rebuild & cleanup the graph if needed */
		if more {
			cfg.Rebuild()
			new(BlockMerge).Apply(cfg)
		}
	}
}
