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
)

// StackLiveness calculates the liveness of each stack slot.
type StackLiveness struct{}

func (self StackLiveness) livein(lr map[Pos]SlotSet, bb *BasicBlock, in map[int]SlotSet, out map[int]SlotSet) SlotSet {
    var ok bool
    var ss SlotSet
    var sp *IrSpill

    /* check for cached live-in sets */
    if ss, ok = in[bb.Id]; ok {
        return ss.clone()
    }

    /* calculate the live-out set of current block */
    tr := bb.Term
    sl := self.liveout(lr, bb, in, out).clone()

    /* assume all terminators are non-definitive */
    if _, ok = tr.(IrDefinitions); ok {
        panic("regalloc: definitions within terminators")
    }

    /* mark live range of the terminator */
    rr := sl.clone()
    lr[pos(bb, _P_term)] = rr

    /* live(i-1) = use(i) ∪ (live(i) - { def(i) }) */
    for i := len(bb.Ins) - 1; i >= 0; i-- {
        ins := bb.Ins[i]
        sp, ok = ins.(*IrSpill)

        /* only account for pointer slots */
        if !ok || !sp.S.IsPtr() {
            lr[pos(bb, i)] = sl.clone()
            continue
        }

        /* handle spill operations */
        switch sp.Op {
            default: {
                panic("stackmap: invalid spill Op")
            }

            /* store operaion marks the value is alive since here */
            case IrSpillStore: {
                if sl.remove(sp.S) {
                    lr[pos(bb, i)] = sl.clone()
                } else {
                    panic(fmt.Sprintf("stackmap: killing non-existing value %s at %s:%d. ins = %s", sp.S, bb, i, bb.Ins[i]))
                }
            }

            /* load operaion marks the value is alive until here */
            case IrSpillReload: {
                if sl.add(sp.S) {
                    lr[pos(bb, i)] = sl.clone()
                }
            }
        }
    }

    /* should not have any Phi nodes */
    if len(bb.Phi) != 0 {
        panic("regalloc: unexpected Phi nodes")
    }

    /* update the cache */
    in[bb.Id] = sl.clone()
    return sl
}

func (self StackLiveness) liveout(lr map[Pos]SlotSet, bb *BasicBlock, in map[int]SlotSet, out map[int]SlotSet) SlotSet {
    var ok bool
    var ss SlotSet
    var it IrSuccessors

    /* check for cached live-out sets */
    if ss, ok = out[bb.Id]; ok {
        return ss
    }

    /* check for return blocks */
    if _, ok = IrTryIntoArchReturn(bb.Term); ok {
        out[bb.Id] = make(SlotSet)
        return ss
    }

    /* create a new register set */
    ss = make(SlotSet)
    it = bb.Term.Successors()

    /* live-out(p) = ∑(live-in(succ(p))) */
    for out[bb.Id] = nil; it.Next(); {
        for sl := range self.livein(lr, it.Block(), in, out) {
            ss.add(sl)
        }
    }

    /* update cache */
    out[bb.Id] = ss
    return ss
}

func (self StackLiveness) liveness(cfg *CFG) {
    for ss := range self.livein(cfg.Func.Liveness, cfg.Root, make(map[int]SlotSet), make(map[int]SlotSet)) {
        panic("stackliveness: live slot at entry: " + ss.String())
    }
}

func (self StackLiveness) Apply(cfg *CFG) {
    cfg.Func.Liveness = make(map[Pos]SlotSet)
    self.liveness(cfg)
}
