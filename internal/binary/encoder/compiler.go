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
    `fmt`
    `reflect`
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/defs`
    `github.com/cloudwego/frugal/internal/rt`
)

type (
    Instr    [2]uint64
    Program  []Instr
    Compiler map[reflect.Type]bool
)

func mkins(op OpCode, to int, iv int64, vt reflect.Type) (v Instr) {
    v[1] = uint64(iv) | mktype(vt)
    v[0] = uint64(op) | uint64(to << 8)
    return
}

func mktype(v reflect.Type) uint64 {
    return uint64(uintptr(unsafe.Pointer(rt.UnpackType(v))))
}

func gettype(v Instr) (p *rt.GoType) {
    *(*uintptr)(unsafe.Pointer(&p)) = uintptr(v[1])
    return
}

func (self Instr) To() int        { return int(self[0]) >> 8 }
func (self Instr) Iv() int64      { return int64(self[1]) }
func (self Instr) Op() OpCode     { return OpCode(self[0] & 0xff) }
func (self Instr) Vt() *rt.GoType { return gettype(self) }

func (self Instr) Disassemble() string {
    switch self.Op() {
        case OP_byte        : return fmt.Sprintf("%-18s0x%02x", self.Op(), self.Iv())
        case OP_word        : return fmt.Sprintf("%-18s0x%04x", self.Op(), self.Iv())
        case OP_long        : return fmt.Sprintf("%-18s0x%08x", self.Op(), self.Iv())
        case OP_size        : fallthrough
        case OP_sint        : fallthrough
        case OP_seek        : fallthrough
        case OP_list_next   : return fmt.Sprintf("%-18s%d", self.Op(), self.Iv())
        case OP_defer       : fallthrough
        case OP_map_begin   : return fmt.Sprintf("%-18s%s", self.Op(), self.Vt())
        case OP_goto        : fallthrough
        case OP_if_nil      : fallthrough
        case OP_map_if_end  : fallthrough
        case OP_list_if_end : return fmt.Sprintf("%-18sL_%d", self.Op(), self.Iv())
        default             : return self.Op().String()
    }
}

func (self Program) pc() int   { return len(self) }
func (self Program) pin(i int) { self[i][0] |= uint64(self.pc() << 8) }

func (self Program) tag(n int) {
    if n >= defs.MaxStack {
        panic("type nesting too deep")
    }
}

func (self *Program) ins(iv Instr)                   { *self = append(*self, iv) }
func (self *Program) add(op OpCode)                  { self.ins(mkins(op, 0, 0, nil)) }
func (self *Program) jmp(op OpCode, to int)          { self.ins(mkins(op, to, 0, nil)) }
func (self *Program) i64(op OpCode, iv int64)        { self.ins(mkins(op, 0, iv, nil)) }
func (self *Program) rtt(op OpCode, vt reflect.Type) { self.ins(mkins(op, 0, 0, vt))   }

func (self Program) Free() {
    freeProgram(self)
}

func (self Program) Disassemble() string {
    nb  := len(self)
    tab := make([]bool, nb + 1)
    ret := make([]string, 0, nb + 1)

    /* prescan to get all the labels */
    for _, ins := range self {
        if _OpBranches[ins.Op()] {
            tab[ins.Iv()] = true
        }
    }

    /* disassemble each instruction */
    for i, ins := range self {
        if !tab[i] {
            ret = append(ret, "\t" + ins.Disassemble())
        } else {
            ret = append(ret, fmt.Sprintf("L_%d:\n\t%s", i, ins.Disassemble()))
        }
    }

    /* add the last label, if needed */
    if tab[nb] {
        ret = append(ret, fmt.Sprintf("L_%d:", nb))
    }

    /* add an "end" indicator, and join all the strings */
    return strings.Join(append(ret, "\tend"), "\n")
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
        self.compilePtr(p, sp, vt.V)
    } else if _, ok := self[vt.S]; !ok {
        self.compileSet(p, sp, vt)
    } else {
        p.rtt(OP_defer, vt.S)
    }
}

func (self Compiler) compileSet(p *Program, sp int, vt *defs.Type) {
    self[vt.S] = true
    self.compileRec(p, sp, vt)
    delete(self, vt.S)
}

func (self Compiler) compileRec(p *Program, sp int, vt *defs.Type) {
    switch vt.T {
        case defs.T_bool   : p.i64(OP_size, 1); p.i64(OP_sint, 1)
        case defs.T_i8     : p.i64(OP_size, 1); p.i64(OP_sint, 1)
        case defs.T_i16    : p.i64(OP_size, 2); p.i64(OP_sint, 2)
        case defs.T_i32    : p.i64(OP_size, 4); p.i64(OP_sint, 4)
        case defs.T_i64    : p.i64(OP_size, 8); p.i64(OP_sint, 8)
        case defs.T_double : p.i64(OP_size, 8); p.i64(OP_sint, 8)
        case defs.T_string : p.i64(OP_size, 4); p.add(OP_vstr)
        case defs.T_binary : p.i64(OP_size, 4); p.add(OP_vstr)
        case defs.T_struct : self.compileStruct  (p, sp, vt)
        case defs.T_map    : self.compileMap     (p, sp, vt)
        case defs.T_set    : self.compileSetList (p, sp, vt.V)
        case defs.T_list   : self.compileSetList (p, sp, vt.V)
        default            : panic("unreachable")
    }
}

func (self Compiler) compilePtr(p *Program, sp int, et *defs.Type) {
    i := p.pc()
    p.add(OP_deref)
    self.compileOne(p, sp, et)
    p.pin(i)
}

func (self Compiler) compileMap(p *Program, sp int, vt *defs.Type) {
    p.tag(sp)
    p.i64(OP_size, 6)
    p.i64(OP_byte, int64(vt.K.Tag()))
    p.i64(OP_byte, int64(vt.V.Tag()))
    p.rtt(OP_map_begin, vt.S)
    i := p.pc()
    p.add(OP_map_if_end)
    p.add(OP_map_key)
    self.compileOne(p, sp + 1, vt.K)
    p.add(OP_map_value)
    self.compileOne(p, sp + 1, vt.V)
    p.add(OP_map_next)
    p.jmp(OP_goto, i)
    p.pin(i)
    p.add(OP_map_end)
}

func (self Compiler) compileSetList(p *Program, sp int, et *defs.Type) {
    p.tag(sp)
    p.i64(OP_size, 5)
    p.i64(OP_byte, int64(et.Tag()))
    p.add(OP_list_begin)
    i := p.pc()
    p.i64(OP_list_if_end, int64(et.S.Size()))
    p.i64(OP_list_next, int64(et.S.Size()))
    self.compileOne(p, sp + 1, et)
    p.add(OP_list_exit)
    p.jmp(OP_goto, i)
    p.pin(i)
    p.add(OP_list_end)
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
        p.i64(OP_byte, 0)
        return
    }

    /* the first field */
    p.tag(sp)
    p.i64(OP_size, 3)
    p.i64(OP_byte, int64(fvs[0].Type.Tag()))
    p.i64(OP_word, int64(fvs[0].ID))
    self.compileOne(p, sp + 1, fvs[0].Type)

    /* remaining fields */
    for i, fv := range fvs[1:] {
        p.i64(OP_size, 3)
        p.i64(OP_seek, int64(fv.F - fvs[i].F))
        p.i64(OP_byte, int64(fv.Type.Tag()))
        p.i64(OP_word, int64(fv.ID))
        self.compileOne(p, sp + 1, fv.Type)
    }

    /* add the STOP field */
    p.i64(OP_size, 1)
    p.i64(OP_byte, 0)
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
    return Optimize(ret), nil
}
