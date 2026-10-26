/*
 * Copyright 2026 CloudWeGo Authors
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
	"errors"
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

func TestDecodeUnknownWireTypes(t *testing.T) {
	for _, invalid := range []byte{0, 1, 5, 9, 16, 17, 128, 221, 254, 255} {
		cases := [][]byte{
			{byte(tLIST), 0, 1, invalid, 0, 0, 0, 1, 0, 0},
			{byte(tMAP), 0, 1, invalid, byte(tBYTE), 0, 0, 0, 1, 0, 0, 0},
			{byte(tMAP), 0, 1, byte(tBYTE), invalid, 0, 0, 0, 1, 0, 0, 0},
		}
		if invalid != byte(tSTOP) {
			cases = append(cases,
				[]byte{invalid, 0, 1, 0},
				[]byte{byte(tSTRUCT), 0, 1, invalid, 0, 1, 0, 0},
			)
		}
		for _, data := range cases {
			var got struct{ _unknownFields []byte }
			_, err := Decode(data, &got)
			assert.True(t, err != nil, "accepted invalid wire type", invalid, data)
		}
	}
}

func TestDecodeUnknownEmptyContainers(t *testing.T) {
	// match thrift.Binary.Skip: ignore element types of empty containers
	for _, elem := range []byte{0, 1, 5, byte(tI32), 254, 255} {
		for _, data := range [][]byte{
			{byte(tLIST), 0, 1, elem, 0, 0, 0, 0, 0},
			{byte(tSET), 0, 1, elem, 0, 0, 0, 0, 0},
			{byte(tMAP), 0, 1, elem, elem, 0, 0, 0, 0, 0},
		} {
			var got struct{ _unknownFields []byte }
			n, err := Decode(data, &got)
			assert.Nil(t, err, data)
			assert.Equal(t, len(data), n, data)
			assert.BytesEqual(t, data[:len(data)-1], got._unknownFields, data)
		}
	}
}

func TestSkipValueSizeAndDepth(t *testing.T) {
	for _, tc := range []struct {
		typ    ttype
		header []byte
	}{
		{tSTRING, nil},
		{tLIST, []byte{byte(tI64)}},
		{tSET, []byte{byte(tI64)}},
		{tMAP, []byte{byte(tI64), byte(tI64)}},
	} {
		for _, length := range [][]byte{{255, 255, 255, 255}, {127, 255, 255, 255}} {
			data := append(append([]byte(nil), tc.header...), length...)
			_, err := skipValue(data, tc.typ, maxSkipDepth)
			assert.True(t, err != nil, "accepted invalid length", length, tc.typ)
		}
	}
	for _, depth := range []int{maxSkipDepth, maxSkipDepth + 1} {
		var data []byte
		for i := 1; i < depth; i++ {
			data = append(data, byte(tSTRUCT), 0, 1)
		}
		data = append(data, make([]byte, depth)...)
		n, err := skipValue(data, tSTRUCT, maxSkipDepth)
		if depth == maxSkipDepth {
			assert.Nil(t, err, depth)
			assert.Equal(t, len(data), n, depth)
		} else {
			assert.True(t, errors.Is(err, errDepthLimitExceeded), depth, err)
		}
	}
}
