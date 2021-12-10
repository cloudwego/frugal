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

package encoder

import (
    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/binary/defs`
)

func (self Compiler) measureOne(p *Program, sp int, vt *defs.Type) {
    if vt.T == defs.T_pointer {
        self.measurePtr(p, sp, vt)
    } else if _, ok := self[vt.S]; !ok {
        self.measureTag(p, sp, vt)
    } else {
        p.rtt(OP_size_defer, vt.S)
    }
}

func (self Compiler) measureTag(p *Program, sp int, vt *defs.Type) {
    self[vt.S] = true
    self.measureRec(p, sp, vt)
    delete(self, vt.S)
}

func (self Compiler) measureRec(p *Program, sp int, vt *defs.Type) {
    switch vt.T {
        case defs.T_bool   : p.i64(OP_size_const, 1)
        case defs.T_i8     : p.i64(OP_size_const, 1)
        case defs.T_i16    : p.i64(OP_size_const, 2)
        case defs.T_i32    : p.i64(OP_size_const, 4)
        case defs.T_i64    : p.i64(OP_size_const, 8)
        case defs.T_double : p.i64(OP_size_const, 8)
        case defs.T_string : p.i64(OP_size_const, 4); p.dyn(OP_size_dyn, atm.PtrSize, 1)
        case defs.T_binary : p.i64(OP_size_const, 4); p.dyn(OP_size_dyn, atm.PtrSize, 1)
        case defs.T_struct : self.measureStruct  (p, sp, vt)
        case defs.T_map    : self.measureMap     (p, sp, vt)
        case defs.T_set    : self.measureSetList (p, sp, vt)
        case defs.T_list   : self.measureSetList (p, sp, vt)
        default            : panic("unreachable")
    }
}

func (self Compiler) measurePtr(p *Program, sp int, vt *defs.Type) {
    i := p.pc()
    n := defs.GetSize(vt.V.S)

    /* check for nil */
    p.tag(sp)
    p.add(OP_if_nil)

    /* element is trivially measuable */
    if n > 0 {
        p.i64(OP_size_const, int64(n))
        p.pin(i)
        return
    }

    /* complex values */
    p.add(OP_make_state)
    p.add(OP_deref)
    self.measureOne(p, sp + 1, vt.V)
    p.add(OP_drop_state)
    p.pin(i)
}

func (self Compiler) measureMap(p *Program, sp int, vt *defs.Type) {
    nk := defs.GetSize(vt.K.S)
    nv := defs.GetSize(vt.V.S)

    /* 6-byte map header */
    p.tag(sp)
    p.i64(OP_size_const, 6)

    /* check for nil maps */
    i := p.pc()
    p.add(OP_if_nil)

    /* key and value are both trivially measuable */
    if nk > 0 && nv > 0 {
        p.i64(OP_size_map, int64(nk + nv))
        p.pin(i)
        return
    }

    /* key or value is trivially measuable */
    if nk > 0 { p.i64(OP_size_map, int64(nk)) }
    if nv > 0 { p.i64(OP_size_map, int64(nv)) }

    /* complex maps */
    p.add(OP_make_state)
    p.rtt(OP_map_begin, vt.S)
    j := p.pc()
    p.add(OP_map_if_end)

    /* complex keys */
    if nk <= 0 {
        p.add(OP_map_key)
        self.measureOne(p, sp + 1, vt.K)
    }

    /* complex values */
    if nv <= 0 {
        p.add(OP_map_value)
        self.measureOne(p, sp + 1, vt.V)
    }

    /* move to the next state */
    p.add(OP_map_next)
    p.jmp(OP_goto, j)
    p.pin(j)
    p.add(OP_map_end)
    p.add(OP_drop_state)
    k := p.pc()
    p.add(OP_goto)
    p.pin(i)
    p.pin(k)
}

func (self Compiler) measureStruct(p *Program, sp int, vt *defs.Type) {
    var err error
    var fvs []defs.Field

    /* struct is trivially measuable */
    if nb := defs.GetSize(vt.S); nb > 0 {
        p.i64(OP_size_const, int64(nb))
        return
    }

    /* resolve the field */
    if fvs, err = defs.ResolveFields(vt.S); err != nil {
        panic(err)
    }

    /* empty structs */
    if len(fvs) == 0 {
        p.i64(OP_size_const, 4)
        return
    }

    /* every field requires at least 3 bytes, plus the 1-byte stop field */
    p.tag(sp)
    p.i64(OP_size_const, 4)
    p.i64(OP_seek, int64(fvs[0].F))
    self.measureOne(p, sp + 1, fvs[0].Type)

    /* remaining fields */
    for i, fv := range fvs[1:] {
        p.i64(OP_size_const, 3)
        p.i64(OP_seek, int64(fv.F - fvs[i].F))
        self.measureOne(p, sp + 1, fv.Type)
    }
}

func (self Compiler) measureSetList(p *Program, sp int, vt *defs.Type) {
    et := vt.V
    nb := defs.GetSize(et.S)

    /* 5-byte list or set header */
    p.tag(sp)
    p.i64(OP_size_const, 5)

    /* element is trivially measuable */
    if nb > 0 {
        p.dyn(OP_size_dyn, atm.PtrSize, int64(nb))
        return
    }

    /* complex lists or sets */
    p.add(OP_make_state)
    p.add(OP_list_begin)
    i := p.pc()
    p.add(OP_list_if_end)
    j := p.pc()
    self.measureOne(p, sp + 1, et)
    p.add(OP_list_decr)
    k := p.pc()
    p.add(OP_list_if_end)
    p.i64(OP_seek, int64(et.S.Size()))
    p.jmp(OP_goto, j)
    p.pin(i)
    p.pin(k)
    p.add(OP_drop_state)
}
