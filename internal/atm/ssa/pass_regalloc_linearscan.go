// +build ignore

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
    `os`
    `sort`
    `strings`

    `github.com/ajstarks/svgo`
    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/davecgh/go-spew/spew`
)

type _LivePoint struct {
    b int
    i int
}

func (self _LivePoint) String() string {
    return fmt.Sprintf("%d:%d", self.b, self.i)
}

func (self _LivePoint) isPriorTo(other _LivePoint) bool {
    return self.b < other.b || (self.b == other.b && self.i < other.i)
}

type _LiveRange struct {
    p []_LivePoint
}

func (self *_LiveRange) last() _LivePoint {
    return self.p[len(self.p) - 1]
}

func (self *_LiveRange) String() string {
    nb := len(self.p)
    buf := make([]string, 0, nb)

    /* add usages */
    for _, u := range self.p {
        buf = append(buf, u.String())
    }

    /* join them together */
    return fmt.Sprintf(
        "{%s}",
        strings.Join(buf, ", "),
    )
}

func liverangemark(regs map[Reg]*_LiveRange, refs []*Reg, b int, i int) {
   for _, r := range refs {
        if r.Kind() != K_zero {
            if lr, ok := regs[*r]; ok {
                lr.p = append(lr.p, _LivePoint { b, i })
            } else {
                regs[*r] = &_LiveRange { p: []_LivePoint {{ b, i }} }
            }
        }
   }
}

type _RegValue struct {
    index int
    isReg bool
}

func vreg(i int) *_RegValue {
    return &_RegValue {
        index: i,
        isReg: true,
    }
}

func vstack(i int) *_RegValue {
    return &_RegValue {
        index: i,
        isReg: false,
    }
}

func (self *_RegValue) String() string {
    if self.isReg {
        return fmt.Sprintf("reg #%d", self.index)
    } else {
        return fmt.Sprintf("slot #%d", self.index)
    }
}

const (
    _SpillLoadReg = x86_64.R8 // dedicated register to use when doing spill-reload
)

// RegAlloc performs register allocation on CFG.
type RegAlloc struct{}

func (RegAlloc) Apply(cfg *CFG) {
    bbs := make([]*BasicBlock, 0, 16)
    regs := make(map[Reg]*_LiveRange)

    /* register precolorer */
    precolor := func(rr []*Reg) {
        for _, r := range rr {
            if r.Kind() == K_arch {
                *r = IrSetArch(Rz, ArchRegs[r.Name()])
            }
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

    /* Phase 1: Serialize all the basic blocks with heuristics */
    for _, bb := range cfg.PostOrder().Reversed() {
        bbs = append(bbs, bb)
    }

    /* Phase 2: Scan all the instructions to determain live ranges */
    for b, bb := range bbs {
        var ok bool
        var use IrUsages
        var def IrDefinitions

        /* should not contain Phi nodes */
        if len(bb.Phi) != 0 {
            panic(fmt.Sprintf("non-empty Phi nodes in bb_%d", bb.Id))
        }

        /* scan instructions */
        for i, v := range bb.Ins {
            if use, ok = v.(IrUsages)      ; ok { liverangemark(regs, use.Usages(), b, i) }
            if def, ok = v.(IrDefinitions) ; ok { liverangemark(regs, def.Definitions(), b, i) }
        }

        /* scan terminators */
        if use, ok = bb.Term.(IrUsages); ok {
            liverangemark(regs, use.Usages(), b, len(bb.Ins))
        }
    }

    /* sort live ranges by usage position */
    for _, rr := range regs {
        sort.Slice(rr.p, func(i int, j int) bool {
            return rr.p[i].isPriorTo(rr.p[j])
        })
    }

    // TODO: remove this
    spew.Config.SortKeys = true
    spew.Dump(regs)
    draw_liverange("/tmp/live_ranges.svg", bbs, regs)

    /* active set */
    sslots := 0
    ranges := make([]Reg, 0, len(regs))
    active := make([]Reg, 0, len(regs))
    ravail := make([]Reg, 0, len(ArchRegs))
    ralloc := make([]int, 0, len(ArchRegs))
    regmap := make(map[Reg]*_RegValue, len(regs))

    /* dump all the registers */
    for r := range regs {
        ranges = append(ranges, r)
    }

    /* calculate allocatable registers */
    for _, r := range ArchRegs {
        if !ArchRegReserved[r] && r != _SpillLoadReg {
            ralloc = append(ralloc, len(ravail))
            ravail = append(ravail, IrSetArch(Rz, r))
        }
    }

    /* sort by increasing starting point */
    sort.SliceStable(ranges, func(i, j int) bool {
        return regs[ranges[i]].p[0].isPriorTo(regs[ranges[j]].p[0])
    })

    /* add to active set */
    addActive := func(r Reg) {
        end := regs[r].last()
        pos := sort.Search(len(active), func(i int) bool { return !regs[active[i]].last().isPriorTo(end) })
        active = append(active, 0)
        copy(active[pos + 1:], active[pos:])
        active[pos] = r
    }

    /* re-spill an already allocated register */
    resetRegAlloc := func(r Reg, s Reg) {
        println("replace reg", r.String(), "=", regmap[s].index)
        println("replace spill", s.String(), "to stack slot", sslots)
        regmap[r] = regmap[s]
        regmap[s] = vstack(sslots)

        /* make a copy of the active intervals */
        abuf := active
        active = active[:0]

        /* filter out the spilled one */
        for _, v := range abuf {
            if v != s {
                active = append(active, v)
            }
        }

        /* add the interval to active list */
        sslots++
        addActive(r)
    }

    /* spill register */
    spillAtInterval := func(r Reg) {
        nregs := len(active)
        spill := active[nregs - 1]

        /* this interval ends after all the active intervals, allocate a stack slot directly */
        if r.Kind() != K_arch && !regs[r].last().isPriorTo(regs[spill].last()) {
            println("direct spill", r.String(), "to stack slot", sslots)
            regmap[r] = vstack(sslots)
            sslots++
            return
        }

        /* don't spill pre-colored registers */
        for spill.Kind() == K_arch {
            nregs--
            spill = active[nregs - 1]
        }

        /* sanity check */
        if !regmap[spill].isReg {
            panic("spill of an already spilled value " + spill.String())
        } else {
            resetRegAlloc(r, spill)
        }
    }

    /* expires dead intervals */
    expireOldIntervals := func(r Reg) {
        for len(active) > 0 && regs[active[0]].last().isPriorTo(regs[r].p[0]) {
            ralloc = insertSortedInts(ralloc, regmap[active[0]].index)
            active = active[1:]
        }
    }

    /* linear scan allocation */
    for _, r := range ranges {
        if expireOldIntervals(r); len(active) == len(ravail) {
            spillAtInterval(r)
        } else {
            println("reg", r.String(), "=", ralloc[0])
            regmap[r], ralloc = vreg(ralloc[0]), ralloc[1:]
            addActive(r)
        }
    }

    println("slot count", sslots)
    spew.Config.SortKeys = true
    spew.Config.DisablePointerMethods = true
    spew.Dump(regmap)
}

func draw_liverange(fn string, bb []*BasicBlock, lr map[Reg]*_LiveRange) {
    leni := 0
    maxw := 0
    maxi := 0
    regs := make([]Reg, 0, len(lr))
    insi := make(map[_LivePoint]int)
    for _, b := range bb {
        for _, v := range b.Ins {
            s := v.String()
            s = strings.TrimSpace(strings.Split(s, "# ")[0])
            if len(s) > maxi {
                maxi = len(s)
            }
        }
        s := b.Term.String()
        s = strings.TrimSpace(strings.Split(s, "# ")[0])
        if len(s) > maxi {
            maxi = len(s)
        }
    }
    for r := range lr {
        s := r.String()
        regs = append(regs, r)
        if len(s) > maxw {
            maxw = len(s)
        }
    }
    insw := maxi * 9 + 120
    regw := (maxw + 1) * 8 + 16
    sort.Slice(regs, func(i, j int) bool {
        k1, k2 := regs[i].Kind(), regs[j].Kind()
        n1, n2 := regs[i].Name(), regs[j].Name()
        return k1 < k2 || (k1 == k2 && n1 < n2) || (k1 == k2 && n1 == n2 && regs[i] < regs[j])
    })
    sort.SliceStable(regs, func(i, j int) bool {
        p1 := lr[regs[i]].p
        p2 := lr[regs[j]].p
        return p1[0].isPriorTo(p2[0])
    })
    for _, b := range bb {
        leni += len(b.Ins) + 1
    }
    fp, err := os.OpenFile(fn, os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0644)
    if err != nil {
        panic(err)
    }
    p := svg.New(fp)
    p.Start(len(regs) * regw + insw + 100, leni * 24 + 100)
    if _, err = fp.WriteString(`<rect width="100%" height="100%" fill="white" />` + "\n"); err != nil {
        panic(err)
    }
    bbi := 0
    for i, b := range bb {
        p.Text(16, 100 + bbi * 24, fmt.Sprintf("bb_%d", b.Id), "fill:gray;font-size:16px;font-family:monospace")
        p.Line(10, 84 + bbi * 24, insw + 5, 84 + bbi * 24, "stroke:lightgray")
        for j, v := range b.Ins {
            s := v.String()
            h := 95 + bbi * 24
            s = strings.TrimSpace(strings.Split(s, "# ")[0])
            insi[_LivePoint { b: i, i: j }] = h
            p.Text(insw, 100 + bbi * 24, s, "fill:black;font-size:16px;font-family:monospace;text-anchor:end")
            p.Line(insw + 10, h, len(regs) * regw + insw + 50, h, "stroke:gray")
            bbi++
        }
        s := b.Term.String()
        h := 95 + bbi * 24
        s = strings.TrimSpace(strings.Split(s, "# ")[0])
        insi[_LivePoint { b: i, i: len(b.Ins) }] = h
        p.Text(insw, 100 + bbi * 24, s, "fill:black;font-size:16px;font-family:monospace;text-anchor:end")
        p.Line(insw + 10, h, len(regs) * regw + insw + 50, h, "stroke:gray")
        bbi++
    }
    for i, r := range regs {
        x := insw + i * regw + 50
        p.Text(x, 70, r.String(), "fill:black;font-size:16px;font-family:monospace;text-anchor:middle")
        p.Line(x, insi[lr[r].p[0]], x, insi[lr[r].p[len(lr[r].p) - 1]], "stroke:black;stroke-width:3")
        for _, pt := range lr[r].p {
            ins := bb[pt.b].Ins
            isdef := false
            if pt.i < len(ins) {
                if def, ok := bb[pt.b].Ins[pt.i].(IrDefinitions); ok {
                    for _, v := range def.Definitions() {
                        if r == *v {
                            isdef = true
                            break
                        }
                    }
                }
            }
            if !isdef {
                p.Circle(x, insi[pt], 4, "fill:black;stroke:black;stroke-width:2")
            } else {
                p.Circle(x, insi[pt], 4, "fill:white;stroke:black;stroke-width:2")
            }
        }
    }
    p.End()
    if err = fp.Close(); err != nil {
        panic(err)
    }
}
