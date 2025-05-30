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
	"fmt"
	"runtime"
	"testing"

	"github.com/cloudwego/gopkg/gridbuf"
	"github.com/stretchr/testify/require"
)

func init() {
	panicIfHackErr()
}

func TestMain(m *testing.M) {
	runtime.GC()
	st0 := runtime.MemStats{}
	runtime.ReadMemStats(&st0)
	defer func() {
		runtime.GC()
		st1 := runtime.MemStats{}
		runtime.ReadMemStats(&st1)
		fmt.Println("Stat ===============")
		fmt.Printf("Before test: %d objects, %d bytes\n", st0.HeapObjects, st0.HeapInuse)
		fmt.Printf("After test: %d objects, %d bytes\n", st1.HeapObjects, st1.HeapInuse)
	}()

	m.Run()
}

func initTestTypesForBenchmark() *TestTypesForBenchmark {
	b1 := true
	s1 := "hello"
	ret := NewTestTypesForBenchmark()
	ret.B1 = &b1
	ret.Str1 = &s1
	ret.Msg1 = &Msg{Type: 1}
	ret.M0 = map[int32]int32{
		1: 2,
		2: 3,
		3: 4,
	}
	ret.M1 = map[string]*Msg{
		"k1":    {Type: 1},
		"k1231": {Type: 2},
		"k233":  {Type: 3},
		"k12":   {Type: 4},
	}
	ret.L0 = []int32{1, 2, 3}
	ret.L1 = []*Msg{{Type: 1}, {Type: 2}, {Type: 3}}
	ret.Set0 = []int32{1, 2, 3}
	ret.Set1 = []string{"AAAA", "BB", "CCCCC"}
	return ret
}

func appendStringField(b []byte, fid uint16, s string) []byte {
	l := uint32(len(s))
	b = append(b, byte(tSTRING), byte(fid>>8), byte(fid), // type and fid
		byte(l>>24), byte(l>>16), byte(l>>8), byte(l)) // len
	return append(b, s...)
}

func BenchmarkAppend(b *testing.B) {
	p := initTestTypesForBenchmark()
	n := EncodedSize(p)
	buf := make([]byte, 0, n)
	b.SetBytes(int64(n))
	for i := 0; i < b.N; i++ {
		_, _ = Append(buf, p)
	}
}

func BenchmarkGridWrite(b *testing.B) {
	p := initTestTypesForBenchmark()
	for i := 0; i < b.N; i++ {
		buf := gridbuf.NewWriteBuffer()
		_ = GridWrite(buf, p)
		_ = buf.Bytes()
		buf.Free()
	}
}

func BenchmarkEncodedSize(b *testing.B) {
	p := initTestTypesForBenchmark()
	_ = EncodedSize(p) // pretouch
	for i := 0; i < b.N; i++ {
		EncodedSize(p)
	}
}

const checkBenchmarkDecodeResult = false

func BenchmarkDecode(b *testing.B) {
	p := initTestTypesForBenchmark()
	n := EncodedSize(p)
	if n <= 0 {
		b.Fatal(n)
	}
	var err error
	buf := make([]byte, n)
	b.SetBytes(int64(n))
	buf, err = Append(buf[:0], p)
	require.NoError(b, err)

	p0 := NewTestTypesForBenchmark()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p0.InitDefault()
		_, _ = Decode(buf, p0)
		if checkBenchmarkDecodeResult {
			require.Equal(b, p0, p)
		}
	}
}

func BenchmarkGridRead(b *testing.B) {
	p := initTestTypesForBenchmark()
	n := EncodedSize(p)
	if n <= 0 {
		b.Fatal(n)
	}
	var err error
	buf := make([]byte, n)
	b.SetBytes(int64(n))
	buf, err = Append(buf[:0], p)
	require.NoError(b, err)

	p0 := NewTestTypesForBenchmark()
	bs := [][]byte{buf}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p0.InitDefault()
		rb := gridbuf.NewReadBuffer(bs)
		_ = GridRead(rb, p0)
		if checkBenchmarkDecodeResult {
			require.Equal(b, p0, p)
		}
		rb.Free()
	}
}
