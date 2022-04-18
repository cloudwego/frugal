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
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/hir`
)

func ri2reg(ri uint8) hir.Register {
    if ri & hir.ArgPointer == 0 {
        return hir.GenericRegister(ri & hir.ArgMask)
    } else {
        return hir.PointerRegister(ri & hir.ArgMask)
    }
}

func minint(a int, b int) int {
    if a < b {
        return a
    } else {
        return b
    }
}

func addptr(p unsafe.Pointer, i int64) unsafe.Pointer {
    return unsafe.Pointer(uintptr(p) + uintptr(i))
}

func regnewref(v Reg) (r *Reg) {
    r = new(Reg)
    *r = v
    return
}

func regsliceref(v []Reg) (r []*Reg) {
    r = make([]*Reg, len(v))
    for i := range v { r[i] = &v[i] }
    return
}
