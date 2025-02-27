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
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
	"github.com/cloudwego/gopkg/protocol/thrift"
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

func (d *tDecoder) Decode(b []byte, base unsafe.Pointer, sd *structDesc, maxdepth int) (int, error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}
	var bs *bitset
	if len(sd.requiredFieldIDs) > 0 {
		bs = bitsetPool.Get().(*bitset)
		defer bitsetPool.Put(bs)
		for _, f := range sd.requiredFieldIDs {
			bs.unset(f)
		}
	}

	var ufs *unknownFields
	if sd.hasUnknownFields {
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

		f := sd.GetField(fid)
		if f == nil || f.Type.WT != tp {
			n, err := thrift.Binary.Skip(b[i:], thrift.TType(tp))
			if err != nil {
				return i, fmt.Errorf("skip unknown field %d of struct %s err: %w", fid, sd.rt.String(), err)
			}
			if ufs != nil {
				ufs.Add(i-fieldHeaderLen, n+fieldHeaderLen) // save off and sz, and copy later
			}
			i += n
			continue
		}
		p := unsafe.Add(base, f.Offset) // pointer to the field

		t := f.Type
		p = d.mallocIfPointer(t, p)
		if t.FixedSize > 0 {
			i += decodeFixedSizeTypes(t.T, b[i:], p)
		} else {
			var n int
			var err error
			if f.NoCopy {
				n, err = decodeStringNoCopy(t, b[i:], p)
			} else {
				n, err = d.decodeType(t, b[i:], p, maxdepth-1)
			}
			if err != nil {
				return i, fmt.Errorf("decode field %d of struct %s err: %w", fid, sd.rt.String(), err)
			}
			i += n
		}
		if bs != nil {
			bs.set(f.ID)
		}
	}
	for _, fid := range sd.requiredFieldIDs {
		if !bs.test(fid) {
			return i, newRequiredFieldNotSetException(lookupFieldName(sd.rt, sd.GetField(fid).Offset))
		}
	}
	if ufs != nil && ufs.Size() > 0 {
		*(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset)) = ufs.Copy(b)
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

func decodeStringNoCopy(t *tType, b []byte, p unsafe.Pointer) (i int, err error) {
	l := int(binary.BigEndian.Uint32(b))
	if l < 0 {
		err = errNegativeSize
		return
	}
	i += 4
	if l == 0 {
		if t.Tag == defs.T_binary {
			*(*[]byte)(p) = []byte{}
		} else {
			*(*string)(p) = ""
		}
		return
	}

	// assert len, panic if []byte shorter than expected.
	_ = b[i+l-1]

	if t.Tag == defs.T_binary {
		h := (*sliceHeader)(p)
		h.Data = uintptr(unsafe.Pointer(&b[i]))
		h.Len = l
		h.Cap = l
	} else { //  convert to str
		h := (*stringHeader)(p)
		h.Data = uintptr(unsafe.Pointer(&b[i]))
		h.Len = l
	}
	i += l
	return
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
		l := int(binary.BigEndian.Uint32(b))
		if l < 0 {
			return 0, errNegativeSize
		}
		i := 4
		if l == 0 {
			if t.Tag == defs.T_binary {
				*(*[]byte)(p) = []byte{}
			} else {
				*(*string)(p) = ""
			}
			return i, nil
		}

		// assert len, panic if []byte shorter than expected.
		_ = b[i+l-1]

		x := d.Malloc(l, 1, 0)
		if t.Tag == defs.T_binary {
			h := (*sliceHeader)(p)
			h.Data = uintptr(x)
			h.Len = l
			h.Cap = l
		} else { //  convert to str
			h := (*stringHeader)(p)
			h.Data = uintptr(x)
			h.Len = l
		}
		copyn(x, b[i:], l)
		i += l
		return i, nil
	case tMAP:
		// map header
		t0, t1, l := ttype(b[0]), ttype(b[1]), int(binary.BigEndian.Uint32(b[2:]))
		if l < 0 {
			return 0, errNegativeSize
		}

		// check types
		kt := t.K
		vt := t.V
		if t0 != kt.WT || t1 != vt.WT {
			return 0, newTypeMismatchKV(kt.WT, vt.WT, t0, t1)
		}

		// decode map

		// tmp vars
		// tmpk = decode(b)
		// tmpv = decode(b)
		// map[tmpk] = tmpv
		tmp := t.MapTmpVarsPool.Get().(*tmpMapVars)
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
		if kt.IsPointer && l > 0 {
			sliceK = d.Malloc(l*kt.V.Size, kt.V.Align, kt.V.MallocAbiType)
		}
		var sliceV unsafe.Pointer
		if vt.IsPointer && l > 0 {
			sliceV = d.Malloc(l*vt.V.Size, vt.V.Align, vt.V.MallocAbiType)
		}

		var n int
		var err error
		i := 6
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
				if n, err = d.decodeType(kt, b[i:], p, maxdepth-1); err != nil {
					break
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
				if n, err = d.decodeType(vt, b[i:], p, maxdepth-1); err != nil {
					break
				} else {
					i += n
				}
			}
			m.SetMapIndex(k, v)
		}
		t.MapTmpVarsPool.Put(tmp) // no defer, it may be in hot path
		return i, nil
	case tLIST, tSET: // NOTE: for tSET, it may be map in the future
		// list header
		tp, l := ttype(b[0]), int(binary.BigEndian.Uint32(b[1:]))
		if l < 0 {
			return 0, errNegativeSize
		}
		// check types
		et := t.V
		if et.WT != tp {
			return 0, newTypeMismatch(et.WT, tp)
		}

		i := 5

		// decode list
		h := (*sliceHeader)(p) // update the slice field
		if l <= 0 {
			h.Zero()
			return i, nil
		}
		x := d.Malloc(l*et.Size, et.Align, et.MallocAbiType) // malloc for slice. make([]Type, l, l)
		h.Data = uintptr(x)
		h.Len = l
		h.Cap = l

		// pre-allocate space for elements if they're pointers
		// like
		// v[i] = &sliceData[i]
		// instead of
		// v[i] = new(type)
		var sliceData unsafe.Pointer
		if et.IsPointer {
			sliceData = d.Malloc(l*et.V.Size, et.V.Align, et.V.MallocAbiType)
		}

		p = x // point to the 1st element, and then decode one by one
		for j := 0; j < l; j++ {
			if j != 0 {
				p = unsafe.Add(p, et.Size) // next element
			}
			vp := p // v[j]

			// p = &sliceData[j], see comment of sliceData above
			if et.IsPointer {
				if j != 0 {
					sliceData = unsafe.Add(sliceData, et.V.Size) // next
				}
				*(*unsafe.Pointer)(p) = sliceData // v[j] = &sliceData[i]
				vp = sliceData                    // &v[j]
			}

			if et.FixedSize > 0 {
				i += decodeFixedSizeTypes(et.T, b[i:], vp)
			} else {
				n, err := d.decodeType(et, b[i:], vp, maxdepth-1)
				if err != nil {
					return i, err
				}
				i += n
			}
		}
		return i, nil
	case tSTRUCT:
		if t.Sd.hasInitFunc {
			f := t.Sd.initFunc // copy on write, reuse itab of iface
			updateIface(unsafe.Pointer(&f), p)
			f.InitDefault()
		}
		return d.Decode(b, p, t.Sd, maxdepth-1)
	}
	return 0, fmt.Errorf("unknown type: %d", t.T)
}
