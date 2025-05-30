/*
 * Copyright 2025 CloudWeGo Authors
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

	"github.com/cloudwego/gopkg/gridbuf"
	"github.com/stretchr/testify/require"
)

func TestGridWriteMapAnyAny(t *testing.T) {
	var funcs map[struct{ k, v ttype }]gridWriteFuncType
	funcs, mapGridWriteFuncs = mapGridWriteFuncs, nil // reset mapGridWriteFuncs to use gridWriteMapAnyAny
	defer func() {
		mapGridWriteFuncs = funcs
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
		M1: map[EnumKey]int8{11: 1},
		M2: map[EnumKey]int16{21: 1},
		M3: map[EnumKey]int32{31: 1},
		M4: map[EnumKey]int64{41: 1},
		M5: map[EnumKey]EnumType{51: 1},
		M6: map[EnumKey]string{61: "1"},
		M7: map[EnumKey]*EmptyStruct{71: {}},
	}

	buf, err := Append(nil, p0)
	require.NoError(t, err)
	_ = buf

	gwritebuf := gridbuf.NewWriteBuffer()
	err = GridWrite(gwritebuf, p0)
	require.NoError(t, err)
	bufs := gwritebuf.Bytes()

	require.Equal(t, string(buf), string(bufs[0]))

	greadbuf := gridbuf.NewReadBuffer(bufs)

	p1 := &TestStruct{}
	err = GridRead(greadbuf, p1)
	require.NoError(t, err)
	require.Equal(t, p0, p1)

	greadbuf.Free()
	gwritebuf.Free()
}
