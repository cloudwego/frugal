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

func insertPhiNodes(bb *BasicBlock, dt DominatorTree) {
    q := lane.NewQueue()
    phi := make(map[Reg]map[int]bool)
    orig := make(map[int]map[Reg]bool)
    defs := make(map[Reg]map[int]*BasicBlock)

    /* find out all the variable origins */
    for q.Enqueue(bb); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        addImmediateDominators(dt.DominatorOf, p, q)

        /* mark all the defination sites */
        for _, ins := range p.Ins {
            if def, ok := ins.(IrDefinations); ok {
                for _, d := range def.Definations() {
                    if k := d.Kind(); k != _K_temp && k != _K_zero {
                        orig[p.Id] = appendReg(orig[p.Id], *d)
                    }
                }
            }
        }
    }

    /* find out all the variable defination sites */
    for q.Enqueue(bb); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        addImmediateDominators(dt.DominatorOf, p, q)

        /* mark all the defination sites */
        for def := range orig[p.Id] {
            defs[def] = appendBlock(defs[def], p)
        }
    }

    /* insert Phi node for every variable */
    for a, w := range defs {
        for len(w) != 0 {
            var k int
            var n *BasicBlock
            var y *BasicBlock

            /* remove some node from worklist */
            for k, n = range w {
                delete(w, k)
                break
            }

            /* insert Phi nodes */
            for _, y = range dt.DominanceFrontier[n.Id] {
                if rem := phi[a]; !rem[y.Id] {
                    id := y.Id
                    src := make(map[*BasicBlock]*Reg)

                    /* mark as processed */
                    if rem == nil {
                        phi[a] = map[int]bool { id: true }
                    } else {
                        rem[id] = true
                    }

                    /* build the Phi node args */
                    for _, pred := range y.Pred {
                        src[pred] = new(Reg)
                        *src[pred] = a
                    }

                    /* insert a new Phi node */
                    y.Phi = append(y.Phi, &IrPhi {
                        R: a,
                        V: src,
                    })

                    /* a node may contain both an ordinary definition and a
                     * Phi node for the same variable */
                    if !orig[y.Id][a] {
                        w[y.Id] = y
                    }
                }
            }
        }
    }
}
