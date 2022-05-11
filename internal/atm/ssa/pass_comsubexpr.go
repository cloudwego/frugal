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

type _VID interface {
    IrDefinations
    vid() []string
}

func (self *IrLoadArg) vid() []string {
    return []string {
        fmt.Sprintf("#%d", self.Id),
    }
}

func (self *IrConstInt) vid() []string {
    return []string {
        fmt.Sprintf("$%d", self.V),
    }
}

func (self *IrConstPtr) vid() []string {
    return []string {
        fmt.Sprintf("$%p", self.P),
    }
}

func (self *IrLEA) vid() []string {
    return []string {
        fmt.Sprintf("(& %s %s)", self.Mem, self.Off),
    }
}

func (self *IrUnaryExpr) vid() []string {
    return []string {
        fmt.Sprintf("(%s %s)", self.Op, self.V),
    }
}

func (self *IrBinaryExpr) vid() []string {
    x := self.X
    y := self.Y

    /* commutative operations, sort the operands */
    switch self.Op {
        case IrOpAdd : fallthrough
        case IrOpMul : fallthrough
        case IrOpAnd : fallthrough
        case IrOpOr  : fallthrough
        case IrOpXor : fallthrough
        case IrCmpEq : fallthrough
        case IrCmpNe : if x > y { x, y = y, x }
    }

    /* build the value ID */
    return []string {
        fmt.Sprintf("(%s %s %s)", self.Op, x, y),
    }
}

func (self *IrBitTestSet) vid() []string {
    return []string {
        fmt.Sprintf("(&# %s %s)", self.X, self.Y),
        fmt.Sprintf("(|# %s %s)", self.X, self.Y),
    }
}

// CSE performs the Common Sub-expression Elimintation optimization.
type CSE struct{}

func (CSE) Apply(cfg *CFG) {
    for {
        done := true
        vals := make(map[string]Reg)

        /* replace all the values with same VID with a copy to the first occurance */
        cfg.ReversePostOrder(func(bb *BasicBlock) {
            for i, v := range bb.Ins {
                var r Reg
                var d _VID
                var ok bool

                /* check if the instruction have VIDs */
                if d, ok = v.(_VID); !ok {
                    continue
                }

                /* calculate the VIDs */
                repc := i
                vids := d.vid()
                defs := d.Definations()

                /* replace each VID with a copy instruction */
                for j, vid := range vids {
                    s := defs[j]
                    r, ok = vals[vid]

                    /* add to definations if not found */
                    if r, ok = vals[vid]; !ok {
                        vals[vid] = *s
                        continue
                    }

                    /* allocate one slot for the new instruction */
                    repc++
                    bb.Ins = append(bb.Ins, nil)
                    copy(bb.Ins[repc + 1:], bb.Ins[repc:])

                    /* insert a new copy instruction */
                    t := *s
                    *s, done = s.zero(), false
                    bb.Ins[repc] = IrCopy(t, r)
                }

                /* all the definations are been replaced */
                if repc == i + len(defs) {
                    bb.Ins = append(bb.Ins[:i], bb.Ins[i + 1:]...)
                }
            }
        })

        /* no modifications in this round */
        if done {
            break
        }
    }
}
