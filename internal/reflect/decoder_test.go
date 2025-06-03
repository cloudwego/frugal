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
	"bytes"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/cloudwego/gopkg/gridbuf"
	"github.com/cloudwego/gopkg/protocol/thrift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestDecode(t *testing.T) {
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
			name:   "case_default",
			update: func(p0 *TestTypes) {},
			test:   func(t *testing.T, p1 *TestTypes) {},
		},
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
			name:   "case_string_binary",
			update: func(p0 *TestTypes) { p0.String_ = "str"; p0.Binary = []byte{1} },
			test: func(t *testing.T, p1 *TestTypes) {
				assert.Equal(t, "str", p1.String_)
				assert.Equal(t, []byte{1}, p1.Binary)
			},
		},
		{
			name:   "case_zero_len_binary",
			update: func(p0 *TestTypes) { p0.Binary = []byte{} },
			test:   func(t *testing.T, p1 *TestTypes) { assert.Equal(t, []byte{}, p1.Binary) },
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
				p0.M2 = map[int32]*Msg{vInt32 - 3: nil, 1: {Type: 2}}
				p0.M3 = map[string]*Msg{"hello": {Type: vInt32 - 4}}
			},
			test: func(t *testing.T, p1 *TestTypes) {
				assert.Equal(t, map[int32]int32{vInt32: vInt32 - 1, 1: 2}, p1.M0)
				assert.Equal(t, map[int32]string{vInt32 - 2: "hello", 1: "2"}, p1.M1)
				assert.Equal(t, map[int32]*Msg{vInt32 - 3: {}, 1: {Type: 2}}, p1.M2)
				assert.Equal(t, map[string]*Msg{"hello": {Type: vInt32 - 4}}, p1.M3)
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
			name: "case_zero_len_list",
			update: func(p0 *TestTypes) {
				p0.L0 = []int32{}
				p0.L1 = []string{}
				p0.L2 = []*Msg{}
			},
			test: func(t *testing.T, p1 *TestTypes) {
				assert.Equal(t, []int32{}, p1.L0)
				assert.Equal(t, []string{}, p1.L1)
				assert.Equal(t, []*Msg{}, p1.L2)
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
		t.Run(name+"_append_decode", func(t *testing.T) {
			p0 := NewTestTypes()
			updatef(p0) // update by testcase func

			n := EncodedSize(p0)
			b, err := Append(nil, p0)
			require.NoError(t, err)
			require.Equal(t, n, len(b))

			// verify by gopkg thrift
			n, err = thrift.Binary.Skip(b, thrift.TType(tSTRUCT))
			require.NoError(t, err)
			require.Equal(t, n, len(b))

			p1 := &TestTypes{}
			n, err = Decode(b, p1)
			require.NoError(t, err)
			require.Equal(t, len(b), n)

			testf(t, p1) // test by testcase func
		})
		t.Run(name+"_grid_write_read", func(t *testing.T) {
			p0 := NewTestTypes()
			updatef(p0) // update by testcase func

			b, err := Append(nil, p0)
			require.NoError(t, err)

			wb := gridbuf.NewWriteBuffer()
			err = GridWrite(wb, p0)
			require.NoError(t, err)
			bs := wb.Bytes()

			rb := gridbuf.NewReadBuffer(bs)
			ufs, err := thrift.GridBuffer.Skip(rb, thrift.TType(tSTRUCT), nil, true)
			require.NoError(t, err)
			require.Equal(t, len(ufs), len(b))

			p1 := &TestTypes{}
			rb = gridbuf.NewReadBuffer(bs)
			err = GridRead(rb, p1)
			require.NoError(t, err)
			testf(t, p1) // test by testcase func
		})
	}
}

