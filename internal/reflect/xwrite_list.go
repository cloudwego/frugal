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
	"github.com/cloudwego/gopkg/xbuf"
)

var xwriteListFuncs = map[ttype]xwriteFuncType{}

func updateXWriteListFunc(t *tType) {
	if t.T != tLIST && t.T != tSET {
		panic("[bug] type mismatch, got: " + ttype2str(t.T))
	}
	f, ok := xwriteListFuncs[t.V.T]
	if ok {
		t.XWriteFunc = f
		return
	}
	t.XWriteFunc = xwriteListAny
}

func registerXWriteListFunc(t ttype, f xwriteFuncType) {
	xwriteListFuncs[t] = f
}

func xwriteListHeader(t *tType, b *xbuf.XWriteBuffer, p unsafe.Pointer) (uint32, unsafe.Pointer) {
	if *(*unsafe.Pointer)(p) == nil {
		thrift.Binary.WriteListBegin(b.MallocN(5), thrift.TType(t.WT), 0)
		return 0, nil
	}
	h := (*sliceHeader)(p)
	n := h.Len
	thrift.Binary.WriteListBegin(b.MallocN(5), thrift.TType(t.WT), n)
	return uint32(n), h.UnsafePointer()
}

func xwriteListAny(t *tType, b *xbuf.XWriteBuffer, p unsafe.Pointer) error {
	t = t.V
	n, vp := xwriteListHeader(t, b, p)
	if n == 0 {
		return nil
	}
	var err error
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size) // move to next element
		}
		err = xwriteAny(t, b, vp)
		if err != nil {
			return err
		}
	}
	return nil
}
