/*
 * Copyright 2023 CloudWeGo Authors
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

package tests

import (
	"testing"

	"github.com/cloudwego/frugal"
	"github.com/stretchr/testify/require"
)

func TestDeepNestedCompact(t *testing.T) {
	for i := 0; i < 100; i++ {
		frugal.EncodedSizeCompact(deepNested)
	}
	n := frugal.EncodedSizeCompact(deepNested)
	buf := make([]byte, n)
	ret, err := frugal.EncodeObjectCompact(buf, nil, deepNested)
	require.NoError(t, err)
	require.Equal(t, n, ret)

	v := &L0{}
	_, err = frugal.DecodeObjectCompact(buf, v)
	require.NoError(t, err)
	require.NotNil(t, v.L1)
	require.NotNil(t, v.L1.L2)

	// Compare Binary vs Compact size
	binSize := frugal.EncodedSize(deepNested)
	t.Logf("DeepNested: Binary=%d Compact=%d (-%.1f%%)",
		binSize, n, float64(binSize-n)/float64(binSize)*100)
}
