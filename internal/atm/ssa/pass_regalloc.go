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
	"strings"
	"unsafe"

	"github.com/cloudwego/frugal/internal/rt"
	"gonum.org/v1/gonum/graph/coloring"
	"gonum.org/v1/gonum/graph/simple"
)

type (
	_RegSet map[Reg]struct{}
)

func (self _RegSet) add(r Reg) _RegSet {
	self[r] = struct{}{}
	return self
}

func (self _RegSet) union(rs _RegSet) {
	for r := range rs {
		self.add(r)
	}
}

func (self _RegSet) remove(r Reg) {
	delete(self, r)
}

func (self _RegSet) subtract(rs _RegSet) {
	for r := range rs {
		self.remove(r)
	}
}

func (self _RegSet) toslice() []Reg {
	nb := len(self)
	rr := make([]Reg, 0, nb)

	/* extract all registers */
	for r := range self {
		rr = append(rr, r)
	}

	/* sort by register ID */
	sort.Slice(rr, func(i int, j int) bool { return rr[i] < rr[j] })
	return rr
}

func (self _RegSet) contains(r Reg) (ret bool) {
	_, ret = self[r]
	return
}

func (self _RegSet) String() string {
	nb := len(self)
	rs := make([]string, 0, nb)

	/* convert every register */
	for _, r := range self.toslice() {
		rs = append(rs, r.String())
	}

	/* join them together */
	return fmt.Sprintf(
		"{%s}",
		strings.Join(rs, ", "),
	)
}

type _RegTabPair struct {
	rr Reg
	rs _RegSet
}

type _RegTab struct {
	p []_RegSet
	m map[Reg]_RegSet
}

func mkregtab() *_RegTab {
	return &_RegTab{
		p: make([]_RegSet, 0, 16),
		m: make(map[Reg]_RegSet, len(ArchRegs)),
	}
}

func (self *_RegTab) reset() {
	for k, v := range self.m {
		self.p = append(self.p, v)
		delete(self.m, k)
		rt.MapClear(v)
	}
}

func (self *_RegTab) pairs() (r []_RegTabPair) {
	r = make([]_RegTabPair, 0, len(self.m))
	for rr, rs := range self.m {
		r = append(r, _RegTabPair{rr, rs})
	}
	sort.Slice(r, func(i int, j int) bool { return r[i].rr < r[j].rr })
	return
}

func (self *_RegTab) alloc(n int) (rs _RegSet) {
	if p := len(self.p); p == 0 {
		rs = make(_RegSet, n)
		return
	} else {
		rs, self.p = self.p[p-1], self.p[:p-1]
		return
	}
}

func (self *_RegTab) clone(s _RegSet) (rs _RegSet) {
	rs = self.alloc(len(s))
	for r := range s {
		rs.add(r)
	}
	return
}

func (self *_RegTab) mkset(r ...Reg) (rs _RegSet) {
	rs = self.alloc(len(r))
	for _, v := range r {
		rs.add(v)
	}
	return
}

func (self *_RegTab) mksetp(r []*Reg) (rs _RegSet) {
	rs = self.alloc(len(r))
	for _, v := range r {
		rs.add(*v)
	}
	return
}

func (self *_RegTab) relate(k Reg, v Reg) {
	if p, ok := self.m[k]; ok {
		p.add(v)
	} else {
		self.m[k] = self.alloc(1).add(v)
	}
}

func (self *_RegTab) remove(r Reg) (rs _RegSet) {
	rs = self.m[r]
	delete(self.m, r)
	return
}

// RegAlloc performs register allocation on CFG.
type RegAlloc struct{}

