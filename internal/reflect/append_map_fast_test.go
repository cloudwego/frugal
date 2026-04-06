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

	"github.com/cloudwego/frugal/internal/assert"
)

func TestAppendMapFastPaths(t *testing.T) {

	type EnumType int64
	type EmptyStruct struct{}

	doTest := func(t *testing.T, p0, p1 any) {
		t.Helper()
		b, err := Append(nil, p0)
		assert.Nil(t, err)
		_, err = Decode(b, p1)
		assert.Nil(t, err)
		assert.DeepEqual(t, p0, p1)
	}

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
		doTest(t, p0, p1)
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
		doTest(t, p0, p1)
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
		doTest(t, p0, p1)
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
		doTest(t, p0, p1)
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
		doTest(t, p0, p1)
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
		doTest(t, p0, p1)
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
		doTest(t, p0, p1)
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
