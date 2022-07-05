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
    `sort`
    `strings`

    `github.com/davecgh/go-spew/spew`
)

type _LiveIntv struct {
    bb    *BasicBlock
    last  int
    first int
}

func (self _LiveIntv) String() string {
    if self.first == self.last {
        return fmt.Sprintf("bb_%d:{%d}", self.bb.Id, self.first)
    } else {
        return fmt.Sprintf("bb_%d:{%d~%d}", self.bb.Id, self.first, self.last)
    }
}

type _LiveRange struct {
    r []_LiveIntv
}

func (self *_LiveRange) mark(bb *BasicBlock, i int) {
    v := bb.Id
    p := sort.Search(len(self.r), func(i int) bool { return self.r[i].bb.Id >= v })

    /* not found, insert a new one */
    if p >= len(self.r) || self.r[p].bb.Id != v {
        self.r = append(self.r, _LiveIntv{})
        copy(self.r[p + 1:], self.r[p:])
        self.r[p] = _LiveIntv { bb: bb, first: i, last: i }
    }

    /* extend live range if needed */
    if self.r[p].last < i {
        self.r[p].last = i
    }
}

func (self *_LiveRange) String() string {
    var s []string
    for _, v := range self.r { s = append(s, v.String()) }
    return strings.Join(s, ", ")
}

type _LiveRanges struct {
    r map[Reg]*_LiveRange
}

func (self *_LiveRanges) dlr(rr Reg) *_LiveRange {
    var ok bool
    var lr *_LiveRange

    /* check for existing ranges */
    if lr, ok = self.r[rr]; ok {
        return lr
    }

    /* check for uninitialized map */
    if self.r == nil {
        self.r = make(map[Reg]*_LiveRange)
    }

    /* create a new one if not exists */
    lr = new(_LiveRange)
    self.r[rr] = lr
    return lr
}

func (self *_LiveRanges) markphi(rr Reg, bb *BasicBlock, i int) {
    if i < 0 || i >= len(bb.Phi) {
        panic("invalid Phi node index")
    } else {
        self.dlr(rr).mark(bb, i - len(bb.Phi))
    }
}

func (self *_LiveRanges) markins(rr Reg, bb *BasicBlock, i int) {
    if i < 0 || i >= len(bb.Ins) {
        panic("invalid instruction index")
    } else {
        self.dlr(rr).mark(bb, i)
    }
}

func (self *_LiveRanges) markterm(rr Reg, bb *BasicBlock) {
    self.dlr(rr).mark(bb, len(bb.Ins))
}

// RegAlloc performs register allocation on CFG.
type RegAlloc struct{}

func (self RegAlloc) liverange(lr *_LiveRanges, bb *BasicBlock, visited map[int]bool) {
    id := bb.Id
    tr := bb.Term

    /* already visited */
    if visited[id] {
        return
    }

    /* mark as visited */
    it := tr.Successors()
    visited[bb.Id] = true

    /* scan Phi nodes */
    for i, p := range bb.Phi {
        for _, v := range p.Usages()      { lr.markphi(*v, bb, i) }
        for _, v := range p.Definitions() { lr.markphi(*v, bb, i) }
    }

    /* scan instructions */
    for i, p := range bb.Ins {
        if u, ok := p.(IrUsages)      ; ok { for _, v := range u.Usages()      { lr.markins(*v, bb, i) } }
        if d, ok := p.(IrDefinitions) ; ok { for _, v := range d.Definitions() { lr.markins(*v, bb, i) } }
    }

    /* scan terminator */
    if u, ok := bb.Term.(IrUsages); ok {
        for _, v := range u.Usages() {
            lr.markterm(*v, bb)
        }
    }

    /* visit all the successors */
    for it.Next() {
        self.liverange(lr, it.Block(), visited)
    }
}

func (self RegAlloc) Apply(cfg *CFG) {
    root := cfg.Root
    visited := make(map[int]bool, cfg.MaxBlock())

    /* Phase 1: Calculate register live ranges */
    dlr := new(_LiveRanges)
    self.liverange(dlr, root, visited)

    spew.Config.SortKeys = true
    spew.Dump(dlr)
}
