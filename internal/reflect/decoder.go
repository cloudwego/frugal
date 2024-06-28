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
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/cloudwego/frugal/internal/binary/defs"
)

const maxDepthLimit = 1023

var decoderPool = sync.Pool{
	New: func() interface{} {
		d := &tDecoder{}
		d.s.init()
		return d
	},
}

type tDecoder struct {
	// noscan span
	// for bool, int8, int16, int32, int64, float64
	// for string, we only use it for (*sliceHeader).Data, not for []string, coz it contains pointer
	s span
}

func (d *tDecoder) Malloc(n, align int, abiType uintptr) unsafe.Pointer {
	if n > defaultDecoderMemSize/8 || abiType != 0 {
		// too large, or it needs GC to scan (MallocAbiType != 0 of tType)
		return mallocgc(uintptr(n), unsafe.Pointer(abiType), abiType != 0)
	}
	return d.s.Malloc(n, align) // only for noscan objects like string.Data, []int etc...
}

func (d *tDecoder) mallocIfPointer(t *tType, p unsafe.Pointer) (ret unsafe.Pointer) {
	if t.IsPointer {
		// we need to malloc the type first before assigning a value to it
		ret = d.Malloc(t.V.Size, t.V.Align, t.V.MallocAbiType)
		*(*unsafe.Pointer)(p) = ret // *p = new(type)
		return
	}
	return p
}

func (d *tDecoder) Decode(b []byte, base unsafe.Pointer, fd *fieldDesc, maxdepth int) (int, error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}
	var bs *bitset
	if len(fd.requiredFields) > 0 {
		bs = bitsetPool.Get().(*bitset)
		defer bitsetPool.Put(bs)
		for _, f := range fd.requiredFields {
			bs.unset(f.ID)
		}
	}

	var ufs *unknownFields
	if fd.hasUnknownFields {
		ufs = unknownFieldsPool.Get().(*unknownFields)
		defer unknownFieldsPool.Put(ufs)
		ufs.Reset()
	}

	i := 0
	for {
		tp := ttype(b[i])
		i++
		if tp == tSTOP {
			break
		}
		fid := binary.BigEndian.Uint16(b[i:])
		i += 2

		f := fd.GetField(fid)
		if f == nil {
			n, err := skipType(tp, b[i:], maxdepth-1)
			if err != nil {
				return i, fmt.Errorf("skip unknown field %d of struct %s err: %w", fid, fd.rt.String(), err)
			}
			if ufs != nil {
				ufs.Add(i-fieldHeaderLen, n+fieldHeaderLen) // save off and sz, and copy later
			}
			i += n
			continue
		}
		t := f.Type
		if t.WT != tp {
			return i, errors.New("type mismatch")
		}
		p := unsafe.Add(base, f.Offset) // pointer to the field
		p = d.mallocIfPointer(t, p)
		if t.FixedSize > 0 {
			i += decodeFixedSizeTypes(t.T, b[i:], p)
		} else {
			n, err := d.decodeType(t, b[i:], p, maxdepth-1)
			if err != nil {
				return i, fmt.Errorf("decode field %d of struct %s err: %w", fid, fd.rt.String(), err)
			}
			i += n
		}
		if bs != nil {
			bs.set(f.ID)
		}
	}
	for _, f := range fd.requiredFields {
		if !bs.test(f.ID) {
			return i, newRequiredFieldNotSetException(lookupFieldName(fd.rt, f.Offset))
		}
	}
	if ufs != nil && ufs.Size() > 0 {
		*(*[]byte)(unsafe.Add(base, fd.unknownFieldsOffset)) = ufs.Copy(b)
	}
	return i, nil
}

