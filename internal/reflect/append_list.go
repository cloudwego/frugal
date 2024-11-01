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

var listAppendFuncs = map[ttype]appendFuncType{}

func updateListAppendFunc(t *tType) {
	if t.T != tLIST && t.T != tSET {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}
	f, ok := listAppendFuncs[t.V.T]
	if ok {
		t.AppendFunc = f
		return
	}
	t.AppendFunc = appendListAny
}

func registerListAppendFunc(t ttype, f appendFuncType) {
	listAppendFuncs[t] = f
}

func appendListHeader(t *tType, b []byte, p unsafe.Pointer) ([]byte, uint32, unsafe.Pointer) {
	if *(*unsafe.Pointer)(p) == nil {
		return append(b, byte(t.WT), 0, 0, 0, 0), 0, nil
	}
	h := (*sliceHeader)(p)
	n := uint32(h.Len)
	return append(b, byte(t.WT),
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n)),
		n, h.UnsafePointer()
}

func appendListAny(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	var err error
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size) // move to next element
		}
		b, err = appendAny(t, b, vp)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}
