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
	"reflect"
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

func TestAppendMapDispatchUsesPredefinedFuncsForBoolScalars(t *testing.T) {
	type TestStruct struct {
		M1 map[bool]bool   `frugal:"1,optional,map<bool:bool>"`
		M2 map[bool]string `frugal:"2,optional,map<bool:string>"`
		M3 map[string]bool `frugal:"3,optional,map<string:bool>"`
	}

	desc, err := getOrcreateStructDesc(reflect.ValueOf(&TestStruct{}))
	assert.Nil(t, err)

	appendMapAnyAnyPtr := reflect.ValueOf(appendMapAnyAny).Pointer()
	assert.True(t, reflect.ValueOf(desc.GetField(1).Type.AppendFunc).Pointer() != appendMapAnyAnyPtr)
	assert.True(t, reflect.ValueOf(desc.GetField(2).Type.AppendFunc).Pointer() != appendMapAnyAnyPtr)
	assert.True(t, reflect.ValueOf(desc.GetField(3).Type.AppendFunc).Pointer() != appendMapAnyAnyPtr)
}

func TestAppendMapBoolScalarsRoundTrip(t *testing.T) {
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

	b, err := Append(nil, p0)
	assert.Nil(t, err)

	p1 := &TestStruct{}
	_, err = Decode(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}

func TestAppendMapAnyAny(t *testing.T) {
	var funcs map[struct{ k, v ttype }]appendFuncType
	funcs, mapAppendFuncs = mapAppendFuncs, nil // reset mapAppendFuncs to use appendMapAnyAny
	defer func() {
		mapAppendFuncs = funcs
	}()

	type EnumKey int64
	type EnumType int64
	type EmptyStruct struct{}

	type TestStruct struct {
		M1 map[EnumKey]int8         `frugal:"1,optional,map<EnumKey:i8>"`
		M2 map[EnumKey]int16        `frugal:"2,optional,map<EnumKey:i16>"`
		M3 map[EnumKey]int32        `frugal:"3,optional,map<EnumKey:i32>"`
		M4 map[EnumKey]int64        `frugal:"4,optional,map<EnumKey:i64>"`
		M5 map[EnumKey]EnumType     `frugal:"5,optional,map<EnumKey:EnumType>"`
		M6 map[EnumKey]string       `frugal:"6,optional,map<EnumKey:string>"`
		M7 map[EnumKey]*EmptyStruct `frugal:"7,optional,map<EnumKey:EmptyStruct>"`
	}

	p0 := &TestStruct{
		M1: map[EnumKey]int8{11: 1, 12: 2},
		M2: map[EnumKey]int16{21: 1, 22: 2},
		M3: map[EnumKey]int32{31: 1, 32: 2},
		M4: map[EnumKey]int64{41: 1, 42: 2},
		M5: map[EnumKey]EnumType{51: 1, 52: 2},
		M6: map[EnumKey]string{61: "1", 62: "2"},
		M7: map[EnumKey]*EmptyStruct{71: {}, 72: {}},
	}

	b, err := Append(nil, p0)
	assert.Nil(t, err)

	p1 := &TestStruct{}
	_, err = Decode(b, p1)
	assert.Nil(t, err)
	assert.DeepEqual(t, p0, p1)
}
