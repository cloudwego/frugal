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
	"sync"
	"unsafe"

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
		if bs != nil { // update bitset if has required fields
			bs.set(f.ID)
		}
		p := unsafe.Add(base, f.Offset) // pointer to the field

		t := f.Type
		p = d.mallocIfPointer(t, p)
		if t.FixedSize > 0 {
			// fixed len types, can be inlined without func call
			i += decodeFixedSizeTypes(t.T, b[i:], p)
			continue
		}

		var n int
		var err error
		if f.NoCopy {
			n, err = decodeStringNoCopy(d, t, b[i:], p)
		} else {
			n, err = t.DecodeFunc(d, t, b[i:], p, maxdepth-1)
		}
		if err != nil {
			return i, fmt.Errorf("decode field %d of struct %s err: %w", fid, sd.rt.String(), err)
		}
		i += n

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

func decodeStruct(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {
	if t.Sd.hasInitFunc {
		f := t.Sd.initFunc // copy on write, reuse itab of iface
		updateIface(unsafe.Pointer(&f), p)
		f.InitDefault()
	}
	return d.Decode(b, p, t.Sd, maxdepth)
}
