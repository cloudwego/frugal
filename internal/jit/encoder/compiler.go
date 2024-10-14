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
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
	"github.com/cloudwego/frugal/internal/jit/rt"
	"github.com/cloudwego/frugal/internal/opts"
)

type Instr struct {
	Op OpCode
	Uv int32
	Iv int64
	To int
	Pr unsafe.Pointer
}

type (
	Program []Instr
)

func (self Instr) Vt() *rt.GoType {
	return (*rt.GoType)(self.Pr)
}

func (self Instr) Str() string {
	return rt.StringFrom(self.Pr, int(self.Uv))
}

func (self Instr) Byte(i int64) int8 {
	return *(*int8)(unsafe.Pointer(uintptr(self.Pr) + uintptr(i)))
}

func (self Instr) Word(i int64) int16 {
	return *(*int16)(unsafe.Pointer(uintptr(self.Pr) + uintptr(i)))
}

func (self Instr) Long(i int64) int32 {
	return *(*int32)(unsafe.Pointer(uintptr(self.Pr) + uintptr(i)))
}

func (self Instr) Quad(i int64) int64 {
	return *(*int64)(unsafe.Pointer(uintptr(self.Pr) + uintptr(i)))
}

func (self Instr) Disassemble() string {
	switch self.Op {
	case OP_size_check:
		fallthrough
	case OP_size_const:
		fallthrough
	case OP_size_map:
		fallthrough
	case OP_seek:
		fallthrough
	case OP_sint:
		fallthrough
	case OP_length:
		return fmt.Sprintf("%-18s%d", self.Op, self.Iv)
	case OP_size_dyn:
		fallthrough
	case OP_memcpy_be:
		return fmt.Sprintf("%-18s%d, %d", self.Op, self.Uv, self.Iv)
	case OP_size_defer:
		fallthrough
	case OP_defer:
		fallthrough
	case OP_map_begin:
		fallthrough
	case OP_unique:
		return fmt.Sprintf("%-18s%s", self.Op, self.Vt())
	case OP_byte:
		return fmt.Sprintf("%-18s0x%02x", self.Op, self.Iv)
	case OP_word:
		return fmt.Sprintf("%-18s0x%04x", self.Op, self.Iv)
	case OP_long:
		return fmt.Sprintf("%-18s0x%08x", self.Op, self.Iv)
	case OP_quad:
		return fmt.Sprintf("%-18s0x%016x", self.Op, self.Iv)
	case OP_map_if_next:
		fallthrough
	case OP_map_if_empty:
		fallthrough
	case OP_list_if_next:
		fallthrough
	case OP_list_if_empty:
		fallthrough
	case OP_goto:
		fallthrough
	case OP_if_nil:
		fallthrough
	case OP_if_hasbuf:
		return fmt.Sprintf("%-18sL_%d", self.Op, self.To)
	case OP_if_eq_imm:
		return fmt.Sprintf("%-18s%d:%d, L_%d", self.Op, self.Iv, self.Uv, self.To)
	case OP_if_eq_str:
		return fmt.Sprintf("%-18s%q, L_%d", self.Op, self.Str(), self.To)
	default:
		return self.Op.String()
	}
}

func (self Program) pc() int   { return len(self) }
func (self Program) pin(i int) { self[i].To = self.pc() }

func (self Program) tag(n int) {
	if n >= defs.StackSize {
		panic("type nesting too deep")
	}
}

func (self *Program) ins(iv Instr)            { *self = append(*self, iv) }
func (self *Program) add(op OpCode)           { self.ins(Instr{Op: op}) }
func (self *Program) jmp(op OpCode, to int)   { self.ins(Instr{Op: op, To: to}) }
func (self *Program) i64(op OpCode, iv int64) { self.ins(Instr{Op: op, Iv: iv}) }
func (self *Program) str(op OpCode, sv string) {
	self.ins(Instr{Op: op, Iv: int64(len(sv)), Pr: rt.StringPtr(sv)})
}

func (self *Program) rtt(op OpCode, vt reflect.Type) {
	self.ins(Instr{Op: op, Pr: unsafe.Pointer(rt.UnpackType(vt))})
}
func (self *Program) dyn(op OpCode, uv int32, iv int64) { self.ins(Instr{Op: op, Uv: uv, Iv: iv}) }

func (self Program) Free() {
	freeProgram(self)
}

func (self Program) Disassemble() string {
	nb := len(self)
	tab := make([]bool, nb+1)
	ret := make([]string, 0, nb+1)

	/* prescan to get all the labels */
	for _, ins := range self {
		if _OpBranches[ins.Op] {
			tab[ins.To] = true
		}
	}

	/* disassemble each instruction */
	for i, ins := range self {
		if !tab[i] {
			ret = append(ret, "    "+ins.Disassemble())
		} else {
			ret = append(ret, fmt.Sprintf("L_%d:\n    %s", i, ins.Disassemble()))
		}
	}

	/* add an "end" indicator, and join all the strings */
	if !tab[nb] {
		return strings.Join(append(ret, "    end"), "\n")
	} else {
		return strings.Join(append(ret, fmt.Sprintf("L_%d:", nb), "    end"), "\n")
	}
}

type Compiler struct {
	o opts.Options
	t map[reflect.Type]bool
}

func CreateCompiler() *Compiler {
	return newCompiler()
}

func (self *Compiler) rescue(ep *error) {
	if val := recover(); val != nil {
		if err, ok := val.(error); ok {
			*ep = err
		} else {
			panic(val)
		}
	}
}

func (self *Compiler) Free() {
	freeCompiler(self)
}

func (self *Compiler) Apply(o opts.Options) *Compiler {
	self.o = o
	return self
}

func (self *Compiler) resetState() *Compiler {
	o := self.o // prevent resetting opts
	resetCompiler(self)
	self.o = o
	return self
}

func (self *Compiler) Compile(vt reflect.Type) (_ Program, err error) {
	ret := newProgram()
	vtp := (*defs.Type)(nil)

	/* parse the type */
	if vtp, err = defs.ParseType(vt, ""); err != nil {
		return nil, err
	}

	/* catch the exceptions, and free the type */
	defer self.rescue(&err)
	defer vtp.Free()

	/* object measuring */
	i := ret.pc()
	ret.add(OP_if_hasbuf)
	self.resetState().measure(&ret, 0, vtp, ret.pc())

	/* object encoding */
	j := ret.pc()
	ret.add(OP_goto)
	ret.pin(i)
	self.resetState().compile(&ret, 0, vtp, ret.pc())

	/* halt the program */
	ret.pin(j)
	ret.add(OP_halt)
	return Optimize(ret), nil
}

func (self *Compiler) CompileAndFree(vt reflect.Type) (ret Program, err error) {
	ret, err = self.Compile(vt)
	self.Free()
	return
}
