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
	"bytes"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/frugal/tests/baseline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudwego/frugal/internal/opts"
	freflect "github.com/cloudwego/frugal/internal/reflect"
	gthrift "github.com/cloudwego/gopkg/protocol/thrift"

	"github.com/cloudwego/frugal/internal/jit"
	jitDecoder "github.com/cloudwego/frugal/internal/jit/decoder"
	jitEncoder "github.com/cloudwego/frugal/internal/jit/encoder"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	samples = []Sample{
		{"small", getSimpleValue(), nil},
		{"medium", getNestingValue(), nil},
		{"large", getNesting2Value(), nil},
	}

	for i, s := range samples {
		v, ok := s.val.(thrift.TStruct)
		if !ok {
			panic("s.val must be thrift.TStruct!")
		}
		mm := thrift.NewTMemoryBuffer()
		mm.Reset()
		if err := v.Write(thrift.NewTBinaryProtocolTransport(mm)); err != nil {
			panic(err)
		}
		samples[i].bytes = mm.Bytes()
		println("sample ", s.name, ", size ", len(samples[i].bytes), "B")
		// println(dumpval(s.val))
	}

	m.Run()
}

type FastCodec interface {
	InitDefault()
	BLength() int
	FastRead(buf []byte) (int, error)
	FastWriteNocopy(buf []byte, w gthrift.NocopyWriter) int
}

type Sample struct {
	name  string
	val   interface{}
	bytes []byte
}

var samples []Sample

var (
	bytesCount  = 16
	stringCount = 16
	listCount   = 8
	mapCount    = 8
)

func getSamples() []Sample {
	return []Sample{
		{samples[0].name, getSimpleValue(), samples[0].bytes},
		{samples[1].name, getNestingValue(), samples[1].bytes},
		{samples[2].name, getNesting2Value(), samples[2].bytes},
	}
}

func getString() string {
	return strings.Repeat("你好,\b\n\r\t世界", stringCount)
}

func getBytes() []byte {
	return bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, bytesCount)
}

func getSimpleValue() *baseline.Simple {
	return &baseline.Simple{
		ByteField:   math.MaxInt8,
		I64Field:    math.MaxInt64,
		DoubleField: math.MaxFloat64,
		I32Field:    math.MaxInt32,
		StringField: getString(),
		BinaryField: getBytes(),
	}
}

func getNestingValue() *baseline.Nesting {
	var ret = &baseline.Nesting{
		String_:         getString(),
		ListSimple:      []*baseline.Simple{},
		Double:          math.MaxFloat64,
		I32:             math.MaxInt32,
		ListI32:         []int32{},
		I64:             math.MaxInt64,
		MapStringString: map[string]string{},
		SimpleStruct:    getSimpleValue(),
		MapI32I64:       map[int32]int64{},
		ListString:      []string{},
		Binary:          getBytes(),
		MapI64String:    map[int64]string{},
		ListI64:         []int64{},
		Byte:            math.MaxInt8,
		MapStringSimple: map[string]*baseline.Simple{},
	}

	for i := 0; i < listCount; i++ {
		ret.ListSimple = append(ret.ListSimple, getSimpleValue())
		ret.ListI32 = append(ret.ListI32, math.MinInt32)
		ret.ListI64 = append(ret.ListI64, math.MinInt64)
		ret.ListString = append(ret.ListString, getString())
	}

	for i := 0; i < mapCount; i++ {
		ret.MapStringString[strconv.Itoa(i)] = getString()
		ret.MapI32I64[int32(i)] = math.MinInt64
		ret.MapI64String[int64(i)] = getString()
		ret.MapStringSimple[strconv.Itoa(i)] = getSimpleValue()
	}

	return ret
}

func getNesting2Value() *baseline.Nesting2 {
	var ret = &baseline.Nesting2{
		MapSimpleNesting: map[*baseline.Simple]*baseline.Nesting{},
		SimpleStruct:     getSimpleValue(),
		Byte:             math.MaxInt8,
		Double:           math.MaxFloat64,
		ListNesting:      []*baseline.Nesting{},
		I64:              math.MaxInt64,
		NestingStruct:    getNestingValue(),
		Binary:           getBytes(),
		String_:          getString(),
		SetNesting:       []*baseline.Nesting{},
		I32:              math.MaxInt32,
	}
	for i := 0; i < mapCount; i++ {
		ret.MapSimpleNesting[getSimpleValue()] = getNestingValue()
	}
	for i := 0; i < listCount; i++ {
		ret.ListNesting = append(ret.ListNesting, getNestingValue())
		x := getNestingValue()
		x.I64 = int64(i)
		ret.SetNesting = append(ret.SetNesting, x)
	}
	return ret
}

func BenchmarkAllSize_BLength_FastCodec(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val.(FastCodec)
			assert.Equal(b, len(s.bytes), v.BLength())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v.BLength()
			}
		})
	}
}

func BenchmarkAllSize_BLength_Frugal_JIT(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val
			assert.Equal(b, jitEncoder.EncodedSize(v), len(s.bytes))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				jitEncoder.EncodedSize(v)
			}
		})
	}
}

func BenchmarkAllSize_BLength_Frugal_Reflect(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val
			assert.Equal(b, freflect.EncodedSize(v), len(s.bytes))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				freflect.EncodedSize(v)
			}
		})
	}
}

func BenchmarkAllSize_Marshal_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val
			mm := thrift.NewTMemoryBuffer()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mm.Reset()
				_ = v.(thrift.TStruct).Write(thrift.NewTBinaryProtocolTransport(mm))
			}
		})
	}
}

