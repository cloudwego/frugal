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

package decoder

import (
    `reflect`
    `unsafe`

    `github.com/cloudwego/frugal/internal/binary/defs`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/cloudwego/frugal/internal/utils`
)

type Instr struct {
    Op OpCode
    Tx defs.Tag
    Id uint16
    To int32
    Iv int64
    Sw *int
    Vt *rt.GoType
}

func mkins(op OpCode, dt defs.Tag, id uint16, to int, iv int64, sw []int, vt reflect.Type) Instr {
    return Instr {
        Op: op,
        Tx: dt,
        Id: id,
        To: int32(to),
        Vt: rt.UnpackType(vt),
        Iv: int64(len(sw)) | iv,
        Sw: (*int)((*rt.GoSlice)(unsafe.Pointer(&sw)).Ptr),
    }
}

type (
    Program  []Instr
    Compiler map[reflect.Type]bool
)

func (self Program) pc() int {
    return len(self)
}

func (self Program) pin(i int) {
    self[i].Iv = int64(self.pc())
}

func (self Program) def(n int) {
    if n >= defs.MaxStack {
        panic("type nesting too deep")
    }
}

func (self *Program) ins(iv Instr)                       { *self = append(*self, iv) }
func (self *Program) add(op OpCode)                      { self.ins(mkins(op, 0, 0, 0, 0, nil, nil)) }
func (self *Program) jmp(op OpCode, to int)              { self.ins(mkins(op, 0, 0, to, 0, nil, nil)) }
func (self *Program) i64(op OpCode, iv int64)            { self.ins(mkins(op, 0, 0, 0, iv, nil, nil)) }
func (self *Program) tab(op OpCode, tv []int)            { self.ins(mkins(op, 0, 0, 0, 0, tv, nil)) }
func (self *Program) tag(op OpCode, vt defs.Tag)         { self.ins(mkins(op, vt, 0, 0, 0, nil, nil)) }
func (self *Program) rtt(op OpCode, vt reflect.Type)     { self.ins(mkins(op, 0, 0, 0, 0, nil, vt)) }
func (self *Program) jcc(op OpCode, vt defs.Tag, to int) { self.ins(mkins(op, vt, 0, to, 0, nil, nil)) }

func (self Program) Free() {
    freeProgram(self)
}

func CreateCompiler() Compiler {
    return newCompiler()
}

func (self Compiler) rescue(ep *error) {
    if val := recover(); val != nil {
        if err, ok := val.(error); ok {
            *ep = err
        } else {
            panic(val)
        }
    }
}

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
        case defs.T_bool   : p.i64(OP_size, 1); p.i64(OP_int, 1)
        case defs.T_i8     : p.i64(OP_size, 1); p.i64(OP_int, 1)
        case defs.T_i16    : p.i64(OP_size, 2); p.i64(OP_int, 2)
        case defs.T_i32    : p.i64(OP_size, 4); p.i64(OP_int, 4)
        case defs.T_i64    : p.i64(OP_size, 8); p.i64(OP_int, 8)
        case defs.T_double : p.i64(OP_size, 8); p.i64(OP_int, 8)
        case defs.T_string : p.i64(OP_size, 4); p.add(OP_str)
        case defs.T_binary : p.i64(OP_size, 4); p.add(OP_bin)
        case defs.T_struct : self.compileStruct  (p, sp, vt)
        case defs.T_map    : self.compileMap     (p, sp, vt)
        case defs.T_set    : self.compileSetList (p, sp, vt.V)
        case defs.T_list   : self.compileSetList (p, sp, vt.V)
        default            : panic("unreachable")
    }
}

func (self Compiler) compilePtr(p *Program, sp int, vt *defs.Type) {
    p.add(OP_make_state)
    p.rtt(OP_deref, vt.S)
    self.compileOne(p, sp, vt.V)
    p.add(OP_drop_state)
}

func (self Compiler) compileMap(p *Program, sp int, vt *defs.Type) {
    p.def(sp)
    p.i64(OP_size, 6)
    p.tag(OP_type, vt.K.Tag())
    p.tag(OP_type, vt.V.Tag())
    p.add(OP_make_state)
    p.rtt(OP_map_begin, vt.S)
    i := p.pc()
    p.add(OP_map_is_done)
    self.compileKey(p, sp + 1, vt)
    self.compileOne(p, sp + 1, vt.V)
    p.add(OP_map_next)
    p.jmp(OP_goto, i)
    p.pin(i)
    p.add(OP_drop_state)
}

func (self Compiler) compileKey(p *Program, sp int, vt *defs.Type) {
    switch vt.K.Tag() {
        case defs.T_bool    : p.rtt(OP_map_set_bool, vt.V.S)
        case defs.T_i8      : p.rtt(OP_map_set_i8, vt.V.S)
        case defs.T_double  : p.rtt(OP_map_set_double, vt.V.S)
        case defs.T_i16     : p.rtt(OP_map_set_i16, vt.V.S)
        case defs.T_i32     : p.rtt(OP_map_set_i32, vt.V.S)
        case defs.T_i64     : p.rtt(OP_map_set_i64, vt.V.S)
        case defs.T_string  : p.rtt(OP_map_set_str, vt.V.S)
        case defs.T_pointer : self.compileKeyPtr(p, sp, vt)
        default             : panic("unreachable")
    }
}

func (self Compiler) compileKeyPtr(p *Program, sp int, vt *defs.Type) {
    pt := vt.K
    st := pt.V

    /* must be a struct */
    if st.Tag() != defs.T_struct {
        panic("map key cannot be non-struct pointers")
    }

    /* construct a new object */
    p.rtt(OP_construct, st.S)
    self.compileOne(p, sp, st)
    p.add(OP_map_set_pointer)
}

func (self Compiler) compileStruct(p *Program, sp int, vt *defs.Type) {
    var fid int
    var rid int
    var err error
    var req []int
    var fvs []defs.Field

    /* resolve the fields */
    if fvs, err = defs.ResolveFields(vt.S); err != nil {
        panic(err)
    }

    /* find the maximum field IDs */
    for _, fv := range fvs {
        id := int(fv.ID)
        fid = utils.MaxInt(fid, id)

        /* required fields */
        if fv.Spec == defs.Required {
            req = append(req, int(fv.ID))
            rid = utils.MaxInt(rid, int(fv.ID))
        }
    }

    /* start parsing the struct */
    p.def(sp)
    p.i64(OP_struct_begin, int64(rid))

    /* switch jump buffer */
    i := p.pc()
    s := make([]int, fid + 1)

    /* check for stop field */
    p.i64(OP_size, 1)
    p.add(OP_struct_read_tag)
    j := p.pc()
    p.add(OP_struct_is_stop)
    p.i64(OP_size, 2)
    p.tab(OP_struct_switch, s)
    k := p.pc()
    p.add(OP_struct_skip)
    p.jmp(OP_goto, i)

    /* assemble every field */
    for _, fv := range fvs {
        s[fv.ID] = p.pc()
        p.jcc(OP_struct_check_type, fv.Type.Tag(), k)
        p.i64(OP_index, int64(fv.F))
        self.compileOne(p, sp + 1, fv.Type)
        p.i64(OP_index, -int64(fv.F))
        p.jmp(OP_goto, i)
    }

    /* check all the required fields */
    p.pin(j)
    p.tab(OP_struct_require, req)
}

func (self Compiler) compileSetList(p *Program, sp int, et *defs.Type) {
    p.def(sp)
    p.i64(OP_size, 5)
    p.tag(OP_type, et.Tag())
    p.add(OP_make_state)
    p.rtt(OP_list_begin, et.S)
    i := p.pc()
    p.add(OP_list_is_done)
    j := p.pc()
    self.compileOne(p, sp + 1, et)
    p.rtt(OP_list_next, et.S)
    k := p.pc()
    p.add(OP_list_is_done)
    p.i64(OP_index, int64(et.S.Size()))
    p.jmp(OP_goto, j)
    p.pin(i)
    p.pin(k)
    p.add(OP_drop_state)
}

func (self Compiler) Free() {
    freeCompiler(self)
}

func (self Compiler) Compile(vt reflect.Type) (ret Program, err error) {
    ret = newProgram()
    vtp := defs.ParseType(vt, "")

    /* catch the exceptions, and free the type */
    defer self.rescue(&err)
    defer vtp.Free()

    /* compile the actual type */
    self.compileOne(&ret, 0, vtp)
    ret.add(OP_halt)
    return ret, nil
}
