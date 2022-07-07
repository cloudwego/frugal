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
    `github.com/davecgh/go-spew/spew`
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
    } else if self.i >= 0 {
        return fmt.Sprintf("bb_%d.ins[%d]", self.b, self.i)
    } else {
        return fmt.Sprintf("bb_%d.phi[bb_%d]", self.b, -self.i)
    }
}

type (
	_RegSet   map[Reg]struct{}
    _LiveRegs map[_Pos]_RegSet
)

func regset(rr ...Reg) (rs _RegSet) {
    rs = make(_RegSet, len(rr))
    for _, r := range rr { rs.add(r) }
    return
}

func regsetp(rr []*Reg) (rs _RegSet) {
    rs = make(_RegSet, len(rr))
    for _, r := range rr { rs.add(*r) }
    return
}

func (self _RegSet) add(r Reg) {
    self[r] = struct{}{}
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

func (self _RegSet) clone() (rs _RegSet) {
    rs = make(_RegSet, len(self))
    for r := range self { rs.add(r) }
    return
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

func (self _RegSet) hasoverlap(rs _RegSet) bool {
    p, q := self, rs
    if len(q) < len(p) { p, q = q, p }
    for r := range p { if _, ok := q[r]; ok { return true } }
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

// RegAlloc performs register allocation on CFG.
type RegAlloc struct{}

func (self RegAlloc) livein(lr _LiveRegs, bb *BasicBlock, pred *BasicBlock, in map[_Pos]_RegSet, out map[int]_RegSet) _RegSet {
    var ok bool
    var rs _RegSet
    var use IrUsages
    var def IrDefinitions

    /* check for cached live-in sets */
    if rs, ok = in[pos(bb, pred.Id)]; ok {
        return rs
    }

    /* calculate the live-out set of current block */
    tr := bb.Term
    regs := self.liveout(lr, bb, in, out).clone()

    /* assume all terminators are non-definitive */
    if _, ok = tr.(IrDefinitions); ok {
        panic("regalloc: definitions within terminators")
    }

    /* scan the terminator */
    if use, ok = tr.(IrUsages); ok {
        regs.union(regsetp(use.Usages()))
        lr[pos(bb, _P_term)] = regs.clone()
    }

    /* scan all instructions backwards */
    for i := len(bb.Ins) - 1; i >= 0; i-- {
        if def, ok = bb.Ins[i].(IrDefinitions) ; ok { regs.subtract(regsetp(def.Definitions())) }
        if use, ok = bb.Ins[i].(IrUsages)      ; ok { regs.union(regsetp(use.Usages())) }
        lr[pos(bb, i)] = regs.clone()
    }

    /* Phi register definitions and usages */
    pdef := make([]Reg, 0, len(bb.Phi))
    puse := make(map[int][]Reg, len(bb.Phi))

    /* scan all Phi nodes */
    for _, v := range bb.Phi {
        pv := v.V
        pdef = append(pdef, v.R)

        /* add all the Phi selections */
        for b, r := range pv {
            puse[b.Id] = append(puse[b.Id], *r)
        }
    }

    /* update the register set with Phi definitions */
    id := -pred.Id - 1
    regs.subtract(regset(pdef...))

    /* update cache for all predecessors */
    for b, r := range puse {
        rs = regs.clone()
        rs.union(regset(r...))
        in[pos(bb, -b - 1)] = rs
    }

    /* should have the register set in cache */
    if rs, ok = in[pos(bb, id)]; ok {
        return rs
    } else {
        return regs
    }
}

func (self RegAlloc) liveout(lr _LiveRegs, bb *BasicBlock, in map[_Pos]_RegSet, out map[int]_RegSet) _RegSet {
    var ok bool
    var rr []Reg
    var rs _RegSet

    /* check for cached live-out sets */
    if rs, ok = out[bb.Id]; ok {
        return rs
    }

    /* check for return blocks */
    if rr, ok = IrTryIntoArchReturn(bb.Term); ok {
        rs = regset(rr...)
        out[bb.Id] = rs
        return rs
    }

    /* create a new register set */
    rs = make(_RegSet)
    it := bb.Term.Successors()

    /* live-out{p} = ∑(live-in{succ(p)}) */
    for out[bb.Id] = nil; it.Next(); {
        rs.union(self.livein(lr, it.Block(), bb, in, out))
    }

    /* update cache */
    out[bb.Id] = rs
    return rs
}

func (self RegAlloc) Apply(cfg *CFG) {
    next := true
    regs := make(_LiveRegs)
    blocks := make(map[int]*BasicBlock, cfg.MaxBlock())
    livein := make(map[_Pos]_RegSet)
    liveout := make(map[int]_RegSet)

    /* collect all basic blocks */
    cfg.PostOrder(func(bb *BasicBlock) {
        blocks[bb.Id] = bb
    })

    /* dummy start block */
    start := &BasicBlock {
        Id   : -1,
        Term : &IrSwitch { Ln: IrLikely(cfg.Root) },
    }

    /* loop until no more retries */
    for next {
        next = false
        rt.MapClear(regs)
        rt.MapClear(livein)
        rt.MapClear(liveout)

        /* Phase 1: Calculate live ranges */
        root := cfg.Root
        self.livein(regs, root, start, livein, liveout)

        spew.Config.SortKeys = true
        spew.Config.DisablePointerMethods = true
        spew.Dump(regs)
    }
}
