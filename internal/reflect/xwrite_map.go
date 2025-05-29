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

	"github.com/cloudwego/gopkg/protocol/thrift"
)

var mapXWriteFuncs = map[struct{ k, v ttype }]xwriteFuncType{}

func updateXWriteMapFunc(t *tType) {
	if t.T != tMAP {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}

	f, ok := mapXWriteFuncs[struct{ k, v ttype }{k: t.K.T, v: t.V.T}]
	if ok {
		t.XWriteFunc = f
		return
	}
	t.XWriteFunc = xwriteMapAnyAny
}

func registerMapXWriteFunc(k, v ttype, f xwriteFuncType) {
	mapXWriteFuncs[struct{ k, v ttype }{k: k, v: v}] = f
}

func xwriteMapHeader(t *tType, b *thrift.XWriteBuffer, p unsafe.Pointer) uint32 {
	var n uint32
	if *(*unsafe.Pointer)(p) != nil {
		n = uint32(maplen(*(*unsafe.Pointer)(p)))
	}
	thrift.XBuffer.WriteMapBegin(b, thrift.TType(t.K.WT), thrift.TType(t.V.WT), int(n))
	return n
}

// this func will be replaced by funcs defined in xwrite_map_gen.go
// see init() in xwrite_map_gen.go for details.
func xwriteMapAnyAny(t *tType, b *thrift.XWriteBuffer, p unsafe.Pointer) error {
	n := xwriteMapHeader(t, b, p)
	if n == 0 {
		return nil
	}

	var err error
	it := newMapIter(rvWithPtr(t.RV, p))
	for kp, vp := it.Next(); kp != nil; kp, vp = it.Next() {
		n--
		err = xwriteAny(t.K, b, kp)
		if err != nil {
			return err
		}
		err = xwriteAny(t.V, b, vp)
		if err != nil {
			return err
		}
	}
	return checkMapN(n)
}
