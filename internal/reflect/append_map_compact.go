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

var mapAppendFuncsCompact = map[struct{ k, v ttype }]appendFuncType{}

func updateMapAppendFuncCompact(t *tType) {
	if t.T != tMAP {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}

	f, ok := mapAppendFuncsCompact[struct{ k, v ttype }{k: t.K.T, v: t.V.T}]
	if ok {
		t.AppendFuncCompact = f
		return
	}
	panic("[bug] missing compact map fast path for " + ttype2str(t.K.T) + ":" + ttype2str(t.V.T))
}

func registerMapAppendFuncCompact(k, v ttype, f appendFuncType) {
	mapAppendFuncsCompact[struct{ k, v ttype }{k: k, v: v}] = f
}

func appendCompactMapHeader(t *tType, b []byte, p unsafe.Pointer) ([]byte, uint32) {
	var n uint32
	if *(*unsafe.Pointer)(p) != nil {
		n = uint32(maplen(*(*unsafe.Pointer)(p)))
	}
	b = appendVarint(b, uint64(n))
	if n != 0 {
		kt := binary2CompactWireType[t.K.T]
		vt := binary2CompactWireType[t.V.T]
		b = append(b, byte(kt<<4)|byte(vt))
	}
	return b, n
}

func compactMapHeaderSize(count int) int {
	n := varintLen(uint64(count))
	if count > 0 {
		n++
	}
	return n
}
