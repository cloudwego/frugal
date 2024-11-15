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
	"errors"
	"unsafe"
)

var mapAppendFuncs = map[struct{ k, v ttype }]appendFuncType{}

func updateMapAppendFunc(t *tType) {
	if t.T != tMAP {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}

	f, ok := mapAppendFuncs[struct{ k, v ttype }{k: t.K.T, v: t.V.T}]
	if ok {
		t.AppendFunc = f
		return
	}
	t.AppendFunc = appendMapAnyAny
}

func registerMapAppendFunc(k, v ttype, f appendFuncType) {
	mapAppendFuncs[struct{ k, v ttype }{k: k, v: v}] = f
}

func checkMapN(n uint32) error {
	if n == 0 {
		return nil
	}
	return errors.New("map size changed during encoding")
}

func appendMapHeader(t *tType, b []byte, p unsafe.Pointer) ([]byte, uint32) {
	var n uint32
	if *(*unsafe.Pointer)(p) != nil {
		n = uint32(maplen(*(*unsafe.Pointer)(p)))
	}
	return append(b, byte(t.K.WT), byte(t.V.WT),
		byte(n>>24), byte(n>>16), byte(n>>8), byte(n)), n
}

// this func will be replaced by funcs defined in append_map_gen.go
// see init() in append_map_gen.go for details.
func appendMapAnyAny(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	b, n := appendMapHeader(t, b, p)
	if n == 0 {
		return b, nil
	}

	var err error
	it := newMapIter(rvWithPtr(t.RV, p))
	for kp, vp := it.Next(); kp != nil; kp, vp = it.Next() {
		n--
		b, err = appendAny(t.K, b, kp)
		if err != nil {
			return b, err
		}
		b, err = appendAny(t.V, b, vp)
		if err != nil {
			return b, err
		}
	}
	return b, checkMapN(n)
}
