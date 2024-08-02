/*
 * Copyright 2022 CloudWeGo Authors
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
	"math"

	"github.com/cloudwego/frugal/internal/defs"
	"github.com/cloudwego/frugal/internal/jit/atm/abi"
)

func (self *Compiler) compile(p *Program, sp int, vt *defs.Type, startpc int) {
	rt := vt.S
	tt := vt.T

	/* only recurse on structs */
	if tt != defs.T_struct {
		self.compileOne(p, sp, vt, startpc)
		return
	}

	/* check for loops */
	if self.t[rt] || !self.o.CanInline(sp, (p.pc()-startpc)*2) {
		p.rtt(OP_defer, rt)
		return
	}

	/* compile the type recursively */
	self.t[rt] = true
	self.compileOne(p, sp, vt, startpc)
	delete(self.t, rt)
}

func (self *Compiler) compileOne(p *Program, sp int, vt *defs.Type, startpc int) {
	switch vt.T {
	case defs.T_bool:
		p.i64(OP_size_check, 1)
		p.i64(OP_sint, 1)
	case defs.T_i8:
		p.i64(OP_size_check, 1)
		p.i64(OP_sint, 1)
	case defs.T_i16:
		p.i64(OP_size_check, 2)
		p.i64(OP_sint, 2)
	case defs.T_i32:
		p.i64(OP_size_check, 4)
		p.i64(OP_sint, 4)
	case defs.T_i64:
		p.i64(OP_size_check, 8)
		p.i64(OP_sint, 8)
	case defs.T_enum:
		p.i64(OP_size_check, 4)
		p.i64(OP_sint, 4)
	case defs.T_double:
		p.i64(OP_size_check, 8)
		p.i64(OP_sint, 8)
	case defs.T_string:
		p.i64(OP_size_check, 4)
		p.i64(OP_length, abi.PtrSize)
		p.dyn(OP_memcpy_be, abi.PtrSize, 1)
	case defs.T_binary:
		p.i64(OP_size_check, 4)
		p.i64(OP_length, abi.PtrSize)
		p.dyn(OP_memcpy_be, abi.PtrSize, 1)
	case defs.T_map:
		self.compileMap(p, sp, vt, startpc)
	case defs.T_set:
		self.compileSeq(p, sp, vt, startpc, true)
	case defs.T_list:
		self.compileSeq(p, sp, vt, startpc, false)
	case defs.T_struct:
		self.compileStruct(p, sp, vt, startpc)
	case defs.T_pointer:
		self.compilePtr(p, sp, vt, startpc)
	default:
		panic("unreachable")
	}
}

func (self *Compiler) compilePtr(p *Program, sp int, vt *defs.Type, startpc int) {
	i := p.pc()
	p.tag(sp)
	p.add(OP_if_nil)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.compile(p, sp+1, vt.V, startpc)
	p.add(OP_drop_state)
	p.pin(i)
}

