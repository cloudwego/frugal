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

func TestDecodeListAny(t *testing.T) {
	type Msg struct {
		S string `frugal:"1,optional,string"`
	}
	type TestStruct struct {
		L1 []int64 `frugal:"1,optional,list<i64>"`
		L2 []*Msg  `frugal:"2,optional,list<Msg>"`
	}

	p := &TestStruct{}
	desc, err := getOrcreateStructDesc(reflect.ValueOf(p))
	assert.NoError(t, err)

	b := make([]byte, 0, 1024)
	x := thrift.BinaryProtocol{}

	d := decoderPool.Get().(*tDecoder)
	defer decoderPool.Put(d)

	// L1
	b = x.AppendListBegin(b, thrift.I64, 3)
	b = x.AppendI64(b, -1)
	b = x.AppendI64(b, 0)
	b = x.AppendI64(b, 1)
	n, err := decodeListAny(d, desc.GetField(1).Type, b, unsafe.Pointer(&p.L1), 1)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, []int64{-1, 0, 1}, p.L1)

	// L2
	b = b[:0]
	b = x.AppendListBegin(b, thrift.STRUCT, 2)
	b = x.AppendFieldBegin(b, thrift.STRING, 1)
	b = x.AppendString(b, "v1")
	b = x.AppendFieldStop(b)
	b = x.AppendFieldBegin(b, thrift.STRING, 1)
	b = x.AppendString(b, "v2")
	b = x.AppendFieldStop(b)
	n, err = decodeListAny(d, desc.GetField(2).Type, b, unsafe.Pointer(&p.L2), 2)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, []*Msg{{S: "v1"}, {S: "v2"}}, p.L2)

	// type mismatch
	b = x.AppendListBegin(b, thrift.I32, 0)
	_, err = decodeListAny(d, desc.GetField(1).Type, b, unsafe.Pointer(&p.L1), 1)
	require.ErrorContains(t, err, "type mismatch")

}
