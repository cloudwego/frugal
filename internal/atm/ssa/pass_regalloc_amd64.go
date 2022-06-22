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

type _LivePoint struct {
    b int
    i int
}

func (self _LivePoint) String() string {
    return fmt.Sprintf("%d:%d", self.b, self.i)
}

func (self _LivePoint) isPrior(other _LivePoint) bool {
    return self.b < other.b || (self.b == other.b && self.i < other.i)
}

type _LiveRange struct {
    p []_LivePoint
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

// RegAlloc performs register allocation on CFG.
type RegAlloc struct{}

func (RegAlloc) Apply(cfg *CFG) {
    bbs := make([]*BasicBlock, 0, 16)
    regs := make(map[Reg]*_LiveRange)

    /* Phase 1: Enumerate all the basic blocks */
    cfg.ReversePostOrder(func(bb *BasicBlock) {
        bbs = append(bbs, bb)
    })

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
            return rr.p[i].isPrior(rr.p[j])
        })
    }

    spew.Config.SortKeys = true
    spew.Dump(regs)
    draw_liverange("/tmp/live_ranges.svg", bbs, regs)
}
