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
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

func TestEmbeddedUnknownFields(t *testing.T) {
	type fields struct {
		_unknownFields []byte
	}
	type first struct {
		fields
		A int32 `frugal:"1,default,i32"`
	}
	type second struct {
		Pad []byte
		fields
	}
	type nested struct {
		A int32 `frugal:"1,default,i32"`
		second
	}
	type ptrEmbed struct {
		Pad []byte
		*fields
	}

	t.Run("value", func(t *testing.T) {
		unknown := fields{_unknownFields: []byte{byte(tBYTE), 0, 3, 9}}
		checkEncodingRoundTrip(t, &first{fields: unknown, A: 1}, &first{})
		checkEncodingRoundTrip(t, &second{fields: unknown}, &second{})
		checkEncodingRoundTrip(t, &nested{A: 1, second: second{fields: unknown}}, &nested{})
	})

	t.Run("nil pointer", func(t *testing.T) {
		v := ptrEmbed{Pad: []byte{byte(tBYTE), 0, 2, 8}}
		before := v
		assert.Equal(t, 1, EncodedSize(&v))
		b, err := Append(nil, &v)
		assert.Nil(t, err)
		assert.BytesEqual(t, []byte{byte(tSTOP)}, b)

		b = []byte{byte(tBYTE), 0, 1, 7, byte(tSTOP)}
		n, err := Decode(b, &v)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.DeepEqual(t, before, v)
		assert.True(t, v.fields == nil)
	})
}
