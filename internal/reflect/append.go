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

	"github.com/cloudwego/gopkg/gridbuf"
	"github.com/cloudwego/gopkg/unsafex"
)

const nocopyWriteThreshold = 4096

func appendStruct(t *tType, b []byte, base unsafe.Pointer, gb *gridbuf.WriteBuffer) ([]byte, error) {
	sd := t.Sd
	if base == nil {
		if cap(b)-len(b) < 1 {
			b = gb.NewBuffer(b, 1)
		}
		return append(b, byte(tSTOP)), nil
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
		if cap(b)-len(b) < 3 {
			b = gb.NewBuffer(b, 3)
		}
		b = append(b, byte(t.WT), byte(f.ID>>8), byte(f.ID))

		// field value
		// the following code should be the same as func `appendAny`
		// manually copy here for inlining:

		if t.IsPointer {
			p = *(*unsafe.Pointer)(p)
		}
		if t.SimpleType { // fast path
			switch t.T {
			case tBYTE, tBOOL:
				if cap(b)-len(b) < 1 {
					b = gb.NewBuffer(b, 1)
				}
				b = append(b, *(*byte)(p)) // for tBOOL, true -> 1, false -> 0
			case tI16:
				if cap(b)-len(b) < 2 {
					b = gb.NewBuffer(b, 2)
				}
				b = appendUint16(b, *((*uint16)(p)))
			case tI32:
				if cap(b)-len(b) < 4 {
					b = gb.NewBuffer(b, 4)
				}
				b = appendUint32(b, *((*uint32)(p)))
			case tENUM:
				if cap(b)-len(b) < 4 {
					b = gb.NewBuffer(b, 4)
				}
				b = appendUint32(b, uint32(*((*int64)(p))))
			case tI64, tDOUBLE:
				if cap(b)-len(b) < 8 {
					b = gb.NewBuffer(b, 8)
				}
				b = appendUint64(b, *((*uint64)(p)))
			case tSTRING:
				s := *((*string)(p))
				if len(s) < nocopyWriteThreshold {
					if cap(b)-len(b) < len(s)+4 {
						b = gb.NewBuffer(b, len(s)+4)
					}
					b = appendUint32(b, uint32(len(s)))
					b = append(b, s...)
				} else {
					if cap(b)-len(b) < 4 {
						b = gb.NewBuffer(b, 4)
					}
					b = appendUint32(b, uint32(len(s)))
					b = gb.WriteDirect(b, unsafex.StringToBinary(s))
				}
			}
		} else {
			b, err = t.AppendFunc(t, b, p, gb)
			if err != nil {
				return b, withFieldErr(err, sd, f)
			}
		}
	}
	if sd.hasUnknownFields {
		xb := *(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset))
		if len(xb) > 0 {
			if cap(b)-len(b) < len(xb) {
				b = gb.NewBuffer(b, len(xb))
			}
			b = append(b, xb...)
		}
	}
	if cap(b)-len(b) < 1 {
		b = gb.NewBuffer(b, 1)
	}
	return append(b, byte(tSTOP)), nil
}

func appendAny(t *tType, b []byte, p unsafe.Pointer, gb *gridbuf.WriteBuffer) ([]byte, error) {
	if t.IsPointer {
		p = *(*unsafe.Pointer)(p)
	}
	if t.SimpleType {
		switch t.T {
		case tBYTE, tBOOL:
			if cap(b)-len(b) < 1 {
				b = gb.NewBuffer(b, 1)
			}
			b = append(b, *(*byte)(p)) // for tBOOL, true -> 1, false -> 0
		case tI16:
			if cap(b)-len(b) < 2 {
				b = gb.NewBuffer(b, 2)
			}
			b = appendUint16(b, *((*uint16)(p)))
		case tI32:
			if cap(b)-len(b) < 4 {
				b = gb.NewBuffer(b, 4)
			}
			b = appendUint32(b, *((*uint32)(p)))
		case tENUM:
			if cap(b)-len(b) < 4 {
				b = gb.NewBuffer(b, 4)
			}
			b = appendUint32(b, uint32(*((*int64)(p))))
		case tI64, tDOUBLE:
			if cap(b)-len(b) < 8 {
				b = gb.NewBuffer(b, 8)
			}
			b = appendUint64(b, *((*uint64)(p)))
		case tSTRING:
			s := *((*string)(p))
			if len(s) < nocopyWriteThreshold {
				if cap(b)-len(b) < len(s)+4 {
					b = gb.NewBuffer(b, len(s)+4)
				}
				b = appendUint32(b, uint32(len(s)))
				b = append(b, s...)
			} else {
				if cap(b)-len(b) < 4 {
					b = gb.NewBuffer(b, 4)
				}
				b = appendUint32(b, uint32(len(s)))
				b = gb.WriteDirect(b, unsafex.StringToBinary(s))
			}
		}
		return b, nil
	} else {
		return t.AppendFunc(t, b, p, gb)
	}
}