func (self RegAlloc) livein(p *_RegTab, lr map[Pos]_RegSet, bb *BasicBlock, in map[int]_RegSet, out map[int]_RegSet) _RegSet {
	var ok bool
	var rs _RegSet
	var use IrUsages
	var def IrDefinitions

	/* check for cached live-in sets */
	if rs, ok = in[bb.Id]; ok {
		return p.clone(rs)
	}

	/* calculate the live-out set of current block */
	tr := bb.Term
	regs := p.clone(self.liveout(p, lr, bb, in, out))

	/* assume all terminators are non-definitive */
	if _, ok = tr.(IrDefinitions); ok {
		panic("regalloc: definitions within terminators")
	}

	/* add the terminator usages if any */
	if use, ok = tr.(IrUsages); ok {
		regs.union(p.mksetp(use.Usages()))
	}

	/* mark live range of the terminator */
	rr := p.clone(regs)
	lr[pos(bb, _P_term)] = rr

	/* live(i-1) = use(i) ∪ (live(i) - { def(i) }) */
	for i := len(bb.Ins) - 1; i >= 0; i-- {
		if def, ok = bb.Ins[i].(IrDefinitions); ok {
			regs.subtract(p.mksetp(def.Definitions()))
		}
		if use, ok = bb.Ins[i].(IrUsages); ok {
			regs.union(p.mksetp(use.Usages()))
		}
		lr[pos(bb, i)] = p.clone(regs)
	}

	/* should not have any Phi nodes */
	if len(bb.Phi) != 0 {
		panic("regalloc: unexpected Phi nodes")
	}

	/* update the cache */
	in[bb.Id] = p.clone(regs)
	return regs
}

func (self RegAlloc) liveout(p *_RegTab, lr map[Pos]_RegSet, bb *BasicBlock, in map[int]_RegSet, out map[int]_RegSet) _RegSet {
	var ok bool
	var rr []Reg
	var rs _RegSet

	/* check for cached live-out sets */
	if rs, ok = out[bb.Id]; ok {
		return rs
	}

	/* check for return blocks */
	if rr, ok = IrTryIntoArchReturn(bb.Term); ok {
		rs = p.mkset(rr...)
		out[bb.Id] = rs
		return rs
	}

	/* create a new register set */
	rs = p.alloc(0)
	it := bb.Term.Successors()

	/* live-out(p) = ∑(live-in(succ(p))) */
	for out[bb.Id] = nil; it.Next(); {
		rs.union(self.livein(p, lr, it.Block(), in, out))
	}

	/* update cache */
	out[bb.Id] = rs
	return rs
}

/* try to choose a different color from reloadRegs */
func (self RegAlloc) colorDiffWithReload(rig *simple.UndirectedGraph, reg Reg, reloadReg map[Reg]int, arch []Reg, colormap map[Reg]int) {
	sameWithReload := false
	for _, c := range reloadReg {
		if c == colormap[reg] {
			sameWithReload = true
			break
		}
	}
	if !sameWithReload {
		return
	}

	/* all possible colors */
	colors := make(map[int]struct{})
	for i := range arch {
		colors[i] = struct{}{}
	}

	/* choose a different color from it's neightbors */
	for r := rig.From(int64(reg)); r.Next(); {
		delete(colors, colormap[Reg(r.Node().ID())])
	}

	/* choose a different color from reloadRegs */
	for _, v := range reloadReg {
		delete(colors, v)
	}

	if len(colors) > 0 {
		/* there have some other colors different with reload regs */
		for c := range colors {
			colormap[reg] = c
			break
		}
	}
}

/* try to choose the same color for reloadReg as reg */
func (self RegAlloc) colorSameWithReg(rig *simple.UndirectedGraph, reloadReg Reg, arch []Reg, colormap map[Reg]int, reg Reg) {
	/* all possible colors */
	colors := make(map[int]struct{})
	for i := range arch {
		colors[i] = struct{}{}
	}

	/* choose a different color from it's neightbors */
	for r := rig.From(int64(reloadReg)); r.Next(); {
		delete(colors, colormap[Reg(r.Node().ID())])
	}

	/* try to choose the same color with reg */
	if _, ok := colors[colormap[reg]]; ok {
		colormap[reloadReg] = colormap[reg]
	}
}

