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
	"strings"
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

// Compact protocol test structs.

type CompactPrimitives struct {
	FBool   bool    `frugal:"1,required"`
	FByte   int8    `frugal:"2,required"`
	I16     int16   `frugal:"3,required"`
	I32     int32   `frugal:"4,required"`
	I64     int64   `frugal:"5,required"`
	Double  float64 `frugal:"6,required"`
	String_ string  `frugal:"7,required"`
	Binary  []byte  `frugal:"8,default"`
	Enum    int64   `frugal:"9,default,i64"`
}

func (p *CompactPrimitives) InitDefault() {}

type CompactWithList struct {
	L0 []int32  `frugal:"1,required,list<i32>"`
	L1 []string `frugal:"2,default,list<string>"`
}

func (p *CompactWithList) InitDefault() {}

type CompactSparseFields struct {
	F1 bool  `frugal:"1,required"`
	F5 int64 `frugal:"5,required"`
	F9 int32 `frugal:"9,required"`
}

func (p *CompactSparseFields) InitDefault() {}

type CompactWithI16List struct {
	L0 []int16 `frugal:"1,required,list<i16>"`
}

func (p *CompactWithI16List) InitDefault() {}

type CompactWithI64List struct {
	L0 []int64 `frugal:"1,required,list<i64>"`
}

func (p *CompactWithI64List) InitDefault() {}

// ---- Tests ----

func TestVarintRoundTrip(t *testing.T) {
	tests := []uint64{0, 1, 127, 128, 255, 16383, 16384, 2097151, 2097152, 1<<63 - 1}
	for _, v := range tests {
		b := appendVarint(nil, v)
		got, n := decodeVarint(b)
		assert.Equal(t, v, got)
		assert.Equal(t, len(b), n)
	}
}

func TestZigzag32RoundTrip(t *testing.T) {
	tests := []int32{0, 1, -1, 127, -127, 128, -128, 16383, -16383, 1<<31 - 1, -(1 << 31)}
	for _, v := range tests {
		b := appendZigzag32(nil, v)
		got, n := decodeZigzag32(b)
		assert.Equal(t, v, got)
		assert.Equal(t, len(b), n)
	}
}

func TestZigzag64RoundTrip(t *testing.T) {
	tests := []int64{0, 1, -1, 127, -127, 128, -128, 16383, -16383, 1<<63 - 1, -(1 << 63)}
	for _, v := range tests {
		b := appendZigzag64(nil, v)
		got, n := decodeZigzag64(b)
		assert.Equal(t, v, got)
		assert.Equal(t, len(b), n)
	}
}

func TestVarintLen(t *testing.T) {
	assert.Equal(t, 1, varintLen(0))
	assert.Equal(t, 1, varintLen(127))
	assert.Equal(t, 2, varintLen(128))
	assert.Equal(t, 2, varintLen(16383))
	assert.Equal(t, 3, varintLen(16384))
}

func TestCompactFieldHeaderSize(t *testing.T) {
	assert.Equal(t, 1, compactFieldHeaderSize(0, 1))
	assert.Equal(t, 1, compactFieldHeaderSize(0, 15))
	assert.Equal(t, 3, compactFieldHeaderSize(0, 16))
	assert.Equal(t, 3, compactFieldHeaderSize(5, 21))
	assert.Equal(t, 1, compactFieldHeaderSize(5, 6))
	// Field ID 0 with lastId=0: delta=0, falls back to 3-byte header
	assert.Equal(t, 3, compactFieldHeaderSize(0, 0))
}

func TestCompactFieldHeaderWriteRead(t *testing.T) {
	tests := []struct{ last, id uint16 }{
		{0, 1}, {0, 15}, {0, 16}, {0, 300},
		{5, 20}, {5, 6}, {100, 115}, {100, 120},
		{0, 0},
	}
	for _, tc := range tests {
		var b []byte
		b = writeCompactFieldHeader(b, tc.last, tc.id, ctI32)
		wt, fid, n := readCompactFieldHeader(b, 0, tc.last)
		assert.Equal(t, ctI32, wt)
		assert.Equal(t, tc.id, fid)
		assert.Equal(t, len(b), n)
	}
}

