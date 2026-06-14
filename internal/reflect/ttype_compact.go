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
	"unsafe"
)

// Compact Protocol uses different wire type IDs than Binary Protocol.
// These are the nibble values (4-bit) packed into field/container headers.
const (
	ctSTOP       ttype = 0x00
	ctBOOL_TRUE  ttype = 0x01
	ctBOOL_FALSE ttype = 0x02
	ctI08        ttype = 0x03
	ctI16        ttype = 0x04
	ctI32        ttype = 0x05
	ctI64        ttype = 0x06
	ctDOUBLE     ttype = 0x07
	ctBINARY     ttype = 0x08
	ctLIST       ttype = 0x09
	ctSET        ttype = 0x0A
	ctMAP        ttype = 0x0B
	ctSTRUCT     ttype = 0x0C
)

// binary2CompactWireType maps Binary Protocol ttype to Compact Protocol wire type.
var binary2CompactWireType = map[ttype]ttype{
	tBOOL:   ctBOOL_TRUE,
	tBYTE:   ctI08,
	tI16:    ctI16,
	tI32:    ctI32,
	tI64:    ctI64,
	tDOUBLE: ctDOUBLE,
	tSTRING: ctBINARY,
	tSTRUCT: ctSTRUCT,
	tMAP:    ctMAP,
	tSET:    ctSET,
	tLIST:   ctLIST,
	tENUM:   ctI32,
}

// Protocol selects the Thrift wire format.
type Protocol uint8

const (
	Binary  Protocol = 0
	Compact Protocol = 1
)

func (t *tType) encodedMapSizeCompact(p unsafe.Pointer) (int, error) {
	if *(*unsafe.Pointer)(p) == nil {
		return compactMapHeaderSize(0), nil
	}

	l := maplen(*(*unsafe.Pointer)(p))
	if l == 0 {
		return compactMapHeaderSize(0), nil
	}

	ret := compactMapHeaderSize(l)

	kt := t.K
	vt := t.V

	it := newMapIter(rvWithPtr(t.RV, p))
	for kp, vp := it.Next(); kp != nil; kp, vp = it.Next() {
		if kt.T == tSTRING {
			ret += varintLen(uint64(len(*(*string)(kp)))) + len(*(*string)(kp))
		} else if kt.T == tBYTE {
			ret += 1
		} else if kt.T == tDOUBLE {
			ret += 8
		} else if kt.FixedSize > 0 {
			switch kt.T {
			case tI16:
				ret += zigzag32Size(int32(*(*int16)(kp)))
			case tI32:
				ret += zigzag32Size(*(*int32)(kp))
			case tI64:
				ret += zigzag64Size(*(*int64)(kp))
			case tENUM:
				ret += zigzag32Size(int32(*(*int64)(kp)))
			case tBOOL:
				ret += 1
			}
		} else {
			n, err := kt.EncodedSizeFuncCompact(kp)
			if err != nil {
				return ret, err
			}
			ret += n
		}

		if vt.T == tSTRING {
			ret += varintLen(uint64(len(*(*string)(vp)))) + len(*(*string)(vp))
		} else if vt.T == tBYTE {
			ret += 1
		} else if vt.T == tDOUBLE {
			ret += 8
		} else if vt.FixedSize > 0 {
			switch vt.T {
			case tI16:
				ret += zigzag32Size(int32(*(*int16)(vp)))
			case tI32:
				ret += zigzag32Size(*(*int32)(vp))
			case tI64:
				ret += zigzag64Size(*(*int64)(vp))
			case tENUM:
				ret += zigzag32Size(int32(*(*int64)(vp)))
			case tBOOL:
				ret += 1
			}
		} else {
			n, err := vt.EncodedSizeFuncCompact(vp)
			if err != nil {
				return ret, err
			}
			ret += n
		}
	}

	return ret, nil
}

func (t *tType) encodedListSizeCompact(p unsafe.Pointer) (int, error) {
	if *(*unsafe.Pointer)(p) == nil {
		return 1, nil
	}
	vt := t.V
	h := (*sliceHeader)(p)
	ret := compactListHeaderSize(h.Len)
	if h.Len == 0 {
		return ret, nil
	}
	if vt.FixedSize > 0 && vt.T == tBYTE {
		return ret + h.Len, nil
	}
	if vt.FixedSize > 0 && vt.T == tDOUBLE {
		return ret + (h.Len * 8), nil
	}
	vp := h.Data
	for i := 0; i < h.Len; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, vt.Size)
		}
		switch vt.T {
		case tBOOL:
			ret += 1
		case tI16:
			ret += zigzag32Size(int32(*(*int16)(vp)))
		case tI32:
			ret += zigzag32Size(*(*int32)(vp))
		case tENUM:
			ret += zigzag32Size(int32(*(*int64)(vp)))
		case tI64:
			ret += zigzag64Size(*(*int64)(vp))
		case tSTRING:
			s := *(*string)(vp)
			ret += varintLen(uint64(len(s))) + len(s)
		default:
			n, err := vt.EncodedSizeFuncCompact(vp)
			if err != nil {
				return ret, err
			}
			ret += n
		}
	}
	return ret, nil
}

func compactListHeaderSize(count int) int {
	if count <= 14 {
		return 1
	}
	return 1 + varintLen(uint64(count))
}
