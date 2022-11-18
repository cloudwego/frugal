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
    `math`
)

const (
    _P_term = math.MaxUint32
)

type Pos struct {
    B *BasicBlock
    I int
}

func pos(bb *BasicBlock, i int) Pos {
    return Pos { bb, i }
}

func (self Pos) String() string {
    if self.I == _P_term {
        return fmt.Sprintf("bb_%d.term", self.B.Id)
    } else {
        return fmt.Sprintf("bb_%d.ins[%d]", self.B.Id, self.I)
    }
}

func (self Pos) isPriorTo(other Pos) bool {
    return self.B.Id < other.B.Id || (self.I < other.I && self.B.Id == other.B.Id)
}
