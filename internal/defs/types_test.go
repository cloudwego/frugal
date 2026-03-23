/*
 * Copyright 2022 CloudWeGo Authors
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

package defs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

func TestTypes_Parsing(t *testing.T) {
	var v map[string][]reflect.SliceHeader
	tt, err := ParseType(reflect.TypeOf(v), "map<string:set<foo.SliceHeader>>")
	assert.Nil(t, err)
	fmt.Println(tt)
}

func TestTypes_MapKeyType(t *testing.T) {
	var v map[*reflect.SliceHeader]int
	tt, err := ParseType(reflect.TypeOf(v), "map<foo.SliceHeader:i64>")
	assert.Nil(t, err)
	fmt.Println(tt)
}

func TestTypes_Enum(t *testing.T) {
	type EnumType int64
	type Int32 int32
	type StructWithEnum struct {
		A EnumType  `frugal:"1,optional,EnumType"`
		B *EnumType `frugal:"2,optional,EnumType"`
		C Int32     `frugal:"3,optional,Int32"`
		D int64     `frugal:"4,optional,i64"`
	}
	ff, err := DoResolveFields(reflect.TypeOf(StructWithEnum{}))
	assert.Nil(t, err)
	assert.Equal(t, 4, len(ff))
	assert.True(t, ff[0].Type.IsEnum())
	assert.Equal(t, T_enum, ff[0].Type.T)
	assert.True(t, ff[1].Type.IsEnum())
	assert.Equal(t, T_pointer, ff[1].Type.T)
	assert.Equal(t, T_enum, ff[1].Type.V.T)
	assert.True(t, !ff[2].Type.IsEnum())
	assert.True(t, !ff[3].Type.IsEnum())

}
