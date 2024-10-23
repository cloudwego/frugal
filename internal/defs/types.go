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

package defs

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

type Tag uint8

const (
	T_bool    Tag = 2
	T_i8      Tag = 3
	T_double  Tag = 4
	T_i16     Tag = 6
	T_i32     Tag = 8
	T_i64     Tag = 10
	T_string  Tag = 11
	T_struct  Tag = 12
	T_map     Tag = 13
	T_set     Tag = 14
	T_list    Tag = 15
	T_enum    Tag = 0x80
	T_binary  Tag = 0x81
	T_pointer Tag = 0x82
)

var wireTags = [256]bool{
	T_bool:   true,
	T_i8:     true,
	T_double: true,
	T_i16:    true,
	T_i32:    true,
	T_i64:    true,
	T_string: true,
	T_struct: true,
	T_map:    true,
	T_set:    true,
	T_list:   true,
}

var keywordTab = [256]string{
	T_bool:   "bool",
	T_i8:     "i8 byte",
	T_double: "double",
	T_i16:    "i16",
	T_i32:    "i32",
	T_i64:    "i64",
	T_string: "string",
	T_binary: "binary",
	T_struct: "struct",
	T_map:    "map",
}

var (
	i64type  = reflect.TypeOf(int64(0))
	bytetype = reflect.TypeOf(byte(0))
)

func T_int() Tag {
	switch IntSize {
	case 4:
		return T_i32
	case 8:
		return T_i64
	default:
		panic("invalid int size")
	}
}

func (self Tag) IsWireTag() bool {
	return wireTags[self]
}

type Type struct {
	T Tag
	K *Type
	V *Type
	S reflect.Type
}

var (
	typePool sync.Pool
)

func newType() *Type {
	if v := typePool.Get(); v == nil {
		return new(Type)
	} else {
		return resetType(v.(*Type))
	}
}

func resetType(p *Type) *Type {
	*p = Type{}
	return p
}

func (self *Type) Tag() Tag {
	switch self.T {
	case T_enum:
		return T_i32
	case T_binary:
		return T_string
	case T_pointer:
		return self.V.Tag()
	default:
		return self.T
	}
}

func (self *Type) IsEnum() bool {
	switch self.T {
	case T_enum:
		return true
	case T_pointer:
		return self.V.IsEnum()
	default:
		return false
	}
}

func (self *Type) Free() {
	typePool.Put(self)
}

func (self *Type) String() string {
	switch self.T {
	case T_bool:
		return "bool"
	case T_i8:
		return "i8"
	case T_double:
		return "double"
	case T_i16:
		return "i16"
	case T_i32:
		return "i32"
	case T_i64:
		return "i64"
	case T_string:
		return "string"
	case T_struct:
		return self.S.Name()
	case T_map:
		return fmt.Sprintf("map<%s:%s>", self.K.String(), self.V.String())
	case T_set:
		return fmt.Sprintf("set<%s>", self.V.String())
	case T_list:
		return fmt.Sprintf("list<%s>", self.V.String())
	case T_enum:
		return "enum"
	case T_binary:
		return "binary"
	case T_pointer:
		return "*" + self.V.String()
	default:
		return fmt.Sprintf("Type(Tag(%d))", self.T)
	}
}

func (self *Type) IsKeyType() bool {
	switch self.T {
	case T_bool:
		return true
	case T_i8:
		return true
	case T_double:
		return true
	case T_i16:
		return true
	case T_i32:
		return true
	case T_i64:
		return true
	case T_string:
		return true
	case T_enum:
		return true
	case T_pointer:
		return self.V.T == T_struct
	default:
		return false
	}
}

func (self *Type) IsValueType() bool {
	return self.T != T_pointer || self.V.T == T_struct
}

func (self *Type) IsSimpleType() bool {
	switch self.T {
	case T_bool:
		return true
	case T_i8:
		return true
	case T_double:
		return true
	case T_i16:
		return true
	case T_i32:
		return true
	case T_i64:
		return true
	case T_string:
		return true
	case T_enum:
		return true
	default:
		return false
	}
}

