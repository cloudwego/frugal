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
	"strconv"
	"testing"
)

func TestAppendMapFastPaths(t *testing.T) {

	type EnumType int64
	type EmptyStruct struct{}

	{
		type TestStruct struct {
			M1 map[int8]int8         `frugal:"1,optional,map<i8:i8>"`
			M2 map[int8]int16        `frugal:"2,optional,map<i8:i16>"`
			M3 map[int8]int32        `frugal:"3,optional,map<i8:i32>"`
			M4 map[int8]int64        `frugal:"4,optional,map<i8:i64>"`
			M5 map[int8]EnumType     `frugal:"5,optional,map<i8:EnumType>"`
			M6 map[int8]string       `frugal:"6,optional,map<i8:string>"`
			M7 map[int8]*EmptyStruct `frugal:"7,optional,map<i8:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[int8]int8{11: 1, 12: 2},
			M2: map[int8]int16{21: 1, 22: 2},
			M3: map[int8]int32{31: 1, 32: 2},
			M4: map[int8]int64{41: 1, 42: 2},
			M5: map[int8]EnumType{51: 1, 52: 2},
			M6: map[int8]string{61: "1", 62: "2"},
			M7: map[int8]*EmptyStruct{71: {}, 72: {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}

	{
		type TestStruct struct {
			M1 map[int16]int8         `frugal:"1,optional,map<i16:i8>"`
			M2 map[int16]int16        `frugal:"2,optional,map<i16:i16>"`
			M3 map[int16]int32        `frugal:"3,optional,map<i16:i32>"`
			M4 map[int16]int64        `frugal:"4,optional,map<i16:i64>"`
			M5 map[int16]EnumType     `frugal:"5,optional,map<i16:EnumType>"`
			M6 map[int16]string       `frugal:"6,optional,map<i16:string>"`
			M7 map[int16]*EmptyStruct `frugal:"7,optional,map<i16:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[int16]int8{11: 1, 12: 2},
			M2: map[int16]int16{21: 1, 22: 2},
			M3: map[int16]int32{31: 1, 32: 2},
			M4: map[int16]int64{41: 1, 42: 2},
			M5: map[int16]EnumType{51: 1, 52: 2},
			M6: map[int16]string{61: "1", 62: "2"},
			M7: map[int16]*EmptyStruct{71: {}, 72: {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[int32]int8         `frugal:"1,optional,map<i32:i8>"`
			M2 map[int32]int16        `frugal:"2,optional,map<i32:i16>"`
			M3 map[int32]int32        `frugal:"3,optional,map<i32:i32>"`
			M4 map[int32]int64        `frugal:"4,optional,map<i32:i64>"`
			M5 map[int32]EnumType     `frugal:"5,optional,map<i32:EnumType>"`
			M6 map[int32]string       `frugal:"6,optional,map<i32:string>"`
			M7 map[int32]*EmptyStruct `frugal:"7,optional,map<i32:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[int32]int8{11: 1, 12: 2},
			M2: map[int32]int16{21: 1, 22: 2},
			M3: map[int32]int32{31: 1, 32: 2},
			M4: map[int32]int64{41: 1, 42: 2},
			M5: map[int32]EnumType{51: 1, 52: 2},
			M6: map[int32]string{61: "1", 62: "2"},
			M7: map[int32]*EmptyStruct{71: {}, 72: {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[int64]int8         `frugal:"1,optional,map<i64:i8>"`
			M2 map[int64]int16        `frugal:"2,optional,map<i64:i16>"`
			M3 map[int64]int32        `frugal:"3,optional,map<i64:i32>"`
			M4 map[int64]int64        `frugal:"4,optional,map<i64:i64>"`
			M5 map[int64]EnumType     `frugal:"5,optional,map<i64:EnumType>"`
			M6 map[int64]string       `frugal:"6,optional,map<i64:string>"`
			M7 map[int64]*EmptyStruct `frugal:"7,optional,map<i64:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[int64]int8{11: 1, 12: 2},
			M2: map[int64]int16{21: 1, 22: 2},
			M3: map[int64]int32{31: 1, 32: 2},
			M4: map[int64]int64{41: 1, 42: 2},
			M5: map[int64]EnumType{51: 1, 52: 2},
			M6: map[int64]string{61: "1", 62: "2"},
			M7: map[int64]*EmptyStruct{71: {}, 72: {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[EnumType]int8         `frugal:"1,optional,map<EnumType:i8>"`
			M2 map[EnumType]int16        `frugal:"2,optional,map<EnumType:i16>"`
			M3 map[EnumType]int32        `frugal:"3,optional,map<EnumType:i32>"`
			M4 map[EnumType]int64        `frugal:"4,optional,map<EnumType:i64>"`
			M5 map[EnumType]EnumType     `frugal:"5,optional,map<EnumType:EnumType>"`
			M6 map[EnumType]string       `frugal:"6,optional,map<EnumType:string>"`
			M7 map[EnumType]*EmptyStruct `frugal:"7,optional,map<EnumType:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[EnumType]int8{11: 1, 12: 2},
			M2: map[EnumType]int16{21: 1, 22: 2},
			M3: map[EnumType]int32{31: 1, 32: 2},
			M4: map[EnumType]int64{41: 1, 42: 2},
			M5: map[EnumType]EnumType{51: 1, 52: 2},
			M6: map[EnumType]string{61: "1", 62: "2"},
			M7: map[EnumType]*EmptyStruct{71: {}, 72: {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[string]int8         `frugal:"1,optional,map<string:i8>"`
			M2 map[string]int16        `frugal:"2,optional,map<string:i16>"`
			M3 map[string]int32        `frugal:"3,optional,map<string:i32>"`
			M4 map[string]int64        `frugal:"4,optional,map<string:i64>"`
			M5 map[string]EnumType     `frugal:"5,optional,map<string:EnumType>"`
			M6 map[string]string       `frugal:"6,optional,map<string:string>"`
			M7 map[string]*EmptyStruct `frugal:"7,optional,map<string:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[string]int8{"11": 1, "12": 2},
			M2: map[string]int16{"21": 1, "22": 2},
			M3: map[string]int32{"31": 1, "32": 2},
			M4: map[string]int64{"41": 1, "42": 2},
			M5: map[string]EnumType{"51": 1, "52": 2},
			M6: map[string]string{"61": "1", "62": "2"},
			M7: map[string]*EmptyStruct{"71": {}, "72": {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[*EmptyStruct]int8         `frugal:"1,optional,map<EmptyStruct:i8>"`
			M2 map[*EmptyStruct]int16        `frugal:"2,optional,map<EmptyStruct:i16>"`
			M3 map[*EmptyStruct]int32        `frugal:"3,optional,map<EmptyStruct:i32>"`
			M4 map[*EmptyStruct]int64        `frugal:"4,optional,map<EmptyStruct:i64>"`
			M5 map[*EmptyStruct]EnumType     `frugal:"5,optional,map<EmptyStruct:EnumType>"`
			M6 map[*EmptyStruct]string       `frugal:"6,optional,map<EmptyStruct:string>"`
			M7 map[*EmptyStruct]*EmptyStruct `frugal:"7,optional,map<EmptyStruct:EmptyStruct>"`
		}
		p0 := &TestStruct{
			M1: map[*EmptyStruct]int8{{}: 1},
			M2: map[*EmptyStruct]int16{{}: 1},
			M3: map[*EmptyStruct]int32{{}: 1},
			M4: map[*EmptyStruct]int64{{}: 1},
			M5: map[*EmptyStruct]EnumType{{}: 1},
			M6: map[*EmptyStruct]string{{}: "1"},
			M7: map[*EmptyStruct]*EmptyStruct{{}: {}},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[float64]int8         `frugal:"1,optional,map<double:i8>"`
			M2 map[float64]int16        `frugal:"2,optional,map<double:i16>"`
			M3 map[float64]int32        `frugal:"3,optional,map<double:i32>"`
			M4 map[float64]int64        `frugal:"4,optional,map<double:i64>"`
			M5 map[float64]EnumType     `frugal:"5,optional,map<double:EnumType>"`
			M6 map[float64]string       `frugal:"6,optional,map<double:string>"`
			M7 map[float64]*EmptyStruct `frugal:"7,optional,map<double:EmptyStruct>"`
			M8 map[float64]bool         `frugal:"8,optional,map<double:bool>"`
			M9 map[float64]float64      `frugal:"9,optional,map<double:double>"`
		}
		p0 := &TestStruct{
			M1: map[float64]int8{1.1: 1, 1.2: 2},
			M2: map[float64]int16{2.1: 1, 2.2: 2},
			M3: map[float64]int32{3.1: 1, 3.2: 2},
			M4: map[float64]int64{4.1: 1, 4.2: 2},
			M5: map[float64]EnumType{5.1: 1, 5.2: 2},
			M6: map[float64]string{6.1: "1", 6.2: "2"},
			M7: map[float64]*EmptyStruct{7.1: {}, 7.2: {}},
			M8: map[float64]bool{8.1: true, 8.2: false},
			M9: map[float64]float64{9.1: 0.5, 9.2: -0.5},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
	{
		type TestStruct struct {
			M1 map[bool][]byte         `frugal:"1,optional,map<bool:binary>"`
			M2 map[int8][]byte         `frugal:"2,optional,map<i8:binary>"`
			M3 map[int16][]byte        `frugal:"3,optional,map<i16:binary>"`
			M4 map[int32][]byte        `frugal:"4,optional,map<i32:binary>"`
			M5 map[int64][]byte        `frugal:"5,optional,map<i64:binary>"`
			M6 map[float64][]byte      `frugal:"6,optional,map<double:binary>"`
			M7 map[EnumType][]byte     `frugal:"7,optional,map<EnumType:binary>"`
			M8 map[string][]byte       `frugal:"8,optional,map<string:binary>"`
			M9 map[*EmptyStruct][]byte `frugal:"9,optional,map<EmptyStruct:binary>"`
		}
		p0 := &TestStruct{
			M1: map[bool][]byte{true: []byte("1"), false: []byte("22")},
			M2: map[int8][]byte{21: []byte("1"), 22: []byte("22")},
			M3: map[int16][]byte{31: []byte("1"), 32: []byte("22")},
			M4: map[int32][]byte{41: []byte("1"), 42: []byte("22")},
			M5: map[int64][]byte{51: []byte("1"), 52: []byte("22"), 53: {}},
			M6: map[float64][]byte{6.1: []byte("1"), 6.2: []byte("22")},
			M7: map[EnumType][]byte{71: []byte("1"), 72: []byte("22")},
			M8: map[string][]byte{"81": []byte("1"), "82": []byte("22"), "83": {}},
			M9: map[*EmptyStruct][]byte{{}: []byte("1")},
		}
		p1 := &TestStruct{}
		checkEncodingRoundTrip(t, p0, p1)
	}
}

func BenchmarkAppendMapFastPath(b *testing.B) {
	type I64I64 struct {
		M map[int64]int64 `frugal:"1,optional,map<i64:i64>"`
	}
	type StringString struct {
		M map[string]string `frugal:"1,optional,map<string:string>"`
	}
	type BoolString struct {
		M map[bool]string `frugal:"1,optional,map<bool:string>"`
	}

	mkI64I64 := func(n int) *I64I64 {
		m := make(map[int64]int64, n)
		for i := 0; i < n; i++ {
			m[int64(i)] = int64(i * 3)
		}
		return &I64I64{M: m}
	}
	mkStringString := func(n int) *StringString {
		m := make(map[string]string, n)
		for i := 0; i < n; i++ {
			s := strconv.Itoa(i)
			m["key-"+s] = "value-" + s
		}
		return &StringString{M: m}
	}
	mkBoolString := func() *BoolString {
		return &BoolString{M: map[bool]string{false: "off", true: "on"}}
	}

	b.Run("I64I64_64", func(b *testing.B) {
		p := mkI64I64(64)
		buf := make([]byte, 0, EncodedSize(p))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Append(buf[:0], p)
		}
	})

	b.Run("StringString_64", func(b *testing.B) {
		p := mkStringString(64)
		buf := make([]byte, 0, EncodedSize(p))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Append(buf[:0], p)
		}
	})

	b.Run("BoolString_2", func(b *testing.B) {
		p := mkBoolString()
		buf := make([]byte, 0, EncodedSize(p))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Append(buf[:0], p)
		}
	})
}
