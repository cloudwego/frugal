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
    `math`
    `sort`
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/oleiade/lane`
)

func isu8(v int64) bool {
    return v >= 0 && v <= math.MaxUint8
}

func isi32(v int64) bool {
    return v >= math.MinInt32 && v <= math.MaxInt32
}

func isp32(p unsafe.Pointer) bool {
    return uintptr(p) <= math.MaxInt32
}

func ri2reg(ri uint8) Reg {
    if ri & hir.ArgPointer == 0 {
        return Rv(hir.GenericRegister(ri & hir.ArgMask))
    } else {
        return Rv(hir.PointerRegister(ri & hir.ArgMask))
    }
}

func ri2regz(ri []uint8) Reg {
    switch len(ri) {
        case 0  : return Rz
        case 1  : return ri2reg(ri[0])
        default : panic("invalid register count")
    }
}

func ri2regs(ri []uint8) []Reg {
    ret := make([]Reg, len(ri))
    for i, r := range ri { ret[i] = ri2reg(r) }
    return ret
}

func minint(a int, b int) int {
    if a < b {
        return a
    } else {
        return b
    }
}

func cmpu64(a uint64, b uint64) int {
    if a < b {
        return -1
    } else if a > b {
        return 1
    } else {
        return 0
    }
}

func addptr(p unsafe.Pointer, i int64) unsafe.Pointer {
    return unsafe.Pointer(uintptr(p) + uintptr(i))
}

func memsizec(n uint8) rune {
    switch n {
        case 1  : return 'b'
        case 2  : return 'w'
        case 4  : return 'l'
        case 8  : return 'q'
        default : panic("unreachable")
    }
}

func stacknew(v interface{}) (r *lane.Stack) {
    r = lane.NewStack()
    r.Push(v)
    return
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

func regslicerepr(v []Reg) string {
    r := make([]string, 0, len(v))
    for _, x := range v  { r = append(r, x.String()) }
    return strings.Join(r, ", ")
}

func regsliceclone(v []Reg) (r []Reg) {
    r = make([]Reg, len(v))
    copy(r, v)
    return
}

func regsliceremove(v []Reg, x Reg) (r []Reg) {
    r = v[:0]
    for _, p := range v { if p != x { r = append(r, p) } }
    return
}

func blockreverse(s []*BasicBlock) {
    for i, j := 0, len(s) - 1; i < j; i, j = i + 1, j - 1 {
        s[i], s[j] = s[j], s[i]
    }
}

func insertSortedInts(mm []int, v int) []int {
    i := sort.SearchInts(mm, v)
    mm = append(mm, 0)
    copy(mm[i + 1:], mm[i:])
    mm[i] = v
    return mm
}
