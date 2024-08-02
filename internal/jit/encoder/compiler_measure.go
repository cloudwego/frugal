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

func (self *Compiler) measure(p *Program, sp int, vt *defs.Type, startpc int) {
	rt := vt.S
	tt := vt.T

	/* only recurse on structs */
	if tt != defs.T_struct {
		self.measureOne(p, sp, vt, startpc)
		return
	}

	/* check for loops with inlining depth limit */
	if self.t[rt] || !self.o.CanInline(sp, (p.pc()-startpc)*2) {
		p.rtt(OP_size_defer, rt)
		return
	}

	/* measure the type recursively */
	self.t[rt] = true
	self.measureOne(p, sp, vt, startpc)
	delete(self.t, rt)
}

func (self *Compiler) measureOne(p *Program, sp int, vt *defs.Type, startpc int) {
	switch vt.T {
	case defs.T_bool:
		p.i64(OP_size_const, 1)
	case defs.T_i8:
		p.i64(OP_size_const, 1)
	case defs.T_i16:
		p.i64(OP_size_const, 2)
	case defs.T_i32:
		p.i64(OP_size_const, 4)
	case defs.T_i64:
		p.i64(OP_size_const, 8)
	case defs.T_enum:
		p.i64(OP_size_const, 4)
	case defs.T_double:
		p.i64(OP_size_const, 8)
	case defs.T_string:
		p.i64(OP_size_const, 4)
		p.dyn(OP_size_dyn, abi.PtrSize, 1)
	case defs.T_binary:
		p.i64(OP_size_const, 4)
		p.dyn(OP_size_dyn, abi.PtrSize, 1)
	case defs.T_map:
		self.measureMap(p, sp, vt, startpc)
	case defs.T_set:
		self.measureSeq(p, sp, vt, startpc)
	case defs.T_list:
		self.measureSeq(p, sp, vt, startpc)
	case defs.T_struct:
		self.measureStruct(p, sp, vt, startpc)
	case defs.T_pointer:
		self.measurePtr(p, sp, vt, startpc)
	default:
		panic("measureOne: unreachable")
	}
}

func (self *Compiler) measurePtr(p *Program, sp int, vt *defs.Type, startpc int) {
	i := p.pc()
	p.tag(sp)
	p.add(OP_if_nil)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.measure(p, sp+1, vt.V, startpc)
	p.add(OP_drop_state)
	p.pin(i)
}

func (self *Compiler) measureMap(p *Program, sp int, vt *defs.Type, startpc int) {
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
		p.i64(OP_size_map, int64(nk+nv))
		p.pin(i)
		return
	}

	/* key or value is trivially measuable */
	if nk > 0 {
		p.i64(OP_size_map, int64(nk))
	}
	if nv > 0 {
		p.i64(OP_size_map, int64(nv))
	}

	/* complex maps */
	j := p.pc()
	p.add(OP_map_if_empty)
	p.add(OP_make_state)
	p.rtt(OP_map_begin, vt.S)
	k := p.pc()

	/* complex keys */
	if nk <= 0 {
		p.add(OP_map_key)
		self.measureItem(p, sp+1, vt.K, startpc)
	}

	/* complex values */
	if nv <= 0 {
		p.add(OP_map_value)
		self.measureItem(p, sp+1, vt.V, startpc)
	}

	/* move to the next state */
	p.add(OP_map_next)
	p.jmp(OP_map_if_next, k)
	p.add(OP_drop_state)
	p.pin(i)
	p.pin(j)
}

