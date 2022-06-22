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

package fuzz

import (
	"fmt"
	"reflect"
	"strconv"

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

var (
	BoolType      = reflect.TypeOf(bool(true))
	OptBoolType   = reflect.TypeOf(&(&struct{ x bool }{x: true}).x)
	ByteType      = reflect.TypeOf(int8(0))
	OptByteType   = reflect.TypeOf(&(&struct{ x int8 }{x: 0}).x)
	I16Type       = reflect.TypeOf(int16(0))
	OptI16Type    = reflect.TypeOf(&(&struct{ x int16 }{x: 0}).x)
	I32Type       = reflect.TypeOf(int32(0))
	OptI32Type    = reflect.TypeOf(&(&struct{ x int32 }{x: 0}).x)
	I64Type       = reflect.TypeOf(int64(0))
	OptI64Type    = reflect.TypeOf(&(&struct{ x int64 }{x: 0}).x)
	DoubleType    = reflect.TypeOf(float64(0))
	OptDoubleType = reflect.TypeOf(&(&struct{ x float64 }{x: 0}).x)
	StringType    = reflect.TypeOf(string("str"))
	OptStringType = reflect.TypeOf(&(&struct{ x string }{x: "0"}).x)
)

var PointerMap = map[Requiredness]map[reflect.Kind]bool{
	Default: {
		reflect.Struct: true,
	},
	Required: {
		reflect.Struct: true,
	},
	Optional: {
		reflect.Int8:    true,
		reflect.Int16:   true,
		reflect.Int32:   true,
		reflect.Int64:   true,
		reflect.Float64: true,
		reflect.String:  true,
		reflect.Struct:  true,
	},
}

func fuzzDynamicStruct(data []byte, tt thrift.TType) (reflect.Type, error) {
	tc := &TypeConstructor{bthrift.Binary}
	ts, _, err := tc.GetType(data, tt)
	if err != nil {
		return nil, err
	}
	return reflect.StructOf([]reflect.StructField{{Name: "Field0", Type: ts.Type, Tag: reflect.StructTag(ts.TypeTag)}}), nil
}

type TypeSpec struct {
	Type    reflect.Type
	TypeTag string
}

type TypeConstructor struct {
	bp bthrift.BTProtocol
}

func (t *TypeConstructor) GetType(buf []byte, fieldType thrift.TType) (ts TypeSpec, length int, err error) {
	switch fieldType {
	case thrift.BOOL:
		_, length, err = t.bp.ReadBool(buf)
		if err != nil {
			return
		}
		return TypeSpec{BoolType, "bool"}, length, nil
	case thrift.BYTE:
		_, length, err = t.bp.ReadByte(buf)
		if err != nil {
			return
		}
		return TypeSpec{ByteType, "byte"}, length, nil
	case thrift.I16:
		_, length, err = t.bp.ReadI16(buf)
		if err != nil {
			return
		}
		return TypeSpec{I16Type, "i16"}, length, nil
	case thrift.I32:
		_, length, err = t.bp.ReadI32(buf)
		if err != nil {
			return
		}
		return TypeSpec{I32Type, "i32"}, length, nil
	case thrift.I64:
		_, length, err = t.bp.ReadI64(buf)
		if err != nil {
			return
		}
		return TypeSpec{I64Type, "i64"}, length, nil
	case thrift.DOUBLE:
		_, length, err = t.bp.ReadDouble(buf)
		if err != nil {
			return
		}
		return TypeSpec{DoubleType, "double"}, length, nil
	case thrift.STRING:
		_, length, err = t.bp.ReadString(buf)
		if err != nil {
			return
		}
		// FIXME: what about binary?
		return TypeSpec{StringType, "string"}, length, nil
	case thrift.STRUCT:
		_, _, err = t.bp.ReadStructBegin(buf)
		if err != nil {
			return
		}
		fields := make([]reflect.StructField, 0)
		var fieldID int
		for {
			_, typeID, _, l, e := t.bp.ReadFieldBegin(buf[length:])
			length += l
			if e != nil {
				err = e
				return
			}
			if typeID == thrift.STOP {
				break
			}
			fts, l, e := t.GetType(buf[length:], typeID)
			length += l
			if e != nil {
				err = e
				return
			}
			l, e = t.bp.ReadFieldEnd(buf[length:])
			length += l
			if e != nil {
				err = e
				return
			}
			gsf := GenerateStructFields(fieldID, Default, fts)
			fields = append(fields, gsf)
			fieldID++
		}
		l, e := t.bp.ReadStructEnd(buf[length:])
		length += l
		if e != nil {
			err = e
			return
		}
		structType := reflect.StructOf(fields)
		return TypeSpec{structType, "ANONYMOUS"}, length, nil
	case thrift.MAP:
		keyType, valueType, size, l, e := t.bp.ReadMapBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		var kts, vts TypeSpec
		for i := 0; i < size; i++ {
			kts, l, e = t.GetType(buf[length:], keyType)
			length += l
			if e != nil {
				err = e
				return
			}
			vts, l, e = t.GetType(buf[length:], valueType)
			length += l
			if e != nil {
				err = e
				return
			}
		}
		if size == 0 { // TODO: use random kv?
			kts = TypeSpec{StringType, "string"}
			vts = TypeSpec{I64Type, "i64"}
		}
		l, e = t.bp.ReadMapEnd(buf[length:])
		length += l
		if e != nil {
			err = e
			return
		}
		return TypeSpec{reflect.MapOf(kts.Type, vts.Type), fmt.Sprintf("map<%s:%s>", kts.TypeTag, vts.TypeTag)}, length, nil
	case thrift.SET:
		elemType, size, l, e := t.bp.ReadSetBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		var ets TypeSpec
		for i := 0; i < size; i++ {
			ets, l, e = t.GetType(buf[length:], elemType)
			length += l
			if e != nil {
				err = e
				return
			}
		}
		if size == 0 { // TODO: use random kv?
			ets = TypeSpec{I64Type, "i64"}
		}
		l, e = t.bp.ReadSetEnd(buf[length:])
		length += l
		if e != nil {
			err = e
			return
		}
		return TypeSpec{reflect.SliceOf(ets.Type), fmt.Sprintf("set<%s>", ets.TypeTag)}, length, nil
	case thrift.LIST:
		elemType, size, l, e := t.bp.ReadListBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		var ets TypeSpec
		for i := 0; i < size; i++ {
			ets, l, e = t.GetType(buf[length:], elemType)
			length += l
			if e != nil {
				err = e
				return
			}
		}
		if size == 0 { // TODO: use random kv?
			ets = TypeSpec{I64Type, "i64"}
		}
		l, e = t.bp.ReadListEnd(buf[length:])
		length += l
		if e != nil {
			err = e
			return
		}
		return TypeSpec{reflect.SliceOf(ets.Type), fmt.Sprintf("list<%s>", ets.TypeTag)}, length, nil
	default:
		return TypeSpec{}, 0, fmt.Errorf("unknown data type: %v", fieldType)
	}
}

func GenerateStructFields(id int, requiredness Requiredness, ts TypeSpec) (ret reflect.StructField) {
	tag := fmt.Sprintf("frugal:\"%d,%s,%s\"", id, RequirednessString[requiredness], ts.TypeTag)
	typ := ts.Type
	if PointerMap[requiredness][typ.Kind()] {
		typ = reflect.PointerTo(ts.Type)
	}
	return reflect.StructField{
		Name: "Field" + strconv.Itoa(id),
		Type: typ,
		Tag:  reflect.StructTag(tag),
	}
}
