// Copyright 2022 ByteDance Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/frugal/fuzz"
	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/semantic"
)

var EmptyStructTypeSpec = &fuzz.TypeSpec{
	Type:    reflect.PointerTo(reflect.StructOf(nil)),
	TypeTag: "ANONYMOUS",
}

var Category2Type = map[parser.Category]thrift.TType{
	parser.Category_Bool:      thrift.BOOL,
	parser.Category_Byte:      thrift.BYTE,
	parser.Category_I16:       thrift.I16,
	parser.Category_I32:       thrift.I32,
	parser.Category_I64:       thrift.I64,
	parser.Category_Double:    thrift.DOUBLE,
	parser.Category_String:    thrift.STRING,
	parser.Category_Struct:    thrift.STRUCT,
	parser.Category_Map:       thrift.MAP,
	parser.Category_Set:       thrift.SET,
	parser.Category_List:      thrift.LIST,
	parser.Category_Enum:      thrift.I32,
	parser.Category_Binary:    thrift.STRING,
	parser.Category_Exception: thrift.STRUCT,
	parser.Category_Union:     thrift.STRUCT,
}

type StructBuilder struct {
	structMap map[*parser.StructLike]*fuzz.TypeSpec
}

func NewStructBuilder() *StructBuilder {
	return &StructBuilder{
		structMap: make(map[*parser.StructLike]*fuzz.TypeSpec),
	}
}

func (s *StructBuilder) BuildThriftStruct(file string) (<-chan (reflect.Type), error) {
	out := make(chan (reflect.Type))
	tree, err := parser.ParseFile(file, nil, true)
	if err != nil {
		return nil, fmt.Errorf("parse thrift file %s failed: %w", file, err)
	}
	checker := semantic.NewChecker(semantic.Options{FixWarnings: true})
	_, err = checker.CheckAll(tree)
	if err != nil {
		return nil, fmt.Errorf("thrift file %s check failed: %w", file, err)
	}
	err = semantic.ResolveSymbols(tree)
	if err != nil {
		return nil, fmt.Errorf("thrift file %s resolve failed: %w", file, err)
	}
	go func() {
		for _, st := range tree.GetStructLikes() {
			rt, err := s.buildStructFromAST(tree, st)
			if err != nil {
				log.Println(fmt.Errorf("build struct from struct %s in thrift file %s failed: %w", st.Name, file, err))
				continue
			}
			out <- rt.Type
		}
		close(out)
	}()
	return out, nil
}

func (s *StructBuilder) buildStructFromAST(tree *parser.Thrift, sl *parser.StructLike) (*fuzz.TypeSpec, error) {
	ret, ok := s.structMap[sl]
	if ok {
		return ret, nil
	}
	// avoid struct circular dependency
	s.structMap[sl] = EmptyStructTypeSpec
	fields := make([]reflect.StructField, len(sl.Fields))
	for i, sf := range sl.Fields {
		t, err := s.buildTypeFromAST(tree, sf.Type, fuzz.Requiredness(sf.Requiredness))
		if err != nil {
			return nil, fmt.Errorf("build struct %s field %s failed: %w", sl.Name, sf.Name, err)
		}
		fields[i] = fuzz.BuildStructField(int16(sf.ID), fuzz.Requiredness(sf.Requiredness), t)
	}
	structType := reflect.StructOf(fields)
	ret = &fuzz.TypeSpec{
		Type:    reflect.PointerTo(structType),
		TypeTag: "ANONYMOUS",
	}
	s.structMap[sl] = ret
	return ret, nil
}

func (s *StructBuilder) buildTypeFromAST(tree *parser.Thrift, typ *parser.Type, requiredness fuzz.Requiredness) (ts *fuzz.TypeSpec, err error) {
	var keySpec, valSpec *fuzz.TypeSpec
	switch typ.Category {
	case parser.Category_Bool:
	case parser.Category_Byte:
	case parser.Category_I16:
	case parser.Category_I32:
	case parser.Category_I64:
	case parser.Category_Double:
	case parser.Category_String:
	case parser.Category_Binary:
		return &fuzz.TypeSpec{Type: fuzz.BinaryType, TypeTag: "binary"}, nil
	case parser.Category_Enum:
		return &fuzz.TypeSpec{Type: fuzz.EnumType, TypeTag: "Enum"}, nil
	case parser.Category_Struct, parser.Category_Exception, parser.Category_Union:
		name := typ.Name
		if typ.Reference != nil {
			name = typ.Reference.Name
			tree = tree.Includes[typ.Reference.Index].Reference
		}
		st, ok := tree.GetStruct(name)
		if !ok {
			return nil, fmt.Errorf("struct-like %s not defined in %s", name, tree.Filename)
		}
		return s.buildStructFromAST(tree, st)
	case parser.Category_Map:
		keySpec, err = s.buildTypeFromAST(tree, typ.KeyType, fuzz.Default)
		if err != nil {
			return nil, err
		}
		valSpec, err = s.buildTypeFromAST(tree, typ.ValueType, fuzz.Default)
		if err != nil {
			return nil, err
		}
	case parser.Category_Set, parser.Category_List:
		valSpec, err = s.buildTypeFromAST(tree, typ.ValueType, fuzz.Default)
		if err != nil {
			return nil, err
		}
	case parser.Category_Typedef:
		typDef, ok := tree.GetTypedef(typ.Name)
		if !ok {
			return nil, fmt.Errorf("typedef %s not defined in %s", typ.Name, tree.Filename)
		}
		return s.buildTypeFromAST(tree, typDef.Type, requiredness)
	default:
		return nil, fmt.Errorf("unknown category type: %v", typ.Category)
	}
	return fuzz.BuildTypeSpec(Category2Type[typ.Category], requiredness, keySpec, valSpec), nil
}
