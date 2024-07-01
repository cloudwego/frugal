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
	"sort"

	"github.com/cloudwego/frugal/internal/rt"
)

type _ValueId struct {
	i int
	v IrNode
	r bool
}

func mkvid(i int, v IrNode) *_ValueId {
	return &_ValueId{
		i: i,
		v: v,
		r: true,
	}
}

type _BlockRef struct {
	bb *BasicBlock
}

func (self *_BlockRef) update(cfg *CFG, bb *BasicBlock) {
	u := bb
	v := self.bb

	/* move them to the same depth */
	for cfg.Depth[u.Id] != cfg.Depth[v.Id] {
		if cfg.Depth[u.Id] > cfg.Depth[v.Id] {
			u = cfg.DominatedBy[u.Id]
		} else {
			v = cfg.DominatedBy[v.Id]
		}
	}

	/* move both nodes until they meet */
	for u != v {
		u = cfg.DominatedBy[u.Id]
		v = cfg.DominatedBy[v.Id]
	}

	/* sanity check */
	if u != nil {
		self.bb = u
	} else {
		panic("reorder: invalid CFG dominator tree")
	}
}

// Reorder moves value closer to it's usage, which reduces register pressure.
type Reorder struct{}

func (Reorder) isMovable(v IrNode) bool {
	var f bool
	var u IrUsages
	var d IrDefinitions

	/* marked as immovable */
	if _, f = v.(IrImmovable); f {
		return false
	}

	/* blacklist all instructions that uses physical registers */
	if u, f = v.(IrUsages); f {
		for _, r := range u.Usages() {
			if r.Kind() == K_arch {
				return false
			}
		}
	}

	/* blacklist all instructions that alters physical registers */
	if d, f = v.(IrDefinitions); f {
		for _, r := range d.Definitions() {
			if r.Kind() == K_arch {
				return false
			}
		}
	}

	/* no such registers, all checked ok */
	return true
}

func (self Reorder) moveInterblock(cfg *CFG) {
	defs := make(map[Reg]*_BlockRef)
	uses := make(map[Pos]*_BlockRef)
	move := make(map[*BasicBlock]int)

	/* usage update routine */
	updateUsage := func(r Reg, bb *BasicBlock) {
		if m, ok := defs[r]; ok {
			if m.bb == nil {
				m.bb = bb
			} else {
				m.update(cfg, bb)
			}
		}
	}

	/* retry until no modifications */
	for move[nil] = 0; len(move) != 0; {
		rt.MapClear(defs)
		rt.MapClear(move)
		rt.MapClear(uses)

		/* Phase 1: Find all movable value definitions */
		cfg.PostOrder().ForEach(func(bb *BasicBlock) {
			for i, v := range bb.Ins {
				var f bool
				var p *_BlockRef
				var d IrDefinitions

				/* value must be movable, and have definitions */
				if d, f = v.(IrDefinitions); !f || !self.isMovable(v) {
					continue
				}

				/* create a new value movement if needed */
				if p, f = uses[pos(bb, i)]; !f {
					p = new(_BlockRef)
					uses[pos(bb, i)] = p
				}

				/* mark all the non-definition sites */
				for _, r := range d.Definitions() {
					if r.Kind() != K_zero {
						defs[*r] = p
					}
				}
			}
		})

		/* Phase 2: Identify the earliest usage locations */
		for _, bb := range cfg.PostOrder().Reversed() {
			var ok bool
			var use IrUsages

			/* search in Phi nodes */
			for _, v := range bb.Phi {
				for b, r := range v.V {
					updateUsage(*r, b)
				}
			}

			/* search in instructions */
			for _, v := range bb.Ins {
				if use, ok = v.(IrUsages); ok {
					for _, r := range use.Usages() {
						updateUsage(*r, bb)
					}
				}
			}

			/* search the terminator */
			if use, ok = bb.Term.(IrUsages); ok {
				for _, r := range use.Usages() {
					updateUsage(*r, bb)
				}
			}
		}

		/* Phase 3: Move value definitions to their usage block */
		for p, m := range uses {
			if m.bb != nil && m.bb != p.B {
				m.bb.Ins = append(m.bb.Ins, p.B.Ins[p.I])
				move[m.bb] = move[m.bb] + 1
				p.B.Ins[p.I] = new(IrNop)
			}
		}

		/* Phase 4: Move values to place */
		for bb, i := range move {
			v := bb.Ins
			n := len(bb.Ins)
			bb.Ins = make([]IrNode, n)
			copy(bb.Ins[i:], v[:n-i])
			copy(bb.Ins[:i], v[n-i:])
		}
	}

	/* Phase 5: Remove all the placeholder NOP instructions */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		ins := bb.Ins
		bb.Ins = bb.Ins[:0]

		/* filter out the NOP instructions */
		for _, v := range ins {
			if _, ok := v.(*IrNop); !ok {
				bb.Ins = append(bb.Ins, v)
			}
		}
	})
}

