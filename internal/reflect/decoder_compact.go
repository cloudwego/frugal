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
	"io"
	"reflect"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
)

var minWireSizeCompact = [256]int8{
	tBOOL:   1,
	tBYTE:   1,
	tI16:    1,
	tI32:    1,
	tI64:    1,
	tDOUBLE: 8,
	tSTRING: 1,
	tSTRUCT: 1,
	tMAP:    1,
	tSET:    1,
	tLIST:   1,
	tENUM:   1,
}

func (d *tDecoder) DecodeCompact(b []byte, base unsafe.Pointer, sd *structDesc, maxdepth int) (int, error) {
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

	var lastId uint16
	i := 0
	for {
		wt, fid, n := readCompactFieldHeader(b, i, lastId)
		i += n
		if wt == ctSTOP {
			break
		}
		if fid <= lastId {
			return i, fmt.Errorf("compact protocol: field id %d out of order (last %d)", fid, lastId)
		}
		lastId = fid

		f := sd.GetField(fid)
		if f == nil {
			headerLen := n // bytes consumed by readCompactFieldHeader
			var valLen int
			// Bool values are inlined in the field header in Compact Protocol;
			// there are zero value bytes to skip.
			if wt == ctBOOL_TRUE || wt == ctBOOL_FALSE {
				valLen = 0
			} else {
				var err error
				valLen, err = skipCompactValue(b[i:], wt)
				if err != nil {
					return i, fmt.Errorf("skip unknown field %d of struct %s err: %w", fid, sd.rt.String(), err)
				}
			}
			if ufs != nil {
				ufs.Add(i-headerLen, valLen+headerLen)
			}
			i += valLen
			continue
		}

		p := unsafe.Add(base, f.Offset)
		t := f.Type

		if t.T == tBOOL {
			p = d.mallocIfPointer(t, p)
			switch wt {
			case ctBOOL_TRUE:
				*(*bool)(p) = true
			case ctBOOL_FALSE:
				*(*bool)(p) = false
			default:
				return i, newTypeMismatch(t.WT, wt)
			}
			if bs != nil {
				bs.set(f.ID)
			}
			continue
		}

		expectedWt := binary2CompactWireType[t.T]
		if wt != expectedWt {
			headerLen := n
			valLen, e := skipCompactValue(b[i:], wt)
			if e != nil {
				return i, e
			}
			if ufs != nil {
				ufs.Add(i-headerLen, valLen+headerLen)
			}
			i += valLen
			continue
		}

		p = d.mallocIfPointer(t, p)
		if i >= len(b) {
			return i, io.ErrShortBuffer
		}
		if t.FixedSize > 0 && t.T == tBYTE {
			*(*byte)(p) = b[i]
			i += 1
		} else if t.FixedSize > 0 && t.T == tDOUBLE {
			if len(b)-i < 8 {
				return i, io.ErrShortBuffer
			}
			*(*uint64)(p) = readUint64LE(b[i:])
			i += 8
		} else if f.NoCopy && t.T == tSTRING {
			n, err := decodeStringNoCopyCompact(t, b[i:], p)
			if err != nil {
				return i, fmt.Errorf("decode field %d of struct %s err: %w", fid, sd.rt.String(), err)
			}
			i += n
		} else {
			n, err := d.decodeTypeCompact(t, b[i:], p, maxdepth-1)
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

func decodeStringNoCopyCompact(t *tType, b []byte, p unsafe.Pointer) (int, error) {
	l64, vn := decodeVarint(b)
	l := int(l64)
	if l < 0 {
		return 0, errNegativeSize
	}
	i := vn
	if l == 0 {
		if t.Tag == defs.T_binary {
			*(*[]byte)(p) = []byte{}
		} else {
			*(*string)(p) = ""
		}
		return i, nil
	}
	if l > len(b)-i {
		return i, newSizeExceedsBufferException(l, len(b)-i)
	}
	if t.Tag == defs.T_binary {
		*(*[]byte)(p) = unsafe.Slice(&b[i], l)
	} else {
		*(*string)(p) = unsafe.String(&b[i], l)
	}
	return i + l, nil
}

func (d *tDecoder) decodeTypeCompact(t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}
	switch t.T {
	case tBYTE:
		if len(b) < 1 {
			return 0, io.ErrShortBuffer
		}
		*(*byte)(p) = b[0]
		return 1, nil
	case tBOOL:
		if len(b) < 1 {
			return 0, io.ErrShortBuffer
		}
		switch ttype(b[0]) {
		case ctBOOL_TRUE:
			*(*bool)(p) = true
		case ctBOOL_FALSE:
			*(*bool)(p) = false
		default:
			return 0, newTypeMismatch(t.WT, ttype(b[0]))
		}
		return 1, nil
	case tI16:
		v, n := decodeZigzag32(b)
		*(*int16)(p) = int16(v)
		return n, nil
	case tI32:
		v, n := decodeZigzag32(b)
		*(*int32)(p) = v
		return n, nil
	case tENUM:
		v, n := decodeZigzag32(b)
		*(*int64)(p) = int64(v)
		return n, nil
	case tI64:
		v, n := decodeZigzag64(b)
		*(*int64)(p) = v
		return n, nil
	case tDOUBLE:
		if len(b) < 8 {
			return 0, io.ErrShortBuffer
		}
		*(*uint64)(p) = readUint64LE(b)
		return 8, nil
	case tSTRING:
		if len(b) < 1 {
			return 0, io.ErrShortBuffer
		}
		l64, vn := decodeVarint(b)
		l := int(l64)
		if l < 0 {
			return 0, errNegativeSize
		}
		i := vn
		if l == 0 {
			if t.Tag == defs.T_binary {
				*(*[]byte)(p) = []byte{}
			} else {
				*(*string)(p) = ""
			}
			return i, nil
		}
		if l > len(b)-i {
			return i, newSizeExceedsBufferException(l, len(b)-i)
		}
		x := d.Malloc(l, 1, 0)
		if t.Tag == defs.T_binary {
			*(*[]byte)(p) = unsafe.Slice((*byte)(x), l)
		} else {
			*(*string)(p) = unsafe.String((*byte)(x), l)
		}
		copy(unsafe.Slice((*byte)(x), l), b[i:])
		i += l
		return i, nil
	case tSTRUCT:
		if t.Sd.hasInitFunc {
			f := t.Sd.initFunc
			updateIface(unsafe.Pointer(&f), p)
			f.InitDefault()
		}
		return d.DecodeCompact(b, p, t.Sd, maxdepth-1)
	case tLIST, tSET:
		return d.decodeListCompact(t, b, p, maxdepth-1)
	case tMAP:
		return d.decodeMapCompact(t, b, p, maxdepth-1)
	default:
		return 0, fmt.Errorf("compact decode: unknown type %d", t.T)
	}
}

func skipCompactValue(b []byte, wt ttype) (int, error) {
	switch wt {
	case ctBOOL_TRUE, ctBOOL_FALSE, ctI08:
		return 1, nil
	case ctI16, ctI32, ctI64:
		_, n := decodeVarint(b)
		return n, nil
	case ctDOUBLE:
		if len(b) < 8 {
			return 0, io.ErrShortBuffer
		}
		return 8, nil
	case ctBINARY:
		l64, vn := decodeVarint(b)
		l := int(l64)
		if l < 0 {
			return 0, errNegativeSize
		}
		if l > len(b)-vn {
			return vn, newSizeExceedsBufferException(l, len(b)-vn)
		}
		return vn + l, nil
	case ctSTRUCT:
		return skipCompactStruct(b)
	case ctLIST, ctSET:
		return skipCompactList(b)
	case ctMAP:
		return skipCompactMap(b)
	default:
		return 0, fmt.Errorf("skip: unknown compact wire type %d", wt)
	}
}

func skipCompactStruct(b []byte) (int, error) {
	i := 0
	var lastId uint16
	for {
		if i >= len(b) {
			return 0, io.ErrShortBuffer
		}
		wt, id, ni := readCompactFieldHeader(b, i, lastId)
		i += ni
		lastId = id
		if wt == ctSTOP {
			break
		}
		if wt == ctBOOL_TRUE || wt == ctBOOL_FALSE {
			continue // bool inlined, no value
		}
		n, err := skipCompactValue(b[i:], wt)
		if err != nil {
			return i, err
		}
		i += n
	}
	return i, nil
}

func skipCompactList(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, io.ErrShortBuffer
	}
	sizeNibble := b[0] >> 4
	et := ttype(b[0] & 0x0F)
	i := 1
	l := int(sizeNibble)
	if sizeNibble == 0xF {
		v, vn := decodeVarint(b[i:])
		l = int(v)
		i += vn
	}
	if l < 0 {
		return 0, errNegativeSize
	}
	for j := 0; j < l; j++ {
		if i >= len(b) {
			return i, io.ErrShortBuffer
		}
		n, err := skipCompactValue(b[i:], et)
		if err != nil {
			return i, err
		}
		i += n
	}
	return i, nil
}

func skipCompactMap(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, io.ErrShortBuffer
	}
	size64, vn := decodeVarint(b)
	l := int(size64)
	i := vn
	if l < 0 {
		return 0, errNegativeSize
	}
	var kt, vt ttype
	if l != 0 {
		if len(b)-i < 1 {
			return 0, io.ErrShortBuffer
		}
		kt = ttype(b[i] >> 4)
		vt = ttype(b[i] & 0x0F)
		i++
	}
	for j := 0; j < l; j++ {
		if i >= len(b) {
			return i, io.ErrShortBuffer
		}
		n, err := skipCompactValue(b[i:], kt)
		if err != nil {
			return i, err
		}
		i += n
		if i >= len(b) {
			return i, io.ErrShortBuffer
		}
		n, err = skipCompactValue(b[i:], vt)
		if err != nil {
			return i, err
		}
		i += n
	}
	return i, nil
}

