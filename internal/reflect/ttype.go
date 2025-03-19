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
	"reflect"
	"strconv"
	"sync"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
)

type ttype uint8

const (
	tSTOP   ttype = 0
	tVOID   ttype = 1
	tBOOL   ttype = 2
	tBYTE   ttype = 3
	tI08    ttype = 3
	tDOUBLE ttype = 4
	tI16    ttype = 6
	tI32    ttype = 8
	tI64    ttype = 10
	tSTRING ttype = 11
	tUTF7   ttype = 11
	tSTRUCT ttype = 12
	tMAP    ttype = 13
	tSET    ttype = 14
	tLIST   ttype = 15
	tUTF8   ttype = 16
	tUTF16  ttype = 17

	// internal use only
	tENUM ttype = 0xfe // XXX: kitex issue, int64, but encode as int32 ...

	// tOTHER, tBINARY, tPTR will not be used when encoding or decoding.
	// It's only for generating code
	tOTHER  = ttype(0xe0) // mapping list, set, map, struct to tOTHER reusing the same func
	tBINARY = ttype(0xe1) // for map type like map[int][]byte when decoding
	tPTR    = ttype(0xe2) // for map type like map[int]unsafe.Pointer
)

var t2s = [256]string{
	tBOOL:   "tBOOL",
	tI08:    "tI08",
	tI16:    "tI16",
	tI32:    "tI32",
	tI64:    "tI64",
	tDOUBLE: "tDOUBLE",
	tSTRING: "tSTRING",
	tSTRUCT: "tSTRUCT",
	tMAP:    "tMAP",
	tSET:    "tSET",
	tLIST:   "tLIST",
	tENUM:   "tENUM",

	tBINARY: "tBINARY",
	tOTHER:  "tOTHER",
	tPTR:    "tPTR",
}

func ttype2wiretype(t ttype) ttype {
	if t == tENUM {
		return tI32
	}
	return t
}

func ttype2str(t ttype) string {
	ret := t2s[t]
	if ret == "" {
		return "unknown[" + strconv.Itoa(int(t)) + "]"
	}
	return ret
}

var simpleTypes = [256]bool{
	tBOOL:   true,
	tBYTE:   true,
	tDOUBLE: true,
	tI16:    true,
	tI32:    true,
	tI64:    true,
	tENUM:   true,
	tSTRING: true,
}

type appendFuncType func(t *tType, b []byte, p unsafe.Pointer) ([]byte, error)

type decodeFuncType func(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error)

type tType struct {
	T ttype
	K *tType
	V *tType

	WT ttype // wiretype tNUM -> tI32

	Tag defs.Tag

	RT    reflect.Type
	Size  int
	Align int

	// for Malloc
	MallocAbiType uintptr // 0 if a type contains no pointer

	// tmp var for reflect.Type, use `rvWithPtr` to copy-on-write
	// only used for newMapIter
	RV reflect.Value

	IsPointer  bool // true if t.Tag == defs.T_pointer
	SimpleType bool // true if simpleTypes[t.T]
	FixedSize  int  // typeToSize[t.T]

	// for tSTRUCT
	Sd *structDesc

	// for tLIST, tSET, tMAP, tSTRUCT
	EncodedSizeFunc func(p unsafe.Pointer) (int, error)
	AppendFunc      appendFuncType
	DecodeFunc      decodeFuncType

	// tMAP only
	MapTmpVarsPool *sync.Pool // for decoder tmp vars
}

// Equal returns true if data of two pointers point to.
func (t *tType) Equal(p0, p1 unsafe.Pointer) bool {
	switch t.T {
	case tBOOL:
		return *(*bool)(p0) == *(*bool)(p1)
	case tBYTE:
		return *(*int8)(p0) == *(*int8)(p1)
	case tDOUBLE:
		return *(*float64)(p0) == *(*float64)(p1)
	case tI16:
		return *(*int16)(p0) == *(*int16)(p1)
	case tI32:
		return *(*int32)(p0) == *(*int32)(p1)
	case tI64, tENUM:
		return *(*int64)(p0) == *(*int64)(p1)
	case tSTRING:
		return *(*string)(p0) == *(*string)(p1)
	}
	return false
}

type ttypesK struct {
	T defs.Tag
	S reflect.Type
}

var ttypes = map[ttypesK]*tType{} // cache for less in-use objects

func newTType(x *defs.Type) *tType {
	k := ttypesK{T: x.T, S: x.S}
	if t := ttypes[k]; t != nil {
		return t
	}
	t := &tType{}

	// newTType is always succuess, update cache as soon as it's created
	// this func is called with lock, no need to add a additional one.
	ttypes[k] = t

	t.T = ttype(x.Tag())
	t.WT = t.T
	t.Tag = x.T
	if x.IsEnum() {
		t.T = tENUM
	}
	t.RT = x.S
	t.Size = int(x.S.Size())
	t.Align = x.S.Align()

	switch t.RT.Kind() {
	case reflect.Array, reflect.Map, reflect.Ptr, reflect.Slice, reflect.String, reflect.Struct:
		t.MallocAbiType = rtTypePtr(t.RT) // pass to mallocgc
	}

	if t.T == tMAP {
		t.RV = reflect.New(t.RT) // alloc on heap, make it addressable
		t.RV = t.RV.Elem()
		t.MapTmpVarsPool = initOrGetMapTmpVarsPool(t)
	}
	t.IsPointer = t.Tag == defs.T_pointer
	t.SimpleType = simpleTypes[t.T]
	t.FixedSize = int(typeToSize[t.T])

	if x.K != nil {
		t.K = newTType(x.K)
	}
	if x.V != nil {
		t.V = newTType(x.V)
	}

	switch t.T {
	case tSTRING:
		t.DecodeFunc = decoderString

	case tMAP:
		t.EncodedSizeFunc = t.encodedMapSize
		t.DecodeFunc = decodeMapAny

	case tLIST, tSET:
		t.EncodedSizeFunc = t.encodedListSize
		t.DecodeFunc = decodeListAny

	case tSTRUCT:
		t.EncodedSizeFunc = t.EncodedSize
		t.DecodeFunc = decodeStruct
	}

	switch t.T {
	case tLIST, tSET:
		updateListAppendFunc(t)
	case tMAP:
		updateMapAppendFunc(t)
		updateMapDecodeFunc(t)
	case tSTRUCT:
		t.AppendFunc = appendStruct
	default:
		t.AppendFunc = appendAny
	}
	if t.IsPointer && t.V.IsPointer {
		// XXX: make it simple... not support it, it's not common
		// The code is not generated by thriftgo?
		// it makes thing complicated, coz we can only get the data recursively.
		panic("doesn't support multilevel pointers like **p")
	}
	return t
}