func decodeFixedSizeTypes(t ttype, b []byte, p unsafe.Pointer) int {
	switch t {
	case tBOOL, tBYTE:
		*(*byte)(p) = b[0] // XXX: for tBOOL 1->true, 2->true/false
		return 1
	case tDOUBLE, tI64:
		*(*uint64)(p) = binary.BigEndian.Uint64(b)
		return 8
	case tI16:
		*(*int16)(p) = int16(binary.BigEndian.Uint16(b))
		return 2
	case tI32:
		*(*int32)(p) = int32(binary.BigEndian.Uint32(b))
		return 4
	case tENUM:
		*(*int64)(p) = int64(int32(binary.BigEndian.Uint32(b)))
		return 4
	default:
		panic("bug")
	}
}

func (d *tDecoder) decodeType(t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}
	if t.FixedSize > 0 {
		return decodeFixedSizeTypes(t.T, b, p), nil
	}
	switch t.T {
	case tSTRING:
		i := 0
		l := int(binary.BigEndian.Uint32(b))
		i += 4
		if l == 0 {
			return i, nil
		}
		x := d.Malloc(l, 1, 0)
		if t.Tag == defs.T_binary {
			h := (*sliceHeader)(p)
			h.Data = x
			h.Len = l
			h.Cap = l
		} else { //  convert to str
			h := (*stringHeader)(p)
			h.Data = x
			h.Len = l
		}
		copyn(x, b[i:], l)
		i += l
		return i, nil
	case tMAP:
		// map header
		t0, t1, l := ttype(b[0]), ttype(b[1]), int(binary.BigEndian.Uint32(b[2:]))
		i := 6

		// check types
		kt := t.K
		vt := t.V
		if t0 != kt.T || t1 != vt.T {
			return 0, errors.New("type mismatch")
		}

		// decode map

		// tmp vars
		// tmpk = decode(b)
		// tmpv = decode(b)
		// map[tmpk] = tmpv
		tmp := t.MapTmpVarsPool.Get().(*tmpMapVars)
		defer t.MapTmpVarsPool.Put(tmp)
		k := tmp.k
		v := tmp.v
		kp := tmp.kp
		vp := tmp.vp
		m := reflect.MakeMapWithSize(t.RT, l)
		*((*uintptr)(p)) = m.Pointer() // p = make(t.RT, l)

		// pre-allocate space for keys and values if they're pointers
		// like
		// kp = &sliceK[i], decode(b, kp)
		// instead of
		// kp = new(type), decode(b, kp)
		var sliceK unsafe.Pointer
		if kt.IsPointer {
			sliceK = d.Malloc(l*kt.V.Size, kt.V.Align, kt.V.MallocAbiType)
		}
		var sliceV unsafe.Pointer
		if vt.IsPointer {
			sliceV = d.Malloc(l*vt.V.Size, vt.V.Align, vt.V.MallocAbiType)
		}

		for j := 0; j < l; j++ {
			p = kp
			if kt.IsPointer { // p = &sliceK[j]
				if j != 0 {
					sliceK = unsafe.Add(sliceK, kt.V.Size) // next
				}
				*(*unsafe.Pointer)(p) = sliceK
				p = sliceK
			}
			if kt.FixedSize > 0 {
				i += decodeFixedSizeTypes(kt.T, b[i:], p)
			} else {
				if n, err := d.decodeType(kt, b[i:], p, maxdepth-1); err != nil {
					return i, err
				} else {
					i += n
				}
			}
			p = vp
			if vt.IsPointer { // p = &sliceV[j]
				if j != 0 { // next
					sliceV = unsafe.Add(sliceV, vt.V.Size)
				}
				*(*unsafe.Pointer)(p) = sliceV
				p = sliceV
			}
			if vt.FixedSize > 0 {
				i += decodeFixedSizeTypes(vt.T, b[i:], p)
			} else {
				if n, err := d.decodeType(vt, b[i:], p, maxdepth-1); err != nil {
					return i, err
				} else {
					i += n
				}
			}
			m.SetMapIndex(k, v)
		}
		return i, nil
	case tLIST, tSET: // NOTE: for tSET, it may be map in the future
		// list header
		tp, l := ttype(b[0]), int(binary.BigEndian.Uint32(b[1:]))
		i := 5

		// check types
		et := t.V
		if et.T != tp {
			return 0, errors.New("type mismatch")
		}

		// decode list
		h := (*sliceHeader)(p) // update the slice field
		h.Data = unsafe.Pointer(nil)
		h.Len = l
		h.Cap = l
		if l <= 0 {
			return i, nil
		}
		x := d.Malloc(l*et.Size, et.Align, et.MallocAbiType) // malloc for slice. make([]Type, l, l)
		h.Data = x
		p = x // point to the 1st element, and then decode one by one
		for j := 0; j < l; j++ {
			vp := d.mallocIfPointer(et, p)
			if et.FixedSize > 0 {
				i += decodeFixedSizeTypes(et.T, b[i:], vp)
			} else {
				n, err := d.decodeType(et, b[i:], vp, maxdepth-1)
				if err != nil {
					return i, err
				}
				i += n
			}
			if j != l-1 {
				p = unsafe.Add(p, et.Size) // next element
			}
		}
		return i, nil
	case tSTRUCT:
		if t.Fd.hasInitFunc {
			f := t.Fd.initFunc // copy on write, reuse itab of iface
			updateIface(unsafe.Pointer(&f), p)
			f.InitDefault()
		}
		return d.Decode(b, p, t.Fd, maxdepth-1)
	}
	return 0, fmt.Errorf("unknown type: %d", t.T)
}

