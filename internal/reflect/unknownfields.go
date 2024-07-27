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
	"runtime"
	"sync"
	"unsafe"
)

type unknownFieldIdx struct {
	off int
	sz  int
}

type unknownFields struct {
	sz   int
	offs []unknownFieldIdx
}

func (p *unknownFields) Reset() {
	p.sz = 0
	p.offs = p.offs[:0]
}

func (p *unknownFields) Add(off, sz int) {
	p.sz += sz
	p.offs = append(p.offs, unknownFieldIdx{off: off, sz: sz})
}

func (p *unknownFields) Size() int {
	return p.sz
}

func (p *unknownFields) Copy(b []byte) []byte {
	sz := p.Size()
	data := mallocgc(uintptr(sz), nil, false) //  without zeroing
	ret := []byte{}
	h := (*sliceHeader)(unsafe.Pointer(&ret))
	h.Data = uintptr(data)
	h.Len = sz
	h.Cap = sz
	off := 0
	for _, x := range p.offs {
		copy(ret[off:], b[x.off:x.off+x.sz])
		off += x.sz
	}
	runtime.KeepAlive(data)
	return ret
}

var unknownFieldsPool = sync.Pool{
	New: func() interface{} {
		return &unknownFields{offs: make([]unknownFieldIdx, 0, 8)}
	},
}