func BenchmarkAllSize_Marshal_FastCodec(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val.(FastCodec)
			buf := make([]byte, v.BLength())
			n := v.FastWriteNocopy(buf, nil)
			require.Equal(b, len(buf), n)
			require.Equal(b, len(s.bytes), n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = v.BLength()
				_ = v.FastWriteNocopy(buf, nil)
			}
		})
	}
}

func BenchmarkAllSize_Marshal_Frugal_JIT(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val
			buf := make([]byte, jitEncoder.EncodedSize(v))
			n, err := jitEncoder.EncodeObject(buf, nil, v)
			require.NoError(b, err)
			require.Equal(b, len(buf), n)
			assert.Equal(b, len(s.bytes), n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = jitEncoder.EncodedSize(v)
				_, _ = jitEncoder.EncodeObject(buf, nil, v)
			}
		})
	}
}

func BenchmarkAllSize_Marshal_Frugal_Reflect(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			v := s.val
			n := freflect.EncodedSize(v)
			buf, err := freflect.Append(make([]byte, 0, n), v)
			require.NoError(b, err)
			require.Equal(b, len(buf), n)
			assert.Equal(b, len(s.bytes), n)
			buf = buf[:0]
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = freflect.EncodedSize(v)
				_, _ = freflect.Append(buf, v)
			}
		})
	}
}

func objectmemclr(in interface{}) {
	switch v := in.(type) {
	case *baseline.Simple:
		*v = baseline.Simple{}
	case *baseline.Nesting:
		*v = baseline.Nesting{}
	case *baseline.Nesting2:
		*v = baseline.Nesting2{}
	default:
		panic("unknown type")
	}
}

func newByEFace(v interface{}) interface{} {
	return reflect.New(reflect.TypeOf(v).Elem()).Interface()
}

func BenchmarkAllSize_Unmarshal_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := bytes.NewBuffer(s.bytes)
			mm := thrift.NewTMemoryBuffer()
			v := newByEFace(s.val).(thrift.TStruct)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				*mm.Buffer = *buf // reset *Buffer to original one
				objectmemclr(v)
				_ = v.Read(thrift.NewTBinaryProtocolTransport(mm))
			}
		})
	}
}

func BenchmarkAllSize_Unmarshal_FastCodec(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			v := newByEFace(s.val).(FastCodec)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				objectmemclr(v)
				_, _ = v.FastRead(buf)
			}
		})
	}
}

func BenchmarkAllSize_Unmarshal_Frugal_JIT(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			v := newByEFace(s.val)
			n, err := jitDecoder.DecodeObject(buf, v)
			require.NoError(b, err)
			require.Equal(b, len(buf), n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				objectmemclr(v)
				_, _ = jitDecoder.DecodeObject(buf, v)
			}
		})
	}
}

func BenchmarkAllSize_Unmarshal_Frugal_Reflect(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			v := newByEFace(s.val)
			n, err := freflect.Decode(buf, v)
			require.NoError(b, err)
			require.Equal(b, len(buf), n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				objectmemclr(v)
				_, _ = freflect.Decode(buf, v)
			}
		})
	}
}

func BenchmarkAllSize_Parallel_Marshal_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := s.val
				mm := thrift.NewTMemoryBuffer()
				for pb.Next() {
					mm.Reset()
					_ = v.(thrift.TStruct).Write(thrift.NewTBinaryProtocolTransport(mm))
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Marshal_FastCodec(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := s.val.(FastCodec)
				buf := make([]byte, v.BLength())
				for pb.Next() {
					v.BLength()
					_ = v.FastWriteNocopy(buf, nil)
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Marshal_Frugal_JIT(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			jit.Pretouch(reflect.TypeOf(s.val), opts.GetDefaultOptions())
			b.SetBytes(int64(len(s.bytes)))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := s.val
				buf := make([]byte, jitEncoder.EncodedSize(v))
				for pb.Next() {
					jitEncoder.EncodedSize(v)
					_, _ = jitEncoder.EncodeObject(buf, nil, v)
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Marshal_Frugal_Reflect(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := s.val
				buf := make([]byte, 0, freflect.EncodedSize(v))
				for pb.Next() {
					freflect.EncodedSize(v)
					_, _ = freflect.Append(buf, v)
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Unmarshal_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := bytes.NewBuffer(s.bytes)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				mm := thrift.NewTMemoryBuffer()
				v := newByEFace(s.val).(thrift.TStruct)
				for pb.Next() {
					*mm.Buffer = *buf // reset *Buffer to original one
					objectmemclr(v)
					_ = v.Read(thrift.NewTBinaryProtocolTransport(mm))
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Unmarshal_FastCodec(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := newByEFace(s.val).(FastCodec)
				for pb.Next() {
					objectmemclr(v)
					_, _ = v.FastRead(buf)
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Unmarshal_Frugal_JIT(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			jit.Pretouch(reflect.TypeOf(s.val), opts.GetDefaultOptions())
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := newByEFace(s.val)
				for pb.Next() {
					objectmemclr(v)
					_, _ = jitDecoder.DecodeObject(buf, v)
				}
			})
		})
	}
}

func BenchmarkAllSize_Parallel_Unmarshal_Frugal_Reflect(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				v := newByEFace(s.val)
				for pb.Next() {
					objectmemclr(v)
					_, _ = freflect.Decode(buf, v)
				}
			})
		})
	}
}