func (self Reorder) moveIntrablock(cfg *CFG) {
	var rbuf []IrNode
	var mbuf []*_ValueId
	var vbuf []*_ValueId
	var addval func(*_ValueId, bool)

	/* reusable states */
	adds := make(map[int]struct{})
	defs := make(map[Reg]*_ValueId)

	/* topology sorter */
	addval = func(v *_ValueId, depsOnly bool) {
		var ok bool
		var use IrUsages
		var val *_ValueId

		/* check if it's been added */
		if _, ok = adds[v.i]; ok {
			return
		}

		/* add all the dependencies recursively */
		if use, ok = v.v.(IrUsages); ok {
			for _, r := range use.Usages() {
				if val, ok = defs[*r]; ok {
					addval(val, false)
				}
			}
		}

		/* add the instruction if needed */
		if !depsOnly {
			rbuf = append(rbuf, v.v)
			adds[v.i] = struct{}{}
		}
	}

	/* process every block */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		rbuf = rbuf[:0]
		mbuf = mbuf[:0]
		vbuf = vbuf[:0]
		rt.MapClear(adds)
		rt.MapClear(defs)

		/* number all instructions */
		for i, v := range bb.Ins {
			id := mkvid(i, v)
			vbuf = append(vbuf, id)

			/* preserve the order of immovable instructions */
			if !self.isMovable(v) {
				mbuf = append(mbuf, id)
			}
		}

		/* mark all non-Phi definitions in this block */
		for _, v := range vbuf {
			if def, ok := v.v.(IrDefinitions); ok {
				for _, r := range def.Definitions() {
					if _, ok = defs[*r]; ok {
						panic(fmt.Sprintf("reorder: multiple definitions for %s in bb_%d", r, bb.Id))
					} else {
						defs[*r] = v
					}
				}
			}
		}

		/* find all the root nodes */
		for _, v := range vbuf {
			if use, ok := v.v.(IrUsages); !ok {
				v.r = false
			} else {
				for _, r := range use.Usages() {
					if v, ok = defs[*r]; ok {
						v.r = false
					}
				}
			}
		}

		/* all the immovable instructions needs to preserve their order */
		for _, v := range mbuf {
			addval(v, false)
		}

		/* add all the root instructions */
		for _, v := range vbuf {
			if v.r {
				addval(v, false)
			}
		}

		/* add remaining instructions */
		for _, v := range vbuf {
			if _, ok := adds[v.i]; !ok {
				addval(v, false)
			}
		}

		/* add the terminator */
		addval(mkvid(-1, bb.Term), true)
		bb.Ins = append(bb.Ins[:0], rbuf...)
	})
}

func (Reorder) moveArgumentLoad(cfg *CFG) {
	var ok bool
	var ir []IrNode
	var vv *IrLoadArg

	/* extract all the argument loads */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		ins := bb.Ins
		bb.Ins = bb.Ins[:0]

		/* scan instructions */
		for _, v := range ins {
			if vv, ok = v.(*IrLoadArg); ok {
				ir = append(ir, vv)
			} else {
				bb.Ins = append(bb.Ins, v)
			}
		}
	})

	/* sort by argument ID */
	sort.Slice(ir, func(i int, j int) bool {
		return ir[i].(*IrLoadArg).I < ir[j].(*IrLoadArg).I
	})

	/* prepend to the root node */
	ins := cfg.Root.Ins
	cfg.Root.Ins = append(ir, ins...)
}

func (self Reorder) Apply(cfg *CFG) {
	self.moveInterblock(cfg)
	self.moveIntrablock(cfg)
	self.moveArgumentLoad(cfg)
}
