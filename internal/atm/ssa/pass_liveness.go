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

type _IrKill struct {
    R []Reg
}

func (*_IrKill) irnode()      {}
func (*_IrKill) irimmovable() {}

func (self *_IrKill) String() string {
    return fmt.Sprintf("kill {%s}", regslicerepr(self.R))
}

func (self *_IrKill) Usages() []*Reg {
    return regsliceref(self.R)
}

type _KillPos struct {
    i  int
    bb *BasicBlock
}

// Liveness determains the live-range of each register, and insert _IrKill to kill an register if needed.
type Liveness struct{}

func (Liveness) Apply(cfg *CFG) {

}

