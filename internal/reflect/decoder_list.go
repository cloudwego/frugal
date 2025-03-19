/*
 * Copyright 2025 CloudWeGo Authors
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
	"unsafe"
)

func decodeListAny(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}

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
	h.Data = x
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
			n, err := et.DecodeFunc(d, et, b[i:], vp, maxdepth-1)
			if err != nil {
				return i, err
			}
			i += n
		}
	}
	return i, nil

}
