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
    `strings`

    `github.com/cloudwego/frugal/internal/rt`
)

type FuncData struct {
    Code     []byte
    Layout   *FuncLayout
    Liveness map[Pos]SlotSet
    StackMap map[uintptr]*rt.StackMap
}

type FuncLayout struct {
    Ins   []IrNode
    Start map[int]int
    Block map[int]*BasicBlock
}

func (self *FuncLayout) String() string {
    ni := len(self.Ins)
    ns := len(self.Start)
    ss := make([]string, 0, ni + ns)

    /* print every instruction */
    for i, ins := range self.Ins {
        if bb, ok := self.Block[i]; !ok {
            ss = append(ss, fmt.Sprintf("%06x |     %s", i, ins))
        } else {
            ss = append(ss, fmt.Sprintf("%06x | bb_%d:", i, bb.Id), fmt.Sprintf("%06x |     %s", i, ins))
        }
    }

    /* join them together */
    return fmt.Sprintf(
        "FuncLayout {\n%s\n}",
        strings.Join(ss, "\n"),
    )
}
