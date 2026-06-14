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

var listAppendFuncsCompact = map[ttype]appendFuncType{}

func updateListAppendFuncCompact(t *tType) {
	if t.T != tLIST && t.T != tSET {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}
	f, ok := listAppendFuncsCompact[t.V.T]
	if ok {
		t.AppendFuncCompact = f
		return
	}
	t.AppendFuncCompact = appendListAnyCompact
}

func registerListAppendFuncCompact(t ttype, f appendFuncType) {
	listAppendFuncsCompact[t] = f
}

func appendCompactListHeader(t *tType, b []byte, p unsafe.Pointer) ([]byte, uint32, unsafe.Pointer) {
	if *(*unsafe.Pointer)(p) == nil {
		return append(b, byte(binary2CompactWireType[t.T])), 0, nil
	}
	h := (*sliceHeader)(p)
	n := uint32(h.Len)
	cwt := binary2CompactWireType[t.T]
	if n <= 14 {
		return append(b, byte((n<<4)|uint32(cwt))), n, h.Data
	}
	b = append(b, byte(0xF0|uint32(cwt)))
	b = appendVarint(b, uint64(n))
	return b, n, h.Data
}
