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

// defaultDecoderMemSize controls the min block mem used to malloc,
// DO NOT increase it mindlessly which would cause mem issue,
// coz objects even use one byte of the mem, it won't be released.
const defaultDecoderMemSize = 2048

type span struct {
	p int
	b unsafe.Pointer
	n int
}

func (s *span) init() {
	sz := defaultDecoderMemSize
	s.p = 0
	s.b = mallocgc(uintptr(sz), nil, false)
	s.n = sz
}

func (s *span) Malloc(n, align int) unsafe.Pointer {
	mask := align - 1
	if s.p+n+mask > s.n {
		sz := defaultDecoderMemSize
		if n+mask > sz {
			sz = n + mask
		}
		s.p = 0
		s.b = mallocgc(uintptr(sz), nil, false)
		s.n = sz
	}
	ret := unsafe.Add(s.b, s.p) // b[p:]
	// memory addr alignment off: aligned(ret) - ret
	off := (uintptr(ret)+uintptr(mask)) & ^uintptr(mask) - uintptr(ret)
	s.p += n + int(off)
	return unsafe.Add(ret, off)
}
