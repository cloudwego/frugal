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
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

type fuzzStruct struct {
	B              bool                `frugal:"1,default,bool"`
	I              *int64              `frugal:"2,optional,i64"`
	S              string              `frugal:"3,default,string,nocopy"`
	M              map[bool]int32      `frugal:"4,default,map<bool:i32>"`
	L              []int64             `frugal:"5,default,list<i64>"`
	P              []*fuzzStruct       `frugal:"6,default,list<fuzzStruct>"`
	V              map[string][]byte   `frugal:"7,default,map<string:binary>"`
	C              map[string]fuzzLeaf `frugal:"8,default,map<string:fuzzLeaf>"`
	_unknownFields []byte
}

type fuzzLeaf struct {
	S string `frugal:"1,default,string"`
}

func FuzzDecode(f *testing.F) {
	for _, b := range [][]byte{
		{}, {0}, {byte(tI64), 0, 2},
		{byte(tBOOL), 0, 1, 2, 0},
		{byte(tMAP), 0, 4, byte(tBOOL), byte(tI32), 0, 0, 0, 1, 2, 0, 0, 0, 7, 0},
		{byte(tLIST), 0, 5, byte(tI64), 0x7f, 0xff, 0xff, 0xff, 0},
	} {
		f.Add(b)
	}

	b, err := Append(nil, &fuzzStruct{
		B: true, S: "hello", M: map[bool]int32{false: 7, true: 8},
		L: []int64{1, 2}, P: []*fuzzStruct{{S: "nested"}},
		V: map[string][]byte{"a": {1, 2}, "b": {3}},
		C: map[string]fuzzLeaf{"leaf": {S: "value"}},
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Fuzz(func(t *testing.T, b []byte) {
		var v fuzzStruct
		n, err := Decode(b, &v)
		if err != nil {
			return
		}
		if n < 0 || n > len(b) {
			t.Fatalf("Decode consumed %d of %d bytes", n, len(b))
		}
		encoded, err := Append(nil, &v)
		if err != nil {
			t.Fatal(err)
		}
		if size := EncodedSize(&v); size != len(encoded) {
			t.Fatalf("EncodedSize = %d, encoded length = %d", size, len(encoded))
		}
		var roundTrip fuzzStruct
		if _, err := Decode(encoded, &roundTrip); err != nil {
			t.Fatal(err)
		}
		assert.BytesEqual(t, v._unknownFields, roundTrip._unknownFields)
	})
}
