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

    `github.com/cloudwego/frugal/internal/rt`
    `github.com/oleiade/lane`
    `gonum.org/v1/gonum/graph`
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

type _RegColor struct {
    r Reg
    c int
}

type _IrSpillOp struct {
    reg   Reg
    tag   Reg
    offs  uintptr
    reload bool
}

func mkSpillOp(reg Reg, tag Reg, reload bool) *_IrSpillOp {
    return &_IrSpillOp {
        reg    : reg,
        tag    : tag,
        offs   : 0,
        reload : reload,
    }
}

func (self *_IrSpillOp) irnode()      {}
func (self *_IrSpillOp) irimpure()    {}
func (self *_IrSpillOp) irimmovable() {}

func (self *_IrSpillOp) Clone() IrNode {
    r := *self
    return &r
}

func (self *_IrSpillOp) String() string {
    if self.reload {
        return fmt.Sprintf("%s = reload %s(SP)", self.reg, self.tag)
    } else {
        return fmt.Sprintf("spill %s -> %s(SP)", self.reg, self.tag)
    }
}

func (self *_IrSpillOp) Usages() []*Reg {
    if self.reload {
        return nil
    } else {
        return []*Reg { &self.reg }
    }
}

func (self *_IrSpillOp) Definitions() []*Reg {
    if !self.reload {
        return nil
    } else {
        return []*Reg { &self.reg }
    }
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

func (self RegAlloc) depthmap(dm map[int]int, bb *BasicBlock, vis map[int]struct{}, path []int) {
    path = append(path, bb.Id)
    vis[bb.Id] = struct{}{}

    /* traverse all the sucessors */
    for it := bb.Term.Successors(); it.Next(); {
        st := -1
        nx := it.Block()

        /* find ID on the path */
        for i, v := range path {
            if v == nx.Id {
                st = i
                break
            }
        }

        /* check for loops, visit the successor if not already */
        if st != -1 {
            for _, id := range path[st:] {
                dm[id]++
            }
        } else {
            if _, ok := vis[nx.Id]; !ok {
                self.depthmap(dm, nx, vis, path)
            }
        }
    }
}

func (self RegAlloc) Apply(cfg *CFG) {
    var spill bool
    var order *lane.Stack

    /* reusable state */
    pool := mkregtab()
    arch := make([]Reg, 0, len(ArchRegs))
    edges := make(map[Reg][]Reg)
    defuse := make(map[Reg]float64)
    livein := make(map[int]_RegSet)
    liveout := make(map[int]_RegSet)
    liveset := make(map[_Pos]_RegSet)
    depthmap := make(map[int]int)
    coalescemap := make(map[Reg]Reg)
    nospillregs := make(map[Reg]struct{})

    /* register precolorer */
    precolor := func(rr []*Reg) {
        for _, r := range rr {
            if r.Kind() == K_arch {
                *r = IrSetArch(Rz, ArchRegs[r.Name()])
            }
        }
    }

    /* register coalescer */
    coalesce := func(rr []*Reg) {
        for _, r := range rr {
            if c, ok := coalescemap[*r]; ok {
                *r = c
            }
        }
    }

    /* def-use counter */
    countdefuse := func(rig *simple.UndirectedGraph, bb int, rr []*Reg) {
        for _, r := range rr {
            if _, ok := nospillregs[*r]; !ok {
                if r.Kind() != K_arch && rig.Node(int64(*r)) != nil {
                    defuse[*r] += math.Pow(10.0, float64(depthmap[bb]))
                }
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

    /* calculate the depth map */
    root := cfg.Root
    self.depthmap(depthmap, root, make(map[int]struct{}), make([]int, 0, 16))

    /* loop until no more retries */
    for {
        pool.reset()
        rt.MapClear(edges)
        rt.MapClear(defuse)
        rt.MapClear(livein)
        rt.MapClear(liveout)
        rt.MapClear(liveset)
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

        /* Phase 3: Coalescing registers */
        next := false
        cols := simple.NewUndirectedGraph()

        /* find out all the coalescing pairs */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            for _, v := range bb.Ins {
                var rx Reg
                var ry Reg
                var ok bool

                /* only look for copy-to-virtual-register instructions */
                if rx, ry, ok = IrArchTryIntoCopy(v); !ok || rx == ry || rx.Kind() == K_arch {
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

        /* try again if coalesce occured */
        if next {
            continue
        }

        /* coloring stack */
        spill = false
        order = lane.NewStack()

        /* Phase 4: Check if the RIG is len(arch)-colorable, and color the graph if possible */
        for rig.Nodes().Len() > 0 {
            var nd graph.Node
            var it graph.Nodes

            /* find the node with a degree less than available
             * registers (and have the smallest ID, to be deterministic) */
            for it = rig.Nodes(); it.Next(); {
                if rig.From(it.Node().ID()).Len() < len(arch) {
                    if nd == nil || nd.ID() > it.Node().ID() {
                        nd = it.Node()
                    }
                }
            }

            /* no such node, then some register must be spilled */
            if nd == nil {
                spill = true
                break
            }

            /* add all the edges */
            for et := rig.From(nd.ID()); et.Next(); {
                rr := Reg(nd.ID())
                edges[rr] = append(edges[rr], Reg(et.Node().ID()))
            }

            /* add the node to stack */
            order.Push(Reg(nd.ID()))
            rig.RemoveNode(nd.ID())
        }

        /* no spilling, the coloring is successful */
        if !spill {
            break
        }

        /* count def and use of each register */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            var ok bool
            var use IrUsages
            var def IrDefinitions

            /* should not have Phi nodes */
            if len(bb.Phi) != 0 {
                panic("regalloc: unexpected Phi node at this point")
            }

            /* process instructions */
            for _, v := range bb.Ins {
                if use, ok = v.(IrUsages)      ; ok { countdefuse(rig, bb.Id, use.Usages()) }
                if def, ok = v.(IrDefinitions) ; ok { countdefuse(rig, bb.Id, def.Definitions()) }
            }

            /* process the terminator */
            if use, ok = bb.Term.(IrUsages); ok {
                countdefuse(rig, bb.Id, use.Usages())
            }
        })

        /* cost-to-degree ratio = [(defs & uses) * 10 ^ loop-nest-depth] / degree */
        for r := range defuse {
            defuse[r] /= float64(rig.From(int64(r)).Len())
        }

        /* the register with lowest spill cost */
        spillr := Rz
        spillc := math.MaxFloat64

        /* find that register */
        for r, c := range defuse {
            if c < spillc {
                spillc = c
                spillr = r
            }
        }

        /* mark the register no-spill */
        if spillr == Rz {
            panic("regalloc: corrupted def-use counters")
        } else {
            nospillregs[spillr] = struct{}{}
        }

        /* Phase 5: Spill the selected register */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            reg := Rz
            ldt := false
            ins := bb.Ins

            /* should not have Phi nodes */
            if len(bb.Phi) != 0 {
                panic("regalloc: unexpected Phi nodes")
            } else {
                bb.Ins = make([]IrNode, 0, len(ins))
            }

            /* register allocation */
            newr := func() Reg {
                reg = cfg.CreateRegister(spillr.Ptr())
                nospillregs[reg] = struct{}{}
                return reg
            }

            /* lazy register allocation */
            getr := func() Reg {
                if reg != Rz {
                    return reg
                } else {
                    return newr()
                }
            }

            /* scan all the instructions */
            for _, v := range ins {
                loads := false
                stores := false

                /* check usages */
                if use, ok := v.(IrUsages); ok {
                    for _, r := range use.Usages() {
                        if *r == spillr {
                            *r = getr()
                            loads = true
                        }
                    }
                }

                /* check definitions */
                if def, ok := v.(IrDefinitions); ok {
                    for _, r := range def.Definitions() {
                        if *r == spillr {
                            stores = true
                            break
                        }
                    }
                }

                /* spill & reload IR */
                ld := mkSpillOp(getr(), spillr, true)
                st := mkSpillOp(spillr, spillr, false)

                /* insert spill & reload instructions */
                switch {
                    case !loads && !stores: bb.Ins = append(bb.Ins, v)
                    case !loads &&  stores: bb.Ins = append(bb.Ins, v, st)
                    case  loads && !stores: bb.Ins = append(bb.Ins, ld, v)
                    case  loads &&  stores: bb.Ins = append(bb.Ins, ld, v, st)
                }
            }

            /* check usages in the terminator */
            if use, ok := bb.Term.(IrUsages); ok {
                for _, r := range use.Usages() {
                    if *r == spillr {
                        *r = getr()
                        ldt = true
                    }
                }
            }

            /* insert the reload if needed */
            if ldt {
                bb.Ins = append(bb.Ins, mkSpillOp(getr(), spillr, true))
            }
        })
    }

    /* register colors */
    regmap := make(map[int]Reg)
    colors := make(map[int]struct{})
    colormap := make(map[Reg]int)
    colortab := make([]_RegColor, 0, len(arch))

    /* assign colors to every virtual register */
    for !order.Empty() {
        cx := math.MaxInt64
        reg := order.Pop().(Reg)
        rt.MapClear(colors)

        /* must not be colored */
        if _, ok := colormap[reg]; ok {
            panic("regalloc: allocation confliction")
        }

        /* all possible colors */
        for c := range arch {
            colors[c] = struct{}{}
        }

        /* choose a different color from it's neightbors */
        for _, r := range edges[reg] {
            delete(colors, colormap[r])
        }

        /* find the lowest available color */
        for c := range colors {
            if c < cx {
                cx = c
            }
        }

        /* the register must exists */
        if cx == math.MaxInt64 {
            panic("regalloc: no available register")
        } else {
            colormap[reg] = cx
        }
    }

    /* dump all register colors */
    for r, c := range colormap {
        colortab = append(colortab, _RegColor { r, c })
    }

    /* sort by register order, ensure all arch-specific registers are at the front */
    sort.Slice(colortab, func(i int, j int) bool {
        return regorder(colortab[i].r) < regorder(colortab[j].r)
    })

    /* Phase 6: Assign physical registers to colors */
    for _, rc := range colortab {
        if rc.r.Kind() == K_arch {
            regmap[rc.c] = rc.r
            regsliceremove(&arch, rc.r)
        } else {
            if _, ok := regmap[rc.c]; !ok {
                regmap[rc.c], arch = arch[0], arch[1:]
            }
        }
    }

    /* register substitution function */
    replaceregs := func(rr []*Reg) {
        for _, r := range rr {
            if c, ok := colormap[*r]; ok && r.Kind() != K_arch {
                if *r, ok = regmap[c]; !ok {
                    panic(fmt.Sprintf("regalloc: no register for color %d", c))
                }
            }
        }
    }

    /* replace all the registers */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        var ok bool
        var use IrUsages
        var def IrDefinitions

        /* should not have Phi nodes */
        if len(bb.Phi) != 0 {
            panic("regalloc: unexpected Phi node at this point")
        }

        /* process instructions */
        for _, v := range bb.Ins {
            if use, ok = v.(IrUsages)      ; ok { replaceregs(use.Usages()) }
            if def, ok = v.(IrDefinitions) ; ok { replaceregs(def.Definitions()) }
        }

        /* process the terminator */
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

    /* remove redundant loadStack where the register isn't modified after being stored to stack */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        storeStack := make(map[Reg]IrStackSlot)
        regModified := make(map[Reg]bool)
        ins := bb.Ins
        bb.Ins = nil
        def := IrDefinitions(nil)

        /* scan every instruction */
        for _, v := range ins {
            if storeIr, ok := v.(*IrAMD64_MOV_store_stack) ; ok {
                storeStack[storeIr.R] = *storeIr.S
                regModified[storeIr.R] = false
                bb.Ins = append(bb.Ins, v)
            } else if loadIr, ok := v.(*IrAMD64_MOV_load_stack) ; ok {
                pos := loadIr.S
                /* if a loadStack instruction loads from the same stackPos to the same register when the register isn't modified, abandon it */
                if s, ok := storeStack[loadIr.R]; ok && s == *pos && regModified[loadIr.R] == false {
                    continue
                }
                bb.Ins = append(bb.Ins, v)
            } else {
                if def, ok = v.(IrDefinitions); ok {
                    for _, r := range def.Definitions() {
                        if _, ok := regModified[*r]; ok && regModified[*r] == false {
                            regModified[*r] = true
                        }
                    }
                }
                bb.Ins = append(bb.Ins, v)
            }
        }
    })

    /* remove redundant loadStack where the register isn't modified after being loaded from stack */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        loadStack := make(map[Reg]IrStackSlot)
        regModified := make(map[Reg]bool)
        ins := bb.Ins
        bb.Ins = nil
        def := IrDefinitions(nil)

        /* scan every instruction */
        for _, v := range ins {
            if loadIr, ok := v.(*IrAMD64_MOV_load_stack) ; ok {
                pos := *loadIr.S
                /* if a loadStack instruction loads from the same stackPos to the same register when the register isn't modified, abandon it */
                if s, ok := loadStack[loadIr.R]; ok && s == pos && regModified[loadIr.R] == false {
                    continue
                }
                loadStack[loadIr.R] = pos
                regModified[loadIr.R] = false
                bb.Ins = append(bb.Ins, v)
            } else {
                if def, ok = v.(IrDefinitions); ok {
                    for _, r := range def.Definitions() {
                        if _, ok := regModified[*r]; ok && regModified[*r] == false {
                            regModified[*r] = true
                        }
                    }
                }
                bb.Ins = append(bb.Ins, v)
            }
        }
    })

    /* remove redundant storeStack where the same stackPos is overwritten before loading */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        var redundantIrPos []int
        storeStack := make(map[IrStackSlot][]int)

        /* scan every instruction */
        for i, v := range bb.Ins {
            if storeIr, ok := v.(*IrAMD64_MOV_store_stack) ; ok {
                pos := storeIr.S
                storeStack[*pos] = append(storeStack[*pos], i)
            } else if loadIr, ok := v.(*IrAMD64_MOV_load_stack) ; ok {
                pos := loadIr.S
                if irPos, ok := storeStack[*pos]; ok {
                    redundantIrPos = append(redundantIrPos, irPos[0 : len(irPos)-1]...)
                    delete(storeStack, *pos)
                }
            }
        }

        for _, irPos := range storeStack {
            if len(irPos) > 1 {
                redundantIrPos = append(redundantIrPos, irPos[0 : len(irPos)-1]...)
            }
        }

        /* abandon redundant storeStack instructions according to their position in the block */
        if len(redundantIrPos) > 0 {
            ins := bb.Ins
            bb.Ins = nil
            sort.Ints(redundantIrPos)
            startPos := 0
            for _, p := range redundantIrPos {
                if startPos < len(ins) {
                    bb.Ins = append(bb.Ins, ins[startPos : p]...)
                    startPos = p + 1
                }
            }
            if startPos < len(ins) {
                bb.Ins = append(bb.Ins, ins[startPos : ]...)
            }
        }
    })
}
