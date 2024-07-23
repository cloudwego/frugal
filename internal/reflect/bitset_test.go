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
	for i := uint16(0); i < ^uint16(0); i++ {
		if i%2 == 0 {
			s.set(i)
		}
		if i%4 == 0 {
			s.unset(i)
		}
	}
	for i := uint16(0); i < ^uint16(0); i++ {
		if i%4 == 0 {
			require.False(t, s.test(i))
		} else if i%2 == 0 {
			require.True(t, s.test(i))
		} else {
			require.False(t, s.test(i))
		}
	}
}
