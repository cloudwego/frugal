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

	"github.com/cloudwego/gopkg/gridbuf"
	"github.com/cloudwego/gopkg/unsafex"
)

const nocopyWriteThreshold = 4096

func gridWriteStruct(t *tType, b *gridbuf.WriteBuffer, base unsafe.Pointer) error {
	sd := t.Sd
	if base == nil {
		b.MallocN(1)[0] = byte(tSTOP)
		return nil
	}
	var err error
	for _, f := range sd.fields {
		t := f.Type
		p := unsafe.Add(base, f.Offset)
		if f.CanSkipEncodeIfNil && *(*unsafe.Pointer)(p) == nil {
			continue
		}
		if f.CanSkipIfDefault && t.Equal(f.Default, p) {
			continue
		}

		// field header
		buf := b.MallocN(3)
		buf[0], buf[1], buf[2] = byte(t.WT), byte(f.ID>>8), byte(f.ID)

		// field value
		// the following code should be the same as func `appendAny`
		// manually copy here for inlining:

		if t.IsPointer {
			p = *(*unsafe.Pointer)(p)
		}
		if t.SimpleType { // fast path
			switch t.T {
			case tBYTE, tBOOL:
				b.MallocN(1)[0] = *(*byte)(p) // for tBOOL, true -> 1, false -> 0
			case tI16:
				binary.BigEndian.PutUint16(b.MallocN(2), *((*uint16)(p)))
			case tI32:
				binary.BigEndian.PutUint32(b.MallocN(4), *((*uint32)(p)))
			case tENUM:
				binary.BigEndian.PutUint32(b.MallocN(4), uint32(*((*int64)(p))))
			case tI64, tDOUBLE:
				binary.BigEndian.PutUint64(b.MallocN(8), *((*uint64)(p)))
			case tSTRING:
				s := *((*string)(p))
				if len(s) < nocopyWriteThreshold {
					buf := b.MallocN(len(s) + 4)
					binary.BigEndian.PutUint32(buf, uint32(len(s)))
					copy(buf[4:], s)
				} else {
					binary.BigEndian.PutUint32(b.MallocN(4), uint32(len(s)))
					b.WriteDirect(unsafex.StringToBinary(s))
				}
			}
		} else {
			err = t.GridWriteFunc(t, b, p)
			if err != nil {
				return withFieldErr(err, sd, f)
			}
		}
	}
	if sd.hasUnknownFields {
		xb := *(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset))
		if len(xb) > 0 {
			b.WriteDirect(xb)
		}
	}
	b.MallocN(1)[0] = byte(tSTOP)
	return nil
}

func gridWriteAny(t *tType, b *gridbuf.WriteBuffer, p unsafe.Pointer) error {
	if t.IsPointer {
		p = *(*unsafe.Pointer)(p)
	}
	if t.SimpleType {
		switch t.T {
		case tBYTE, tBOOL:
			b.MallocN(1)[0] = *(*byte)(p) // for tBOOL, true -> 1, false -> 0
		case tI16:
			binary.BigEndian.PutUint16(b.MallocN(2), *((*uint16)(p)))
		case tI32:
			binary.BigEndian.PutUint32(b.MallocN(4), *((*uint32)(p)))
		case tENUM:
			binary.BigEndian.PutUint32(b.MallocN(4), uint32(*((*int64)(p))))
		case tI64, tDOUBLE:
			binary.BigEndian.PutUint64(b.MallocN(8), *((*uint64)(p)))
		case tSTRING:
			s := *((*string)(p))
			if len(s) < nocopyWriteThreshold {
				buf := b.MallocN(len(s) + 4)
				binary.BigEndian.PutUint32(buf, uint32(len(s)))
				copy(buf[4:], s)
			} else {
				binary.BigEndian.PutUint32(b.MallocN(4), uint32(len(s)))
				b.WriteDirect(unsafex.StringToBinary(s))
			}
		}
		return nil
	} else {
		return t.GridWriteFunc(t, b, p)
	}
}
