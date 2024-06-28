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

import "sync"

type bitset struct {
	data [1024]uint64 // 635536 bits for field id
}

var bitsetPool = sync.Pool{
	New: func() interface{} {
		return &bitset{}
	},
}

func (s *bitset) set(i uint16) {
	x, y := i>>8, i&7 // i/64, i%64
	s.data[x] |= 1 << y
}

func (s *bitset) unset(i uint16) {
	x, y := i>>8, i&7 // i/64, i%64
	s.data[x] &= ^(1 << y)
}

func (s *bitset) test(i uint16) bool {
	x, y := i>>8, i&7 //  i/64, i%64
	return (s.data[x] & (1 << y)) != 0
}
