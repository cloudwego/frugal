/*
 * Copyright 2022 ByteDance Inc.
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

//go:generate kitex -thrift frugal_tag baseline.thrift
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
	`unsafe`

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/frugal"
	`github.com/cloudwego/frugal/internal/rt`
	"github.com/cloudwego/frugal/testdata/kitex_gen/baseline"
	"github.com/cloudwego/kitex/pkg/protocol/bthrift"
	"github.com/stretchr/testify/assert"
	thriftiter "github.com/thrift-iterator/go"
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

type FastAPI interface {
	BLength() int
	FastRead(buf []byte) (int, error)
	FastWriteNocopy(buf []byte, binaryWriter bthrift.BinaryWriter) int
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
	listCount       = 8
	mapCount        = 8
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

func BenchmarkMarshalAllSize_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			var v = s.val
			var mm = thrift.NewTMemoryBuffer()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mm.Reset()
				_ = v.(thrift.TStruct).Write(thrift.NewTBinaryProtocolTransport(mm))
			}
		})
	}
}

func BenchmarkMarshalAllSize_ThriftIterator(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			var v = s.val
			act, err := thriftiter.Marshal(v)
			if err != nil {
				b.Fatal(err)
			}
			assert.Equal(b, len(s.bytes), len(act))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = thriftiter.Marshal(v)
			}
		})
	}
}

func BenchmarkMarshalAllSize_KitexFast(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			var v = s.val.(FastAPI)
			buf := make([]byte, v.BLength())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v.BLength()
				_ = v.FastWriteNocopy(buf, nil)
			}
		})
	}
}

func BenchmarkMarshalAllSize_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			var v = s.val
			buf := make([]byte, frugal.EncodedSize(v))
			act, err := frugal.EncodeObject(buf, nil, v)
			if err != nil {
				b.Fatal(err)
			}
			assert.Equal(b, len(s.bytes), act)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				frugal.EncodedSize(v)
				_, _ = frugal.EncodeObject(buf, nil, v)
			}
		})
	}
}

//go:noescape
//go:linkname typedmemclr runtime.typedmemclr
//goland:noinspection GoUnusedParameter
func typedmemclr(typ *rt.GoType, ptr unsafe.Pointer)

func objectmemclr(v interface{}) {
	p := rt.UnpackEface(v)
	typedmemclr(rt.PtrElem(p.Type), p.Value)
}

func BenchmarkUnmarshalAllSize_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()
			mm := thrift.NewTMemoryBuffer()
			var v = reflect.New(rtype).Interface()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mm.Reset()
				objectmemclr(v)
				_, _ = mm.Write(buf)
				_ = v.(thrift.TStruct).Read(thrift.NewTBinaryProtocolTransport(mm))
			}
		})
	}
}

func BenchmarkUnmarshalAllSize_ThriftIterator(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()
			var v = reflect.New(rtype).Interface()
			err := thriftiter.Unmarshal(buf, v)
			if err != nil {
				b.Fatal(err)
			}
			// assert.Equal(b, s.val, v)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				objectmemclr(v)
				_ = thriftiter.Unmarshal(buf, v)
			}
		})
	}
}

func BenchmarkUnmarshalAllSize_KitexFast(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()
			var v = reflect.New(rtype).Interface()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				objectmemclr(v)
				_, _ = v.(FastAPI).FastRead(buf)
			}
		})
	}

}

func BenchmarkUnmarshalAllSize_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()
			var v = reflect.New(rtype).Interface()
			act, err := frugal.DecodeObject(buf, v)
			if err != nil {
				b.Fatal(err)
			}
			assert.Equal(b, len(s.bytes), act)
			assert.Equal(b, s.val.(FastAPI).BLength(), frugal.EncodedSize(v))
			b.SetBytes(int64(len(buf)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				objectmemclr(v)
				_, _ = frugal.DecodeObject(buf, v)
			}
		})
	}
}

func BenchmarkMarshalAllSize_Parallel_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = s.val
				var mm = thrift.NewTMemoryBuffer()
				for pb.Next() {
					mm.Reset()
					_ = v.(thrift.TStruct).Write(thrift.NewTBinaryProtocolTransport(mm))
				}
			})
		})
	}
}

func BenchmarkMarshalAllSize_Parallel_ThriftIterator(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			var v = s.val
			act, err := thriftiter.Marshal(v)
			if err != nil {
				b.Fatal(err)
			}
			assert.Equal(b, len(s.bytes), len(act))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = s.val
				for pb.Next() {
					_, _ = thriftiter.Marshal(v)
				}
			})
		})
	}
}

func BenchmarkMarshalAllSize_Parallel_KitexFast(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = s.val.(FastAPI)
				buf := make([]byte, v.BLength())
				for pb.Next() {
					v.BLength()
					_ = v.FastWriteNocopy(buf, nil)
				}
			})
		})
	}
}

func BenchmarkMarshalAllSize_Parallel_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			var v = s.val
			buf := make([]byte, frugal.EncodedSize(v))
			act, err := frugal.EncodeObject(buf, nil, v)
			if err != nil {
				b.Fatal(err)
			}
			assert.Equal(b, len(s.bytes), act)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = s.val
				buf := make([]byte, frugal.EncodedSize(v))
				for pb.Next() {
					frugal.EncodedSize(v)
					_, _ = frugal.EncodeObject(buf, nil, v)
				}
			})
		})
	}
}

func BenchmarkUnmarshalAllSize_Parallel_ApacheThrift(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				mm := thrift.NewTMemoryBuffer()
				var v = reflect.New(rtype).Interface()
				for pb.Next() {
					mm.Reset()
					objectmemclr(v)
					_, _ = mm.Write(buf)
					_ = v.(thrift.TStruct).Read(thrift.NewTBinaryProtocolTransport(mm))
				}
			})
		})
	}
}

func BenchmarkUnmarshalAllSize_Parallel_ThriftIterator(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()
			var v = reflect.New(rtype).Interface()
			err := thriftiter.Unmarshal(buf, v)
			if err != nil {
				b.Fatal(err)
			}

			// assert.Equal(b, s.val, v)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = reflect.New(rtype).Interface()
				for pb.Next() {
					objectmemclr(v)
					_ = thriftiter.Unmarshal(buf, v)
				}
			})
		})
	}
}

func BenchmarkUnmarshalAllSize_Parallel_KitexFast(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			rtype := reflect.TypeOf(s.val).Elem()
			buf := s.bytes
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = reflect.New(rtype).Interface()
				for pb.Next() {
					objectmemclr(v)
					_, _ = v.(FastAPI).FastRead(buf)
				}
			})
		})
	}
}

func BenchmarkUnmarshalAllSize_Parallel_Frugal(b *testing.B) {
	for _, s := range getSamples() {
		b.Run(s.name, func(b *testing.B) {
			defer func() {
				if e := recover(); e != nil {
					b.Fatal(e)
				}
			}()
			b.SetBytes(int64(len(s.bytes)))
			buf := s.bytes
			rtype := reflect.TypeOf(s.val).Elem()
			var v = reflect.New(rtype).Interface()
			act, err := frugal.DecodeObject(buf, v)
			if err != nil {
				b.Fatal(err)
			}
			assert.Equal(b, len(s.bytes), act)
			assert.Equal(b, s.val.(FastAPI).BLength(), frugal.EncodedSize(v))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var v = reflect.New(rtype).Interface()
				for pb.Next() {
					objectmemclr(v)
					_, err = frugal.DecodeObject(buf, v)
				}
			})
		})
	}
}
