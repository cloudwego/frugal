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
	"sort"
	"strings"

	"github.com/cloudwego/gopkg/protocol/thrift"
)

var TypeSize = map[thrift.TType]int{
	thrift.BOOL:   1,
	thrift.BYTE:   1,
	thrift.I16:    2,
	thrift.I32:    4,
	thrift.I64:    8,
	thrift.DOUBLE: 8,
	thrift.STRING: 4,
	thrift.LIST:   5,
	thrift.SET:    5,
	thrift.MAP:    6,
	thrift.STRUCT: 1,
}

func isValidType(t thrift.TType) bool {
	if t < thrift.BOOL || t > thrift.LIST || t == thrift.TType(5) || t == thrift.TType(7) || t == thrift.TType(9) {
		return false
	}
	return true
}

// Check checks if buf is valid thrift binary-buffered binary.
func Check(buf []byte, fieldTypeID thrift.TType) (typ *Type, length int, err error) {
	defer func() {
		if panic_err := recover(); panic_err != nil {
			if strings.Contains(fmt.Sprint(panic_err), "slice bounds out of range") {
				err = fmt.Errorf(fmt.Sprint(panic_err))
			}
		}
	}()
	var l int
	typ = &Type{TypeID: fieldTypeID}
	switch fieldTypeID {
	case thrift.BOOL:
		_, l, err = thrift.Binary.ReadBool(buf)
		length += l
		return
	case thrift.BYTE:
		_, l, err = thrift.Binary.ReadByte(buf)
		length += l
		return
	case thrift.I16:
		_, l, err = thrift.Binary.ReadI16(buf)
		length += l
		return
	case thrift.I32:
		_, l, err = thrift.Binary.ReadI32(buf)
		length += l
		return
	case thrift.I64:
		_, l, err = thrift.Binary.ReadI64(buf)
		length += l
		return
	case thrift.DOUBLE:
		_, l, err = thrift.Binary.ReadDouble(buf)
		length += l
		return
	case thrift.STRING:
		_, l, err = thrift.Binary.ReadString(buf)
		length += l
		return
	case thrift.STRUCT:
		fields := make(map[int16]*Type)
		for {
			typeID, id, l, e := thrift.Binary.ReadFieldBegin(buf[length:])
			length += l
			if e != nil {
				err = e
				return
			}
			if typeID == thrift.STOP {
				break
			}
			fType, l, e := Check(buf[length:], typeID)
			if e != nil {
				err = e
				return
			}
			length += l
			if _, ok := fields[id]; ok {
				err = fmt.Errorf("duplicate field id: %d", id)
				return
			}
			fields[id] = fType
		}
		typ.Fields = fields
		return
	case thrift.MAP:
		keyTypeID, valTypeID, size, l, e := thrift.Binary.ReadMapBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		if !isValidType(keyTypeID) {
			return nil, 0, fmt.Errorf("unknown data type %d", keyTypeID)
		}
		if !isValidType(valTypeID) {
			return nil, 0, fmt.Errorf("unknown data type %d", keyTypeID)
		}
		if keyTypeID == thrift.LIST || keyTypeID == thrift.SET || keyTypeID == thrift.MAP {
			return nil, 0, fmt.Errorf("map key cannot be container")
		}
		if length+size*(TypeSize[keyTypeID]+TypeSize[valTypeID]) >= len(buf) {
			return nil, 0, fmt.Errorf("size not enough")
		}
		var keyTypes, valTypes Types
		for i := 0; i < size; i++ {
			keyType, l, e := Check(buf[length:], keyTypeID)
			if e != nil {
				err = e
				return
			}
			length += l
			keyTypes = append(keyTypes, keyType)
			valType, l, e := Check(buf[length:], valTypeID)
			length += l
			if e != nil {
				err = e
				return
			}
			valTypes = append(valTypes, valType)
		}
		if size == 0 {
			keyTypes = append(keyTypes, &Type{TypeID: keyTypeID})
			valTypes = append(valTypes, &Type{TypeID: valTypeID})
		}
		if keyTypes.Conflict() {
			return nil, 0, fmt.Errorf("map key type conflict")
		}
		if valTypes.Conflict() {
			return nil, 0, fmt.Errorf("map value type conflict")
		}
		// NOTE: considering combine types to one complete type
		typ.KeyType = keyTypes[0]
		typ.ValType = valTypes[0]
		return
	case thrift.SET:
		elemTypeID, size, l, e := thrift.Binary.ReadSetBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		if !isValidType(elemTypeID) {
			return nil, 0, fmt.Errorf("unknown data type %d", elemTypeID)
		}
		if length+size*TypeSize[elemTypeID] >= len(buf) {
			return nil, 0, fmt.Errorf("size not enough")
		}
		strs := make([]string, size)
		var elemTypes Types
		for i := 0; i < size; i++ {
			elemType, l, e := Check(buf[length:], elemTypeID)
			if e != nil {
				err = e
				return
			}
			strs[i] = string(buf[length : length+l])
			length += l
			elemTypes = append(elemTypes, elemType)
		}
		// set element check
		if size >= 2 {
			sort.Strings(strs)
			if elemTypeID == thrift.BOOL {
				for i := 0; i < len(strs)-1; i++ {
					if strs[i] == string([]byte{1}) && strs[i+1] == string([]byte{1}) {
						return nil, 0, fmt.Errorf("set element duplicated")
					}
					if strs[i] != string([]byte{1}) && strs[i+1] != string([]byte{1}) {
						return nil, 0, fmt.Errorf("set element duplicated")
					}
				}
			} else {
				for i := 0; i < len(strs)-1; i++ {
					if strs[i] == strs[i+1] {
						return nil, 0, fmt.Errorf("set element duplicated")
					}
				}
			}
		}
		if size == 0 {
			elemTypes = append(elemTypes, &Type{TypeID: elemTypeID})
		}
		if elemTypes.Conflict() {
			return nil, 0, fmt.Errorf("set element type conflict")
		}
		// NOTE: considering combine types to one complete type
		typ.ValType = elemTypes[0]
		return
	case thrift.LIST:
		elemTypeID, size, l, e := thrift.Binary.ReadListBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		if !isValidType(elemTypeID) {
			return nil, 0, fmt.Errorf("unknown data type %d", elemTypeID)
		}
		if length+size*TypeSize[elemTypeID] >= len(buf) {
			return nil, 0, fmt.Errorf("size not enough")
		}
		var elemTypes Types
		for i := 0; i < size; i++ {
			elemType, l, e := Check(buf[length:], elemTypeID)
			if e != nil {
				err = e
				return
			}
			length += l
			elemTypes = append(elemTypes, elemType)
		}
		if size == 0 {
			elemTypes = append(elemTypes, &Type{TypeID: elemTypeID})
		}
		if elemTypes.Conflict() {
			return nil, 0, fmt.Errorf("list element type conflict")
		}
		// NOTE: considering combine types to one complete type
		typ.ValType = elemTypes[0]
		return
	default:
		return nil, 0, fmt.Errorf("unknown data type %d", fieldTypeID)
	}
}
