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
	"sort"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/kitex/pkg/protocol/bthrift"
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
	if t < thrift.BOOL || t > thrift.LIST || t == thrift.TType(9) || t == thrift.TType(5) {
		return false
	}
	return true
}

// Check checks if buf is valid thrift binary-buffered binary.
func Check(buf []byte, fieldType thrift.TType) (length int, err error) {
	var l int
	switch fieldType {
	case thrift.BOOL:
		_, l, err = bthrift.Binary.ReadBool(buf)
		length += l
		return
	case thrift.BYTE:
		_, l, err = bthrift.Binary.ReadByte(buf)
		length += l
		return
	case thrift.I16:
		_, l, err = bthrift.Binary.ReadI16(buf)
		length += l
		return
	case thrift.I32:
		_, l, err = bthrift.Binary.ReadI32(buf)
		length += l
		return
	case thrift.I64:
		_, l, err = bthrift.Binary.ReadI64(buf)
		length += l
		return
	case thrift.DOUBLE:
		_, l, err = bthrift.Binary.ReadDouble(buf)
		length += l
		return
	case thrift.STRING:
		_, l, err = bthrift.Binary.ReadString(buf)
		length += l
		return
	case thrift.STRUCT:
		_, l, err = bthrift.Binary.ReadStructBegin(buf)
		length += l
		if err != nil {
			return
		}
		ids := make([]int16, 0, 16)
		for {
			_, typeID, id, l, e := bthrift.Binary.ReadFieldBegin(buf[length:])
			length += l
			if e != nil {
				err = e
				return
			}
			if typeID == thrift.STOP {
				break
			}
			for _, old := range ids {
				if old == id {
					return 0, fmt.Errorf("duplicated field id")
				}
			}
			ids = append(ids, id)
			l, e = Check(buf[length:], typeID)
			length += l
			if e != nil {
				err = e
				return
			}
			l, e = bthrift.Binary.ReadFieldEnd(buf[length:])
			length += l
			if e != nil {
				err = e
				return
			}
		}
		l, e := bthrift.Binary.ReadStructEnd(buf[length:])
		length += l
		if e != nil {
			err = e
		}
		return
	case thrift.MAP:
		keyType, valueType, size, l, e := bthrift.Binary.ReadMapBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		if !isValidType(keyType) {
			return 0, fmt.Errorf("unknown data type %d", keyType)
		}
		if !isValidType(valueType) {
			return 0, fmt.Errorf("unknown data type %d", keyType)
		}
		if length+size*(TypeSize[keyType]+TypeSize[valueType]) >= len(buf) {
			return 0, fmt.Errorf("size not enough")
		}
		for i := 0; i < size; i++ {
			l, e := Check(buf[length:], keyType)
			length += l
			if e != nil {
				err = e
				return
			}
			l, e = Check(buf[length:], valueType)
			length += l
			if e != nil {
				err = e
				return
			}
		}
		l, e = bthrift.Binary.ReadMapEnd(buf[length:])
		length += l
		if e != nil {
			err = e
		}
		return
	case thrift.SET:
		elemType, size, l, e := bthrift.Binary.ReadSetBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		if !isValidType(elemType) {
			return 0, fmt.Errorf("unknown data type %d", elemType)
		}
		if length+size*TypeSize[elemType] >= len(buf) {
			return 0, fmt.Errorf("size not enough")
		}
		strs := make([]string, size)
		for i := 0; i < size; i++ {
			l, e = Check(buf[length:], elemType)
			strs[i] = string(buf[length : length+l])
			length += l
			if e != nil {
				err = e
				return
			}
		}
		if size >= 2 {
			sort.Strings(strs)
			for i := 0; i < len(strs)-2; i++ {
				if strs[i] == strs[i+1] {
					return 0, fmt.Errorf("set element duplicated")
				}
			}
		}
		l, e = bthrift.Binary.ReadSetEnd(buf[length:])
		length += l
		if e != nil {
			err = e
		}
		return
	case thrift.LIST:
		elemType, size, l, e := bthrift.Binary.ReadListBegin(buf)
		length += l
		if e != nil {
			err = e
			return
		}
		if !isValidType(elemType) {
			return 0, fmt.Errorf("unknown data type %d", elemType)
		}
		if length+size*TypeSize[elemType] >= len(buf) {
			return 0, fmt.Errorf("size not enough")
		}
		for i := 0; i < size; i++ {
			l, e = Check(buf[length:], elemType)
			length += l
			if e != nil {
				err = e
				return
			}
		}
		l, e = bthrift.Binary.ReadListEnd(buf[length:])
		length += l
		if e != nil {
			err = e
		}
		return
	default:
		return 0, fmt.Errorf("unknown data type %d", fieldType)
	}
}