func (self *Compiler) measureSeq(p *Program, sp int, vt *defs.Type, startpc int) {
	et := vt.V
	nb := defs.GetSize(et.S)

	/* 5-byte list or set header */
	p.tag(sp)
	p.i64(OP_size_const, 5)

	/* check for nil slice */
	i := p.pc()
	p.add(OP_if_nil)

	/* element is trivially measuable */
	if nb > 0 {
		p.dyn(OP_size_dyn, abi.PtrSize, int64(nb))
		p.pin(i)
		return
	}

	/* complex lists or sets */
	j := p.pc()
	p.add(OP_list_if_empty)
	p.add(OP_make_state)
	p.add(OP_list_begin)
	k := p.pc()
	p.add(OP_goto)
	r := p.pc()
	p.i64(OP_seek, int64(et.S.Size()))
	p.pin(k)
	self.measureItem(p, sp+1, et, startpc)
	p.add(OP_list_decr)
	p.jmp(OP_list_if_next, r)
	p.add(OP_drop_state)
	p.pin(i)
	p.pin(j)
}

func (self *Compiler) measureItem(p *Program, sp int, vt *defs.Type, startpc int) {
	tag := vt.T
	elem := vt.V

	/* special handling for pointers */
	if tag != defs.T_pointer {
		self.measure(p, sp, vt, startpc)
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
	self.measure(p, sp+1, elem, startpc)
	p.add(OP_drop_state)
	j := p.pc()
	p.add(OP_goto)
	p.pin(i)
	p.i64(OP_size_const, 1)
	p.pin(j)
}

func (self *Compiler) measureStruct(p *Program, sp int, vt *defs.Type, startpc int) {
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

	/* 1-byte stop field */
	p.tag(sp)
	p.i64(OP_size_const, 1)

	/* measure every field */
	for _, fv := range fvs {
		p.i64(OP_seek, int64(fv.F))
		self.measureField(p, sp+1, fv, startpc)
		p.i64(OP_seek, -int64(fv.F))
	}
}

func (self *Compiler) measureField(p *Program, sp int, fv defs.Field, startpc int) {
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
				self.measureStructDefault(p, sp, fv, startpc)
			} else {
				self.measureStructRequired(p, sp, fv, startpc)
			}
		}

	/* struct types, only available in hand-written structs */
	case defs.T_struct:
		{
			self.measureStructRequired(p, sp, fv, startpc)
		}

	/* sequential types */
	case defs.T_map:
		fallthrough
	case defs.T_set:
		fallthrough
	case defs.T_list:
		{
			if fv.Spec == defs.Optional {
				self.measureStructIterable(p, sp, fv, startpc)
			} else {
				self.measureStructRequired(p, sp, fv, startpc)
			}
		}

	/* pointers */
	case defs.T_pointer:
		{
			if fv.Spec == defs.Optional {
				self.measureStructOptional(p, sp, fv, startpc)
			} else if fv.Type.V.T == defs.T_struct {
				self.measureStructPointer(p, sp, fv, startpc)
			} else {
				panic("fatal: non-optional non-struct pointers")
			}
		}
	}
}

func (self *Compiler) measureStructDefault(p *Program, sp int, fv defs.Field, startpc int) {
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

	/* measure if it's not the default value */
	p.i64(OP_size_const, 3)
	self.measure(p, sp, fv.Type, startpc)
	p.pin(i)
}

func (self *Compiler) measureStructPointer(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	p.add(OP_if_nil)
	p.i64(OP_size_const, 3)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.measure(p, sp+1, fv.Type.V, startpc)
	p.add(OP_drop_state)
	j := p.pc()
	p.add(OP_goto)
	p.pin(i)
	p.i64(OP_size_const, 4)
	p.pin(j)
}

func (self *Compiler) measureStructIterable(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	p.add(OP_if_nil)
	p.i64(OP_size_const, 3)
	self.measure(p, sp, fv.Type, startpc)
	p.pin(i)
}

func (self *Compiler) measureStructOptional(p *Program, sp int, fv defs.Field, startpc int) {
	i := p.pc()
	p.add(OP_if_nil)
	p.i64(OP_size_const, 3)
	p.add(OP_make_state)
	p.add(OP_deref)
	self.measure(p, sp+1, fv.Type.V, startpc)
	p.add(OP_drop_state)
	p.pin(i)
}

func (self *Compiler) measureStructRequired(p *Program, sp int, fv defs.Field, startpc int) {
	p.i64(OP_size_const, 3)
	self.measure(p, sp, fv.Type, startpc)
}
