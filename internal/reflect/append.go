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

import "unsafe"

func appendStruct(t *tType, b []byte, base unsafe.Pointer) ([]byte, error) {
	sd := t.Sd
	if base == nil {
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
				b = append(b, *(*byte)(p)) // for tBOOL, true -> 1, false -> 0
			case tI16:
				b = appendUint16(b, *((*uint16)(p)))
			case tI32:
				b = appendUint32(b, *((*uint32)(p)))
			case tENUM:
				b = appendUint32(b, uint32(*((*int64)(p))))
			case tI64, tDOUBLE:
				b = appendUint64(b, *((*uint64)(p)))
			case tSTRING:
				s := *((*string)(p))
				b = appendUint32(b, uint32(len(s)))
				b = append(b, s...)
			}
		} else {
			b, err = t.AppendFunc(t, b, p)
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
	return append(b, byte(tSTOP)), nil
}

func appendAny(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	if t.IsPointer {
		p = *(*unsafe.Pointer)(p)
	}
	if t.SimpleType {
		switch t.T {
		case tBYTE, tBOOL:
			b = append(b, *(*byte)(p)) // for tBOOL, true -> 1, false -> 0
		case tI16:
			b = appendUint16(b, *((*uint16)(p)))
		case tI32:
			b = appendUint32(b, *((*uint32)(p)))
		case tENUM:
			b = appendUint32(b, uint32(*((*int64)(p))))
		case tI64, tDOUBLE:
			b = appendUint64(b, *((*uint64)(p)))
		case tSTRING:
			s := *((*string)(p))
			b = appendUint32(b, uint32(len(s)))
			b = append(b, s...)
		}
		return b, nil
	} else {
		return t.AppendFunc(t, b, p)
	}
}
