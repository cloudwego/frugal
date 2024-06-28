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

func TestSpan(t *testing.T) {
	s := &span{}
	s.init()
	for i := 0; i < 10; i++ {
		// reset align to 8
		p := s.Malloc(16, 8)
		require.Equal(t, uintptr(0), uintptr(p)%8)

		n0 := s.p
		p = s.Malloc(8, 4)
		n1 := s.p
		require.Equal(t, uintptr(0), uintptr(p)%4)
		require.Equal(t, n0+8, n1) // coz it's already 8 byte aligned

		p = s.Malloc(4, 2)
		n2 := s.p
		require.Equal(t, uintptr(0), uintptr(p)%2)
		require.Equal(t, n1+4, n2) // coz it's already 4 byte aligned

		_ = s.Malloc(2, 1)
		n3 := s.p
		require.Equal(t, n2+2, n3) // coz it's already 2 byte aligned

		p = s.Malloc(8, 8)
		n4 := s.p
		require.Equal(t, uintptr(0), uintptr(p)%8)
		require.Equal(t, n3+8+2, n4) // 4 + 2 % 8 == 6, 8 - 6 = 2
	}

	// test large malloc
	p := s.Malloc(2*defaultDecoderMemSize, 2)
	require.Equal(t, uintptr(0), uintptr(p)%2)
	p = s.Malloc(4, 4)
	require.Equal(t, uintptr(0), uintptr(p)%4)
}