func TestCompactStruct_Primitives(t *testing.T) {
	p := &CompactPrimitives{
		FBool:   true,
		FByte:   0x55,
		I16:     1234,
		I32:     56789,
		I64:     -999999999,
		Double:  3.14159,
		String_: "hello compact",
	}

	n := EncodedSizeCompact(p)
	assert.True(t, n > 0)
	buf := make([]byte, n)
	ret, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)
	assert.Equal(t, n, len(ret))

	p2 := &CompactPrimitives{}
	i, err := DecodeCompact(ret, p2)
	assert.Nil(t, err)
	assert.Equal(t, n, i)

	assert.Equal(t, p.FBool, p2.FBool)
	assert.Equal(t, p.FByte, p2.FByte)
	assert.Equal(t, p.I16, p2.I16)
	assert.Equal(t, p.I32, p2.I32)
	assert.Equal(t, p.I64, p2.I64)
	assert.Equal(t, p.Double, p2.Double)
	assert.Equal(t, p.String_, p2.String_)
}

func TestCompactStruct_BoolTrue(t *testing.T) {
	p := &CompactPrimitives{FBool: true}
	buf, err := AppendCompact(nil, p)
	assert.Nil(t, err)
	// delta=1 in high nibble, ctBOOL_TRUE=0x01 in low nibble: 0x10 | 0x01 = 0x11
	assert.Equal(t, byte(0x11), buf[0])

	p2 := &CompactPrimitives{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.Equal(t, true, p2.FBool)
}

func TestCompactStruct_BoolFalse(t *testing.T) {
	p := &CompactPrimitives{FBool: false}
	buf, err := AppendCompact(nil, p)
	assert.Nil(t, err)
	// delta=1 in high nibble, ctBOOL_FALSE=0x02 in low nibble: 0x10 | 0x02 = 0x12
	assert.Equal(t, byte(0x12), buf[0])

	p2 := &CompactPrimitives{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.Equal(t, false, p2.FBool)
}

func TestCompactStruct_ListOfInt32(t *testing.T) {
	p := &CompactWithList{L0: []int32{1, 2, 3, 100, -50}}

	n := EncodedSizeCompact(p)
	assert.True(t, n > 0)
	buf := make([]byte, n)
	ret, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)
	assert.Equal(t, n, len(ret))

	p2 := &CompactWithList{}
	_, err = DecodeCompact(ret, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.L0, p2.L0)
}

func TestCompactStruct_ListOfString(t *testing.T) {
	p := &CompactWithList{L1: []string{"hello", "world"}}

	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	ret, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &CompactWithList{}
	_, err = DecodeCompact(ret, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.L1, p2.L1)
}

func TestCompactStruct_ListOfInt16(t *testing.T) {
	p := &CompactWithI16List{L0: []int16{1, -1, 32767, -32768}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &CompactWithI16List{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.L0, p2.L0)
}

func TestCompactStruct_ListOfInt64(t *testing.T) {
	p := &CompactWithI64List{L0: []int64{0, 1, -1, 1<<62 - 1, -(1 << 62)}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &CompactWithI64List{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.L0, p2.L0)
}

func TestCompactStruct_SparseFields(t *testing.T) {
	p := &CompactSparseFields{F1: true, F5: 123456789, F9: -999}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	// field 1 bool true: delta=1, ctBOOL_TRUE=1 => 0x11
	assert.Equal(t, byte(0x11), buf[0])
	// field 5 i64: delta=4 (5-1), ctI64=6 => 0x46

	p2 := &CompactSparseFields{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.Equal(t, p.F1, p2.F1)
	assert.Equal(t, p.F5, p2.F5)
	assert.Equal(t, p.F9, p2.F9)
}

func TestCompactNilStruct(t *testing.T) {
	type Empty struct {
	}
	p := &Empty{}
	n := EncodedSizeCompact(p)
	assert.Equal(t, 1, n)
	b, err := AppendCompact(nil, p)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(b))
	assert.Equal(t, byte(ctSTOP), b[0])
}

func assertCompactPanic(t *testing.T, f func()) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got nil")
		}
		s, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if !strings.Contains(s, "compact protocol") && !strings.Contains(s, "not implemented") {
			t.Fatalf("expected panic containing 'compact protocol', got: %q", s)
		}
	}()
	f()
}

