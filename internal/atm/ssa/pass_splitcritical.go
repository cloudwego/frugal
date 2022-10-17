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

type _CrEdge struct {
    to   *BasicBlock
    from *BasicBlock
}

// SplitCritical splits critical edges (those that go from a block with
// more than one outedge to a block with more than one inedge) by inserting
// an empty block.
//
// PhiProp wants a critical-edge-free CFG so it can safely remove Phi nodes
// for RegAlloc.
type SplitCritical struct{}

func (SplitCritical) Apply(cfg *CFG) {
    var nb int
    var edges []_CrEdge

    /* find all critical edges */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        if len(bb.Pred) > 1 {
            for _, p := range bb.Pred {
                nb = 0
                tr := p.Term.Successors()

                /* check for successors */
                for nb < 2 && tr.Next() {
                    nb++
                }

                /* the predecessor have more than 1 successors, this is a critcal edge */
                if nb > 1 {
                    edges = append(edges, _CrEdge {
                        to   : bb,
                        from : p,
                    })
                }
            }
        }
    })

    /* insert empty block between the edges */
    for _, e := range edges {
        bb := cfg.CreateBlock()
        bb.Term = IrArchJump(e.to)
        bb.Pred = []*BasicBlock { e.from }

        /* update the successor */
        for it := e.from.Term.Successors(); it.Next(); {
            if it.Block() == e.to {
                it.UpdateBlock(bb)
                break
            }
        }

        /* update the predecessor */
        for i, p := range e.to.Pred {
            if p == e.from {
                e.to.Pred[i] = bb
                break
            }
        }

        /* update the Phi nodes */
        for _, p := range e.to.Phi {
            for b, r := range p.V {
                if b == e.from {
                    p.V[bb] = r
                    delete(p.V, b)
                    break
                }
            }
        }
    }

    /* rebuild the CFG if needed */
    if len(edges) != 0 {
        cfg.Rebuild()
    }
}
