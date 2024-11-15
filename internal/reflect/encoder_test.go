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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewMsg().BLength()
const encodedMsgSize = 15

// NewTestTypes().BLength()
const encodedTestTypesSize = 176

// NewTestTypesOptional().BLength()
const encodedTestTypesOptionalSize = 1

// NewTestTypesWithDefault().BLength()
const encodedTestTypesWithDefault = 25

func TestEncode(t *testing.T) {
	// default NewXXXX cases
	require.Equal(t, encodedTestTypesSize, EncodedSize(NewTestTypes()))
	require.Equal(t, encodedTestTypesWithDefault, EncodedSize(NewTestTypesWithDefault()))
	require.Equal(t, encodedTestTypesOptionalSize, EncodedSize(NewTestTypesOptional()))

	type testcase struct {
		name   string
		update func(p *TestTypesOptional)
		expect int // Encode or EncodedSize
	}

	b := make([]byte, 1024)

	fhdr := fieldHeaderLen
	lhdr := listHeaderLen
	mhdr := mapHeaderLen
	shdr := strHeaderLen

	testcases := []testcase{
		{
			name:   "case_bool",
			update: func(p *TestTypesOptional) { v := true; p.FBool = &v },
			expect: encodedTestTypesOptionalSize + fhdr + 1,
		},
		{
			name:   "case_string",
			update: func(p *TestTypesOptional) { v := "str"; p.String_ = &v },
			expect: encodedTestTypesOptionalSize + fhdr + shdr + 3,
		},
		{
			name:   "case_map_with_primitive_types",
			update: func(p *TestTypesOptional) { p.M0 = map[int32]int32{1: 2, 3: 4} },
			expect: encodedTestTypesOptionalSize + fhdr + mhdr + 2*(4+4),
		},
		{
			name:   "case_map_with_i32_string",
			update: func(p *TestTypesOptional) { p.M1 = map[int32]string{1: "2", 2: "3"} },
			expect: encodedTestTypesOptionalSize + fhdr + mhdr + 2*(4+(shdr+1)),
		},
		{
			name:   "case_map_with_string_struct",
			update: func(p *TestTypesOptional) { p.M3 = map[string]*Msg{"1": nil, "2": {Type: 3}} }, // 36
			expect: encodedTestTypesOptionalSize + fhdr + mhdr + (shdr + 1 + 1) + (shdr + 1 + encodedMsgSize),
		},
		{
			name:   "case_map_with_i32_list",
			update: func(p *TestTypesOptional) { p.ML = map[int32][]int32{1: {1, 2}, 2: {3, 4}} },
			expect: encodedTestTypesOptionalSize + fhdr + mhdr + 2*(4+lhdr+2*4),
		},
		{
			name:   "case_list_with_i32",
			update: func(p *TestTypesOptional) { p.L0 = []int32{1, 2} },
			expect: encodedTestTypesOptionalSize + fhdr + lhdr + 2*4,
		},
		{
			name:   "case_list_with_string",
			update: func(p *TestTypesOptional) { p.L1 = []string{"1", "2"} },
			expect: encodedTestTypesOptionalSize + fhdr + lhdr + 2*(shdr+1),
		},
		{
			name:   "case_list_with_struct",
			update: func(p *TestTypesOptional) { p.L2 = []*Msg{{Type: 1}, {Type: 2}} },
			expect: encodedTestTypesOptionalSize + fhdr + lhdr + 2*encodedMsgSize,
		},
		{
			name:   "case_list_with_map",
			update: func(p *TestTypesOptional) { p.LM = []map[int32]int32{{1: 2}} },
			expect: encodedTestTypesOptionalSize + fhdr + lhdr + mhdr + 4 + 4,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewTestTypesOptional()
			tc.update(p)
			assert.Equal(t, tc.expect, EncodedSize(p))
			x, err := Append(b[:0], p)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expect, len(x))
			}
		})
	}
}

func TestEncodeStructOther(t *testing.T) {
	assert.Equal(t, encodedMsgSize, EncodedSize(Msg{})) // indirect type
	assert.Equal(t, 1, EncodedSize((*Msg)(nil)))        // nil
	b, err := Append(nil, Msg{})
	assert.NoError(t, err)
	assert.Equal(t, encodedMsgSize, len(b))
}

func TestEncodeUnknownFields(t *testing.T) {
	type Msg1 struct { // without S0, I1
		I0 int32  `thrift:"i0,2" frugal:"2,default,i32"`
		S1 string `thrift:"s1,4" frugal:"4,default,string"`

		_unknownFields []byte
	}

	m := &Msg1{I0: 123, S1: "Hello"}
	m._unknownFields = []byte("helloworld")

	n := EncodedSize(m)
	b, err := Append(nil, m)
	require.NoError(t, err)
	assert.Equal(t, n, len(b))
	assert.Contains(t, string(b), string(append([]byte("helloworld")[:], byte(tSTOP))))
}
