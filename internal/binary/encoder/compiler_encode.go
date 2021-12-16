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

func (self Compiler) compile(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    rt := vt.S
    tt := vt.T

    /* check for loops, recursive only on structs */
    if self[rt] && tt == defs.T_struct {
        p.rtt(OP_defer, rt)
        return
    }

    /* measure the type recursively */
    self[rt] = true
    self.compileOne(p, sp, vt, req)
}

func (self Compiler) compileOne(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    switch vt.T {
        case defs.T_bool    : p.i64(OP_size_check, 1); p.i64(OP_sint, 1)
        case defs.T_i8      : p.i64(OP_size_check, 1); p.i64(OP_sint, 1)
        case defs.T_i16     : p.i64(OP_size_check, 2); p.i64(OP_sint, 2)
        case defs.T_i32     : p.i64(OP_size_check, 4); p.i64(OP_sint, 4)
        case defs.T_i64     : p.i64(OP_size_check, 8); p.i64(OP_sint, 8)
        case defs.T_double  : p.i64(OP_size_check, 8); p.i64(OP_sint, 8)
        case defs.T_string  : p.i64(OP_size_check, 4); p.i64(OP_length, atm.PtrSize); p.dyn(OP_memcpy_be, atm.PtrSize, 1)
        case defs.T_binary  : p.i64(OP_size_check, 4); p.i64(OP_length, atm.PtrSize); p.dyn(OP_memcpy_be, atm.PtrSize, 1)
        case defs.T_map     : self.compileMap     (p, sp, vt, req)
        case defs.T_set     : self.compileSetList (p, sp, vt, req, true)
        case defs.T_list    : self.compileSetList (p, sp, vt, req, false)
        case defs.T_struct  : self.compileStruct  (p, sp, vt)
        case defs.T_pointer : self.compilePtr     (p, sp, vt, req, self.compile, []byte{0})
        default             : panic("unreachable")
    }
}

func (self Compiler) compilePtr(p *Program, sp int, vt *defs.Type, req defs.Requiredness, compile _CompilerAction, nilv []byte) {
    i := p.pc()
    p.add(OP_if_nil)
    p.add(OP_make_state)
    p.add(OP_deref)
    compile(p, sp, vt.V, req)
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
    p.i64(OP_size_check, int64(len(nilv)))
    p.buf(nilv)
    p.pin(j)
}

func (self Compiler) compileMap(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
    kt := vt.K
    et := vt.V

    /* 6-byte map header */
    if p.tag(sp); req != defs.Optional {
        p.i64(OP_size_check, 6)
        p.i64(OP_byte, int64(kt.Tag()))
        p.i64(OP_byte, int64(et.Tag()))
    }

    /* check for nil maps */
    i := p.pc()
    p.add(OP_if_nil)

    /* check for optional map header */
    if req == defs.Optional {
        p.i64(OP_size_check, 6)
        p.i64(OP_byte, int64(kt.Tag()))
        p.i64(OP_byte, int64(et.Tag()))
    }

    /* encode the map */
    p.add(OP_map_len)
    p.add(OP_make_state)
    p.rtt(OP_map_begin, vt.S)
    j := p.pc()
    p.add(OP_map_if_end)
    p.add(OP_map_key)
    self.compile(p, sp + 1, kt, defs.Default)
    p.add(OP_map_value)
    self.compile(p, sp + 1, et, defs.Default)
    p.add(OP_map_next)
    p.jmp(OP_goto, j)
    p.pin(j)
    p.add(OP_map_end)
    p.add(OP_drop_state)

    /* map is optional */
    if req == defs.Optional {
        p.pin(i)
        return
    }

    /* encode the length for nil maps */
    k := p.pc()
    p.add(OP_goto)
    p.pin(i)
    p.i64(OP_long, 0)
    p.pin(k)
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
    p.i64(OP_seek, int64(fvs[0].F))
    self.compileField(p, sp + 1, fvs[0])

    /* remaining fields */
    for i, fv := range fvs[1:] {
        f := fvs[i].F
        p.i64(OP_seek, int64(fv.F - f))
        self.compileField(p, sp + 1, fv)
    }

    /* add the STOP field */
    p.i64(OP_size_check, 1)
    p.i64(OP_byte, 0)
}

func (self Compiler) compileField(p *Program, sp int, fv defs.Field) {
    if fv.Type.T == defs.T_pointer {
        self.compileFieldPtr(p, sp, fv)
    } else {
        self.compileFieldStd(fv.ID)(p, sp, fv.Type, fv.Spec)
    }
}

func (self Compiler) compileFieldPtr(p *Program, sp int, fv defs.Field) {
    self.compilePtr(p, sp, fv.Type, fv.Spec, self.compileFieldStd(fv.ID), []byte {
        byte(fv.Type.Tag()),
        byte(fv.ID >> 8),
        byte(fv.ID),
        0,
    })
}

func (self Compiler) compileFieldStd(id uint16) _CompilerAction {
    return func(p *Program, sp int, vt *defs.Type, req defs.Requiredness) {
        p.i64(OP_size_check, 3)
        p.i64(OP_byte, int64(vt.Tag()))
        p.i64(OP_word, int64(id))
        self.compile(p, sp, vt, req)
    }
}

func (self Compiler) compileSetList(p *Program, sp int, vt *defs.Type, req defs.Requiredness, verifyUnique bool) {
    nb := -1
    et := vt.V
    tt := et.Tag()

    /* 5-byte set or list header */
    if p.tag(sp); req != defs.Optional {
        p.i64(OP_size_check, 5)
        p.i64(OP_byte, int64(tt))
        p.i64(OP_length, atm.PtrSize)
    }

    /* check for nil slice */
    i := p.pc()
    p.add(OP_if_nil)

    /* check for optional list or set header */
    if req == defs.Optional {
        p.i64(OP_size_check, 5)
        p.i64(OP_byte, int64(tt))
        p.i64(OP_length, atm.PtrSize)
    }

    /* special case of primitive sets or lists */
    switch tt {
        case defs.T_bool   : nb = 1
        case defs.T_i8     : nb = 1
        case defs.T_i16    : nb = 2
        case defs.T_i32    : nb = 4
        case defs.T_i64    : nb = 8
        case defs.T_double : nb = 8
    }

    /* check if this is the special case */
    if nb != -1 {
        p.dyn(OP_memcpy_be, atm.PtrSize, int64(nb))
        p.pin(i)
        return
    }

    /* complex sets or lists */
    p.add(OP_make_state)
    p.add(OP_list_begin)
    j := p.pc()
    p.add(OP_list_if_end)
    k := p.pc()
    self.compile(p, sp + 1, et, defs.Default)
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
