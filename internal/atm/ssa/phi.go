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
    `sort`

    `github.com/oleiade/lane`
)

type _PhiDesc struct {
    r Reg
    b []*BasicBlock
}

func appendReg(buf map[Reg]bool, r Reg) map[Reg]bool {
    if buf == nil {
        return map[Reg]bool { r: true }
    } else {
        buf[r] = true
        return buf
    }
}

func appendBlock(buf map[int]*BasicBlock, bb *BasicBlock) map[int]*BasicBlock {
    if buf == nil {
        return map[int]*BasicBlock { bb.Id: bb }
    } else {
        buf[bb.Id] = bb
        return buf
    }
}

func insertPhiNodes(dt *DominatorTree) {
    q := lane.NewQueue()
    phi := make(map[Reg]map[int]bool)
    orig := make(map[int]map[Reg]bool)
    defs := make(map[Reg]map[int]*BasicBlock)

    /* find out all the variable origins */
    for q.Enqueue(dt.Root); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        addImmediateDominators(dt.DominatorOf, p, q)

        /* mark all the defination sites */
        for _, ins := range p.Ins {
            if def, ok := ins.(IrDefinations); ok {
                for _, d := range def.Definations() {
                    if k := d.Kind(); k != K_zero {
                        orig[p.Id] = appendReg(orig[p.Id], *d)
                    }
                }
            }
        }
    }

    /* find out all the variable defination sites */
    for q.Enqueue(dt.Root); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        addImmediateDominators(dt.DominatorOf, p, q)

        /* mark all the defination sites */
        for def := range orig[p.Id] {
            defs[def] = appendBlock(defs[def], p)
        }
    }

    /* reserve buffer for Phi descriptors */
    nb := len(defs)
    pd := make([]_PhiDesc, nb)

    /* dump the descriptors */
    for r, v := range defs {
        n := len(v)
        b := make([]*BasicBlock, 0, n)

        /* dump the blocks */
        for _, p := range v {
            b = append(b, p)
        }

        /* sort blocks by ID */
        sort.Slice(b, func(i int, j int) bool {
            return b[i].Id < b[j].Id
        })

        /* add the descriptor */
        pd = append(pd, _PhiDesc {
            r: r,
            b: b,
        })
    }

    /* sort descriptors by register */
    sort.Slice(pd, func(i int, j int) bool {
        return pd[i].r < pd[j].r
    })

    /* insert Phi node for every variable */
    for _, p := range pd {
        for len(p.b) != 0 {
            n := p.b[0]
            p.b = p.b[1:]

            /* insert Phi nodes */
            for _, y := range dt.DominanceFrontier[n.Id] {
                if rem := phi[p.r]; !rem[y.Id] {
                    id := y.Id
                    src := make(map[*BasicBlock]*Reg)

                    /* mark as processed */
                    if rem != nil {
                        rem[id] = true
                    } else {
                        phi[p.r] = map[int]bool { id: true }
                    }

                    /* build the Phi node args */
                    for _, pred := range y.Pred {
                        src[pred] = new(Reg)
                        *src[pred] = p.r
                    }

                    /* insert a new Phi node */
                    y.Phi = append(y.Phi, &IrPhi {
                        R: p.r,
                        V: src,
                    })

                    /* a node may contain both an ordinary definition and a
                     * Phi node for the same variable */
                    if !orig[y.Id][p.r] {
                        p.b = append(p.b, y)
                    }
                }
            }
        }
    }
}
