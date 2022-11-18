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
)

type FuncData struct {
    Layout   *FuncLayout
    Liveness map[Pos]SlotSet
}

type FuncLayout struct {
    Ins   []IrNode
    Start map[int]int
}

func (self *FuncLayout) String() string {
    rev := make(map[int]int, len(self.Start))
    buf := make([]string, 0, len(self.Ins) + len(self.Start))

    /* mark all the starting position for each basic block */
    for i, p := range self.Start {
        rev[p] = i
    }

    /* print every instruction */
    for i, ins := range self.Ins {
        if bb, ok := rev[i]; !ok {
            buf = append(buf, fmt.Sprintf("%06x |     %s", i, ins))
        } else {
            buf = append(buf, fmt.Sprintf("%06x | bb_%d:", i, bb), fmt.Sprintf("%06x |     %s", i, ins))
        }
    }

    /* join them together */
    return fmt.Sprintf(
        "FuncLayout {\n%s\n}",
        strings.Join(buf, "\n"),
    )
}
