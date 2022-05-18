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
    `github.com/oleiade/lane`
)

type _Renamer struct {
    count map[Reg]int
    stack map[Reg][]int
}

func newRenamer() _Renamer {
    return _Renamer {
        count: make(map[Reg]int),
        stack: make(map[Reg][]int),
    }
}

func (self _Renamer) popr(r Reg) {
    if n := len(self.stack[r]); n != 0 {
        self.stack[r] = self.stack[r][:n - 1]
    }
}

func (self _Renamer) topr(r Reg) int {
    if n := len(self.stack[r]); n == 0 {
        return 0
    } else {
        return self.stack[r][n - 1]
    }
}

func (self _Renamer) pushr(r Reg) (i int) {
    i = self.count[r]
    self.count[r] = i + 1
    self.stack[r] = append(self.stack[r], i)
    return
}

func (self _Renamer) renameuses(ins IrNode) {
    if u, ok := ins.(IrUsages); ok {
        for _, a := range u.Usages() {
            *a = a.Derive(self.topr(*a))
        }
    }
}

func (self _Renamer) renamedefs(ins IrNode, buf *[]Reg) {
    if s, ok := ins.(IrDefinitions); ok {
        for _, def := range s.Definitions() {
            *buf = append(*buf, *def)
            *def = def.Derive(self.pushr(*def))
        }
    }
}

func (self _Renamer) renameblock(dt *DominatorTree, bb *BasicBlock) {
    var r Reg
    var d []Reg
    var n IrNode

    /* rename Phi nodes */
    for _, n = range bb.Phi {
        self.renamedefs(n, &d)
    }

    /* rename body */
    for _, n = range bb.Ins {
        self.renameuses(n)
        self.renamedefs(n, &d)
    }

    /* get the successor iterator */
    tr := bb.Term
    it := tr.Successors()

    /* rename terminators */
    self.renameuses(tr)
    self.renamedefs(tr, &d)

    /* rename all the Phi node of it's successors */
    for it.Next() {
        for _, phi := range it.Block().Phi {
            r = *phi.V[bb]
            phi.V[bb] = regnewref(r.Derive(self.topr(r)))
        }
    }

    /* rename all it's children in the dominator tree */
    for _, p := range dt.DominatorOf[bb.Id] {
        self.renameblock(dt, p)
    }

    /* pop the definations */
    for _, s := range d {
        self.popr(s)
    }
}

func renameRegisters(dt *DominatorTree) {
    rr := newRenamer()
    rr.renameblock(dt, dt.Root)
    normalizeRegisters(dt)
}

func assignRegisters(rr []*Reg, rm map[Reg]Reg) {
    for _, r := range rr {
        if r.Kind() != K_zero {
            if v, ok := rm[*r]; ok {
                panic("register redefined: " + r.String())
            } else {
                v = r.Normalize(len(rm))
                *r, rm[*r] = v, v
            }
        }
    }
}

func replaceRegisters(rr []*Reg, rm map[Reg]Reg) {
    for _, r := range rr {
        if r.Kind() != K_zero {
            if v, ok := rm[*r]; ok {
                *r = v
            } else {
                panic("use of undefined register: " + r.String())
            }
        }
    }
}

func normalizeRegisters(dt *DominatorTree) {
    q := lane.NewQueue()
    r := make(map[Reg]Reg)

    /* find all the register definations */
    for q.Enqueue(dt.Root); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        addImmediateDominated(dt.DominatorOf, p, q)

        /* assign Phi nodes */
        for _, n := range p.Phi {
            assignRegisters(n.Definitions(), r)
        }

        /* assign instructions */
        for _, n := range p.Ins {
            if d, ok := n.(IrDefinitions); ok {
                assignRegisters(d.Definitions(), r)
            }
        }

        /* assign terminators */
        if d, ok := p.Term.(IrDefinitions); ok {
            assignRegisters(d.Definitions(), r)
        }
    }

    /* normalize each block */
    for q.Enqueue(dt.Root); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        addImmediateDominated(dt.DominatorOf, p, q)

        /* replace Phi nodes */
        for _, n := range p.Phi {
            replaceRegisters(n.Usages(), r)
        }

        /* replace instructions */
        for _, n := range p.Ins {
            if u, ok := n.(IrUsages); ok {
                replaceRegisters(u.Usages(), r)
            }
        }

        /* replace terminators */
        if u, ok := p.Term.(IrUsages); ok {
            replaceRegisters(u.Usages(), r)
        }
    }
}
