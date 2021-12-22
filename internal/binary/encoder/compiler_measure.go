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

func (self Compiler) measure(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    rt := vt.S
    tt := vt.T

    /* check for loops, recursive only on structs */
    if self[rt] && tt == defs.T_struct {
        p.rtt(OP_size_defer, rt)
        return
    }

    /* measure the type recursively */
    self[rt] = true
    self.measureOne(p, sp, vt, req)
    delete(self, rt)
}

func (self Compiler) measureOne(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    switch vt.T {
        case defs.T_bool    : p.i64(OP_size_const, 1)
        case defs.T_i8      : p.i64(OP_size_const, 1)
        case defs.T_i16     : p.i64(OP_size_const, 2)
        case defs.T_i32     : p.i64(OP_size_const, 4)
        case defs.T_i64     : p.i64(OP_size_const, 8)
        case defs.T_double  : p.i64(OP_size_const, 8)
        case defs.T_string  : p.i64(OP_size_const, 4); p.dyn(OP_size_dyn, atm.PtrSize, 1)
        case defs.T_binary  : p.i64(OP_size_const, 4); p.dyn(OP_size_dyn, atm.PtrSize, 1)
        case defs.T_map     : self.measureMap     (p, sp, vt, req)
        case defs.T_set     : self.measureSetList (p, sp, vt, req)
        case defs.T_list    : self.measureSetList (p, sp, vt, req)
        case defs.T_struct  : self.measureStruct  (p, sp, vt)
        case defs.T_pointer : self.measurePtr     (p, sp, vt, req, 1, self.measure)
        default             : panic("measureOne: unreachable")
    }
}

func (self Compiler) measurePtr(p *Program, sp int, vt *defs.Type, req defs.Requiredness, nilv int64, measure _CompilerAction) {
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
    measure(p, sp + 1, vt.V, req)
    p.add(OP_drop_state)

    /* non-struct pointer or optional field */
    if req == defs.Optional || vt.V.T != defs.T_struct {
        p.pin(i)
        return
    }

    /* structs always need a 0x00 terminator */
    j := p.pc()
    p.add(OP_goto)
    p.pin(i)
    p.i64(OP_size_const, nilv)
    p.pin(j)
}

func (self Compiler) measureMap(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    nk := defs.GetSize(vt.K.S)
    nv := defs.GetSize(vt.V.S)

    /* 6-byte map header */
    if p.tag(sp); req != defs.Optional {
        p.i64(OP_size_const, 6)
    }

    /* check for nil maps */
    i := p.pc()
    p.add(OP_if_nil)

    /* check for optional map header */
    if req == defs.Optional {
        p.i64(OP_size_const, 6)
    }

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
        self.measure(p, sp + 1, vt.K, defs.Default)
    }

    /* complex values */
    if nv <= 0 {
        p.add(OP_map_value)
        self.measure(p, sp + 1, vt.V, defs.Default)
    }

    /* move to the next state */
    p.add(OP_map_next)
    p.jmp(OP_goto, j)
    p.pin(j)
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
    p.i64(OP_size_const, 1)
    p.i64(OP_seek, int64(fvs[0].F))
    self.measureField(p, sp + 1, fvs[0])

    /* remaining fields */
    for i, fv := range fvs[1:] {
        p.i64(OP_seek, int64(fv.F - fvs[i].F))
        self.measureField(p, sp + 1, fv)
    }
}

func (self Compiler) measureField(p *Program, sp int, fv defs.Field) {
    if fv.Type.T == defs.T_pointer {
        self.measureFieldPtr(p, sp, fv)
    } else {
        self.measureFieldStd(p, sp, fv.Type, fv.Spec)
    }
}

func (self Compiler) measureFieldPtr(p *Program, sp int, fv defs.Field) {
    self.measurePtr(p, sp, fv.Type, fv.Spec, 4, self.measureFieldStd)
}

func (self Compiler) measureFieldStd(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    p.i64(OP_size_const, 3)
    self.measure(p, sp, vt, req)
}

func (self Compiler) measureSetList(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    et := vt.V
    nb := defs.GetSize(et.S)

    /* 5-byte list or set header */
    if p.tag(sp); req != defs.Optional {
        p.i64(OP_size_const, 5)
    }

    /* check for nil slice */
    i := p.pc()
    p.add(OP_if_nil)

    /* element is trivially measuable */
    if nb > 0 {
        p.dyn(OP_size_dyn, atm.PtrSize, int64(nb))
        p.pin(i)
        return
    }

    /* check for optional list or set header */
    if req == defs.Optional {
        p.i64(OP_size_const, 5)
    }

    /* complex lists or sets */
    p.add(OP_make_state)
    p.add(OP_list_begin)
    j := p.pc()
    p.add(OP_list_if_end)
    k := p.pc()
    self.measure(p, sp + 1, et, defs.Default)
    p.add(OP_list_decr)
    r := p.pc()
    p.add(OP_list_if_end)
    p.i64(OP_seek, int64(et.S.Size()))
    p.jmp(OP_goto, k)
    p.pin(j)
    p.pin(r)
    p.add(OP_drop_state)
    p.pin(i)
}