func skipstr(b []byte) int {
	return 4 + int(binary.BigEndian.Uint32(b))
}

// skipType skips over the value for the given type using Go implementation.
func skipType(t ttype, b []byte, maxdepth int) (int, error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}
	if n := typeToSize[t]; n > 0 {
		return int(n), nil
	}
	switch t {
	case tSTRING:
		return skipstr(b), nil
	case tMAP:
		i := 6
		kt, vt, sz := ttype(b[0]), ttype(b[1]), int32(binary.BigEndian.Uint32(b[2:]))
		if sz < 0 {
			return 0, errInvalidData
		}
		ksz, vsz := int(typeToSize[kt]), int(typeToSize[vt])
		if ksz > 0 && vsz > 0 {
			return i + (int(sz) * (ksz + vsz)), nil
		}
		for j := int32(0); j < sz; j++ {
			if ksz > 0 {
				i += ksz
			} else if kt == tSTRING {
				i += skipstr(b[i:])
			} else if n, err := skipType(kt, b[i:], maxdepth-1); err != nil {
				return i, err
			} else {
				i += n
			}
			if vsz > 0 {
				i += vsz
			} else if vt == tSTRING {
				i += skipstr(b[i:])
			} else if n, err := skipType(vt, b[i:], maxdepth-1); err != nil {
				return i, err
			} else {
				i += n
			}
		}
		return i, nil
	case tLIST, tSET:
		i := 5
		vt, sz := ttype(b[0]), int32(binary.BigEndian.Uint32(b[1:]))
		if sz < 0 {
			return 0, errInvalidData
		}
		if typeToSize[vt] > 0 {
			return i + int(sz)*int(typeToSize[vt]), nil
		}
		for j := int32(0); j < sz; j++ {
			if vt == tSTRING {
				i += skipstr(b[i:])
			} else if n, err := skipType(vt, b[i:], maxdepth-1); err != nil {
				return i, err
			} else {
				i += n
			}
		}
		return i, nil
	case tSTRUCT:
		i := 0
		for {
			ft := ttype(b[i])
			i += 1 // ttype
			if ft == tSTOP {
				return i, nil
			}
			i += 2 // Field ID
			if typeToSize[ft] > 0 {
				i += int(typeToSize[ft])
			} else if n, err := skipType(ft, b[i:], maxdepth-1); err != nil {
				return i, err
			} else {
				i += n
			}
		}
	default:
		return 0, newUnknownDataTypeException(t)
	}
}
