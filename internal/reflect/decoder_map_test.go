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
	"reflect"
	"testing"
	"unsafe"

	"github.com/cloudwego/gopkg/protocol/thrift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeMapAny(t *testing.T) {
	type Msg struct {
		S string `frugal:"1,optional,string"`
	}
	type TestStruct struct {
		M1 map[int64]int64 `frugal:"1,optional,map<i64:i64>"`
		M2 map[*Msg]*Msg   `frugal:"2,optional,map<Msg:Msg>"`
	}
	p := &TestStruct{}
	desc, err := getOrcreateStructDesc(reflect.ValueOf(p))
	assert.NoError(t, err)

	b := make([]byte, 0, 1024)
	x := thrift.BinaryProtocol{}

	d := decoderPool.Get().(*tDecoder)
	defer decoderPool.Put(d)

	// M1
	b = x.AppendMapBegin(b, thrift.I64, thrift.I64, 1)
	b = x.AppendI64(b, 1)
	b = x.AppendI64(b, -1)
	n, err := decodeMapAny(d, desc.GetField(1).Type, b, unsafe.Pointer(&p.M1), 1)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, map[int64]int64{1: -1}, p.M1)

	// M2
	b = b[:0]
	b = x.AppendMapBegin(b, thrift.STRUCT, thrift.STRUCT, 2)
	b = x.AppendFieldBegin(b, thrift.STRING, 1)
	b = x.AppendString(b, "key1")
	b = x.AppendFieldStop(b)
	b = x.AppendFieldBegin(b, thrift.STRING, 1)
	b = x.AppendString(b, "val1")
	b = x.AppendFieldStop(b)
	b = x.AppendFieldBegin(b, thrift.STRING, 1)
	b = x.AppendString(b, "key2")
	b = x.AppendFieldStop(b)
	b = x.AppendFieldBegin(b, thrift.STRING, 1)
	b = x.AppendString(b, "val2")
	b = x.AppendFieldStop(b)
	n, err = decodeMapAny(d, desc.GetField(2).Type, b, unsafe.Pointer(&p.M2), 2)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, 2, len(p.M2))
	for k, v := range p.M2 {
		switch k.S {
		case "key1":
			require.Equal(t, "val1", v.S)
		case "key2":
			require.Equal(t, "val2", v.S)
		default:
			t.Fatal("BUG")
		}
	}

	// type mismatch
	b = x.AppendMapBegin(b[:0], thrift.I64, thrift.I32, 1)
	_, err = decodeMapAny(d, desc.GetField(1).Type, b, unsafe.Pointer(&p.M1), 1)
	require.ErrorContains(t, err, "type mismatch")
}
