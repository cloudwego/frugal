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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitset(t *testing.T) {
	s := &bitset{}
	for i := 0; i <= 0xffff; i++ {
		s.set(uint16(i))
	}
	for i := 0; i <= 0xffff; i++ {
		require.True(t, s.test(uint16(i)))
	}
	for i, v := range s.data { // all bits set
		require.Equal(t, ^uint64(0), v, i)
	}
	for i := 0; i <= 0xffff; i++ {
		s.unset(uint16(i))
	}
	for i, v := range s.data { // all bits unset
		require.Equal(t, uint64(0), v, i)
	}
}

func BenchmarkBitset(b *testing.B) {
	s := &bitset{}
	for i := 0; i < b.N; i++ {
		s.set(uint16(i))
	}
	for i := 0; i < b.N; i++ {
		s.test(uint16(i))
	}
	for i := 0; i < b.N; i++ {
		s.unset(uint16(i))
	}
}
