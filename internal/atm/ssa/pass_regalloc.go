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
    `math`
    `sort`
    `strings`

    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/rt`
    `gonum.org/v1/gonum/graph`
    `gonum.org/v1/gonum/graph/coloring`
    `gonum.org/v1/gonum/graph/simple`
    `gonum.org/v1/gonum/graph/topo`
)

const (
    _P_term = math.MaxUint32
)

type _Pos struct {
    i int
    b int
}

func pos(bb *BasicBlock, i int) _Pos {
    return _Pos {
        i: i,
        b: bb.Id,
    }
}

func (self _Pos) String() string {
    if self.i == _P_term {
        return fmt.Sprintf("bb_%d.term", self.b)
    } else {
        return fmt.Sprintf("bb_%d.ins[%d]", self.b, self.i)
    }
}

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

func (self _RegSet) hasoverlap(rs _RegSet) bool {
    p, q := self, rs
    if len(q) < len(p) { p, q = q, p }
    for r := range p { if q.contains(r) { return true } }
    return false
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
    return &_RegTab {
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
    for rr, rs := range self.m { r = append(r, _RegTabPair { rr, rs }) }
    sort.Slice(r, func(i int, j int) bool { return r[i].rr < r[j].rr })
    return
}

func (self *_RegTab) alloc(n int) (rs _RegSet) {
    if p := len(self.p); p == 0 {
        rs = make(_RegSet, n)
        return
    } else {
        rs, self.p = self.p[p - 1], self.p[:p - 1]
        return
    }
}

func (self *_RegTab) clone(s _RegSet) (rs _RegSet) {
    rs = self.alloc(len(s))
    for r := range s { rs.add(r) }
    return
}

func (self *_RegTab) mkset(r ...Reg) (rs _RegSet) {
    rs = self.alloc(len(r))
    for _, v := range r { rs.add(v) }
    return
}

func (self *_RegTab) mksetp(r []*Reg) (rs _RegSet) {
    rs = self.alloc(len(r))
    for _, v := range r { rs.add(*v) }
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

func (self RegAlloc) livein(p *_RegTab, lr map[_Pos]_RegSet, bb *BasicBlock, in map[int]_RegSet, out map[int]_RegSet) _RegSet {
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
        if def, ok = bb.Ins[i].(IrDefinitions) ; ok { regs.subtract(p.mksetp(def.Definitions())) }
        if use, ok = bb.Ins[i].(IrUsages)      ; ok { regs.union(p.mksetp(use.Usages())) }
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

func (self RegAlloc) liveout(p *_RegTab, lr map[_Pos]_RegSet, bb *BasicBlock, in map[int]_RegSet, out map[int]_RegSet) _RegSet {
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

func (self RegAlloc) Apply(cfg *CFG) {
    var k int
    var colors map[int64]int

    /* reusable state */
    pool := mkregtab()
    arch := make([]Reg, 0, len(ArchRegs))
    idmap := make(map[int]Reg)
    slots := make(map[Reg]int)
    defuse := make(map[Reg]float64)
    livein := make(map[int]_RegSet)
    liveout := make(map[int]_RegSet)
    liveset := make(map[_Pos]_RegSet)
    spillset := make(map[Reg]bool)
    nospillset := make(map[Reg]bool)
    coalescemap := make(map[Reg]Reg)

    /* stack slot allocator */
    getslot := func(r Reg) uintptr {
        var i int
        var ok bool

        /* check if it exists, if not, allocate a new one */
        if i, ok = slots[r]; !ok {
            i = len(slots)
            slots[r] = i
        }

        /* convert to slot offset */
        i *= abi.PtrSize
        return uintptr(i)
    }

    /* register precolorer */
    precolor := func(rr []*Reg) {
        for _, r := range rr {
            if r.Kind() == K_arch {
                *r = IrSetArch(Rz, ArchRegs[r.Name()])
            }
        }
    }

    /* register def-use counter */
    countreg := func(rr []*Reg) {
        for _, r := range rr {
            if spillset[*r] && !nospillset[*r] {
                defuse[*r]++
            }
        }
    }

    /* register coalescer */
    coalesce := func(rr []*Reg) {
        for _, p := range rr {
            if r, ok := coalescemap[*p]; ok {
                *p = r
            }
        }
    }

    /* calculate allocatable registers */
    for _, r := range ArchRegs {
        if !ArchRegReserved[r] {
            arch = append(arch, IrSetArch(Rz, r))
        }
    }

    /* precolor all arch-specific registers */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        ok := false
        use := IrUsages(nil)
        def := IrDefinitions(nil)

        /* scan all the instructions */
        for _, v := range bb.Ins {
            if use, ok = v.(IrUsages)      ; ok { precolor(use.Usages()) }
            if def, ok = v.(IrDefinitions) ; ok { precolor(def.Definitions()) }
        }

        /* scan the terminator */
        if use, ok = bb.Term.(IrUsages); ok {
            precolor(use.Usages())
        }
    })

    /* loop until no more retries */
    for {
        pool.reset()
        rt.MapClear(defuse)
        rt.MapClear(livein)
        rt.MapClear(liveout)
        rt.MapClear(liveset)
        rt.MapClear(spillset)
        rt.MapClear(coalescemap)

        /* Phase 1: Calculate live ranges */
        lr := self.livein(pool, liveset, cfg.Root, livein, liveout)
        rig := simple.NewUndirectedGraph()

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
            for i := 0; i < nr - 1; i++ {
                for j := i + 1; j < nr; j++ {
                    p, _ := rig.NodeWithID(int64(rr[i]))
                    q, _ := rig.NodeWithID(int64(rr[j]))
                    rig.SetEdge(rig.NewEdge(p, q))
                }
            }
        }

        /* force every arch-specific register interfering with each other */
        for i := 0; i < len(arch); i++ {
            for j := i + 1; j < len(arch); j++ {
                p, _ := rig.NodeWithID(int64(arch[i]))
                q, _ := rig.NodeWithID(int64(arch[j]))
                rig.SetEdge(rig.NewEdge(p, q))
            }
        }

        /* Phase 3: Coalescing registers */
        next := false
        cols := simple.NewUndirectedGraph()

        /* find out all the coalescing pairs */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            for _, v := range bb.Ins {
                var rx Reg
                var ry Reg
                var ok bool

                /* only look for copy instructions */
                if rx, ry, ok = IrArchTryIntoCopy(v); !ok || rx == ry {
                    continue
                }

                /* make sure Y is the node with a lower degree */
                if rig.From(int64(rx)).Len() < rig.From(int64(ry)).Len() {
                    rx, ry = ry, rx
                }

                /* determain whether it's safe to coalesce using George's heuristic */
                for p := rig.From(int64(ry)); p.Next(); {
                    if t := p.Node().ID(); t == int64(rx) || len(arch) <= rig.From(t).Len() && !rig.HasEdgeBetween(t, int64(rx)) {
                        ok = false
                        break
                    }
                }

                /* check if it can be coalesced */
                if !ok {
                    continue
                }

                /* check for pre-colored registers */
                if rx.Kind() == K_arch && ry.Kind() == K_arch {
                    panic(fmt.Sprintf("regalloc: arch-specific register confliction: %s and %s", rx, ry))
                }

                /* add to colaescing graph */
                p, _ := cols.NodeWithID(int64(rx))
                q, _ := cols.NodeWithID(int64(ry))
                cols.SetEdge(cols.NewEdge(p, q))
            }
        })

        /* calculate the substitution set */
        for _, cc := range topo.ConnectedComponents(cols) {
            var r0 Reg
            var rx []graph.Node

            /* sort by register, arch-specific register always goes first */
            sort.Slice(cc, func(i int, j int) bool {
                x, y := Reg(cc[i].ID()), Reg(cc[j].ID())
                return regorder(x) < regorder(y)
            })

            /* extract the target register */
            rx = cc[1:]
            r0 = Reg(cc[0].ID())

            /* substitute to the first register */
            for _, node := range rx {
                next = true
                coalescemap[Reg(node.ID())] = r0
            }
        }

        /* coalesce if needed */
        if next {
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
                    if use, ok = v.(IrUsages)      ; ok { coalesce(use.Usages()) }
                    if def, ok = v.(IrDefinitions) ; ok { coalesce(def.Definitions()) }
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
        if next {
            continue
        }

        /* Phase 4: Check if the RIG is len(arch)-colorable, and color the graph if possible */
        if k, colors = coloring.RecursiveLargestFirst(rig); k <= len(arch) {
            break
        }

        /* calculate the spilling candidates */
        for _, rr := range coloring.Sets(colors) {
            var ok bool
            var ri int64

            /* check if it contains an arch-specific register */
            for _, ri = range rr {
                if Reg(ri).Kind() == K_arch {
                    ok = true
                    break
                }
            }

            /* if not, add all the registers to spill candidates */
            if !ok {
                for _, ri = range rr {
                    spillset[Reg(ri)] = true
                }
            }
        }

        /* Phase 5: Evaluate spilling cost, and spill the cheapest register */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            var ok bool
            var use IrUsages
            var def IrDefinitions

            /* should not contain any Phi nides */
            if len(bb.Phi) != 0 {
                panic("regalloc: unexpected Phi node")
            }

            /* scan every instruction */
            for _, v := range bb.Ins {
                if use, ok = v.(IrUsages)      ; ok { countreg(use.Usages()) }
                if def, ok = v.(IrDefinitions) ; ok { countreg(def.Definitions()) }
            }

            /* scan the terminator */
            if use, ok = bb.Term.(IrUsages); ok {
                countreg(use.Usages())
            }
        })

        /* sanity check */
        if len(defuse) == 0 {
            panic("regalloc: invalid spilling configuration")
        }

        /* calculate the cost-to-degree ratio for every register */
        for r := range defuse {
            defuse[r] /= float64(rig.From(int64(r)).Len())
        }

        /* the reference and the cost of the register to spill */
        spillr := Rz
        spillc := math.MaxFloat64

        /* find the register with lowest cost-to-degree ratio */
        for r, c := range defuse {
            if spillc > c {
                spillc = c
                spillr = r
            }
        }
        println("spill", spillr.String(), "k", k)

        /* insert spill and reload instructions for the spilled register */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            ins := bb.Ins
            bb.Ins = make([]IrNode, 0, len(ins))

            /* scan every instruction */
            for _, v := range ins {
                sl := getslot(spillr)
                use, ok := v.(IrUsages)

                /* insert reload if needed */
                if ok {
                    if rr := regslicefindall(use.Usages(), spillr); len(rr) != 0 {
                        rx := cfg.CreateRegister(spillr.Ptr())
                        bb.Ins = append(bb.Ins, IrArchLoadStack(rx, sl, IrSlotLocal))
                        regslicereplaceall(rr, rx)
                        nospillset[rx] = true
                    }
                }

                /* add the original instruction */
                bb.Ins = append(bb.Ins, v)
                def, ok := v.(IrDefinitions)

                /* insert spill if needed */
                if ok {
                    if rr := regslicefindall(def.Definitions(), spillr); len(rr) != 0 {
                        rx := cfg.CreateRegister(spillr.Ptr())
                        bb.Ins = append(bb.Ins, IrArchStoreStack(rx, sl, IrSlotLocal))
                        regslicereplaceall(rr, rx)
                        nospillset[rx] = true
                    }
                }
            }

            /* scan the terminator */
            if p, ok := bb.Term.(IrUsages); ok {
                if rr := regslicefindall(p.Usages(), spillr); len(rr) != 0 {
                    rx := cfg.CreateRegister(spillr.Ptr())
                    bb.Ins = append(bb.Ins, IrArchLoadStack(rx, getslot(spillr), IrSlotLocal))
                    regslicereplaceall(rr, rx)
                    nospillset[rx] = true
                }
            }
        })
    }

    /* sanity check */
    if k > len(arch) {
        panic(fmt.Sprintf("regalloc: more colors than available: %d > %d", k, len(arch)))
    }
    return

    /* assign IDs to precolored registers */
    for ri, c := range colors {
        if r := Reg(ri); r.Kind() == K_arch {
            if _, ok := idmap[c]; ok {
                panic("regalloc: confliction of arch-specific register: " + r.String())
            } else {
                arch, idmap[c] = regsliceremove(arch, r), r
            }
        }
    }

    /* assign remaining registers */
    for ri, c := range colors {
        if r := Reg(ri); r.Kind() != K_arch {
            if _, ok := idmap[c]; !ok {
                arch, idmap[c] = arch[1:], arch[0]
            }
        }
    }

    /* register remapping routine */
    remapregs := func(rr []*Reg) {
        for _, r := range rr {
            if r.Kind() != K_zero {
                if id, ok := colors[int64(*r)]; !ok {
                    panic("regalloc: uncolored register: " + r.String())
                } else if rv, ok := idmap[id]; !ok {
                    panic("regalloc: unmapped register: " + r.String())
                } else {
                    *r = rv
                }
            }
        }
    }

    /* Phase 6: Remap all the registers */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        var ok bool
        var ia *IrAlias
        var use IrUsages
        var def IrDefinitions

        /* should not contain any Phi nodes */
        if len(bb.Phi) != 0 {
            panic(fmt.Sprintf("regalloc: unexpecte Phi node in bb_%d", bb.Id))
        }

        /* replace every instruction */
        for _, v := range bb.Ins {
            if use, ok = v.(IrUsages)     ; ok { remapregs(use.Usages()) }
            if def, ok = v.(IrDefinitions); ok { remapregs(def.Definitions()) }
        }

        /* alias node should not exists after regalloc */
        for i, v := range bb.Ins {
            if ia, ok = v.(*IrAlias); ok {
                bb.Ins[i] = IrArchCopy(ia.R, ia.V)
            }
        }

        /* remap the terminator */
        if use, ok = bb.Term.(IrUsages); ok {
            remapregs(use.Usages())
        }
    })
}