func ParseType(vt reflect.Type, def string) (*Type, error) {
	var i int
	return doParseType(vt, def, &i, true)
}

func isident(c byte) bool {
	return isident0(c) || c >= '0' && c <= '9'
}

func isident0(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func readToken(src string, i *int, eofok bool) (string, error) {
	p := *i
	n := len(src)

	/* skip the spaces */
	for p < n && unicode.IsSpace(rune(src[p])) {
		p++
	}

	/* check for EOF */
	if p == n {
		if eofok {
			return "", nil
		} else {
			return "", ESyntax(p, src, "unexpected EOF")
		}
	}

	/* skip the character */
	q := p
	p++

	/* check for identifiers */
	if isident0(src[q]) {
		for p < n && isident(src[p]) {
			p++
		}
	}

	/* slice the source */
	*i = p
	return src[q:p], nil
}

func mkMistyped(pos int, src string, tv string, tag Tag, vt reflect.Type) SyntaxError {
	if tag != T_struct {
		return ESyntax(pos, src, fmt.Sprintf("type mismatch, %s expected, got %s", keywordTab[tag], tv))
	} else {
		return ESyntax(pos, src, fmt.Sprintf("struct name mismatch, %s expected, got %s", vt.Name(), tv))
	}
}

func doParseType(vt reflect.Type, def string, i *int, allowPtrs bool) (*Type, error) {
	var tag Tag
	var err error
	var ret *Type

	/* dereference the pointer if possible */
	if ret = newType(); vt.Kind() == reflect.Ptr {
		ret.S = vt
		ret.T = T_pointer

		/* prohibit nested pointers */
		if !allowPtrs {
			return nil, EType(ret.V.S, "nested pointer is not allowed")
		}

		/* parse the pointer element recursively */
		if ret.V, err = doParseType(vt.Elem(), def, i, false); err != nil {
			return nil, err
		} else {
			return ret, nil
		}
	}

	/* check for value kind */
	switch vt.Kind() {
	case reflect.Bool:
		tag = T_bool
	case reflect.Int:
		tag = T_int()
	case reflect.Int8:
		tag = T_i8
	case reflect.Int16:
		tag = T_i16
	case reflect.Int32:
		tag = T_i32
	case reflect.Int64:
		tag = T_i64
	case reflect.Uint:
		return nil, EUseOther(vt, "int")
	case reflect.Uint8:
		return nil, EUseOther(vt, "int8")
	case reflect.Uint16:
		return nil, EUseOther(vt, "int16")
	case reflect.Uint32:
		return nil, EUseOther(vt, "int32")
	case reflect.Uint64:
		return nil, EUseOther(vt, "int64")
	case reflect.Float32:
		return nil, EUseOther(vt, "float64")
	case reflect.Float64:
		tag = T_double
	case reflect.Array:
		return nil, EUseOther(vt, "[]"+vt.Elem().String())
	case reflect.Map:
		tag = T_map
	case reflect.Slice:
		break
	case reflect.String:
		tag = T_string
	case reflect.Struct:
		tag = T_struct
	default:
		return nil, EType(vt, "unsupported type")
	}

	/* it's a slice, check for byte slice */
	if tag == 0 {
		if et := vt.Elem(); et == bytetype {
			tag = T_binary
		} else if def == "" {
			return nil, ESetList(*i, def, et)
		} else {
			return doParseSlice(vt, et, def, i, ret)
		}
	}

	/* match the type if any */
	if def != "" {
		if tv, et := readToken(def, i, false); et != nil {
			return nil, et
		} else if !strings.Contains(keywordTab[tag], tv) {
			if !isident0(tv[0]) {
				return nil, mkMistyped(*i-len(tv), def, tv, tag, vt)
			} else if ok, ex := doMatchStruct(vt, def, i, &tv); ex != nil {
				return nil, ex
			} else if !ok {
				return nil, mkMistyped(*i-len(tv), def, tv, tag, vt)
			} else if tag == T_i64 && vt != i64type {
				tag = T_enum
			}
		}
	}

	/* simple types */
	if tag != T_map {
		ret.S = vt
		ret.T = tag
		return ret, nil
	}

	/* map begin */
	if def != "" {
		if tk, et := readToken(def, i, false); et != nil {
			return nil, et
		} else if tk != "<" {
			return nil, ESyntax(*i-len(tk), def, "'<' expected")
		}
	}

	/* parse the key type */
	if ret.K, err = doParseType(vt.Key(), def, i, true); err != nil {
		return nil, err
	}

	/* validate map key */
	if !ret.K.IsKeyType() {
		return nil, EType(ret.K.S, "not a valid map key type")
	}

	/* key-value delimiter */
	if def != "" {
		if tk, et := readToken(def, i, false); et != nil {
			return nil, et
		} else if tk != ":" {
			return nil, ESyntax(*i-len(tk), def, "':' expected")
		}
	}

	/* parse the value type */
	if ret.V, err = doParseType(vt.Elem(), def, i, true); err != nil {
		return nil, err
	}

	/* map end */
	if def != "" {
		if tk, et := readToken(def, i, false); et != nil {
			return nil, et
		} else if tk != ">" {
			return nil, ESyntax(*i-len(tk), def, "'>' expected")
		}
	}

	/* check for list elements */
	if !ret.V.IsValueType() {
		return nil, EType(ret.V.S, "non-struct pointers are not valid map value types")
	}

	/* set the type tag */
	ret.S = vt
	ret.T = T_map
	return ret, nil
}

func doParseSlice(vt reflect.Type, et reflect.Type, def string, i *int, rt *Type) (*Type, error) {
	var err error
	var tok string

	/* read the next token */
	if tok, err = readToken(def, i, false); err != nil {
		return nil, err
	}

	/* identify "set" or "list" */
	if tok == "set" {
		rt.T = T_set
	} else if tok == "list" {
		rt.T = T_list
	} else {
		return nil, ESyntax(*i-len(tok), def, `"set" or "list" expected`)
	}

	/* list or set begin */
	if tok, err = readToken(def, i, false); err != nil {
		return nil, err
	} else if tok != "<" {
		return nil, ESyntax(*i-len(tok), def, "'<' expected")
	}

	/* set or list element */
	if rt.V, err = doParseType(et, def, i, true); err != nil {
		return nil, err
	}

	/* list or set end */
	if tok, err = readToken(def, i, false); err != nil {
		return nil, err
	} else if tok != ">" {
		return nil, ESyntax(*i-len(tok), def, "'>' expected")
	}

	/* check for list elements */
	if !rt.V.IsValueType() {
		return nil, EType(rt.V.S, "non-struct pointers are not valid list/set elements")
	}

	/* set the type */
	rt.S = vt
	return rt, nil
}

func doMatchStruct(vt reflect.Type, def string, i *int, tv *string) (bool, error) {
	var err error
	var tok string

	/* mark the starting position */
	sp := *i
	tn := vt.Name()

	/* read the next token */
	if tok, err = readToken(def, &sp, true); err != nil {
		return false, err
	}

	/* anonymous struct */
	if tn == "" && vt.Kind() == reflect.Struct {
		return true, nil
	}

	/* just a simple type with no qualifiers */
	if tok == "" || tok == ":" || tok == ">" {
		return tn == *tv, nil
	}

	/* otherwise, it must be a "." */
	if tok != "." {
		return false, ESyntax(sp, def, "'.' or '>' expected")
	}

	/* must be an identifier */
	if *tv, err = readToken(def, &sp, false); err != nil {
		return false, err
	} else if !isident0((*tv)[0]) {
		return false, ESyntax(sp, def, "struct name expected")
	}

	/* update parsing position */
	*i = sp
	return tn == *tv, nil
}
