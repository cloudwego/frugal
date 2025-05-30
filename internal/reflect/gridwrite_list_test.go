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
	"unsafe"

	"github.com/cloudwego/gopkg/gridbuf"
	"github.com/cloudwego/gopkg/protocol/thrift"
	"github.com/stretchr/testify/require"
)

func TestGridWriteListAny(t *testing.T) {
	typ := &tType{T: tLIST}
	typ.V = &tType{T: tI64, WT: tI64, Size: 8, SimpleType: true}
	v := []int64{1, 2}

	b := gridbuf.NewWriteBuffer()
	err := gridWriteListAny(typ, b, unsafe.Pointer(&v))
	require.NoError(t, err)

	x := thrift.BinaryProtocol{}
	expectb := x.AppendListBegin(nil, thrift.I64, 2)
	expectb = x.AppendI64(expectb, 1)
	expectb = x.AppendI64(expectb, 2)
	require.Equal(t, expectb, b.Bytes()[0])

	// empty case
	v = nil
	b = gridbuf.NewWriteBuffer()
	err = gridWriteListAny(typ, b, unsafe.Pointer(&v))
	require.NoError(t, err)
	expectb = x.AppendListBegin(nil, thrift.I64, 0)
	require.Equal(t, expectb, b.Bytes()[0])
}
