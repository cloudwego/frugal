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

	"github.com/cloudwego/frugal/internal/defs"
)

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

func decoderString(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, _ int) (int, error) {
	return _decoderString(d, t, b, p, false)
}

func decodeStringNoCopy(d *tDecoder, t *tType, b []byte, p unsafe.Pointer) (int, error) {
	return _decoderString(d, t, b, p, true)
}

func _decoderString(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, nocopy bool) (int, error) {
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

	x := unsafe.Pointer(nil)
	if nocopy {
		x = unsafe.Pointer(&b[i])
	} else {
		x = d.Malloc(l, 1, 0)
		copyn(x, b[i:], l)
	}

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
	i += l
	return i, nil
}