func (self *Compiler) compileMap(p *Program, sp int, vt *defs.Type, startpc int) {
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
	self.compileItem(p, sp+1, kt, startpc)
	p.add(OP_map_value)
	self.compileItem(p, sp+1, et, startpc)
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

func (self *Compiler) compileSeq(p *Program, sp int, vt *defs.Type, startpc int, verifyUnique bool) {
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
	case defs.T_bool:
		nb = 1
	case defs.T_i8:
		nb = 1
	case defs.T_i16:
		nb = 2
	case defs.T_i32:
		nb = 4
	case defs.T_i64:
		nb = 8
	case defs.T_double:
		nb = 8
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
	self.compileItem(p, sp+1, et, startpc)
	p.add(OP_list_decr)
	p.jmp(OP_list_if_next, r)
	p.add(OP_drop_state)
	p.pin(i)
	p.pin(j)
}

func (self *Compiler) compileItem(p *Program, sp int, vt *defs.Type, startpc int) {
	tag := vt.T
	elem := vt.V

	/* special handling for pointers */
	if tag != defs.T_pointer {
		self.compile(p, sp, vt, startpc)
		return
	}

	/* must be pointer struct at this point */
	if elem.T != defs.T_struct {
		panic("fatal: non-struct pointers within container elements")
	}

	/* always add the STOP field for structs */
	i := p.pc()
	p.tag(sp)
	p.add(OP_if_nil)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.compile(p, sp+1, elem, startpc)
	p.add(OP_drop_state)
	j := p.pc()
	p.add(OP_goto)
	p.pin(i)
	p.i64(OP_size_check, 1)
	p.i64(OP_byte, 0)
	p.pin(j)
}

func (self *Compiler) compileStruct(p *Program, sp int, vt *defs.Type, startpc int) {
	var err error
	var fvs []defs.Field

	/* resolve the field */
	if fvs, err = defs.ResolveFields(vt.S); err != nil {
		panic(err)
	}

	/* compile every field */
	for _, fv := range fvs {
		p.tag(sp)
		p.i64(OP_seek, int64(fv.F))
		self.compileStructField(p, sp+1, fv, startpc)
		p.i64(OP_seek, -int64(fv.F))
	}

	/* add the STOP field */
	p.i64(OP_size_check, 1)
	p.i64(OP_byte, 0)
}

func (self *Compiler) compileStructField(p *Program, sp int, fv defs.Field, startpc int) {
	switch fv.Type.T {
	default:
		{
			panic("fatal: invalid field type: " + fv.Type.String())
		}

	/* non-pointer types */
	case defs.T_bool:
		fallthrough
	case defs.T_i8:
		fallthrough
	case defs.T_double:
		fallthrough
	case defs.T_i16:
		fallthrough
	case defs.T_i32:
		fallthrough
	case defs.T_i64:
		fallthrough
	case defs.T_string:
		fallthrough
	case defs.T_enum:
		fallthrough
	case defs.T_binary:
		{
			if fv.Default.IsValid() && fv.Spec == defs.Optional {
				self.compileStructDefault(p, sp, fv, startpc)
			} else {
				self.compileStructRequired(p, sp, fv, startpc)
			}
		}

	/* struct types, only available in hand-written structs */
	case defs.T_struct:
		{
			self.compileStructRequired(p, sp, fv, startpc)
		}

	/* sequential types */
	case defs.T_map:
		fallthrough
	case defs.T_set:
		fallthrough
	case defs.T_list:
		{
			if fv.Spec == defs.Optional {
				self.compileStructIterable(p, sp, fv, startpc)
			} else {
				self.compileStructRequired(p, sp, fv, startpc)
			}
		}

	/* pointers */
	case defs.T_pointer:
		{
			if fv.Spec == defs.Optional {
				self.compileStructOptional(p, sp, fv, startpc)
			} else if fv.Type.V.T == defs.T_struct {
				self.compileStructPointer(p, sp, fv, startpc)
			} else {
				panic("fatal: non-optional non-struct pointers")
			}
		}
	}
}

func (self *Compiler) compileStructDefault(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	t := fv.Type.T

	/* check for default values */
	switch t {
	case defs.T_bool:
		p.dyn(OP_if_eq_imm, 1, bool2i64(fv.Default.Bool()))
	case defs.T_i8:
		p.dyn(OP_if_eq_imm, 1, fv.Default.Int())
	case defs.T_double:
		p.dyn(OP_if_eq_imm, 8, int64(math.Float64bits(fv.Default.Float())))
	case defs.T_i16:
		p.dyn(OP_if_eq_imm, 2, fv.Default.Int())
	case defs.T_i32:
		p.dyn(OP_if_eq_imm, 4, fv.Default.Int())
	case defs.T_i64:
		p.dyn(OP_if_eq_imm, 8, fv.Default.Int())
	case defs.T_string:
		p.str(OP_if_eq_str, fv.Default.String())
	case defs.T_enum:
		p.dyn(OP_if_eq_imm, 4, fv.Default.Int())
	case defs.T_binary:
		p.str(OP_if_eq_str, mem2str(fv.Default.Bytes()))
	default:
		panic("unreachable")
	}

	/* compile if it's not the default value */
	self.compileStructFieldBegin(p, fv, 3)
	self.compile(p, sp, fv.Type, startpc)
	p.pin(i)
}

func (self *Compiler) compileStructPointer(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	p.add(OP_if_nil)
	self.compileStructFieldBegin(p, fv, 4)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.compile(p, sp+1, fv.Type.V, startpc)
	p.add(OP_drop_state)
	j := p.pc()
	p.add(OP_goto)
	p.pin(i)
	self.compileStructFieldBegin(p, fv, 4)
	p.i64(OP_byte, 0)
	p.pin(j)
}

func (self *Compiler) compileStructIterable(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	p.add(OP_if_nil)
	self.compileStructFieldBegin(p, fv, 3)
	self.compile(p, sp, fv.Type, startpc)
	p.pin(i)
}

func (self *Compiler) compileStructOptional(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	p.add(OP_if_nil)
	self.compileStructFieldBegin(p, fv, 3)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.compile(p, sp+1, fv.Type.V, startpc)
	p.add(OP_drop_state)
	p.pin(i)
}

func (self *Compiler) compileStructRequired(p *Program, sp int, fv defs.Field, startpc int) {
	self.compileStructFieldBegin(p, fv, 3)
	self.compile(p, sp, fv.Type, startpc)
}

func (self *Compiler) compileStructFieldBegin(p *Program, fv defs.Field, nb int64) {
	p.i64(OP_size_check, nb)
	p.i64(OP_byte, int64(fv.Type.Tag()))
	p.i64(OP_word, int64(fv.ID))
}
