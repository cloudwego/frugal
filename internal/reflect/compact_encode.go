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
	"unsafe"
)

func appendAnyCompact(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	if t.IsPointer {
		p = *(*unsafe.Pointer)(p)
	}
	if t.T == tBOOL {
		if *(*bool)(p) {
			return append(b, byte(ctBOOL_TRUE)), nil
		}
		return append(b, byte(ctBOOL_FALSE)), nil
	}
	return t.AppendFuncCompact(t, b, p)
}

func appendStructCompact(t *tType, b []byte, base unsafe.Pointer) ([]byte, error) {
	sd := t.Sd
	if base == nil {
		return append(b, byte(ctSTOP)), nil
	}
	var lastId uint16
	var err error
	for _, f := range sd.fields {
		ft := f.Type
		p := unsafe.Add(base, f.Offset)
		if f.CanSkipEncodeIfNil && *(*unsafe.Pointer)(p) == nil {
			continue
		}
		if f.CanSkipIfDefault && ft.Equal(f.Default, p) {
			continue
		}

		wt := binary2CompactWireType[ft.T]
		if ft.T == tBOOL {
			if ft.IsPointer {
				p = *(*unsafe.Pointer)(p)
			}
			if *(*bool)(p) {
				wt = ctBOOL_TRUE
			} else {
				wt = ctBOOL_FALSE
			}
			b = writeCompactFieldHeader(b, lastId, f.ID, wt)
			lastId = f.ID
			continue
		}

		b = writeCompactFieldHeader(b, lastId, f.ID, wt)
		lastId = f.ID

		if ft.IsPointer {
			p = *(*unsafe.Pointer)(p)
		}
		if ft.SimpleType {
			switch ft.T {
			case tBYTE:
				b = append(b, *(*byte)(p))
			case tI16:
				b = appendZigzag32(b, int32(*(*int16)(p)))
			case tI32:
				b = appendZigzag32(b, *(*int32)(p))
			case tENUM:
				b = appendZigzag32(b, int32(*(*int64)(p)))
			case tI64:
				b = appendZigzag64(b, *(*int64)(p))
			case tDOUBLE:
				b = appendUint64LE(b, *(*uint64)(p))
			case tSTRING:
				s := *(*string)(p)
				b = appendVarint(b, uint64(len(s)))
				b = append(b, s...)
			}
		} else {
			b, err = ft.AppendFuncCompact(ft, b, p)
			if err != nil {
				return b, withFieldErr(err, sd, f)
			}
		}
	}
	if sd.hasUnknownFields {
		xb := *(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset))
		if len(xb) > 0 {
			b = append(b, xb...)
		}
	}
	return append(b, byte(ctSTOP)), nil
}

func appendListAnyCompact(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	vt := t.V
	b, n, vp := appendCompactListHeader(vt, b, p)
	if n == 0 {
		if t.T == tSET {
			h := (*sliceHeader)(p)
			if h.Len > 0 {
				return b, checkUniqueness(vt, h)
			}
		}
		return b, nil
	}
	if t.T == tSET {
		h := (*sliceHeader)(p)
		if err := checkUniqueness(vt, h); err != nil {
			return b, err
		}
	}
	var err error
	t = vt
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b, err = appendAnyCompact(t, b, vp)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func (t *tType) encodedSizeCompact(base unsafe.Pointer) (int, error) {
	sd := t.Sd
	if t.T != 0 {
		base = *(*unsafe.Pointer)(base)
	}
	if base == nil {
		return 1, nil
	}
	ret := 0
	var lastId uint16
	for _, f := range sd.fields {
		p := unsafe.Add(base, f.Offset)
		if f.CanSkipEncodeIfNil && *(*unsafe.Pointer)(p) == nil {
			continue
		}
		ft := f.Type
		if f.CanSkipIfDefault && ft.Equal(f.Default, p) {
			continue
		}

		ret += compactFieldHeaderSize(lastId, f.ID)
		lastId = f.ID

		if ft.T == tBOOL {
			continue
		}

		kvp := p
		if ft.IsPointer && ft.T != tSTRUCT && ft.T != tLIST && ft.T != tSET && ft.T != tMAP {
			kvp = *(*unsafe.Pointer)(kvp)
		}
		switch ft.T {
		case tBYTE:
			ret += 1
		case tI16:
			ret += zigzag32Size(int32(*(*int16)(kvp)))
		case tI32:
			ret += zigzag32Size(*(*int32)(kvp))
		case tENUM:
			ret += zigzag32Size(int32(*(*int64)(kvp)))
		case tI64:
			ret += zigzag64Size(*(*int64)(kvp))
		case tDOUBLE:
			ret += 8
		case tSTRING:
			s := *(*string)(kvp)
			ret += varintLen(uint64(len(s))) + len(s)
		case tSTRUCT, tLIST, tSET, tMAP:
			if ft.EncodedSizeFuncCompact == nil {
				return 0, fmt.Errorf("compact encoded size not implemented for %s", ttype2str(ft.T))
			}
			n, err := ft.EncodedSizeFuncCompact(p)
			if err != nil {
				return ret, err
			}
			ret += n
		}
	}
	if sd.hasUnknownFields {
		ret += len(*(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset)))
	}
	ret += 1
	return ret, nil
}