func (self RegAlloc) Apply(cfg *CFG) {
	var rig *simple.UndirectedGraph
	var pass int
	var arch []Reg
	var colormap map[Reg]int

	/* reusable state */
	pool := mkregtab()
	regmap := make(map[int]Reg)
	livein := make(map[int]_RegSet)
	liveout := make(map[int]_RegSet)
	liveset := make(map[Pos]_RegSet)
	archcolors := make(map[int64]int, len(ArchRegs))
	coalescemap := make(map[Reg]Reg)
	invcoalescemap := make(map[Reg][]Reg)

	/* register coalescer */
	coalesce := func(rr []*Reg) {
		for _, r := range rr {
			if c, ok := coalescemap[*r]; ok {
				*r = c
			}
		}
	}

	/* calculate allocatable registers */
	for _, r := range ArchRegs {
		if !ArchRegReserved[r] {
			arch = append(arch, IrSetArch(Rz, r))
		}
	}

	/* allocate colors to the registers */
	for i, r := range arch {
		archcolors[int64(r)] = i
	}

	/* register precolorer */
	precolor := func(rr []*Reg) {
		for _, r := range rr {
			if r.Kind() == K_arch {
				*r = IrSetArch(Rz, ArchRegs[r.Name()])
			}
		}
	}

	/* precolor all physical registers */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		ok := false
		use := IrUsages(nil)
		def := IrDefinitions(nil)

		/* scan all the instructions */
		for _, v := range bb.Ins {
			if use, ok = v.(IrUsages); ok {
				precolor(use.Usages())
			}
			if def, ok = v.(IrDefinitions); ok {
				precolor(def.Definitions())
			}
		}

		/* scan the terminator */
		if use, ok = bb.Term.(IrUsages); ok {
			precolor(use.Usages())
		}
	})

	/* loop until no more retries */
	for {
		pool.reset()
		rt.MapClear(livein)
		rt.MapClear(liveout)
		rt.MapClear(liveset)
		rt.MapClear(coalescemap)
		rt.MapClear(invcoalescemap)

		/* expand operands but not in the first pass */
		if pass != 0 {
			new(OperandAlloc).Apply(cfg)
		}

		/* Phase 1: Calculate live ranges */
		lr := self.livein(pool, liveset, cfg.Root, livein, liveout)
		rig = simple.NewUndirectedGraph()

		/* sanity check: no registers live at the entry point */
		if len(lr) != 0 {
			panic("regalloc: live registers at entry: " + lr.String())
		}

		/* Phase 2: Build register interference graph */
		for _, rs := range liveset {
			rr := rs.toslice()
			nr := len(rr)

			/* special case of a single live register */
			if nr == 1 && rig.Node(int64(rr[0])) == nil {
				p, _ := rig.NodeWithID(int64(rr[0]))
				rig.AddNode(p)
				continue
			}

			/* create every edge */
			for i := 0; i < nr-1; i++ {
				for j := i + 1; j < nr; j++ {
					p, _ := rig.NodeWithID(int64(rr[i]))
					q, _ := rig.NodeWithID(int64(rr[j]))
					rig.SetEdge(rig.NewEdge(p, q))
				}
			}
		}

		/* make sure all physical registers are in the RIG */
		for _, r := range arch {
			if p, ok := rig.NodeWithID(int64(r)); ok {
				rig.AddNode(p)
			}
		}

		/* Phase 3: Coalescing registers, find out all the coalescing pairs */
		cfg.PostOrder().ForEach(func(bb *BasicBlock) {
			for _, v := range bb.Ins {
				var r0 Reg
				var r1 Reg
				var rx Reg
				var ry Reg
				var ok bool

				/* only interested in copy instructions */
				if rx, ry, ok = IrArchTryIntoCopy(v); !ok {
					continue
				}

				/* locate the coalescing source register */
				if r0, ok = coalescemap[rx]; ok {
					rx = r0
				}
				if r1, ok = coalescemap[ry]; ok {
					ry = r1
				}

				/* check if the two registers can be coalesced */
				if rx == ry || rig.HasEdgeBetween(int64(rx), int64(ry)) || (rx.Kind() == K_arch && ry.Kind() == K_arch) {
					continue
				}

				/* make sure Y is the node with a lower degree */
				if rig.From(int64(rx)).Len() < rig.From(int64(ry)).Len() {
					rx, ry = ry, rx
				}

				/* all the adjacent nodes of Y */
				p := rig.From(int64(ry))
				ok = true

				/* determain whether it's safe to coalesce using George's heuristic */
				for p.Next() {
					if t := p.Node().ID(); len(arch) <= rig.From(t).Len() && !rig.HasEdgeBetween(t, int64(rx)) {
						ok = false
						break
					}
				}

				/* check if it can be coalesced */
				if !ok {
					continue
				}

				/* r0 is the target register, r1 will be coalesced */
				if regorder(rx) < regorder(ry) {
					r0 = rx
					r1 = ry
				} else {
					r0 = ry
					r1 = rx
				}

				/* invcoalescemap's key is the target reg (named r0), it's value is the reg list coalesced to r0 */
				for _, r := range invcoalescemap[r1] {
					coalescemap[r] = r0
				}

				/* also mark the inverse relationship */
				invcoalescemap[r0] = append(invcoalescemap[r0], r1)
				invcoalescemap[r0] = append(invcoalescemap[r0], invcoalescemap[r1]...)

				/* coalesce r0 and r1 (replace r1 by r0) in the interference graph */
				for p = rig.From(int64(r1)); p.Next(); {
					if t := p.Node().ID(); !rig.HasEdgeBetween(t, int64(r0)) {
						nt, _ := rig.NodeWithID(t)
						n0, _ := rig.NodeWithID(int64(r0))
						rig.SetEdge(rig.NewEdge(n0, nt))
					}
				}

				/* mark as coalesced */
				coalescemap[r1] = r0
				rig.RemoveNode(int64(r1))
			}
		})

		/* coalesce if needed */
		if len(coalescemap) != 0 {
			cfg.PostOrder().ForEach(func(bb *BasicBlock) {
				var ok bool
				var use IrUsages
				var def IrDefinitions

				/* should not have Phi nodes */
				if len(bb.Phi) != 0 {
					panic("regalloc: unexpected Phi nodes")
				}

				/* scan instructions */
				for _, v := range bb.Ins {
					if use, ok = v.(IrUsages); ok {
						coalesce(use.Usages())
					}
					if def, ok = v.(IrDefinitions); ok {
						coalesce(def.Definitions())
					}
				}

				/* scan terminator */
				if use, ok = bb.Term.(IrUsages); ok {
					coalesce(use.Usages())
				}
			})
		}

		/* remove copies to itself */
		cfg.PostOrder().ForEach(func(bb *BasicBlock) {
			ins := bb.Ins
			bb.Ins = bb.Ins[:0]

			/* filter the instructions */
			for _, p := range ins {
				if rd, rs, ok := IrArchTryIntoCopy(p); !ok || rd != rs {
					bb.Ins = append(bb.Ins, p)
				}
			}
		})

		/* try again if coalesce occured */
		if len(coalescemap) != 0 {
			continue
		}

		/* Phase 4: Attempt to color the RIG */
		k, m, _ := coloring.WelshPowell(rig, archcolors)
		colormap = *(*map[Reg]int)(unsafe.Pointer(&m))

		/* check for len(arch)-colorablity */
		if k <= len(arch) {
			break
		}

		/* the second pass should always be colorable */
		if pass++; pass >= 2 {
			panic("regalloc: this CFG may not colorable")
		}

		/* calculate the color sets */
		colors := coloring.Sets(m)
		colorset := *(*map[int][]Reg)(unsafe.Pointer(&colors))

		/* Phase 4: Spill excess registers to stack */
		for i, r := range arch {
			rr := Rz
			ok := false
			rs := colorset[i]

			/* remove from color map */
			for _, rr = range rs {
				if colormap[rr] != i {
					panic("regalloc: color mismatch for register " + rr.String())
				} else if delete(colormap, rr); rr == r {
					ok = true
				}
			}

			/* remove from color set */
			if delete(colorset, i); !ok {
				panic("regalloc: invalid coloring: missing register " + r.String())
			}
		}

		/* spill those without a physical register */
		cfg.PostOrder().ForEach(func(bb *BasicBlock) {
			var cc int
			var rr Reg
			var ok bool
			var use IrUsages

			/* should not contain Phi nodes */
			if len(bb.Phi) != 0 {
				panic("regalloc: unexpected Phi node")
			}

			/* allocate buffer for new instructions */
			ins := bb.Ins
			bb.Ins = make([]IrNode, 0, len(ins))

			/* scan every instructions */
			for _, v := range ins {
				var s Reg
				var t Reg
				var r *Reg
				var d IrDefinitions

				/* clear the register map */
				for c := range regmap {
					delete(regmap, c)
				}

				/* reload as needed */
				if use, ok = v.(IrUsages); ok {
					for _, r = range use.Usages() {
						if cc, ok = colormap[*r]; ok {
							if rr, ok = regmap[cc]; ok {
								*r = rr
							} else {
								*r = cfg.CreateRegister(r.Ptr())
								bb.Ins = append(bb.Ins, IrCreateSpill(*r, cc, IrSpillReload))
								regmap[cc] = *r
							}
						}
					}
				}

				/* add the instruction itself */
				i := len(bb.Ins)
				bb.Ins = append(bb.Ins, v)

				/* no definitions */
				if d, ok = v.(IrDefinitions); !ok {
					continue
				}

				/* spill as needed */
				for _, r = range d.Definitions() {
					if cc, ok = colormap[*r]; ok {
						if t, s, ok = IrArchTryIntoCopy(v); !ok {
							bb.Ins = append(bb.Ins, IrCreateSpill(*r, cc, IrSpillStore))
						} else {
							bb.Ins[i] = IrCreateSpillEx(s, t.Ptr(), cc, IrSpillStore)
						}
					}
				}
			}

			/* clear the register map for terminator */
			for c := range regmap {
				delete(regmap, c)
			}

			/* scan for reloads in the terminator */
			if use, ok = bb.Term.(IrUsages); ok {
				for _, r := range use.Usages() {
					if cc, ok = colormap[*r]; ok {
						if rr, ok = regmap[cc]; ok {
							*r = rr
						} else {
							*r = cfg.CreateRegister(r.Ptr())
							bb.Ins = append(bb.Ins, IrCreateSpill(*r, cc, IrSpillReload))
							regmap[cc] = *r
						}
					}
				}
			}
		})
	}

	/* finetune color allocation plan */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		reloadRegs := make(map[Reg]int)
		spillSlots := make(map[int]Reg)
		slotReg := make(map[int]Reg)

		/* process instructions */
		for _, v := range bb.Ins {
			if spillIr, ok := v.(*IrSpill); ok && spillIr.Op == IrSpillReload {
				if _, ok = reloadRegs[spillIr.R]; !ok {
					/* try to assign the same color to reloadReg and spillReg with the same slot */
					if r, ok := spillSlots[spillIr.S.ID()]; ok {
						self.colorSameWithReg(rig, spillIr.R, arch, colormap, r)
					} else if r, ok := slotReg[spillIr.S.ID()]; ok {
						/* try to assign the same color to reloadRegs with the same slot */
						self.colorSameWithReg(rig, spillIr.R, arch, colormap, r)
					} else {
						/* try to assign different color to reloadRegs with different slots */
						self.colorDiffWithReload(rig, spillIr.R, reloadRegs, arch, colormap)
					}
					slotReg[spillIr.S.ID()] = spillIr.R
					reloadRegs[spillIr.R] = colormap[spillIr.R]
				}
			} else if ok && spillIr.Op == IrSpillStore {
				spillSlots[spillIr.S.ID()] = spillIr.R
			} else {
				/* try to choose color different from reload regs for defined regs */
				if def, ok := v.(IrDefinitions); ok {
					for _, r := range def.Definitions() {
						self.colorDiffWithReload(rig, *r, reloadRegs, arch, colormap)
					}
				}
			}
		}
	})

	/* register substitution routine */
	replaceregs := func(rr []*Reg) {
		for _, r := range rr {
			if c, ok := colormap[*r]; ok {
				*r = arch[c]
			}
		}
	}

	/* Phase 5: Replace all the virtual registers with physical registers */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var ok bool
		var use IrUsages
		var def IrDefinitions

		/* should not contain Phi nodes */
		if len(bb.Phi) != 0 {
			panic("regalloc: unexpected Phi node")
		}

		/* replace instructions */
		for _, v := range bb.Ins {
			if use, ok = v.(IrUsages); ok {
				replaceregs(use.Usages())
			}
			if def, ok = v.(IrDefinitions); ok {
				replaceregs(def.Definitions())
			}
		}

		/* replace the terminator */
		if use, ok = bb.Term.(IrUsages); ok {
			replaceregs(use.Usages())
		}
	})

	/* remove copies to itself */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		ins := bb.Ins
		bb.Ins = bb.Ins[:0]

		/* filter the instructions */
		for _, p := range ins {
			if rd, rs, ok := IrArchTryIntoCopy(p); !ok || rd != rs {
				bb.Ins = append(bb.Ins, p)
			}
		}
	})

	/* remove redundant spill and reload instructions where both the register and stack aren't modified */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		/* register to stack slot is a one to one mapping */
		regSlot := make(map[Reg]int)
		slotReg := make(map[int]Reg)
		ins := bb.Ins
		bb.Ins = nil
		def := IrDefinitions(nil)

		/* scan every instruction */
		for _, v := range ins {
			if spillIr, ok := v.(*IrSpill); ok {
				/* if there has been a one-one mapping between the register and stack slot, abandon this spill/reload instruction */
				if s, ok := regSlot[spillIr.R]; ok && s == spillIr.S.ID() {
					if r, ok := slotReg[spillIr.S.ID()]; ok && r == spillIr.R {
						continue
					}
				}

				/* delete old mapping relations */
				delete(regSlot, slotReg[spillIr.S.ID()])
				delete(slotReg, regSlot[spillIr.R])

				/* establish new mapping relations */
				slotReg[spillIr.S.ID()] = spillIr.R
				regSlot[spillIr.R] = spillIr.S.ID()
				bb.Ins = append(bb.Ins, v)
			} else {
				if def, ok = v.(IrDefinitions); ok {
					for _, r := range def.Definitions() {
						/* delete old mapping relations */
						delete(slotReg, regSlot[*r])
						delete(regSlot, *r)
					}
				}
				bb.Ins = append(bb.Ins, v)
			}
		}
	})

	/* remove redundant spill where the same stack slot is overwritten before loading */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var redundantIrPos []int
		storeStack := make(map[int][]int)

		/* scan every instruction */
		for i, v := range bb.Ins {
			if spillIr, ok := v.(*IrSpill); ok && spillIr.Op == IrSpillStore {
				storeStack[spillIr.S.ID()] = append(storeStack[spillIr.S.ID()], i)
			} else if ok && spillIr.Op == IrSpillReload {
				if irPos, ok := storeStack[spillIr.S.ID()]; ok {
					redundantIrPos = append(redundantIrPos, irPos[0:len(irPos)-1]...)
					delete(storeStack, spillIr.S.ID())
				}
			}
		}

		for _, irPos := range storeStack {
			if len(irPos) > 1 {
				redundantIrPos = append(redundantIrPos, irPos[0:len(irPos)-1]...)
			}
		}

		/* abandon redundant spill instructions according to their position in the block */
		if len(redundantIrPos) > 0 {
			ins := bb.Ins
			bb.Ins = nil
			sort.Ints(redundantIrPos)
			startPos := 0
			for _, p := range redundantIrPos {
				bb.Ins = append(bb.Ins, ins[startPos:p]...)
				startPos = p + 1
			}
			if startPos < len(ins) {
				bb.Ins = append(bb.Ins, ins[startPos:]...)
			}
		}
	})

	regSliceToSet := func(rr []*Reg) _RegSet {
		rs := make(_RegSet, 0)
		for _, r := range rr {
			rs.add(*r)
		}
		return rs
	}

	/* remove redundant reload where the register isn't used after being reloaded */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var ok bool
		var use IrUsages
		var def IrDefinitions
		var spillIr *IrSpill
		var removePos []int
		rs := make(_RegSet, 0)

		/* add the terminator usages if any */
		if use, ok = bb.Term.(IrUsages); ok {
			rs.union(regSliceToSet(use.Usages()))
		}

		/* live(i-1) = use(i) ∪ (live(i) - { def(i) }) */
		for i := len(bb.Ins) - 1; i >= 0; i-- {
			if def, ok = bb.Ins[i].(IrDefinitions); ok {
				if spillIr, ok = bb.Ins[i].(*IrSpill); ok && spillIr.Op == IrSpillReload {
					/* if the reloaded reg isn't used afterwards, record its position and then remove it */
					if !rs.contains(spillIr.R) {
						removePos = append(removePos, i)
						continue
					}
				}
				rs.subtract(regSliceToSet(def.Definitions()))
			}
			if use, ok = bb.Ins[i].(IrUsages); ok {
				rs.union(regSliceToSet(use.Usages()))
			}
		}

		if len(removePos) > 0 {
			ins := bb.Ins
			bb.Ins = nil
			sort.Ints(removePos)
			startPos := 0
			for _, p := range removePos {
				bb.Ins = append(bb.Ins, ins[startPos:p]...)
				startPos = p + 1
			}
			if startPos < len(ins) {
				bb.Ins = append(bb.Ins, ins[startPos:]...)
			}
		}
	})
}
