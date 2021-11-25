/*
 * Copyright 2021 ByteDance Inc.
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

package atm

import (
    `fmt`
    `strings`
)

type Program struct {
    Head *Instr
}

func (self Program) Free() {
    for p, q := self.Head, self.Head; p != nil; p = q {
        q = p.Ln
        freeInstr(p)
    }
}

func (self Program) Disassemble() string {
    ret := make([]string, 0, 64)
    ref := make(map[*Instr]string)

    /* scan all the branch target */
    for p := self.Head; p != nil; p = p.Ln {
        if p.isBranch() {
            if p.Op != OP_bsw {
                if _, ok := ref[p.Br]; !ok {
                    ref[p.Br] = fmt.Sprintf("L_%d", len(ref))
                }
            } else {
                for _, lb := range p.Sw() {
                    if _, ok := ref[lb]; !ok {
                        ref[lb] = fmt.Sprintf("L_%d", len(ref))
                    }
                }
            }
        }
    }

    /* dump all the instructions */
    for p := self.Head; p != nil; p = p.Ln {
        if vv, ok := ref[p]; !ok {
            ret = append(ret, "\t" + p.disassemble(ref))
        } else {
            ret = append(ret, vv + ":", "\t" + p.disassemble(ref))
        }
    }

    /* join them together */
    return strings.Join(ret, "\n")
}
