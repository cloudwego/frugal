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
	"encoding/binary"
	"errors"
	"unsafe"
)

type tEncoder struct {
	// ...
}

// Encode encodes a struct to buf.
// base is the pointer of the struct
func (e *tEncoder) Encode(b []byte, base unsafe.Pointer, fd *fieldDesc) (int, error) {
	if base == nil {
		// kitex will encode nil struct with a single byte tSTOP
		b[0] = byte(tSTOP)
		return 1, nil
	}
	i := 0
	for _, f := range fd.fields {
		t := f.Type
		p := unsafe.Add(base, f.Offset)
		if f.CanSkipEncodeIfNil && *(*unsafe.Pointer)(p) == nil {
			continue
		}
		if f.CanSkipIfDefault && t.Equal(f.Default, p) {
			continue
		}

		// field header
		b[i] = byte(t.WT)
		binary.BigEndian.PutUint16(b[i+1:], f.ID)
		i += fieldHeaderLen

		// field value
		if t.SimpleType { // fast path
			if t.IsPointer {
				p = *(*unsafe.Pointer)(p)
			}
			i += encodeSimpleTypes(t.T, b[i:], p)
		} else if t.T == tSTRUCT {
			// tSTRUCT always is pointer?
			n, err := e.Encode(b[i:], *(*unsafe.Pointer)(p), t.Fd)
			if err != nil {
				return i, err
			}
			i += n
		} else {
			n, err := e.encodeContainerType(t, b[i:], p)
			if err != nil {
				return i, err
			}
			i += n
		}
	}
	if fd.hasUnknownFields {
		xb := *(*[]byte)(unsafe.Add(base, fd.unknownFieldsOffset))
		if len(xb) > 0 {
			i += copy(b[i:], xb)
		}
	}
	b[i] = byte(tSTOP)
	i++
	return i, nil
}

func (e *tEncoder) encodeContainerType(t *tType, b []byte, p unsafe.Pointer) (int, error) {
	switch t.T {
	case tMAP:
		kt := t.K
		vt := t.V
		// map header
		b[0] = byte(kt.WT)
		b[1] = byte(vt.WT)
		if *(*unsafe.Pointer)(p) == nil {
			b[5], b[4], b[3], b[2] = 0, 0, 0, 0
			return mapHeaderLen, nil
		}
		binary.BigEndian.PutUint32(b[2:], uint32(maplen(*(*unsafe.Pointer)(p))))
		i := mapHeaderLen
		mv := rvWithPtr(t.RV, p)
		it := newMapIter(mv)
		for kp, vp := it.Next(); kp != nil; kp, vp = it.Next() {
			// Key
			// SimpleType or tSTRUCT
			if kt.SimpleType { // fast path
				i += encodeSimpleTypes(kt.T, b[i:], kp)
			} else {
				n, err := e.Encode(b[i:], *(*unsafe.Pointer)(kp), kt.Fd)
				if err != nil {
					return i, err
				}
				i += n
			}

			// Value
			if vt.SimpleType { // fast path
				i += encodeSimpleTypes(vt.T, b[i:], vp)
			} else if vt.T == tSTRUCT {
				// tSTRUCT always is pointer?
				n, err := e.Encode(b[i:], *(*unsafe.Pointer)(vp), vt.Fd)
				if err != nil {
					return i, err
				}
				i += n
			} else { // tLIST, tSET, tMAP, unlikely...
				n, err := e.encodeContainerType(vt, b[i:], vp)
				if err != nil {
					return i, err
				}
				i += n
			}
		}
		return i, nil
	case tLIST, tSET: // NOTE: for tSET, it may be map in the future
		// list header
		vt := t.V
		b[0] = byte(vt.WT)
		if *(*unsafe.Pointer)(p) == nil {
			b[4], b[3], b[2], b[1] = 0, 0, 0, 0
			return listHeaderLen, nil
		}
		h := (*sliceHeader)(p)
		if t.T == tSET { // for tSET, check duplicated items
			if err := checkUniqueness(t.V, h); err != nil {
				return listHeaderLen, err
			}
		}
		binary.BigEndian.PutUint32(b[1:], uint32(h.Len))
		i := listHeaderLen
		vp := h.Data
		// list elements
		for j := 0; j < h.Len; j++ {
			if j != 0 {
				vp = unsafe.Add(vp, vt.Size) // move to next element
			}
			if vt.SimpleType { // fast path
				i += encodeSimpleTypes(vt.T, b[i:], vp)
			} else if vt.T == tSTRUCT {
				// tSTRUCT always is pointer?
				n, err := e.Encode(b[i:], *(*unsafe.Pointer)(vp), vt.Fd)
				if err != nil {
					return i, err
				}
				i += n
			} else { // tLIST, tSET, tMAP, unlikely...
				n, err := e.encodeContainerType(vt, b[i:], vp)
				if err != nil {
					return i, err
				}
				i += n
			}
		}

		return i, nil
	}
	return 0, errors.New("unknown type")
}

// NOTE: PLEASE ADD CODE CAREFULLY
// can inline encodeSimpleTypes with cost 78 (budget 80)
func encodeSimpleTypes(t ttype, b []byte, p unsafe.Pointer) int {
	switch t {
	case tBYTE, tBOOL:
		b[0] = *((*byte)(p)) // for tBOOL, true -> 1, false -> 0
		return 1
	case tI16:
		binary.BigEndian.PutUint16(b, uint16(*((*int16)(p))))
		return 2
	case tI32:
		binary.BigEndian.PutUint32(b, uint32(*((*int32)(p))))
		return 4
	case tENUM:
		binary.BigEndian.PutUint32(b, uint32(*((*int64)(p))))
		return 4
	case tI64, tDOUBLE:
		binary.BigEndian.PutUint64(b, *((*uint64)(p)))
		return 8
	case tSTRING:
		x := *((*string)(p))
		binary.BigEndian.PutUint32(b, uint32(len(x)))
		return 4 + copy(b[4:], x)
	}
	panic("bug")
}
