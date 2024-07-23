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
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {

	rand.Seed(time.Now().Unix())

	type testcase struct {
		name   string
		update func(p *TestTypes)
		test   func(t *testing.T, p1 *TestTypes)
	}

	var (
		vInt16   = int16(rand.Uint32() & 0xffff)
		vInt32   = int32(rand.Uint32())
		vInt64   = int64(rand.Uint64())
		vFloat64 = math.Float64frombits(rand.Uint64())
	)

	for math.IsNaN(vFloat64) { // fix test failure
		vFloat64 = math.Float64frombits(rand.Uint64())
	}

	testcases := []testcase{
		{
			name:   "case_bool",
			update: func(p0 *TestTypes) { p0.FBool = true },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, true, p1.FBool) },
		},
		{
			name:   "case_string",
			update: func(p0 *TestTypes) { p0.String_ = "hello" },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, "hello", p1.String_) },
		},
		{
			name:   "case_byte",
			update: func(p0 *TestTypes) { p0.FByte = 0x55 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, int8(0x55), p1.FByte) },
		},
		{
			name:   "case_int8",
			update: func(p0 *TestTypes) { p0.I8 = 0x55 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, int8(0x55), p1.I8) },
		},
		{
			name:   "case_int16",
			update: func(p0 *TestTypes) { p0.I16 = vInt16 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, vInt16, p1.I16) },
		},
		{
			name:   "case_int32",
			update: func(p0 *TestTypes) { p0.I32 = vInt32 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, vInt32, p1.I32) },
		},
		{
			name:   "case_int64",
			update: func(p0 *TestTypes) { p0.I64 = -vInt64 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, -vInt64, p1.I64) },
		},
		{
			name:   "case_float64",
			update: func(p0 *TestTypes) { p0.Double = vFloat64 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, vFloat64, p1.Double) },
		},
		{
			name:   "case_string",
			update: func(p0 *TestTypes) { p0.String_ = "str" },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, "str", p1.String_) },
		},
		{
			name:   "case_enum",
			update: func(p0 *TestTypes) { p0.Enum = Numberz(vInt32) },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, Numberz(vInt32), p1.Enum) },
		},
		{
			name:   "case_typedef",
			update: func(p0 *TestTypes) { p0.UID = vInt64 },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, vInt64, p1.UID) },
		},
		{
			name:   "case_struct",
			update: func(p0 *TestTypes) { p0.S = &Msg{Type: vInt32} },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, &Msg{Type: vInt32}, p1.S) },
		},
		{
			name: "case_map",
			update: func(p0 *TestTypes) {
				p0.M0 = map[int32]int32{vInt32: vInt32 - 1, 1: 2}
				p0.M1 = map[int32]string{vInt32 - 2: "hello", 1: "2"}
				p0.M2 = map[int32]*Msg{vInt32 - 3: nil, 1: &Msg{Type: 2}}
				p0.M3 = map[string]*Msg{"hello": &Msg{Type: vInt32 - 4}}
			},
			test: func(t *testing.T, p1 *TestTypes) {
				assert.Equal(t, map[int32]int32{vInt32: vInt32 - 1, 1: 2}, p1.M0)
				assert.Equal(t, map[int32]string{vInt32 - 2: "hello", 1: "2"}, p1.M1)
				assert.Equal(t, map[int32]*Msg{vInt32 - 3: &Msg{}, 1: &Msg{Type: 2}}, p1.M2)
				assert.Equal(t, map[string]*Msg{"hello": &Msg{Type: vInt32 - 4}}, p1.M3)
			},
		},
		{
			name: "case_list",
			update: func(p0 *TestTypes) {
				p0.L0 = []int32{1, 2}
				p0.L1 = []string{"hello", "world"}
				p0.L2 = []*Msg{nil, {Type: vInt32}}
			},
			test: func(t *testing.T, p1 *TestTypes) {
				assert.Equal(t, []int32{1, 2}, p1.L0)
				assert.Equal(t, []string{"hello", "world"}, p1.L1)
				assert.Equal(t, []*Msg{{}, {Type: vInt32}}, p1.L2)
			},
		},
		{
			name: "case_set",
			update: func(p0 *TestTypes) {
				p0.S0 = []int32{1, 2}
				p0.S1 = []string{"hello", "world"}
			},
			test: func(t *testing.T, p1 *TestTypes) {
				assert.Equal(t, []int32{1, 2}, p1.S0)
				assert.Equal(t, []string{"hello", "world"}, p1.S1)
			},
		},
	}
	for _, tc := range testcases {
		name := tc.name
		updatef := tc.update
		testf := tc.test
		t.Run(name, func(t *testing.T) {
			p0 := NewTestTypes()
			updatef(p0) // update by testcase func

			b := make([]byte, EncodedSize(p0))
			n, err := Encode(b, p0)
			require.NoError(t, err)
			require.Equal(t, len(b), n)

			n, err = skipType(tSTRUCT, b, maxDepthLimit)
			require.NoError(t, err)
			require.Equal(t, len(b), n)

			p1 := &TestTypes{}
			n, err = Decode(b, p1)
			require.NoError(t, err)
			require.Equal(t, len(b), n)

			testf(t, p1) // test by testcase func
		})
	}
}

func TestDecodeRequired(t *testing.T) {
	type S0 struct {
		V *bool `frugal:"1,optional,bool"`
	}
	type S1 struct {
		V bool `frugal:"1,required,bool"`
	}
	v := true
	p0 := &S0{}
	b := make([]byte, 10)
	n, err := Encode(b, p0)
	require.NoError(t, err)
	b = b[:n]
	p1 := &S1{}
	_, err = Decode(b, p1)
	require.Equal(t, newRequiredFieldNotSetException("V"), err)

	p0.V = &v
	n, err = Encode(b[:10], p0)
	require.NoError(t, err)
	b = b[:n]

	n, err = Decode(b, p1)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, true, p1.V)
}

func TestDecodeUnknownFields(t *testing.T) {
	type Msg0 struct {
		I0 int32  `thrift:"i0,2" frugal:"2,default,i32"`
		S0 string `thrift:"s0,3" frugal:"3,default,string"`
		S1 string `thrift:"s1,4" frugal:"4,default,string"`
		I1 int32  `thrift:"i1,5" frugal:"5,default,i32"`
	}

	type Msg1 struct { // without S0, I1
		I0 int32  `thrift:"i0,2" frugal:"2,default,i32"`
		S1 string `thrift:"s1,4" frugal:"4,default,string"`

		_unknownFields []byte
	}

	msg := Msg0{I0: 1, S0: "s0", S1: "s1", I1: 2}
	b := make([]byte, EncodedSize(msg))
	_, _ = Encode(b, msg)

	p := &Msg1{}
	_, _ = Decode(b, p)

	assert.Equal(t, msg.I0, p.I0)
	assert.Equal(t, msg.S1, p.S1)

	sz := fieldHeaderLen + strHeaderLen + len(msg.S0) + fieldHeaderLen + 4
	testb := make([]byte, sz)
	testb = appendStringField(testb[:0], 3, msg.S0)
	testb = appendInt32Field(testb, 5, uint32(msg.I1))
	assert.Equal(t, sz, len(testb))
	assert.Equal(t, testb, p._unknownFields)
}
