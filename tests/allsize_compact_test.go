/*
 * Copyright 2022 CloudWeGo Authors
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	freflect "github.com/cloudwego/frugal/internal/reflect"
)

func BenchmarkAllSizeCompact_Marshal_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			v := s.val
			n := freflect.EncodedSizeCompact(v)
			b.SetBytes(int64(n))
			buf, err := freflect.AppendCompact(make([]byte, 0, n), v)
			require.NoError(b, err)
			require.Equal(b, len(buf), n)
			buf = buf[:0]
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = freflect.EncodedSizeCompact(v)
				_, _ = freflect.AppendCompact(buf, v)
			}
		})
	}
}

func BenchmarkAllSizeCompact_Unmarshal_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			v := s.val
			n := freflect.EncodedSizeCompact(v)
			buf, err := freflect.AppendCompact(make([]byte, 0, n), v)
			require.NoError(b, err)
			b.SetBytes(int64(n))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v2 := newByEFace(v)
				objectmemclr(v2)
				_, _ = freflect.DecodeCompact(buf, v2)
			}
		})
	}
}

func BenchmarkAllSizeCompact_Parallel_Marshal_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			v := s.val
			n := freflect.EncodedSizeCompact(v)
			b.SetBytes(int64(n))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				buf := make([]byte, 0, n)
				for pb.Next() {
					freflect.EncodedSizeCompact(v)
					_, _ = freflect.AppendCompact(buf, v)
				}
			})
		})
	}
}

func BenchmarkAllSizeCompact_Parallel_Unmarshal_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			v := s.val
			n := freflect.EncodedSizeCompact(v)
			buf, err := freflect.AppendCompact(make([]byte, 0, n), v)
			require.NoError(b, err)
			b.SetBytes(int64(n))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					v2 := newByEFace(v)
					objectmemclr(v2)
					_, _ = freflect.DecodeCompact(buf, v2)
				}
			})
		})
	}
}

func TestCompactAllSize_RoundTrip(t *testing.T) {
	for _, s := range getSamples() {
		t.Run(s.name, func(t *testing.T) {
			v := s.val
			n := freflect.EncodedSizeCompact(v)
			buf, err := freflect.AppendCompact(make([]byte, 0, n), v)
			require.NoError(t, err)
			require.Equal(t, len(buf), n)

			v2 := newByEFace(v)
			dn, err := freflect.DecodeCompact(buf, v2)
			require.NoError(t, err)
			require.Equal(t, len(buf), dn)

			binSize := freflect.EncodedSize(v)
			t.Logf("%s: Binary=%d Compact=%d (-%.1f%%)",
				s.name, binSize, n, float64(binSize-n)/float64(binSize)*100)
		})
	}
}

func TestCompactAllSize_SizeMatches(t *testing.T) {
	for _, s := range getSamples() {
		t.Run(s.name, func(t *testing.T) {
			v := s.val
			est := freflect.EncodedSizeCompact(v)
			buf, err := freflect.AppendCompact(make([]byte, 0, est), v)
			require.NoError(t, err)
			assert.Equal(t, est, len(buf))
		})
	}
}
