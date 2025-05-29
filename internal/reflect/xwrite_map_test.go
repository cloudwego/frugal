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

	"github.com/cloudwego/gopkg/protocol/thrift"
	"github.com/stretchr/testify/require"
)

func TestXWriteMapAnyAny(t *testing.T) {
	var funcs map[struct{ k, v ttype }]xwriteFuncType
	funcs, mapXWriteFuncs = mapXWriteFuncs, nil // reset mapXWriteFuncs to use xwriteMapAnyAny
	defer func() {
		mapXWriteFuncs = funcs
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

	b := thrift.NewXWriteBuffer()
	err := XWrite(b, p0)
	require.NoError(t, err)
	bufs := b.Bytes()
	_ = bufs
	b.Free()

	//p1 := &TestStruct{}
	//_, err = Decode(b, p1)
	//require.NoError(t, err)
	//require.Equal(t, p0, p1)
}