func TestDecodeOptional(t *testing.T) {
	type testcase struct {
		name   string
		update func(p *TestTypesOptional)
		test   func(t *testing.T, p1 *TestTypesOptional)
	}

	var (
		vInt16   = int16(rand.Uint32() & 0xffff)
		vInt32   = int32(rand.Uint32())
		vInt64   = int64(rand.Uint64())
		vFloat64 = math.Float64frombits(rand.Uint64())
		vTrue    = true
		vString  = "hello"
		vByte    = int8(0x55)
		vEnum    = Numberz(int32(rand.Uint32()))
	)

	for math.IsNaN(vFloat64) { // fix test failure
		vFloat64 = math.Float64frombits(rand.Uint64())
	}

	testcases := []testcase{
		{
			name:   "case_bool",
			update: func(p0 *TestTypesOptional) { p0.FBool = &vTrue },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vTrue, *p1.FBool) },
		},
		{
			name:   "case_string",
			update: func(p0 *TestTypesOptional) { p0.String_ = &vString },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vString, *p1.String_) },
		},
		{
			name:   "case_byte",
			update: func(p0 *TestTypesOptional) { p0.FByte = &vByte },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vByte, *p1.FByte) },
		},
		{
			name:   "case_int8",
			update: func(p0 *TestTypesOptional) { p0.I8 = &vByte },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vByte, *p1.I8) },
		},
		{
			name:   "case_int16",
			update: func(p0 *TestTypesOptional) { p0.I16 = &vInt16 },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vInt16, *p1.I16) },
		},
		{
			name:   "case_int32",
			update: func(p0 *TestTypesOptional) { p0.I32 = &vInt32 },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vInt32, *p1.I32) },
		},
		{
			name:   "case_int64",
			update: func(p0 *TestTypesOptional) { p0.I64 = &vInt64 },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vInt64, *p1.I64) },
		},
		{
			name:   "case_float64",
			update: func(p0 *TestTypesOptional) { p0.Double = &vFloat64 },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vFloat64, *p1.Double) },
		},
		{
			name:   "case_enum",
			update: func(p0 *TestTypesOptional) { p0.Enum = &vEnum },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vEnum, *p1.Enum) },
		},
		{
			name:   "case_typedef",
			update: func(p0 *TestTypesOptional) { p0.UID = &vInt64 },
			test:   func(t *testing.T, p1 *TestTypesOptional) { assert.Equal(t, vInt64, *p1.UID) },
		},
	}
	for _, tc := range testcases {
		name := tc.name
		updatef := tc.update
		testf := tc.test
		t.Run(name+"_append_decode", func(t *testing.T) {
			p0 := NewTestTypesOptional()
			updatef(p0) // update by testcase func

			n := EncodedSize(p0)
			b, err := Append(nil, p0)
			require.NoError(t, err)
			require.Equal(t, n, len(b))

			// verify by gopkg thrift
			n, err = thrift.Binary.Skip(b, thrift.TType(tSTRUCT))
			require.NoError(t, err)
			require.Equal(t, n, len(b))

			p1 := &TestTypesOptional{}
			n, err = Decode(b, p1)
			require.NoError(t, err)
			require.Equal(t, len(b), n)

			testf(t, p1) // test by testcase func
		})
		t.Run(name+"_grid_write_read", func(t *testing.T) {
			p0 := NewTestTypesOptional()
			updatef(p0) // update by testcase func

			b, err := Append(nil, p0)
			require.NoError(t, err)

			wb := gridbuf.NewWriteBuffer()
			err = GridWrite(wb, p0)
			require.NoError(t, err)
			bs := wb.Bytes()

			rb := gridbuf.NewReadBuffer(bs)
			ufs, err := thrift.GridBuffer.Skip(rb, thrift.TType(tSTRUCT), nil, true)
			require.NoError(t, err)
			require.Equal(t, len(ufs), len(b))

			p1 := &TestTypesOptional{}
			rb = gridbuf.NewReadBuffer(bs)
			err = GridRead(rb, p1)
			require.NoError(t, err)
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
	b, err := Append(nil, p0)
	require.NoError(t, err)
	p1 := &S1{}
	_, err = Decode(b, p1)
	require.Equal(t, newRequiredFieldNotSetException("V"), err)

	p0.V = &v
	b, err = Append(nil, p0)
	require.NoError(t, err)

	n, err := Decode(b, p1)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, true, p1.V)
}

func TestGridReadWriteRequired(t *testing.T) {
	type S0 struct {
		V *bool `frugal:"1,optional,bool"`
	}
	type S1 struct {
		V bool `frugal:"1,required,bool"`
	}
	v := true
	p0 := &S0{}
	wb := gridbuf.NewWriteBuffer()
	err := GridWrite(wb, p0)
	require.NoError(t, err)
	p1 := &S1{}
	rb := gridbuf.NewReadBuffer(wb.Bytes())
	err = GridRead(rb, p1)
	require.Equal(t, newRequiredFieldNotSetException("V"), err)

	p0.V = &v
	wb = gridbuf.NewWriteBuffer()
	err = GridWrite(wb, p0)
	require.NoError(t, err)

	rb = gridbuf.NewReadBuffer(wb.Bytes())
	err = GridRead(rb, p1)
	require.NoError(t, err)
	require.Equal(t, true, p1.V)
}

func TestDecodeUnknownFields(t *testing.T) {
	type Msg0 struct {
		I0 int32  `thrift:"i0,2" frugal:"2,default,i32"`
		S0 string `thrift:"s0,3" frugal:"3,default,string"`
	}

	type Msg1 struct { // without S0
		I0 int32 `thrift:"i0,2" frugal:"2,default,i32"`

		_unknownFields []byte
	}

	msg := Msg0{I0: 1, S0: "s0"}
	b := make([]byte, EncodedSize(msg))
	_, _ = Append(b[:0], msg)

	p := &Msg1{}
	_, _ = Decode(b, p)

	assert.Equal(t, msg.I0, p.I0)

	sz := fieldHeaderLen + strHeaderLen + len(msg.S0)
	testb := make([]byte, sz)
	testb = appendStringField(testb[:0], 3, msg.S0)
	assert.Equal(t, sz, len(testb))
	assert.Equal(t, testb, p._unknownFields)
}

func TestGridReadWriteUnknownFields(t *testing.T) {
	type Msg0 struct {
		I0 int32  `thrift:"i0,2" frugal:"2,default,i32"`
		S0 string `thrift:"s0,3" frugal:"3,default,string"`
	}

	type Msg1 struct { // without S0
		I0 int32 `thrift:"i0,2" frugal:"2,default,i32"`

		_unknownFields []byte
	}

	msg := Msg0{I0: 1, S0: "s0"}
	wb := gridbuf.NewWriteBuffer()
	_ = GridWrite(wb, msg)
	bs := wb.Bytes()

	p := &Msg1{}
	rb := gridbuf.NewReadBuffer(bs)
	_ = GridRead(rb, p)

	assert.Equal(t, msg.I0, p.I0)

	sz := fieldHeaderLen + strHeaderLen + len(msg.S0)
	testb := make([]byte, sz)
	testb = appendStringField(testb[:0], 3, msg.S0)
	assert.Equal(t, sz, len(testb))
	assert.Equal(t, testb, p._unknownFields)
}

func TestDecodeNoCopy(t *testing.T) {
	type Msg struct {
		A string `frugal:"1,default,string,nocopy"`
		B string `frugal:"2,default,string"`
		C []byte `frugal:"3,default,binary,nocopy"`
	}

	strA := "strA"
	strB := "strB"
	strC := "strC"
	sz := 3*(fieldHeaderLen+strHeaderLen) + len(strA) + len(strB) + len(strC)
	b := make([]byte, 0, sz)
	b = appendStringField(b, 1, strA)
	b = appendStringField(b, 2, strB)
	b = appendStringField(b, 3, strC)
	b = append(b, 0) // tSTOP

	p := &Msg{}
	_, err := Decode(b, p)
	require.NoError(t, err)

	assert.Equal(t, strA, p.A)
	assert.Equal(t, strB, p.B)
	assert.Equal(t, strC, string(p.C))

	// update original buffer
	// coz it's nocopy, the fields of p will be changed implicitly as well.
	// update strA -> xtrA, strB -> xtrB, strC -> xtrC
	for _, s := range []string{strA, strB, strC} {
		i := bytes.Index(b, []byte(s))
		b[i] = 'x'
	}
	assert.Equal(t, "xtrA", p.A)
	assert.Equal(t, "strB", p.B) // p.B has no `nocopy` option
	assert.Equal(t, "xtrC", string(p.C))

	type Msg2 struct {
		A int32 `frugal:"4,default,i32,nocopy"`
	}
	p2 := &Msg2{}
	_, err = Decode(b, p2)
	require.Error(t, err)
	_ = p2
}
