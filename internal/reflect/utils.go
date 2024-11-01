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
	"runtime"
	"sync"
	"unsafe"
)

// copyn copies n bytes from src to dst addr.
// it mainly used by decoder of tSTRING
func copyn(dst unsafe.Pointer, src []byte, n int) {
	var b []byte
	hdr := (*sliceHeader)(unsafe.Pointer(&b))
	hdr.Data = uintptr(dst)
	hdr.Cap = n
	hdr.Len = n
	copy(b, src)
	runtime.KeepAlive(dst)
}

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
		var vv []bool
		*(*sliceHeader)(unsafe.Pointer(&vv)) = *h
		uniq = checkUniquenessBool(vv)
	case tI08:
		var vv []int8
		*(*sliceHeader)(unsafe.Pointer(&vv)) = *h
		uniq = checkUniquenessInt8(vv)
	case tI16:
		var vv []int16
		*(*sliceHeader)(unsafe.Pointer(&vv)) = *h
		uniq = checkUniquenessInt16(vv)
	case tI32:
		var vv []int32
		*(*sliceHeader)(unsafe.Pointer(&vv)) = *h
		uniq = checkUniquenessInt32(vv)
	case tI64, tENUM:
		var vv []int64
		*(*sliceHeader)(unsafe.Pointer(&vv)) = *h
		uniq = checkUniquenessInt64(vv)
	case tDOUBLE:
		var vv []float64
		*(*sliceHeader)(unsafe.Pointer(&vv)) = *h
		uniq = checkUniquenessFloat64(vv)
	case tSTRING:
		var ss []string
		*(*sliceHeader)(unsafe.Pointer(&ss)) = *h
		uniq = checkUniquenessString(ss)
	default: // tSTRUCT?
		// NOTE: tSTRUCT is not always comparable, and it's not common for set
		return nil
	}
	if !uniq {
		return fmt.Errorf("%s error writing set field: slice is not unique", t.RT.String())
	}
	return nil
}

// XXX: checkUniqueness funcs use generics when we start to use newer go versions.
func checkUniquenessBool(vv []bool) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

func checkUniquenessInt8(vv []int8) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

func checkUniquenessInt16(vv []int16) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

func checkUniquenessInt32(vv []int32) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

func checkUniquenessInt64(vv []int64) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

func checkUniquenessFloat64(vv []float64) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			if vv[i] == vv[j] {
				return false
			}
		}
	}
	return true
}

func checkUniquenessString(vv []string) bool {
	l := len(vv)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
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
