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
    Instr    uint64
    Program  []Instr
    Compiler map[reflect.Type]bool
)

func mkins(op OpCode, et defs.Tag, kt defs.Tag, id uint16, to int, nb int, vt reflect.Type) Instr {
    return Instr(
        (uint64(id) << 48) |
        (uint64(et) << 24) |
        (uint64(kt) << 16) |
        (uint64(to) <<  8) |
        (mkoffs(nb) <<  8) |
        (mktype(vt) <<  8) |
        (uint64(op) <<  0),
    )
}

func mkoffs(v int) uint64 {
    if v < defs.MinInt56 || v > defs.MaxInt56 {
        panic("value exceeds 56-bit integer range")
    } else {
        return uint64(v)
    }
}

func mktype(v reflect.Type) uint64 {
    if p := uintptr(unsafe.Pointer(rt.UnpackType(v))); p > defs.MaxUint56 {
        panic("pointer exceeds 56-bit address space")
    } else {
        return uint64(p)
    }
}

func gettype(v Instr) (p *rt.GoType) {
    *(*uintptr)(unsafe.Pointer(&p)) = uintptr(v)
    return
}

func (self Instr) Op() OpCode     { return OpCode(self) }
func (self Instr) To() int        { return int(self >> 8) }
func (self Instr) Nb() int        { return int(self) / 256 }
func (self Instr) Et() uint8      { return uint8(self >> 24) }
func (self Instr) Kt() uint8      { return uint8(self >> 16) }
func (self Instr) Id() uint16     { return uint16(self >> 48) }
func (self Instr) Vt() *rt.GoType { return gettype(self >> 8) }

func (self Instr) Disassemble() string {
    switch self.Op() {
        case OP_goto           : fallthrough
        case OP_follow         : fallthrough
        case OP_map_check_key  : fallthrough
        case OP_list_advance   : return fmt.Sprintf("%-18sL_%d", self.Op(), self.To())
        case OP_defer          : return fmt.Sprintf("%-18s%s", self.Op(), self.Vt())
        case OP_offset         : return fmt.Sprintf("%-18s%d", self.Op(), self.Nb())
        case OP_list_begin     : return fmt.Sprintf("%-18s%d", self.Op(), self.Et())
        case OP_field_begin    : return fmt.Sprintf("%-18s%d: %d", self.Op(), self.Id(), self.Et())
        case OP_map_begin      : return fmt.Sprintf("%-18s%d -> %d", self.Op(), self.Kt(), self.Et())
        default                : return self.Op().String()
    }
}

func (self *Program) pc() int   { return len(*self) }
func (self *Program) pin(i int) { (*self)[i] = ((*self)[i] & 0xff) | Instr(uint64(self.pc()) << 8) }

func (self *Program) add(op OpCode)                            { self.ins(mkins(op, 0, 0, 0, 0, 0, nil)) }
func (self *Program) jmp(op OpCode, to int)                    { self.ins(mkins(op, 0, 0, 0, to, 0, nil)) }
func (self *Program) idx(op OpCode, nb int)                    { self.ins(mkins(op, 0, 0, 0, 0, nb, nil)) }
func (self *Program) seq(op OpCode, et defs.Tag)               { self.ins(mkins(op, et, 0, 0, 0, 0, nil)) }
func (self *Program) rtt(op OpCode, vt reflect.Type)           { self.ins(mkins(op, 0, 0, 0, 0, 0, vt))   }
func (self *Program) fid(op OpCode, et defs.Tag, id uint16)    { self.ins(mkins(op, et, 0, id, 0, 0, nil)) }
func (self *Program) kvs(op OpCode, kt defs.Tag, et defs.Tag)  { self.ins(mkins(op, et, kt, 0, 0, 0, nil)) }

func (self *Program) tag(n int) {
    if n >= defs.MaxStack {
        panic("type nesting too deep")
    }
}

func (self *Program) ins(iv Instr) {
    if len(*self) >= defs.MaxUint56 {
        panic("program too long")
    } else {
        *self = append(*self, iv)
    }
}

func (self *Program) Disassemble() string {
    nb  := len(*self)
    tab := make([]bool, nb + 1)
    ret := make([]string, 0, nb + 1)

    /* prescan to get all the labels */
    for _, ins := range *self {
        if _OpBranches[ins.Op()] {
            tab[ins.To()] = true
        }
    }

    /* disassemble each instruction */
    for i, ins := range *self {
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
        case defs.T_bool   : p.add(OP_bool)
        case defs.T_i8     : p.add(OP_i8)
        case defs.T_double : p.add(OP_double)
        case defs.T_i16    : p.add(OP_i16)
        case defs.T_i32    : p.add(OP_i32)
        case defs.T_i64    : p.add(OP_i64)
        case defs.T_string : p.add(OP_binary)
        case defs.T_binary : p.add(OP_binary)
        case defs.T_struct : self.compileStruct  (p, sp, vt)
        case defs.T_map    : self.compileMap     (p, sp, vt.V, vt.K)
        case defs.T_set    : self.compileSetList (p, sp, vt.V)
        case defs.T_list   : self.compileSetList (p, sp, vt.V)
        default            : panic("unreachable")
    }
}

func (self Compiler) compilePtr(p *Program, sp int, et *defs.Type) {
    i := p.pc()
    p.add(OP_follow)
    self.compileOne(p, sp, et)
    p.pin(i)
}

func (self Compiler) compileMap(p *Program, sp int, et *defs.Type, kt *defs.Type) {
    p.tag(sp)
    p.add(OP_save)
    p.kvs(OP_map_begin, kt.T, et.T)
    i := p.pc()
    p.add(OP_map_check_key)
    self.compileOne(p, sp + 1, kt)
    p.add(OP_map_value_next)
    self.compileOne(p, sp + 1, et)
    p.jmp(OP_goto, i)
    p.pin(i)
    p.add(OP_drop)
}

func (self Compiler) compileSetList(p *Program, sp int, et *defs.Type) {
    p.tag(sp)
    p.add(OP_save)
    p.seq(OP_list_begin, et.T)
    i := p.pc()
    p.add(OP_list_advance)
    self.compileOne(p, sp + 1, et)
    p.jmp(OP_goto, i)
    p.pin(i)
    p.add(OP_drop)
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
        p.add(OP_field_stop)
        return
    }

    /* the first field */
    p.tag(sp)
    p.add(OP_save)
    p.fid(OP_field_begin, fvs[0].Type.T, fvs[0].ID)
    self.compileOne(p, sp + 1, fvs[0].Type)

    /* remaining fields */
    for i, fv := range fvs[1:] {
        p.idx(OP_offset, fv.F - fvs[i].F)
        p.fid(OP_field_begin, fv.Type.T, fv.ID)
        self.compileOne(p, sp + 1, fv.Type)
    }

    /* add the STOP field */
    p.add(OP_field_stop)
    p.add(OP_drop)
}

func (self Compiler) Compile(vt reflect.Type) (ret *Program, err error) {
    ret = new(Program)
    vtp := defs.ParseType(vt, "")

    /* catch the exceptions, and free the type */
    defer self.rescue(&err)
    defer vtp.Free()

    /* compile the actual type */
    self.compileOne(ret, 0, vtp)
    return
}