func (d *tDecoder) decodeListCompact(t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {
	if len(b) < 1 {
		return 0, io.ErrShortBuffer
	}
	et := t.V
	sizeNibble := b[0] >> 4
	tp := ttype(b[0] & 0x0F)
	i := 1
	l := int(sizeNibble)
	if sizeNibble == 0xF {
		v, vn := decodeVarint(b[i:])
		l = int(v)
		i += vn
	}
	if l < 0 {
		return 0, errNegativeSize
	}

	h := (*sliceHeader)(p)
	if l <= 0 {
		h.Zero()
		return i, nil
	}
	if remain := len(b) - i; l > remain/int(minWireSizeCompact[et.T]) {
		return i, newSizeExceedsBufferException(l, remain)
	}

	x := d.Malloc(l*et.Size, et.Align, et.MallocAbiType)
	h.Data = x
	h.Len = l
	h.Cap = l

	var sliceData unsafe.Pointer
	if et.IsPointer {
		sliceData = d.Malloc(l*et.V.Size, et.V.Align, et.V.MallocAbiType)
	}

	vp := x
	for j := 0; j < l; j++ {
		if j != 0 {
			vp = unsafe.Add(vp, et.Size)
		}
		ep := vp
		if et.IsPointer {
			if j != 0 {
				sliceData = unsafe.Add(sliceData, et.V.Size)
			}
			*(*unsafe.Pointer)(vp) = sliceData
			ep = sliceData
		}

		if et.FixedSize > 0 && et.T == tBYTE {
			*(*byte)(ep) = b[i]
			i += 1
		} else if et.FixedSize > 0 && et.T == tDOUBLE {
			*(*uint64)(ep) = readUint64LE(b[i:])
			i += 8
		} else {
			n, err := d.decodeTypeCompact(et, b[i:], ep, maxdepth-1)
			if err != nil {
				return i, err
			}
			i += n
		}
	}
	expectedWt := binary2CompactWireType[et.T]
	if tp != expectedWt {
		return 0, newTypeMismatch(et.WT, tp)
	}
	return i, nil
}

func (d *tDecoder) decodeMapCompact(t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {
	if len(b) < 1 {
		return 0, io.ErrShortBuffer
	}

	kt := t.K
	vt := t.V

	size64, vn := decodeVarint(b)
	l := int(size64)
	i := vn

	if l < 0 {
		return 0, errNegativeSize
	}

	var t0, t1 ttype
	if l != 0 {
		if len(b)-i < 1 {
			return 0, io.ErrShortBuffer
		}
		t0 = binary2CompactWireType[kt.T]
		t1 = binary2CompactWireType[vt.T]
		wt0 := ttype(b[i] >> 4)
		wt1 := ttype(b[i] & 0x0F)
		if wt0 != t0 || wt1 != t1 {
			return 0, newTypeMismatchKV(kt.WT, vt.WT, wt0, wt1)
		}
		i++
	}

	if remain := len(b) - i; l > remain/(int(minWireSizeCompact[kt.T])+int(minWireSizeCompact[vt.T])) {
		return i, newSizeExceedsBufferException(l, remain)
	}

	tmp := t.MapTmpVarsPool.Get().(*tmpMapVars)
	k := tmp.k
	v := tmp.v
	kp := tmp.kp
	vp := tmp.vp
	m := reflect.MakeMapWithSize(t.RT, l)

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
	for j := 0; j < l; j++ {
		tmp := kp
		if kt.IsPointer {
			if j != 0 {
				sliceK = unsafe.Add(sliceK, kt.V.Size)
			}
			*(*unsafe.Pointer)(tmp) = sliceK
			tmp = sliceK
		}
		if kt.FixedSize > 0 && kt.T == tBYTE {
			*(*byte)(tmp) = b[i]
			i += 1
		} else if kt.FixedSize > 0 && kt.T == tDOUBLE {
			*(*uint64)(tmp) = readUint64LE(b[i:])
			i += 8
		} else {
			if n, err = d.decodeTypeCompact(kt, b[i:], tmp, maxdepth-1); err != nil {
				break
			}
			i += n
		}
		tmp = vp
		if vt.IsPointer {
			if j != 0 {
				sliceV = unsafe.Add(sliceV, vt.V.Size)
			}
			*(*unsafe.Pointer)(tmp) = sliceV
			tmp = sliceV
		}
		if vt.FixedSize > 0 && vt.T == tBYTE {
			*(*byte)(tmp) = b[i]
			i += 1
		} else if vt.FixedSize > 0 && vt.T == tDOUBLE {
			*(*uint64)(tmp) = readUint64LE(b[i:])
			i += 8
		} else {
			if n, err = d.decodeTypeCompact(vt, b[i:], tmp, maxdepth-1); err != nil {
				break
			}
			i += n
		}
		m.SetMapIndex(k, v)
	}

	if err == nil {
		*(*unsafe.Pointer)(p) = m.UnsafePointer()
	}
	t.MapTmpVarsPool.Put(tmp)
	return i, err
}
