/*
 * Copyright 2024 CloudWeGo Authors
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

package reflect

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// only be used when NewRequiredFieldNotSetException
func lookupFieldName(rt reflect.Type, offset uintptr) string {
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Offset == offset {
			return f.Name
		}
	}
	return "unknown"
}

func withFieldErr(err error, sd *structDesc, f *tField) error {
	return fmt.Errorf("%q field %d err: %w", sd.Name(), f.ID, err)
}

func checkUniqueness(t *tType, h *sliceHeader) error {
	var uniq bool
	switch t.T {
	case tBOOL:
		uniq = checkUnique(unsafe.Slice((*bool)(h.Data), h.Len))
	case tI08:
		uniq = checkUnique(unsafe.Slice((*int8)(h.Data), h.Len))
	case tI16:
		uniq = checkUnique(unsafe.Slice((*int16)(h.Data), h.Len))
	case tI32:
		uniq = checkUnique(unsafe.Slice((*int32)(h.Data), h.Len))
	case tI64, tENUM:
		uniq = checkUnique(unsafe.Slice((*int64)(h.Data), h.Len))
	case tDOUBLE:
		uniq = checkUnique(unsafe.Slice((*float64)(h.Data), h.Len))
	case tSTRING:
		uniq = checkUnique(unsafe.Slice((*string)(h.Data), h.Len))
	default: // tSTRUCT?
		// NOTE: tSTRUCT is not always comparable, and it's not common for set
		return nil
	}
	if !uniq {
		return fmt.Errorf("%s error writing set field: slice is not unique", t.RT.String())
	}
	return nil
}

func checkUnique[T comparable](vv []T) bool {
	for i := range vv {
		for j := i + 1; j < len(vv); j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

type tmpMapVars struct {
	k  reflect.Value  // t.K.RT
	kp unsafe.Pointer // *t.K.RT

	v  reflect.Value  // t.V.RT
	vp unsafe.Pointer // *t.V.RT
}

func initOrGetMapTmpVarsPool(t *tType) *sync.Pool {
	if t.T != tMAP {
		return nil
	}
	return &sync.Pool{
		New: func() interface{} {
			m := &tmpMapVars{}
			m.k = reflect.New(t.K.RT)
			m.kp = m.k.UnsafePointer()
			m.k = m.k.Elem() // make m.k addressable with type t.K.RT
			m.v = reflect.New(t.V.RT)
			m.vp = m.v.UnsafePointer()
			m.v = m.v.Elem() // make m.v addressable with type t.V.RT
			return m
		},
	}
}

func appendUint16(b []byte, v uint16) []byte {
	return append(b,
		byte(v>>8),
		byte(v),
	)
}

func appendUint32(b []byte, v uint32) []byte {
	return append(b,
		byte(v>>24),
		byte(v>>16),
		byte(v>>8),
		byte(v),
	)
}

func appendUint64(b []byte, v uint64) []byte {
	return append(b,
		byte(v>>56),
		byte(v>>48),
		byte(v>>40),
		byte(v>>32),
		byte(v>>24),
		byte(v>>16),
		byte(v>>8),
		byte(v),
	)
}