const (
	fieldHeaderLen = 1 + 2     // type + id
	mapHeaderLen   = 1 + 1 + 4 // k type, v type, map len
	listHeaderLen  = 1 + 4     // elem type, list len
	strHeaderLen   = 4         // str len
)

func encodedStringSize(p unsafe.Pointer) int {
	// string type in list or map, it's always non-pointer
	// so we no need to do the check of t.IsPointer
	return strHeaderLen + len(*(*string)(p))
}

func (t *tType) EncodedSize(base unsafe.Pointer) (int, error) {
	sd := t.Sd
	if t.T != 0 { // not from reflect.EncodedSize
		// for field of a struct, value of a map, or elem of a list,
		// it's a pointer to struct pointer, then we have to convert it to struct pointer
		base = *(*unsafe.Pointer)(base)
	}
	if base == nil {
		return 1, nil // tSTOP
	}
	ret := sd.fixedLenFieldSize
	for _, i := range sd.varLenFields {
		f := sd.fields[i]
		p := unsafe.Add(base, f.Offset)
		if f.CanSkipEncodeIfNil && *(*unsafe.Pointer)(p) == nil {
			continue
		}
		t := f.Type
		if f.CanSkipIfDefault && t.Equal(f.Default, p) {
			continue
		}
		if n := t.FixedSize; n > 0 {
			ret += (fieldHeaderLen + int(n))
			// fast skip types like tBOOL, tBYTE, tDOUBLE, tI16, tI32, tI64
			continue
		}
		if t.T == tSTRING {
			if t.IsPointer {
				p = *(*unsafe.Pointer)(p)
			}
			ret += fieldHeaderLen + encodedStringSize(p)
			continue
		}
		ret += fieldHeaderLen
		n, err := t.EncodedSizeFunc(p) // tLIST, tSET, tMAP, tSTRUCT
		if err != nil {
			return ret, err
		}
		ret += n
	}
	if sd.hasUnknownFields {
		ret += len(*(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset)))
	}
	ret += 1 // tSTOP
	return ret, nil
}

func (t *tType) encodedMapSize(p unsafe.Pointer) (int, error) {
	if *(*unsafe.Pointer)(p) == nil {
		// We always encode nil map for required or default requiredness
		return mapHeaderLen, nil // 0-len map
	}

	kt, doneK := t.K, false
	vt, doneV := t.V, false
	l := maplen(*(*unsafe.Pointer)(p))
	if l == 0 {
		return mapHeaderLen, nil // 0-len map
	}
	ret := mapHeaderLen
	if kt.FixedSize > 0 {
		ret += l * kt.FixedSize
		doneK = true
	}
	if vt.FixedSize > 0 {
		ret += l * vt.FixedSize
		doneV = true
	}
	if doneK && doneV {
		return ret, nil // fast path
	}

	// we already skipped primitive types.
	// need to handle tSTRING, tMAP, tLIST, tSET or tSTRUCT
	it := newMapIter(rvWithPtr(t.RV, p))
	for kp, vp := it.Next(); kp != nil; kp, vp = it.Next() {
		// Key
		// tSTRING, tSTRUCT
		if !doneK {
			if kt.T == tSTRING {
				ret += encodedStringSize(kp)
			} else {
				n, err := kt.EncodedSize(kp)
				if err != nil {
					return ret, err
				}
				ret += n
			}
		}
		if doneV {
			continue
		}
		// Value
		// tSTRING, tMAP, tLIST, tSET or tSTRUCT
		if vt.T == tSTRING {
			ret += encodedStringSize(vp)
		} else {
			n, err := vt.EncodedSizeFunc(vp)
			if err != nil {
				return ret, err
			}
			ret += n
		}
	}

	return ret, nil
}

func (t *tType) encodedListSize(p unsafe.Pointer) (int, error) {
	if *(*unsafe.Pointer)(p) == nil {
		return listHeaderLen, nil // 0-len list
	}
	vt := t.V
	h := (*sliceHeader)(p)
	if vt.FixedSize > 0 {
		return listHeaderLen + (h.Len * vt.FixedSize), nil
	}
	ret := listHeaderLen
	if h.Len == 0 {
		return ret, nil
	}
	vp := h.Data
	for i := 0; i < h.Len; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, vt.Size) //  move to next element
		}
		if vt.T == tSTRING {
			ret += encodedStringSize(vp)
		} else {
			n, err := vt.EncodedSizeFunc(vp)
			if err != nil {
				return ret, err
			}
			ret += n
		}
	}
	return ret, nil
}
