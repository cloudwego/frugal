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
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
    `github.com/davecgh/go-spew/spew`
    `gonum.org/v1/gonum/graph/coloring`
    `gonum.org/v1/gonum/graph/simple`
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
    next := true
    pool := mkregtab()
    arch := make([]Reg, 0, len(ArchRegs))
    idmap := make(map[int]Reg)
    livein := make(map[int]_RegSet)
    liveout := make(map[int]_RegSet)
    liveset := make(map[_Pos]_RegSet)

    /* register precolorer */
    precolor := func(rr []*Reg) {
        for _, r := range rr {
            if r.Kind() == K_arch {
                *r = IrSetArch(Rz, ArchRegs[r.Name()])
            }
        }
    }

    // TODO: enable register coalescing
    // /* register coalescer */
    // coalesce := func(rr []*Reg, rs Reg, rd Reg) {
    //     for _, r := range rr {
    //         if *r == rs {
    //             *r = rd
    //         }
    //     }
    // }

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
    for next {
        next = false
        pool.reset()
        rt.MapClear(livein)
        rt.MapClear(liveout)
        rt.MapClear(liveset)

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

            /* create every edge */
            for i := 0; i < nr - 1; i++ {
                for j := i + 1; j < nr; j++ {
                    p, _ := rig.NodeWithID(int64(rr[i]))
                    q, _ := rig.NodeWithID(int64(rr[j]))
                    rig.SetEdge(rig.NewEdge(p, q))
                }
            }
        }

        // TODO: enable register coalescing
        // /* Phase 3: Coalescing one pair of register */
        // for it := cfg.PostOrder(); !next && it.Next(); {
        //     for _, v := range it.Block().Ins {
        //         var rx Reg
        //         var ry Reg
        //         var ok bool
        //
        //         /* only look for copy instructions */
        //         if rx, ry, ok = IrArchTryIntoCopy(v); !ok || rx == ry {
        //             continue
        //         }
        //
        //         /* make sure Y is the node with a lower degree */
        //         if rig.From(int64(rx)).Len() < rig.From(int64(ry)).Len() {
        //             rx, ry = ry, rx
        //         }
        //
        //         /* determain whether it's safe to coalesce using George's heuristic */
        //         for p := rig.From(int64(ry)); p.Next(); {
        //             if t := p.Node().ID(); t == int64(rx) || len(arch) <= rig.From(t).Len() && !rig.HasEdgeBetween(t, int64(rx)) {
        //                 ok = false
        //                 break
        //             }
        //         }
        //
        //         /* check if it can be coalesced */
        //         if !ok {
        //             continue
        //         }
        //
        //         /* check for pre-colored registers */
        //         switch kx, ky := rx.Kind(), ry.Kind(); {
        //             case kx != K_arch && ky != K_arch: break
        //             case kx != K_arch && ky == K_arch: break
        //             case kx == K_arch && ky != K_arch: rx, ry = ry, rx
        //             case kx == K_arch && ky == K_arch: panic(fmt.Sprintf("regalloc: arch-specific register confliction: %s and %s", rx, ry))
        //         }
        //
        //         /* replace all the register references */
        //         cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        //             var use IrUsages
        //             var def IrDefinitions
        //
        //             /* should not have Phi nodes here */
        //             if len(bb.Phi) != 0 {
        //                 panic("regalloc: unexpected Phi node")
        //             }
        //
        //             /* scan every instruction */
        //             for _, p := range bb.Ins {
        //                 if use, ok = p.(IrUsages)      ; ok { coalesce(use.Usages(), rx, ry) }
        //                 if def, ok = p.(IrDefinitions) ; ok { coalesce(def.Definitions(), rx, ry) }
        //             }
        //
        //             /* scan the terminator */
        //             if use, ok = bb.Term.(IrUsages); ok {
        //                 coalesce(use.Usages(), rx, ry)
        //             }
        //         })
        //
        //         /* remove register copies to itself */
        //         cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        //             ins := bb.Ins
        //             bb.Ins = bb.Ins[:0]
        //
        //             /* filter the instructions */
        //             for _, p := range ins {
        //                 if rd, rs, ok := IrArchTryIntoCopy(p); !ok || rd != rs {
        //                     bb.Ins = append(bb.Ins, p)
        //                 }
        //             }
        //         })
        //
        //         /* need to start over */
        //         next = true
        //         break
        //     }
        // }
        //
        // /* try again if coalescing occured */
        // if next {
        //     continue
        // }

        /* Phase 4: Check if the RIG is len(arch)-colorable, and color the graph if possible */
        if k, colors = coloring.RecursiveLargestFirst(rig); k <= len(arch) {
            break
        }

        /* Phase 5: Evaluate spilling cost, and spill the cheapest register */
        panic("regalloc: not implemented: spill")
    }

    /* sanity check */
    if k > len(arch) {
        panic(fmt.Sprintf("regalloc: more colors than available: %d > %d", k, len(arch)))
    }

    // TODO: remove debug code
    cbuf := make([]struct{r Reg; c int}, 0, len(colors))
    nbuf := make([]struct{c int; r []int64}, 0, len(colors))
    for ci, r := range coloring.Sets(colors) { nbuf = append(nbuf, struct{c int; r []int64}{ci, r}) }
    for ri, c := range colors { cbuf = append(cbuf, struct{r Reg; c int}{Reg(ri), c}) }
    sort.Slice(cbuf, func(i, j int) bool { return cbuf[i].r < cbuf[j].r })
    sort.Slice(nbuf, func(i, j int) bool { return nbuf[i].c < nbuf[j].c })
    println(k)
    for _, v := range cbuf {
        println(v.r.String(), "=", v.c)
    }
    for _, v := range nbuf {
        println(v.c, "=", regslicerepr(*(*[]Reg)(unsafe.Pointer(&v.r))))
    }

    /* assign IDs to precolored registers */
    for ri, id := range colors {
        if r := Reg(ri); r.Kind() == K_arch {
            if _, ok := idmap[id]; ok {
                panic("regalloc: confliction of arch-specific register: " + r.String())
            } else {
                arch, idmap[id] = regsliceremove(arch, r), r
            }
        }
    }

    /* assign remaining registers */
    for ri, id := range colors {
        if r := Reg(ri); r.Kind() != K_arch {
            if _, ok := idmap[id]; !ok {
                arch, idmap[id] = arch[1:], arch[0]
            }
        }
    }

    // TODO: remove debug code
    spew.Config.SortKeys = true
    spew.Config.DisablePointerMethods = true
    spew.Dump(idmap)

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

        /* remap the terminator */
        if use, ok = bb.Term.(IrUsages); ok {
            remapregs(use.Usages())
        }
    })
}
