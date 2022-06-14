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
	"strconv"

	"github.com/cloudwego/frugal/fuzz"
	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/semantic"
)

type StructBuilder struct{}

func (s *StructBuilder) BuildThriftStruct(thrift string) (<-chan (reflect.Type), error) {
	out := make(chan (reflect.Type))
	tree, err := parser.ParseFile(thrift, nil, true)
	if err != nil {
		return nil, fmt.Errorf("parse thrift file %s failed: %w", thrift, err)
	}
	checker := semantic.NewChecker(semantic.Options{FixWarnings: true})
	_, err = checker.CheckAll(tree)
	if err != nil {
		return nil, fmt.Errorf("thrift file %s check failed: %w", thrift, err)
	}
	err = semantic.ResolveSymbols(tree)
	if err != nil {
		return nil, fmt.Errorf("thrift file %s resolve failed: %w", thrift, err)
	}
	go func() {
		for _, st := range tree.GetStructLikes() {
			rt, err := s.buildStructFromAST(tree, st)
			if err != nil {
				log.Println(fmt.Errorf("build struct from struct %s in thrift file %s failed: %w", st.Name, thrift, err))
				continue
			}
			out <- rt
		}
	}()
	return out, nil
}

func (s *StructBuilder) buildStructFromAST(tree *parser.Thrift, sl *parser.StructLike) (reflect.Type, error) {
	fields := make([]reflect.StructField, len(sl.Fields))
	for i, sf := range sl.Fields {
		t, err := s.buildTypeFromAST(tree, *sf.Type, fuzz.Requiredness(sf.Requiredness))
		if err != nil {
			return nil, fmt.Errorf("build struct %s field %s failed: %w", sl.Name, sf.Name, err)
		}
		fields[i] = reflect.StructField{
			Name: "Field" + strconv.Itoa(i),
			Type: t,
		}
	}
	return reflect.StructOf(fields), nil
}

func (s *StructBuilder) buildTypeFromAST(tree *parser.Thrift, typ parser.Type, requiredness fuzz.Requiredness) (rt reflect.Type, err error) {
	switch typ.Category {
	case parser.Category_Bool:
		return fuzz.BoolType, nil
	case parser.Category_Byte:
		return fuzz.ByteType, nil
	case parser.Category_I16:
		return fuzz.I16Type, nil
	case parser.Category_I32:
		return fuzz.I32Type, nil
	case parser.Category_I64:
		return fuzz.I64Type, nil
	case parser.Category_Double:
		return fuzz.DoubleType, nil
	case parser.Category_String:
		return fuzz.StringType, nil
	case parser.Category_Struct:
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
		kt, err := s.buildTypeFromAST(tree, *typ.KeyType, fuzz.Default)
		if err != nil {
			return nil, err
		}
		vt, err := s.buildTypeFromAST(tree, *typ.KeyType, fuzz.Default)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(kt, vt), nil
	case parser.Category_Set, parser.Category_List:
		et, err := s.buildTypeFromAST(tree, *typ.ValueType, fuzz.Default)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(et), nil
	default:
		return nil, fmt.Errorf("unknown data type: %v", typ.Category)
	}
}
