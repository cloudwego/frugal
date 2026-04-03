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

	"github.com/cloudwego/frugal/internal/assert"
)

func TestAppendListFastPaths(t *testing.T) {

	type EnumType int64

	type Msg struct {
		X int64 `frugal:"1,default,i64"`
		Y int64 `frugal:"2,default,i64"`
	}

	type TestStruct struct {
		L1 []int8     `frugal:"1,optional,list<i8>"`
		L2 []int16    `frugal:"2,optional,list<i16>"`
		L3 []int32    `frugal:"3,optional,list<i32>"`
		L4 []int64    `frugal:"4,optional,list<i64>"`
		L5 []EnumType `frugal:"5,optional,list<EnumType>"`
		L6 []string   `frugal:"6,optional,list<string>"`
		L7 []*Msg     `frugal:"7,optional,list<Msg>"`
	}

	var p0, p1 *TestStruct
	var b []byte
	var err error

	p0 = &TestStruct{
		L1: []int8{11, 12},
		L2: []int16{21, 22},
		L3: []int32{31, 32},
		L4: []int64{41, 42},
		L5: []EnumType{51, 52},
		L6: []string{"61", "62"},
		L7: []*Msg{{X: 71, Y: 72}, {X: 73, Y: 74}},
	}
	b, err = Append(nil, p0)
	assert.Nil(t, err)

	p1 = &TestStruct{}
	_, err = Decode(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)

	// Empty list
	p0 = &TestStruct{
		L1: []int8{},
		L2: []int16{},
		L3: []int32{},
		L4: []int64{},
		L5: []EnumType{},
		L6: []string{},
		L7: []*Msg{},
	}
	b, err = Append(nil, p0)
	assert.Nil(t, err)

	p1 = &TestStruct{}
	_, err = Decode(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}
