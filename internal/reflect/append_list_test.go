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
	"unsafe"

	"github.com/cloudwego/frugal/internal/assert"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

func TestAppendListAny(t *testing.T) {
	typ := &tType{T: tLIST}
	typ.V = &tType{T: tI64, WT: tI64, Size: 8, SimpleType: true}
	v := []int64{1, 2}
	b, err := appendListAny(typ, nil, unsafe.Pointer(&v))
	assert.Nil(t, err)

	x := thrift.BinaryProtocol{}
	expectb := x.AppendListBegin(nil, thrift.I64, 2)
	expectb = x.AppendI64(expectb, 1)
	expectb = x.AppendI64(expectb, 2)
	assert.BytesEqual(t, expectb, b)

	// empty case
	v = nil
	b, err = appendListAny(typ, nil, unsafe.Pointer(&v))
	assert.Nil(t, err)
	expectb = x.AppendListBegin(nil, thrift.I64, 0)
	assert.BytesEqual(t, expectb, b)
}
