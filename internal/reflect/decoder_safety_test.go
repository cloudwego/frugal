/*
 * Copyright 2026 CloudWeGo Authors
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
	"errors"
	"io"
	"math"
	"reflect"
	"testing"
	"unsafe"

	"github.com/cloudwego/frugal/internal/assert"
)

func TestDecodeBool(t *testing.T) {
	type msg struct {
		Value bool           `frugal:"1,default,bool"`
		Ptr   *bool          `frugal:"2,optional,bool"`
		List  []bool         `frugal:"3,default,list<bool>"`
		Map   map[bool]int32 `frugal:"4,default,map<bool:i32>"`
	}
	for _, raw := range []byte{0, 1, 2, 3, 255} {
		data := []byte{
			byte(tBOOL), 0, 1, raw,
			byte(tBOOL), 0, 2, raw,
			byte(tLIST), 0, 3, byte(tBOOL), 0, 0, 0, 1, raw,
			byte(tMAP), 0, 4, byte(tBOOL), byte(tI32), 0, 0, 0, 1, raw, 0, 0, 0, 7,
			byte(tSTOP),
		}
		var got msg
		_, err := Decode(data, &got)
		assert.Nil(t, err, raw)
		want := raw == 1
		assert.DeepEqual(t, msg{Value: want, Ptr: &want, List: []bool{want}, Map: map[bool]int32{want: 7}}, got, raw)
	}
}

func TestDecodeTruncatedFixedPointer(t *testing.T) {
	type enum int64
	cases := []any{
		struct {
			V *bool `frugal:"1,optional,bool"`
		}{new(bool)},
		struct {
			V *int8 `frugal:"1,optional,i8"`
		}{new(int8)},
		struct {
			V *int16 `frugal:"1,optional,i16"`
		}{new(int16)},
		struct {
			V *int32 `frugal:"1,optional,i32"`
		}{new(int32)},
		struct {
			V *int64 `frugal:"1,optional,i64"`
		}{new(int64)},
		struct {
			V *float64 `frugal:"1,optional,double"`
		}{new(float64)},
		struct {
			V *enum `frugal:"1,optional,enum"`
		}{new(enum)},
	}
	for _, value := range cases {
		data, err := Append(nil, value)
		assert.Nil(t, err)
		for size := 0; size < len(data)-1; size++ {
			got := reflect.New(reflect.TypeOf(value))
			n, err := Decode(data[:size], got.Interface())
			assert.True(t, errors.Is(err, io.ErrShortBuffer) && n <= size, value, size, n, err)
			// pointer must stay nil on error because d.s does not zero memory
			assert.True(t, got.Elem().Field(0).IsNil(), value, size)
		}
	}
}

func TestDecodeTruncatedContainers(t *testing.T) {
	type nested struct {
		S string `frugal:"1,default,string"`
	}
	type msg struct {
		A []int64             `frugal:"1,default,list<i64>"`
		B map[string]int64    `frugal:"2,default,map<string:i64>"`
		C map[int64]string    `frugal:"3,default,map<i64:string>"`
		D *nested             `frugal:"4,default,nested"`
		E []*nested           `frugal:"5,default,list<nested>"`
		F map[string][]string `frugal:"6,default,map<string:list<string>>"`
		G map[int64]int64     `frugal:"7,default,map<i64:i64>"`
	}
	data, err := Append(nil, &msg{
		A: []int64{1, 2}, B: map[string]int64{"a long key": 3},
		C: map[int64]string{4: "a long value", 5: "another value"},
		D: &nested{S: "nested"}, E: []*nested{{S: "list"}},
		F: map[string][]string{"key": {"one", "two"}},
		G: map[int64]int64{6: 7},
	})
	assert.Nil(t, err)
	// every truncated prefix must fail when decoded or skipped
	for size := range data {
		for _, v := range []any{&msg{}, &struct{ _unknownFields []byte }{}} {
			n, err := Decode(data[:size], v)
			assert.True(t, err != nil && n <= size, size, n, err)
		}
	}
	var got struct{ _unknownFields []byte }
	n, err := Decode(data, &got)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)
	assert.BytesEqual(t, data[:len(data)-1], got._unknownFields)
}

func TestDecodeSizeOverflow(t *testing.T) {
	large := math.MaxInt/2 + 1
	for _, tc := range []struct {
		name string
		typ  *tType
		data []byte
	}{
		{"slice", &tType{T: tLIST, V: &tType{WT: tBYTE, Size: large}}, []byte{byte(tBYTE), 0, 0, 0, 2, 0, 0}},
		{"slice_pointers", &tType{T: tLIST, V: &tType{WT: tSTRUCT, Size: 8, IsPointer: true, V: &tType{Size: large}}}, []byte{byte(tSTRUCT), 0, 0, 0, 2, 0, 0}},
		{"map_keys", &tType{T: tMAP, K: &tType{WT: tSTRUCT, IsPointer: true, V: &tType{Size: large}}, V: &tType{WT: tBYTE}}, []byte{byte(tSTRUCT), byte(tBYTE), 0, 0, 0, 2, 0, 0, 0, 0}},
		{"map_values", &tType{T: tMAP, K: &tType{WT: tBYTE}, V: &tType{WT: tSTRUCT, IsPointer: true, V: &tType{Size: large}}}, []byte{byte(tBYTE), byte(tSTRUCT), 0, 0, 0, 2, 0, 0, 0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output sliceHeader
			var decoder tDecoder
			_, err := decoder.decodeType(tc.typ, tc.data, unsafe.Pointer(&output), maxDepthLimit)
			assertSizeLimit(t, err)
			assert.Equal(t, sliceHeader{}, output)
		})
	}
}

func TestDecodeMapStructValueReset(t *testing.T) {
	type Leaf struct {
		S string `frugal:"1,optional,string"`
		N *int32 `frugal:"2,optional,i32"`
	}
	type S struct {
		M map[string]Leaf `frugal:"1,default,map<string:Leaf>"`
	}
	n := int32(7)
	// entry "b" is empty and must not keep fields from entry "a" or pooled tmp vars
	checkEncodingRoundTrip(t, &S{M: map[string]Leaf{"a": {S: "secret", N: &n}}}, &S{})
	data := []byte{
		byte(tMAP), 0, 1, byte(tSTRING), byte(tSTRUCT), 0, 0, 0, 2,
		0, 0, 0, 1, 'a', byte(tSTRING), 0, 1, 0, 0, 0, 6, 's', 'e', 'c', 'r', 'e', 't', byte(tSTOP),
		0, 0, 0, 1, 'b', byte(tSTOP),
		byte(tSTOP),
	}
	var got S
	_, err := Decode(data, &got)
	assert.Nil(t, err)
	assert.DeepEqual(t, map[string]Leaf{"a": {S: "secret"}, "b": {}}, got.M)
	checkEncodingRoundTrip(t, &S{M: map[string]Leaf{"c": {}}}, &S{})
}
