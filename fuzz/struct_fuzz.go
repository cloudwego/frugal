// Copyright 2022 CloudWeGo Authors
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

package fuzz

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/kitex/pkg/protocol/bthrift"
)

type Requiredness int

const (
	Default Requiredness = iota
	Required
	Optional
)

var RequirednessString = [...]string{
	Default:  "default",
	Required: "required",
	Optional: "optional",
}

var PointerMap = map[Requiredness]map[reflect.Kind]bool{
	Default: {
		reflect.Struct: true,
	},
	Required: {
		reflect.Struct: true,
	},
	Optional: {
		reflect.Bool:    true,
		reflect.Int8:    true,
		reflect.Int16:   true,
		reflect.Int32:   true,
		reflect.Int64:   true,
		reflect.Float64: true,
		reflect.String:  true,
		reflect.Struct:  true,
	},
}

type Enum int64

var (
	BoolType   = reflect.TypeOf(bool(true))
	ByteType   = reflect.TypeOf(int8(0))
	I16Type    = reflect.TypeOf(int16(0))
	I32Type    = reflect.TypeOf(int32(0))
	I64Type    = reflect.TypeOf(int64(0))
	DoubleType = reflect.TypeOf(float64(0))
	StringType = reflect.TypeOf(string("str"))
	BinaryType = reflect.TypeOf([]byte{0})
	EnumType   = reflect.TypeOf(Enum(0))
)

func fuzzDynamicStruct(typ *Type) (reflect.Type, error) {
	tc := &TypeConstructor{bthrift.Binary}
	ts, err := tc.GetType(typ)
	if err != nil {
		return nil, err
	}
	return reflect.StructOf([]reflect.StructField{BuildStructField(0, Default, ts)}), nil
}

type TypeSpec struct {
	Type    reflect.Type
	TypeTag string
}

type TypeConstructor struct {
	bp bthrift.BTProtocol
}

func (t *TypeConstructor) GetType(typ *Type) (ts *TypeSpec, err error) {
	var keySpec, valSpec *TypeSpec
	switch typ.TypeID {
	case thrift.BOOL, thrift.BYTE, thrift.I16, thrift.I32, thrift.I64, thrift.DOUBLE:
	case thrift.STRING:
		// FIXME: what about binary?
	case thrift.STRUCT:
		fields := make([]reflect.StructField, 0)
		for fieldID, fieldTyp := range typ.Fields {
			fts, e := t.GetType(fieldTyp)
			if e != nil {
				err = e
				return
			}
			gsf := BuildStructField(fieldID, Default, fts)
			fields = append(fields, gsf)
		}
		structType := reflect.StructOf(fields)
		return &TypeSpec{reflect.PointerTo(structType), "ANONYMOUS"}, nil
	case thrift.MAP:
		if typ.KeyType != nil {
			keySpec, err = t.GetType(typ.KeyType)
			if err != nil {
				return
			}
		}
		if typ.ValType != nil {
			valSpec, err = t.GetType(typ.ValType)
			if err != nil {
				return
			}
		}
	case thrift.SET, thrift.LIST:
		if typ.ValType != nil {
			valSpec, err = t.GetType(typ.ValType)
			if err != nil {
				return
			}
		}
	default:
		return nil, fmt.Errorf("unknown data type: %v", typ.TypeID)
	}
	return BuildTypeSpec(typ.TypeID, Default, keySpec, valSpec), nil
}

func BuildTypeSpec(t thrift.TType, requiredness Requiredness, keySpec, valSpec *TypeSpec) *TypeSpec {
	var typ reflect.Type
	var tag string
	switch t {
	case thrift.BOOL:
		typ = BoolType
		tag = "bool"
	case thrift.BYTE:
		typ = ByteType
		tag = "byte"
	case thrift.I16:
		typ = I16Type
		tag = "i16"
	case thrift.I32:
		typ = I32Type
		tag = "i32"
	case thrift.I64:
		typ = I64Type
		tag = "i64"
	case thrift.DOUBLE:
		typ = DoubleType
		tag = "double"
	case thrift.STRING:
		typ = StringType
		tag = "string"
	case thrift.MAP:
		if keySpec == nil {
			keySpec = &TypeSpec{Type: I64Type, TypeTag: "i64"}
		}
		if valSpec == nil {
			valSpec = &TypeSpec{Type: I64Type, TypeTag: "i64"}
		}
		typ = reflect.MapOf(keySpec.Type, valSpec.Type)
		tag = fmt.Sprintf("map<%s:%s>", keySpec.TypeTag, valSpec.TypeTag)
	case thrift.SET:
		if valSpec == nil {
			valSpec = &TypeSpec{Type: I64Type, TypeTag: "i64"}
		}
		typ = reflect.SliceOf(valSpec.Type)
		tag = fmt.Sprintf("set<%s>", valSpec.TypeTag)
	case thrift.LIST:
		if valSpec == nil {
			valSpec = &TypeSpec{Type: I64Type, TypeTag: "i64"}
		}
		typ = reflect.SliceOf(valSpec.Type)
		tag = fmt.Sprintf("list<%s>", valSpec.TypeTag)
	case thrift.STRUCT: // unknown struct
		typ = reflect.StructOf(nil)
		tag = "ANONYMOUS"
	default:
		panic("unreachable code" + t.String())
	}
	if PointerMap[requiredness][typ.Kind()] {
		typ = reflect.PointerTo(typ)
	}
	return &TypeSpec{Type: typ, TypeTag: tag}
}

func BuildStructField(id int16, requiredness Requiredness, ts *TypeSpec) (ret reflect.StructField) {
	tag := fmt.Sprintf("frugal:\"%d,%s,%s\"", id, RequirednessString[requiredness], ts.TypeTag)
	typ := ts.Type
	if PointerMap[requiredness][typ.Kind()] {
		typ = reflect.PointerTo(ts.Type)
	}
	var pkgPath string
	name := "Field" + strconv.Itoa(int(id))
	if id < 0 {
		name = "field" + strings.ReplaceAll(strconv.Itoa(int(id)), "-", "_")
		pkgPath = "anonymous"
	}
	return reflect.StructField{
		Name:    name,
		Type:    typ,
		Tag:     reflect.StructTag(tag),
		PkgPath: pkgPath,
	}
}
