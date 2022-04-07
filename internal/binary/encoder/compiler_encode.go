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

package encoder

import (
    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/binary/defs`
)

func (self Compiler) compile(p *Program, sp int, vt *defs.Type) {
    rt := vt.S
    tt := vt.T

    /* check for loops, recursive only on structs with inlining depth limit */
    if tt == defs.T_struct && (self[rt] || sp >= _MAX_STACK || p.pc() >= _MAX_ILBUF) {
        p.rtt(OP_defer, rt)
        return
    }

    /* compile the type recursively */
    self[rt] = true
    self.compileOne(p, sp, vt)
}

func (self Compiler) compileOne(p *Program, sp int, vt *defs.Type) {
    switch vt.T {
        case defs.T_bool    : p.i64(OP_size_check, 1); p.i64(OP_sint, 1)
        case defs.T_i8      : p.i64(OP_size_check, 1); p.i64(OP_sint, 1)
        case defs.T_i16     : p.i64(OP_size_check, 2); p.i64(OP_sint, 2)
        case defs.T_i32     : p.i64(OP_size_check, 4); p.i64(OP_sint, 4)
        case defs.T_i64     : p.i64(OP_size_check, 8); p.i64(OP_sint, 8)
        case defs.T_enum    : p.i64(OP_size_check, 4); p.i64(OP_sint, 4)
        case defs.T_double  : p.i64(OP_size_check, 8); p.i64(OP_sint, 8)
        case defs.T_string  : p.i64(OP_size_check, 4); p.i64(OP_length, abi.PtrSize); p.dyn(OP_memcpy_be, abi.PtrSize, 1)
        case defs.T_binary  : p.i64(OP_size_check, 4); p.i64(OP_length, abi.PtrSize); p.dyn(OP_memcpy_be, abi.PtrSize, 1)
        case defs.T_map     : self.compileMap(p, sp, vt)
        case defs.T_set     : self.compileSeq(p, sp, vt, true)
        case defs.T_list    : self.compileSeq(p, sp, vt, false)
        case defs.T_struct  : self.compileStruct(p, sp, vt)
        case defs.T_pointer : self.compilePtr(p, sp, vt)
        default             : panic("unreachable")
    }
}

func (self Compiler) compilePtr(p *Program, sp int, vt *defs.Type) {
    i := p.pc()
    p.tag(sp)
    p.add(OP_if_nil)
    p.add(OP_make_state)
    p.add(OP_deref)
    self.compile(p, sp + 1, vt.V)
    p.add(OP_drop_state)
    p.pin(i)
}

func (self Compiler) compileMap(p *Program, sp int, vt *defs.Type) {
    kt := vt.K
    et := vt.V

    /* 6-byte map header */
    p.tag(sp)
    p.i64(OP_size_check, 6)
    p.i64(OP_byte, int64(kt.Tag()))
    p.i64(OP_byte, int64(et.Tag()))

    /* check for nil maps */
    i := p.pc()
    p.add(OP_if_nil)

    /* encode the map */
    p.add(OP_map_len)
    j := p.pc()
    p.add(OP_map_if_empty)
    p.add(OP_make_state)
    p.rtt(OP_map_begin, vt.S)
    k := p.pc()
    p.add(OP_map_key)
    self.compile(p, sp + 1, kt)
    p.add(OP_map_value)
    self.compile(p, sp + 1, et)
    p.add(OP_map_next)
    p.jmp(OP_map_if_next, k)
    p.add(OP_drop_state)

    /* encode the length for nil maps */
    r := p.pc()
    p.add(OP_goto)
    p.pin(i)
    p.i64(OP_long, 0)
    p.pin(j)
    p.pin(r)
}

func (self Compiler) compileSeq(p *Program, sp int, vt *defs.Type, verifyUnique bool) {
    nb := -1
    et := vt.V

    /* 5-byte set or list header */
    p.tag(sp)
    p.i64(OP_size_check, 5)
    p.i64(OP_byte, int64(et.Tag()))
    p.i64(OP_length, abi.PtrSize)

    /* check for nil slice */
    i := p.pc()
    p.add(OP_if_nil)

    /* special case of primitive sets or lists */
    switch et.T {
        case defs.T_bool   : nb = 1
        case defs.T_i8     : nb = 1
        case defs.T_i16    : nb = 2
        case defs.T_i32    : nb = 4
        case defs.T_i64    : nb = 8
        case defs.T_double : nb = 8
    }

    /* check for uniqueness if needed */
    if verifyUnique {
        p.rtt(OP_unique, et.S)
    }

    /* check if this is the special case */
    if nb != -1 {
        p.dyn(OP_memcpy_be, abi.PtrSize, int64(nb))
        p.pin(i)
        return
    }

    /* complex sets or lists */
    j := p.pc()
    p.add(OP_list_if_empty)
    p.add(OP_make_state)
    p.add(OP_list_begin)
    k := p.pc()
    p.add(OP_goto)
    r := p.pc()
    p.i64(OP_seek, int64(et.S.Size()))
    p.pin(k)
    self.compile(p, sp + 1, et)
    p.add(OP_list_decr)
    p.jmp(OP_list_if_next, r)
    p.add(OP_drop_state)
    p.pin(i)
    p.pin(j)
}

func (self Compiler) compileStruct(p *Program, sp int, vt *defs.Type) {
    var off int
    var err error
    var fvs []defs.Field

    /* resolve the field */
    if fvs, err = defs.ResolveFields(vt.S); err != nil {
        panic(err)
    }

    /* compile every field */
    for _, fv := range fvs {
        p.tag(sp)
        p.i64(OP_seek, int64(fv.F - off))
        self.compileStructField(p, sp + 1, fv)
        off = fv.F
    }

    /* add the STOP field */
    p.i64(OP_size_check, 1)
    p.i64(OP_byte, 0)
}

func (self Compiler) compileStructField(p *Program, sp int, fv defs.Field) {
    switch fv.Type.T {
        default: {
            panic("fatal: invalid field type: " + fv.Type.String())
        }

        /* non-pointer types */
        case defs.T_bool   : fallthrough
        case defs.T_i8     : fallthrough
        case defs.T_double : fallthrough
        case defs.T_i16    : fallthrough
        case defs.T_i32    : fallthrough
        case defs.T_i64    : fallthrough
        case defs.T_string : fallthrough
        case defs.T_struct : fallthrough
        case defs.T_enum   : fallthrough
        case defs.T_binary : self.compileStructRequired(p, sp, fv)

        /* sequencial types */
        case defs.T_map  : fallthrough
        case defs.T_set  : fallthrough
        case defs.T_list : {
            if fv.Spec != defs.Optional {
                self.compileStructRequired(p, sp, fv)
            } else {
                self.compileStructIterable(p, sp, fv)
            }
        }

        /* pointers */
        case defs.T_pointer: {
            if fv.Spec == defs.Optional {
                self.compileStructOptional(p, sp, fv)
            } else if fv.Type.V.T == defs.T_struct {
                self.compileStructPointer(p, sp, fv)
            } else {
                panic("fatal: non-optional non-struct pointers")
            }
        }
    }
}

func (self Compiler) compileStructPointer(p *Program, sp int, fv defs.Field) {
    i := p.pc()
    p.add(OP_if_nil)
    self.compileStructFieldBegin(p, fv, 4)
    p.add(OP_make_state)
    p.add(OP_deref)
    self.compile(p, sp, fv.Type.V)
    p.add(OP_drop_state)
    j := p.pc()
    p.add(OP_goto)
    p.pin(i)
    self.compileStructFieldBegin(p, fv, 4)
    p.i64(OP_byte, 0)
    p.pin(j)
}

func (self Compiler) compileStructOptional(p *Program, sp int, fv defs.Field) {
    i := p.pc()
    p.add(OP_if_nil)
    self.compileStructFieldBegin(p, fv, 3)
    p.add(OP_make_state)
    p.add(OP_deref)
    self.compile(p, sp, fv.Type.V)
    p.add(OP_drop_state)
    p.pin(i)
}

func (self Compiler) compileStructRequired(p *Program, sp int, fv defs.Field) {
    self.compileStructFieldBegin(p, fv, 3)
    self.compile(p, sp, fv.Type)
}

func (self Compiler) compileStructIterable(p *Program, sp int, fv defs.Field) {
    i := p.pc()
    p.add(OP_if_nil)
    self.compileStructFieldBegin(p, fv, 3)
    self.compile(p, sp, fv.Type)
    p.pin(i)
}

func (self Compiler) compileStructFieldBegin(p *Program, fv defs.Field, nb int64) {
    p.i64(OP_size_check, nb)
    p.i64(OP_byte, int64(fv.Type.Tag()))
    p.i64(OP_word, int64(fv.ID))
}
