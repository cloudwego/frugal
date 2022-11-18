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
)

type _Successor struct {
    bb   *BasicBlock
    prob Likeliness
}

// Layout flattens the CFG into a linear FuncLayout
type Layout struct{}

func (self Layout) flatten(fn *FuncLayout, bb *BasicBlock) {
    var ok bool
    var nx []_Successor

    /* check for visited blocks */
    if _, ok = fn.Start[bb.Id]; ok {
        return
    } else {
        fn.Start[bb.Id] = len(fn.Ins)
    }

    /* add instructions and the terminator */
    fn.Ins = append(fn.Ins, bb.Ins...)
    fn.Ins = append(fn.Ins, bb.Term)

    /* get all it's successors */
    for it := bb.Term.Successors(); it.Next(); {
        nx = append(nx, _Successor {
            bb   : it.Block(),
            prob : it.Likeliness(),
        })
    }

    /* sort the likely blocks at front */
    sort.Slice(nx, func(i int, j int) bool {
        return nx[i].prob == Likely && nx[j].prob == Unlikely
    })

    /* visit all the successors */
    for _, v := range nx {
        self.flatten(fn, v.bb)
    }
}

func (self Layout) Apply(cfg *CFG) {
    cfg.Func.Layout = new(FuncLayout)
    cfg.Func.Layout.Start = make(map[int]int, cfg.MaxBlock())

    /* remove all virtual instructions */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = bb.Ins[:0]

        /* filter the instructions */
        for _, v := range ins {
            if _, ok := v.(*IrEntry)       ; ok { continue }
            if _, ok := v.(*IrClobberList) ; ok { continue }
            bb.Ins = append(bb.Ins, v)
        }
    })

    /* retry until no more intermediate blocks found */
    for {
        var rt bool
        var nb *BasicBlock

        /* collapse all the intermediate blocks */
        cfg.PostOrder().ForEach(func(bb *BasicBlock) {
            tr := bb.Term
            it := tr.Successors()

            /* scan every successors */
            for it.Next() {
                nx := it.Block()
                st := nx.Term.Successors()

                /* must have exactly 1 predecessor, exactly 1 successor, and no instructions
                 * if so, update the successor to skip the intermediate block */
                if st.Next() && len(nx.Ins) == 0 && len(nx.Pred) == 1 {
                    if nb = st.Block(); !st.Next() {
                        rt = true
                        it.UpdateBlock(nb)
                    }
                }
            }
        })

        /* rebuild the CFG if needed */
        if !rt {
            break
        } else {
            cfg.Rebuild()
        }
    }

    /* flatten the CFG */
    root := cfg.Root
    self.flatten(cfg.Func.Layout, root)
}