func TestCompactStruct_MapInt32String(t *testing.T) {
	type WithMap struct {
		M map[int32]string `frugal:"1,required"`
	}
	p := &WithMap{M: map[int32]string{1: "one", 2: "two", 3: "three"}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &WithMap{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.M, p2.M)
}

func TestCompactStruct_MapInt32Int64(t *testing.T) {
	type WithMap struct {
		M map[int32]int64 `frugal:"1,required"`
	}
	p := &WithMap{M: map[int32]int64{1: 100, -5: -999, 0: 0}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &WithMap{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.M, p2.M)
}

func TestCompactStruct_EmptyMap(t *testing.T) {
	type WithMap struct {
		M map[int32]string `frugal:"1,required"`
	}
	p := &WithMap{M: map[int32]string{}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &WithMap{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(p2.M))
}

func TestCompactStruct_ListOfMap(t *testing.T) {
	type WithNested struct {
		L []map[int32]string `frugal:"1,required,list<map<i32:string>>"`
	}
	p := &WithNested{L: []map[int32]string{
		{1: "a", 2: "b"},
		{3: "c"},
	}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &WithNested{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.True(t, len(p.L) == len(p2.L))
	for i := range p.L {
		assert.DeepEqual(t, p.L[i], p2.L[i])
	}
}

func TestCompactStruct_MapOfList(t *testing.T) {
	type WithNested struct {
		M map[int32][]int32 `frugal:"1,required,map<i32:list<i32>>"`
	}
	p := &WithNested{M: map[int32][]int32{
		1: {10, 20},
		2: {30},
	}}
	n := EncodedSizeCompact(p)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)

	p2 := &WithNested{}
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)
	assert.DeepEqual(t, p.M, p2.M)
}

func TestCompactStruct_FullTestTypes(t *testing.T) {
	p := NewTestTypes()
	p.FBool = true
	p.FByte = 0x55
	p.I8 = -8
	p.I16 = 1234
	p.I32 = 56789
	p.I64 = -999999999
	p.Double = 3.14159
	p.String_ = "hello compact"
	p.Binary = []byte{0xde, 0xad, 0xbe, 0xef}
	p.Enum = Numberz_TEN
	p.UID = 42
	p.S = &Msg{Message: "nested", Type: 99}
	p.M0 = map[int32]int32{1: 100, 2: 200}
	p.M1 = map[int32]string{1: "one", 2: "two"}
	p.M2 = map[int32]*Msg{1: {Message: "a", Type: 1}}
	p.M3 = map[string]*Msg{"key1": {Message: "b", Type: 2}}
	p.L0 = []int32{10, 20, 30}
	p.L1 = []string{"hello", "world"}
	p.L2 = []*Msg{{Message: "x", Type: 1}}
	p.S0 = []int32{1, 2, 3}
	p.S1 = []string{"a", "b"}
	p.LM = []map[int32]int32{{1: 10}, {2: 20}}
	p.ML = map[int32][]int32{1: {10, 20}}
	p.MS = map[int32][]int32{1: {30, 40}}

	n := EncodedSizeCompact(p)
	assert.True(t, n > 0)
	buf := make([]byte, n)
	_, err := AppendCompact(buf[:0], p)
	assert.Nil(t, err)
	assert.Equal(t, n, len(buf))

	p2 := NewTestTypes()
	_, err = DecodeCompact(buf, p2)
	assert.Nil(t, err)

	// Compare all fields
	assert.Equal(t, p.FBool, p2.FBool)
	assert.Equal(t, p.FByte, p2.FByte)
	assert.Equal(t, p.I8, p2.I8)
	assert.Equal(t, p.I16, p2.I16)
	assert.Equal(t, p.I32, p2.I32)
	assert.Equal(t, p.I64, p2.I64)
	assert.Equal(t, p.Double, p2.Double)
	assert.Equal(t, p.String_, p2.String_)
	assert.BytesEqual(t, p.Binary, p2.Binary)
	assert.Equal(t, p.Enum, p2.Enum)
	assert.Equal(t, p.UID, p2.UID)
	assert.True(t, p2.S != nil)
	assert.Equal(t, p.S.Message, p2.S.Message)
	assert.Equal(t, p.S.Type, p2.S.Type)
	assert.DeepEqual(t, p.M0, p2.M0)
	assert.DeepEqual(t, p.M1, p2.M1)
	assert.True(t, len(p2.M2) == 1)
	assert.Equal(t, p.M2[int32(1)].Message, p2.M2[int32(1)].Message)
	assert.True(t, len(p2.M3) == 1)
	assert.Equal(t, p.M3["key1"].Message, p2.M3["key1"].Message)
	assert.DeepEqual(t, p.L0, p2.L0)
	assert.DeepEqual(t, p.L1, p2.L1)
	assert.True(t, len(p2.L2) == 1)
	assert.Equal(t, p.L2[0].Message, p2.L2[0].Message)
	assert.DeepEqual(t, p.S0, p2.S0)
	assert.DeepEqual(t, p.S1, p2.S1)
	assert.DeepEqual(t, p.LM, p2.LM)
	assert.DeepEqual(t, p.ML, p2.ML)
	assert.DeepEqual(t, p.MS, p2.MS)
}

func TestBinaryAPI_StillWorks(t *testing.T) {
	p := NewTestTypes()
	p.FBool = true
	p.FByte = 0x55
	p.I16 = 1234
	p.I32 = 56789
	p.I64 = 999999999
	p.String_ = "hello world"

	n := EncodedSize(p)
	assert.True(t, n > 0)

	b, err := Append(make([]byte, 0, n), p)
	assert.Nil(t, err)
	assert.Equal(t, n, len(b))

	p2 := NewTestTypes()
	_, err = Decode(b, p2)
	assert.Nil(t, err)

	assert.Equal(t, p.FBool, p2.FBool)
	assert.Equal(t, p.FByte, p2.FByte)
	assert.Equal(t, p.I16, p2.I16)
	assert.Equal(t, p.I32, p2.I32)
	assert.Equal(t, p.I64, p2.I64)
	assert.Equal(t, p.String_, p2.String_)
}

func TestCompactBinary_SizeComparison(t *testing.T) {
	p := NewTestTypes()
	p.FBool = true
	p.FByte = 0x55
	p.I8 = -8
	p.I16 = 1234
	p.I32 = 56789
	p.I64 = -999999999
	p.Double = 3.14159
	p.String_ = "hello compact"
	p.Binary = []byte{0xde, 0xad, 0xbe, 0xef}
	p.Enum = Numberz_TEN
	p.UID = 42
	p.S = &Msg{Message: "nested", Type: 99}
	p.M0 = map[int32]int32{1: 100, 2: 200}
	p.M1 = map[int32]string{1: "one", 2: "two"}
	p.L0 = []int32{10, 20, 30}
	p.L1 = []string{"hello", "world"}

	binSize := EncodedSize(p)
	compactSize := EncodedSizeCompact(p)
	assert.True(t, compactSize < binSize,
		"Compact size (%d) should be smaller than Binary size (%d)", compactSize, binSize)

	t.Logf("Binary=%d bytes, Compact=%d bytes, saved=%d bytes (%.1f%%)",
		binSize, compactSize, binSize-compactSize,
		float64(binSize-compactSize)/float64(binSize)*100)
}

func TestCompactSizeVsActual(t *testing.T) {
	p := NewTestTypes()
	p.FBool = true
	p.FByte = 0x55
	p.I8 = -8
	p.I16 = 1234
	p.I32 = 56789
	p.I64 = -999999999
	p.Double = 3.14159
	p.String_ = "hello compact"
	p.Binary = []byte{0xde, 0xad, 0xbe, 0xef}
	p.Enum = Numberz_TEN
	p.UID = 42
	p.S = &Msg{Message: "nested", Type: 99}
	p.M0 = map[int32]int32{1: 100, 2: 200}
	p.M1 = map[int32]string{1: "one", 2: "two"}
	p.M2 = map[int32]*Msg{1: {Message: "a", Type: 1}}
	p.M3 = map[string]*Msg{"key1": {Message: "b", Type: 2}}
	p.L0 = []int32{10, 20, 30}
	p.L1 = []string{"hello", "world"}
	p.L2 = []*Msg{{Message: "x", Type: 1}}
	p.S0 = []int32{1, 2, 3}
	p.S1 = []string{"a", "b"}
	p.LM = []map[int32]int32{{1: 10}, {2: 20}}
	p.ML = map[int32][]int32{1: {10, 20}}
	p.MS = map[int32][]int32{1: {30, 40}}

	estimated := EncodedSizeCompact(p)
	actual, err := AppendCompact(nil, p)
	assert.Nil(t, err)
	assert.Equal(t, estimated, len(actual), "estimated=%d actual=%d", estimated, len(actual))
}

func TestCompactOptional(t *testing.T) {
	type S struct {
		F1 *bool   `frugal:"1,optional"`
		F2 *int32  `frugal:"2,optional,i32"`
		F3 *string `frugal:"3,optional,string"`
	}
	// nil optional: no field encoded
	p := &S{}
	n := EncodedSizeCompact(p)
	assert.Equal(t, 1, n) // just STOP
	b, _ := AppendCompact(nil, p)
	assert.Equal(t, 1, len(b))
	assert.Equal(t, byte(ctSTOP), b[0])

	// set optional: field encoded
	v := int32(42)
	p.F2 = &v
	b, _ = AppendCompact(nil, p)
	p2 := &S{}
	_, err := DecodeCompact(b, p2)
	assert.Nil(t, err)
	assert.True(t, p2.F2 != nil)
	assert.Equal(t, v, *p2.F2)
}

func TestCompactRequired(t *testing.T) {
	type S0 struct {
		V *bool `frugal:"1,optional,bool"`
	}
	type S1 struct {
		V bool `frugal:"1,required,bool"`
	}
	v := true
	p0 := &S0{}
	b, _ := AppendCompact(nil, p0)

	p1 := &S1{}
	_, err := DecodeCompact(b, p1)
	assert.DeepEqual(t, newRequiredFieldNotSetException("V"), err)

	p0.V = &v
	b, _ = AppendCompact(nil, p0)
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
}

func TestCompactDecodeShortBuffer(t *testing.T) {
	type S struct {
		F1 string `frugal:"1,required,string"`
	}
	raw := []byte{0x18, 0x01, 0x41, 0x00}
	p := &S{}
	_, err := DecodeCompact(raw, p)
	assert.Nil(t, err)
	assert.Equal(t, "A", p.F1)

	_, err = DecodeCompact([]byte{0x18}, p)
	assert.True(t, err != nil)
}

func TestCompactAppendListFastPaths(t *testing.T) {
	type EnumType int64
	type Simple struct {
		L1 []int16    `frugal:"1,required,list<i16>"`
		L2 []int32    `frugal:"2,required,list<i32>"`
		L3 []int64    `frugal:"3,required,list<i64>"`
		L4 []string   `frugal:"4,required,list<string>"`
		L5 []int8     `frugal:"5,required,list<i8>"`
		L6 []float64   `frugal:"6,required,list<double>"`
		L7 []EnumType `frugal:"7,required,list<EnumType>"`
	}
	p0 := &Simple{
		L1: []int16{1, -1, 32767},
		L2: []int32{1, -1, 2147483647},
		L3: []int64{1, -1, 1<<62 - 1},
		L4: []string{"hello", "world"},
		L5: []int8{-128, 0, 127},
		L6: []float64{3.14, -2.718, 0.0},
		L7: []EnumType{10, -20, 300},
	}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &Simple{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)

	// Empty lists (covers n==0 branches)
	p0 = &Simple{
		L1: []int16{},
		L2: []int32{},
		L3: []int64{},
		L4: []string{},
		L5: []int8{},
		L6: []float64{},
		L7: []EnumType{},
	}
	b, err = AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 = &Simple{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}

func TestCompactAppendStruct(t *testing.T) {
	type Inner struct {
		A int32  `frugal:"1,required,i32"`
		B string `frugal:"2,default,string"`
	}
	type Outer struct {
		F1 bool   `frugal:"1,required"`
		F2 *Inner `frugal:"2,optional"`
		F3 int64  `frugal:"3,required,i64"`
	}
	p0 := &Outer{F1: true, F2: &Inner{A: 42, B: "hello"}, F3: -999}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &Outer{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.Equal(t, p0.F1, p1.F1)
	assert.Equal(t, p0.F3, p1.F3)
	assert.True(t, p1.F2 != nil)
	assert.Equal(t, p0.F2.A, p1.F2.A)
	assert.Equal(t, p0.F2.B, p1.F2.B)
}

func TestCompactAppendMapAnyAny(t *testing.T) {
	type WithMap struct {
		M map[string]string `frugal:"1,required"`
	}
	p0 := &WithMap{M: map[string]string{"a": "1", "b": "2"}}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &WithMap{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0.M, p1.M)
}

func TestCompactEncodeOnlyUnknownFields(t *testing.T) {
	type Msg struct {
		_unknownFields []byte
	}
	m := &Msg{_unknownFields: []byte("helloworld")}
	n := EncodedSizeCompact(m)
	b, err := AppendCompact(nil, m)
	assert.Nil(t, err)
	assert.Equal(t, n, len(b))
	assert.BytesEqual(t, append([]byte("helloworld"), byte(ctSTOP)), b)
}

func TestCompactEncodeOnlyUnknownFieldsNamedType(t *testing.T) {
	type UnknownFieldsType []byte
	type Msg struct {
		_unknownFields UnknownFieldsType
	}
	m := &Msg{_unknownFields: UnknownFieldsType("helloworld")}
	n := EncodedSizeCompact(m)
	b, err := AppendCompact(nil, m)
	assert.Nil(t, err)
	assert.Equal(t, n, len(b))
	assert.BytesEqual(t, append([]byte("helloworld"), byte(ctSTOP)), b)
}

func TestCompactAppendMapBoolScalarsRoundTrip(t *testing.T) {
	type TestStruct struct {
		M1 map[bool]bool   `frugal:"1,optional,map<bool:bool>"`
		M2 map[bool]string `frugal:"2,optional,map<bool:string>"`
		M3 map[string]bool `frugal:"3,optional,map<string:bool>"`
	}
	p0 := &TestStruct{
		M1: map[bool]bool{false: true, true: false},
		M2: map[bool]string{false: "off", true: "on"},
		M3: map[string]bool{"disabled": false, "enabled": true},
	}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &TestStruct{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}

func TestCompactAppendListAny(t *testing.T) {
	type WithList struct {
		L []map[int32]string `frugal:"1,required,list<map<i32:string>>"`
	}
	p0 := &WithList{
		L: []map[int32]string{
			{1: "a"},
			{2: "b", 3: "c"},
		},
	}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &WithList{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}

func TestCompactEncodeUnknownFields(t *testing.T) {
	type S1 struct {
		F1 int32  `frugal:"1,required,i32"`
		F2 string `frugal:"2,default,string"`
	}
	type S2 struct {
		F1 int32 `frugal:"1,required,i32"`
	}

	p1 := &S1{F1: 42, F2: "hello"}
	b, _ := AppendCompact(nil, p1)

	p2 := &S2{}
	n, err := DecodeCompact(b, p2)
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)
	assert.Equal(t, int32(42), p2.F1)
}

func TestCompactAppendListGenericBool(t *testing.T) {
	type S struct {
		L []bool `frugal:"1,required,list<bool>"`
	}
	p0 := &S{L: []bool{true, false, true, false, true}}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &S{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)

	// Empty list
	p0 = &S{L: []bool{}}
	b, err = AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 = &S{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}

func TestCompactAppendMapGenericBoolI32(t *testing.T) {
	type S struct {
		M map[bool]int32 `frugal:"1,required,map<bool:i32>"`
	}
	p0 := &S{M: map[bool]int32{true: 100, false: -50}}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &S{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)

	// Empty map
	p0 = &S{M: map[bool]int32{}}
	b, err = AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 = &S{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(p1.M))
}

func TestCompactListLargeHeader(t *testing.T) {
	// n > 14 hits the 0xF0 extended header branch in appendCompactListHeader
	type S struct {
		L []int32 `frugal:"1,required,list<i32>"`
	}
	elems := make([]int32, 20)
	for i := range elems {
		elems[i] = int32(i)
	}
	p0 := &S{L: elems}
	b, err := AppendCompact(nil, p0)
	assert.Nil(t, err)
	p1 := &S{}
	_, err = DecodeCompact(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}

func TestCompactNilPointer(t *testing.T) {
	type Empty struct{}
	var p *Empty = nil
	n := EncodedSizeCompact(p)
	assert.Equal(t, 1, n)
	b, err := AppendCompact(nil, p)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(b))
	assert.Equal(t, byte(ctSTOP), b[0])
}

func TestCompactDecodeErrors(t *testing.T) {
	// Not a pointer
	_, err := DecodeCompact([]byte{}, struct{}{})
	assert.True(t, err != nil)
	assert.True(t, strings.Contains(err.Error(), "not a pointer"))

	// Nil pointer
	_, err = DecodeCompact([]byte{}, (*struct{ X int32 })(nil))
	assert.True(t, err != nil)
	assert.True(t, strings.Contains(err.Error(), "nil pointer"))

	// Pointer to non-struct
	n := 42
	_, err = DecodeCompact([]byte{}, &n)
	assert.True(t, err != nil)
	assert.True(t, strings.Contains(err.Error(), "not a pointer to a struct"))
}

func TestCompactEncodedSizeValue(t *testing.T) {
	// Pass a struct value (not pointer) to hit the pool path in EncodedSizeCompact
	type S struct {
		F1 int32  `frugal:"1,required,i32"`
		F2 string `frugal:"2,default,string"`
	}
	v := S{F1: 42, F2: "hello"}
	n := EncodedSizeCompact(v)
	assert.True(t, n > 1)

	// Round-trip through AppendCompact (also with value, not pointer)
	b, err := AppendCompact(nil, v)
	assert.Nil(t, err)
	assert.Equal(t, n, len(b))

	p2 := &S{}
	_, err = DecodeCompact(b, p2)
	assert.Nil(t, err)
	assert.Equal(t, v.F1, p2.F1)
	assert.Equal(t, v.F2, p2.F2)
}
