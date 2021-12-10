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

func (self Compiler) compileOne(p *Program, sp int, vt *defs.Type) {
    if vt.T == defs.T_pointer {
        self.compilePtr(p, sp, vt)
    } else if _, ok := self[vt.S]; !ok {
        self.compileTag(p, sp, vt)
    } else {
        p.rtt(OP_defer, vt.S)
    }
}

func (self Compiler) compileTag(p *Program, sp int, vt *defs.Type) {
    self[vt.S] = true
    self.compileRec(p, sp, vt)
    delete(self, vt.S)
}

func (self Compiler) compileRec(p *Program, sp int, vt *defs.Type) {
    switch vt.T {
        case defs.T_bool   : p.i64(OP_size_check, 1); p.i64(OP_sint, 1)
        case defs.T_i8     : p.i64(OP_size_check, 1); p.i64(OP_sint, 1)
        case defs.T_i16    : p.i64(OP_size_check, 2); p.i64(OP_sint, 2)
        case defs.T_i32    : p.i64(OP_size_check, 4); p.i64(OP_sint, 4)
        case defs.T_i64    : p.i64(OP_size_check, 8); p.i64(OP_sint, 8)
        case defs.T_double : p.i64(OP_size_check, 8); p.i64(OP_sint, 8)
        case defs.T_string : p.i64(OP_size_check, 4); p.i64(OP_length, atm.PtrSize); p.dyn(OP_memcpy_be, atm.PtrSize, 1)
        case defs.T_binary : p.i64(OP_size_check, 4); p.i64(OP_length, atm.PtrSize); p.dyn(OP_memcpy_be, atm.PtrSize, 1)
        case defs.T_struct : self.compileStruct  (p, sp, vt)
        case defs.T_map    : self.compileMap     (p, sp, vt)
        case defs.T_set    : self.compileSetList (p, sp, vt.V)
        case defs.T_list   : self.compileSetList (p, sp, vt.V)
        default            : panic("unreachable")
    }
}

func (self Compiler) compilePtr(p *Program, sp int, vt *defs.Type) {
    i := p.pc()
    p.add(OP_if_nil)
    p.add(OP_make_state)
    p.add(OP_deref)
    self.compileOne(p, sp, vt.V)
    p.add(OP_drop_state)
    p.pin(i)
}

func (self Compiler) compileMap(p *Program, sp int, vt *defs.Type) {
    p.tag(sp)
    p.i64(OP_size_check, 6)
    p.i64(OP_byte, int64(vt.K.Tag()))
    p.i64(OP_byte, int64(vt.V.Tag()))
    i := p.pc()
    p.add(OP_if_nil)
    p.add(OP_map_len)
    p.add(OP_make_state)
    p.rtt(OP_map_begin, vt.S)
    j := p.pc()
    p.add(OP_map_if_end)
    p.add(OP_map_key)
    self.compileOne(p, sp + 1, vt.K)
    p.add(OP_map_value)
    self.compileOne(p, sp + 1, vt.V)
    p.add(OP_map_next)
    p.jmp(OP_goto, j)
    p.pin(j)
    p.add(OP_map_end)
    p.add(OP_drop_state)
    k := p.pc()
    p.add(OP_goto)
    p.pin(i)
    p.i64(OP_long, 0)
    p.pin(k)
}

func (self Compiler) compileSetList(p *Program, sp int, et *defs.Type) {
    nb := -1
    tt := et.Tag()

    /* special case of primitive sets or lists */
    switch tt {
        case defs.T_bool   : nb = 1
        case defs.T_i8     : nb = 1
        case defs.T_i16    : nb = 2
        case defs.T_i32    : nb = 4
        case defs.T_i64    : nb = 8
        case defs.T_double : nb = 8
    }

    /* set or list header */
    p.i64(OP_size_check, 5)
    p.i64(OP_byte, int64(tt))
    p.i64(OP_length, atm.PtrSize)

    /* check if this is the special case */
    if nb != -1 {
        p.dyn(OP_memcpy_be, atm.PtrSize, int64(nb))
        return
    }

    /* complex sets or lists */
    p.tag(sp)
    p.add(OP_make_state)
    p.add(OP_list_begin)
    i := p.pc()
    p.add(OP_list_if_end)
    j := p.pc()
    self.compileOne(p, sp + 1, et)
    p.add(OP_list_decr)
    k := p.pc()
    p.add(OP_list_if_end)
    p.i64(OP_seek, int64(et.S.Size()))
    p.jmp(OP_goto, j)
    p.pin(i)
    p.pin(k)
    p.add(OP_drop_state)
}

func (self Compiler) compileStruct(p *Program, sp int, vt *defs.Type) {
    var err error
    var fvs []defs.Field

    /* resolve the field */
    if fvs, err = defs.ResolveFields(vt.S); err != nil {
        panic(err)
    }

    /* empty structs */
    if len(fvs) == 0 {
        p.i64(OP_size_check, 1)
        p.i64(OP_byte, 0)
        return
    }

    /* every field requires at least 3 bytes */
    p.tag(sp)
    p.i64(OP_size_check, 3)
    p.i64(OP_seek, int64(fvs[0].F))
    p.i64(OP_byte, int64(fvs[0].Type.Tag()))
    p.i64(OP_word, int64(fvs[0].ID))
    self.compileOne(p, sp + 1, fvs[0].Type)

    /* remaining fields */
    for i, fv := range fvs[1:] {
        f := fvs[i].F
        p.i64(OP_size_check, 3)
        p.i64(OP_seek, int64(fv.F - f))
        p.i64(OP_byte, int64(fv.Type.Tag()))
        p.i64(OP_word, int64(fv.ID))
        self.compileOne(p, sp + 1, fv.Type)
    }

    /* add the STOP field */
    p.i64(OP_size_check, 1)
    p.i64(OP_byte, 0)
}
