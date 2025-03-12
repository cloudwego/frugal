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
	"reflect"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
)

type decodeFuncKey struct {
	k, v ttype
}

var mapDecodeFuncs = map[decodeFuncKey]decodeFuncType{}

func decoderType(t *tType) ttype {
	if t.IsPointer {
		return tPTR
	}
	if t.T == tSTRING && t.Tag == defs.T_binary {
		return tBINARY
	}
	return t.T
}

func updateMapDecodeFunc(t *tType) {
	if t.T != tMAP {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}
	kt, vt := decoderType(t.K), decoderType(t.V)
	f, ok := mapDecodeFuncs[decodeFuncKey{kt, vt}]
	if ok {
		t.DecodeFunc = f
		return
	}
	t.DecodeFunc = decodeMapAny
}

func registerMapDecodeFunc(k, v ttype, f decodeFuncType) {
	mapDecodeFuncs[decodeFuncKey{k, v}] = f
}

func decodeMapHeader(t *tType, b []byte) (l int, err error) {
	t0, t1, l := ttype(b[0]), ttype(b[1]), int(binary.BigEndian.Uint32(b[2:]))
	if l < 0 {
		return 0, errNegativeSize
	}
	if t0 != t.K.WT || t1 != t.V.WT {
		return 0, newTypeMismatchKV(t.K.WT, t.V.WT, t0, t1)
	}
	return l, nil
}

func decodeMapAny(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, maxdepth int) (i int, err error) {
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}

	// map header
	l, err := decodeMapHeader(t, b)
	if err != nil {
		return 0, err
	}

	kt := t.K
	vt := t.V

	// tmp vars
	// tmpk = decode(b)
	// tmpv = decode(b)
	// map[tmpk] = tmpv
	tmp := t.MapTmpVarsPool.Get().(*tmpMapVars)
	k := tmp.k
	v := tmp.v
	kp := tmp.kp
	vp := tmp.vp
	m := reflect.MakeMapWithSize(t.RT, l)
	*((*unsafe.Pointer)(p)) = m.UnsafePointer() // p = make(t.RT, l)

	// pre-allocate space for keys and values if they're pointers
	// like
	// kp = &sliceK[i], decode(b, kp)
	// instead of
	// kp = new(type), decode(b, kp)
	var sliceK unsafe.Pointer
	if kt.IsPointer && l > 0 {
		sliceK = d.Malloc(l*kt.V.Size, kt.V.Align, kt.V.MallocAbiType)
	}
	var sliceV unsafe.Pointer
	if vt.IsPointer && l > 0 {
		sliceV = d.Malloc(l*vt.V.Size, vt.V.Align, vt.V.MallocAbiType)
	}
	n := 0
	i = 6
	for j := 0; j < l; j++ {
		p = kp
		if kt.IsPointer { // p = &sliceK[j]
			if j != 0 {
				sliceK = unsafe.Add(sliceK, kt.V.Size) // next
			}
			*(*unsafe.Pointer)(p) = sliceK
			p = sliceK
		}

		if kt.FixedSize > 0 {
			i += decodeFixedSizeTypes(kt.T, b[i:], p)
		} else {
			n, err = kt.DecodeFunc(d, kt, b[i:], p, maxdepth-1)
			if err != nil {
				break
			}
			i += n
		}

		p = vp
		if vt.IsPointer { // p = &sliceV[j]
			if j != 0 { // next
				sliceV = unsafe.Add(sliceV, vt.V.Size)
			}
			*(*unsafe.Pointer)(p) = sliceV
			p = sliceV
		}

		if vt.FixedSize > 0 {
			i += decodeFixedSizeTypes(vt.T, b[i:], p)
		} else {
			n, err = vt.DecodeFunc(d, vt, b[i:], p, maxdepth-1)
			if err != nil {
				break
			}
			i += n
		}

		m.SetMapIndex(k, v)
	}
	t.MapTmpVarsPool.Put(tmp) // no defer, it may be in hot path
	return
}
