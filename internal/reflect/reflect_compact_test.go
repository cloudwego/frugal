package reflect

import (
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

func BenchmarkCompactAppend(b *testing.B) {
	p := initTestTypesForBenchmark()
	n := EncodedSizeCompact(p)
	buf := make([]byte, 0, n)
	b.SetBytes(int64(n))
	for i := 0; i < b.N; i++ {
		_, _ = AppendCompact(buf, p)
	}
}

func BenchmarkCompactEncodedSize(b *testing.B) {
	p := initTestTypesForBenchmark()
	_ = EncodedSizeCompact(p)
	for i := 0; i < b.N; i++ {
		EncodedSizeCompact(p)
	}
}

func BenchmarkCompactDecode(b *testing.B) {
	p := initTestTypesForBenchmark()
	n := EncodedSizeCompact(p)
	if n <= 0 {
		b.Fatal(n)
	}
	var err error
	buf := make([]byte, n)
	b.SetBytes(int64(n))
	buf, err = AppendCompact(buf[:0], p)
	assert.Nil(b, err)

	p0 := NewTestTypesForBenchmark()
	for i := 0; i < b.N; i++ {
		p0.InitDefault()
		_, _ = DecodeCompact(buf, p0)
	}
}
