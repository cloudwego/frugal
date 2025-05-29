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
	"unsafe"

	"github.com/cloudwego/gopkg/protocol/thrift"
)

func xwriteStruct(t *tType, b *thrift.XWriteBuffer, base unsafe.Pointer) error {
	sd := t.Sd
	if base == nil {
		thrift.XBuffer.WriteFieldStop(b)
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
		thrift.XBuffer.WriteFieldBegin(b, thrift.TType(t.WT), int16(f.ID))

		// field value
		// the following code should be the same as func `appendAny`
		// manually copy here for inlining:

		if t.IsPointer {
			p = *(*unsafe.Pointer)(p)
		}
		if t.SimpleType { // fast path
			switch t.T {
			case tBYTE, tBOOL:
				thrift.XBuffer.WriteByte(b, *(*int8)(p)) // for tBOOL, true -> 1, false -> 0
			case tI16:
				thrift.XBuffer.WriteI16(b, *((*int16)(p)))
			case tI32:
				thrift.XBuffer.WriteI32(b, *((*int32)(p)))
			case tENUM:
				thrift.XBuffer.WriteI32(b, int32(*((*int64)(p))))
			case tI64, tDOUBLE:
				thrift.XBuffer.WriteI64(b, *((*int64)(p)))
			case tSTRING:
				s := *((*string)(p))
				thrift.XBuffer.WriteString(b, s)
			}
		} else {
			err = t.XWriteFunc(t, b, p)
			if err != nil {
				return withFieldErr(err, sd, f)
			}
		}
	}
	if sd.hasUnknownFields {
		xb := *(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset))
		if len(xb) > 0 {
			thrift.XBuffer.RawWrite(b, xb)
		}
	}
	thrift.XBuffer.WriteFieldStop(b)
	return nil
}

func xwriteAny(t *tType, b *thrift.XWriteBuffer, p unsafe.Pointer) error {
	if t.IsPointer {
		p = *(*unsafe.Pointer)(p)
	}
	if t.SimpleType {
		switch t.T {
		case tBYTE, tBOOL:
			thrift.XBuffer.WriteByte(b, *(*int8)(p)) // for tBOOL, true -> 1, false -> 0
		case tI16:
			thrift.XBuffer.WriteI16(b, *((*int16)(p)))
		case tI32:
			thrift.XBuffer.WriteI32(b, *((*int32)(p)))
		case tENUM:
			thrift.XBuffer.WriteI32(b, int32(*((*int64)(p))))
		case tI64, tDOUBLE:
			thrift.XBuffer.WriteI64(b, *((*int64)(p)))
		case tSTRING:
			s := *((*string)(p))
			thrift.XBuffer.WriteString(b, s)
		}
		return nil
	} else {
		return t.XWriteFunc(t, b, p)
	}
}
