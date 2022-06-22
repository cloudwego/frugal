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

// BlockMerge merges redundant intermediate blocks (blocks with a single
// outgoing edge which goes to another block with a single incoming edge).
type BlockMerge struct{}

func (BlockMerge) Apply(cfg *CFG) {
    for {
        var ok bool
        var rt bool
        var sw *IrSwitch

        /* check every block */
        cfg.PostOrder(func(bb *BasicBlock) {
            if sw, ok = bb.Term.(*IrSwitch); ok && len(sw.Br) == 0 && len(sw.Ln.To.Pred) == 1 {
                rt = true
                bb.Ins = append(bb.Ins, sw.Ln.To.Ins...)
                bb.Term = sw.Ln.To.Term

                /* must not have Phi nodes */
                if len(sw.Ln.To.Phi) != 0 {
                    panic("invalid Phi node found in intermediate blocks")
                }

                /* get the successor iterator */
                tr := sw.Ln.To.Term
                it := tr.Successors()

                /* update all predecessors references */
                for it.Next() {
                    rb := it.Block()
                    pp := rb.Pred

                    /* update predecessor list */
                    for i, p := range pp {
                        if p == sw.Ln.To {
                            pp[i] = bb
                        }
                    }

                    /* update in Phi nodes */
                    for _, v := range rb.Phi {
                        v.V[bb] = v.V[sw.Ln.To]
                        delete(v.V, sw.Ln.To)
                    }
                }
            }
        })

        /* rebuild the dominator tree, and retry if needed */
        if cfg.Rebuild(); !rt {
            break
        }
    }
}

