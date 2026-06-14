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
	"errors"
	"io"
	"math"
	"testing"
	"unsafe"

	"github.com/cloudwego/frugal/internal/assert"
	"github.com/cloudwego/frugal/internal/defs"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

func TestDecodeTypeCompact_Scalars(t *testing.T) {
	d := &tDecoder{}

	t.Run("bool_true", func(t *testing.T) {
		var v bool
		n, err := d.decodeTypeCompact(&tType{T: tBOOL}, []byte{byte(ctBOOL_TRUE)}, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, true, v)
	})

	t.Run("bool_false", func(t *testing.T) {
		var v bool
		n, err := d.decodeTypeCompact(&tType{T: tBOOL}, []byte{byte(ctBOOL_FALSE)}, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, false, v)
	})

	t.Run("bool_type_mismatch", func(t *testing.T) {
		var v bool
		_, err := d.decodeTypeCompact(&tType{T: tBOOL}, []byte{0x00}, unsafe.Pointer(&v), 1)
		assert.True(t, err != nil)
	})

	t.Run("byte", func(t *testing.T) {
		var v byte
		n, err := d.decodeTypeCompact(&tType{T: tBYTE}, []byte{0x55}, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, byte(0x55), v)
	})

	t.Run("i16", func(t *testing.T) {
		var v int16
		b := appendZigzag32(nil, -12345)
		n, err := d.decodeTypeCompact(&tType{T: tI16}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, int16(-12345), v)
	})

	t.Run("i32", func(t *testing.T) {
		var v int32
		b := appendZigzag32(nil, -99999)
		n, err := d.decodeTypeCompact(&tType{T: tI32}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, int32(-99999), v)
	})

	t.Run("enum", func(t *testing.T) {
		var v int64
		b := appendZigzag32(nil, int32(-77777))
		n, err := d.decodeTypeCompact(&tType{T: tENUM}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, int64(-77777), v)
	})

	t.Run("i64", func(t *testing.T) {
		var v int64
		b := appendZigzag64(nil, -(1 << 50))
		n, err := d.decodeTypeCompact(&tType{T: tI64}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, -(1 << 50), v)
	})

	t.Run("double", func(t *testing.T) {
		var v float64
		b := appendUint64LE(nil, math.Float64bits(3.14159))
		n, err := d.decodeTypeCompact(&tType{T: tDOUBLE}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, 8, n)
		assert.Equal(t, 3.14159, v)
	})

	t.Run("string", func(t *testing.T) {
		var v string
		s := "hello compact"
		b := appendVarint(nil, uint64(len(s)))
		b = append(b, s...)
		n, err := d.decodeTypeCompact(&tType{T: tSTRING, Tag: defs.T_string}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, s, v)
	})

	t.Run("binary", func(t *testing.T) {
		var v []byte
		raw := []byte{0xde, 0xad, 0xbe, 0xef}
		b := appendVarint(nil, uint64(len(raw)))
		b = append(b, raw...)
		n, err := d.decodeTypeCompact(&tType{T: tSTRING, Tag: defs.T_binary}, b, unsafe.Pointer(&v), 1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.BytesEqual(t, raw, v)
	})
}

func TestDecodeTypeCompact_ShortBuffer(t *testing.T) {
	d := &tDecoder{}

	// bool: needs at least 1 byte
	_, err := d.decodeTypeCompact(&tType{T: tBOOL}, nil, nil, 1)
	assert.True(t, err == io.ErrShortBuffer)

	// byte: needs at least 1 byte
	_, err = d.decodeTypeCompact(&tType{T: tBYTE}, nil, nil, 1)
	assert.True(t, err == io.ErrShortBuffer)

	// string: needs at least 1 byte for varint
	_, err = d.decodeTypeCompact(&tType{T: tSTRING, Tag: defs.T_string}, nil, nil, 1)
	assert.True(t, err == io.ErrShortBuffer)

	// double: needs at least 8 bytes
	_, err = d.decodeTypeCompact(&tType{T: tDOUBLE}, []byte{0}, unsafe.Pointer(new(float64)), 1)
	assert.True(t, err == io.ErrShortBuffer)

	// depth limit exceeded
	var v int32
	b := appendZigzag32(nil, 42)
	_, err = d.decodeTypeCompact(&tType{T: tI32}, b, unsafe.Pointer(&v), 0)
	assert.True(t, err == errDepthLimitExceeded)
}

func TestDecodeStringNoCopyCompact(t *testing.T) {
	// NoCopy path: called from DecodeCompact when f.NoCopy && t.T == tSTRING
	t.Run("normal_string", func(t *testing.T) {
		s := "hello nocopy"
		raw := appendVarint(nil, uint64(len(s)))
		raw = append(raw, s...)
		var v string
		n, err := decodeStringNoCopyCompact(&tType{Tag: defs.T_string}, raw, unsafe.Pointer(&v))
		assert.Nil(t, err)
		assert.Equal(t, len(raw), n)
		assert.Equal(t, s, v)
	})

	t.Run("empty_string", func(t *testing.T) {
		raw := appendVarint(nil, 0)
		var v string
		n, err := decodeStringNoCopyCompact(&tType{Tag: defs.T_string}, raw, unsafe.Pointer(&v))
		assert.Nil(t, err)
		assert.Equal(t, len(raw), n)
		assert.Equal(t, "", v)
	})

	t.Run("empty_binary", func(t *testing.T) {
		raw := appendVarint(nil, 0)
		var v []byte
		n, err := decodeStringNoCopyCompact(&tType{Tag: defs.T_binary}, raw, unsafe.Pointer(&v))
		assert.Nil(t, err)
		assert.Equal(t, len(raw), n)
		assert.BytesEqual(t, []byte{}, v)
	})

	t.Run("short_buffer", func(t *testing.T) {
		var v string
		// Declare non-zero length but truncate data
		b := appendVarint(nil, 5)
		b = append(b, 'h', 'e') // only 2 of 5 bytes
		_, err := decodeStringNoCopyCompact(&tType{Tag: defs.T_string}, b, unsafe.Pointer(&v))
		assertSizeLimitCompact(t, err)
	})

	t.Run("negative_size", func(t *testing.T) {
		// zigzag encode -1 gives all 1s, decode as uint64 gives large > 2^63
		b := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
		var v string
		_, err := decodeStringNoCopyCompact(&tType{Tag: defs.T_string}, b, unsafe.Pointer(&v))
		assert.True(t, err == errNegativeSize)
	})
}

func TestDecodeCompactSizeExceedsBuffer(t *testing.T) {
	// string: declared length exceeds remaining buffer
	t.Run("string", func(t *testing.T) {
		type S struct {
			F string `frugal:"1,required,string"`
		}
		raw := []byte{0x18, 0x10, 0x41} // field 1, length varint continues
		_, err := DecodeCompact(raw, &S{})
		assert.True(t, err != nil)
	})

	// list: element count far exceeds remaining buffer
	t.Run("list", func(t *testing.T) {
		type S struct {
			L []int32 `frugal:"1,required,list<i32>"`
		}
		// 0x19 = field 1, ctLIST. 0xF5 = extended size, ctI32. Then huge varint.
		raw := []byte{0x19, 0xF5, 0x80, 0x80, 0x80, 0x01, 0x00}
		_, err := DecodeCompact(raw, &S{})
		assertSizeLimitCompact(t, err)
	})

	// map: entry count far exceeds remaining buffer
	t.Run("map", func(t *testing.T) {
		type S struct {
			M map[int32]int32 `frugal:"1,required,map<i32:i32>"`
		}
		raw := []byte{0x18, 0x80, 0x80, 0x80, 0x01, 0x00}
		_, err := DecodeCompact(raw, &S{})
		assert.True(t, err != nil)
	})
}

func TestSkipCompact(t *testing.T) {
	// Skip unknown struct field
	t.Run("struct", func(t *testing.T) {
		type Inner struct {
			X int32  `frugal:"1,required,i32"`
			Y string `frugal:"2,required,string"`
		}
		type Outer struct {
			F1 *Inner `frugal:"1,optional"`
		}
		type Reader struct{} // reads nothing, all fields unknown

		p0 := &Outer{F1: &Inner{X: 42, Y: "hello"}}
		b, err := AppendCompact(nil, p0)
		assert.Nil(t, err)

		p1 := &Reader{}
		n, err := DecodeCompact(b, p1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
	})

	// Skip unknown list field
	t.Run("list", func(t *testing.T) {
		type Outer struct {
			L []string `frugal:"1,required,list<string>"`
		}
		type Reader struct{}

		p0 := &Outer{L: []string{"a", "b", "c"}}
		b, err := AppendCompact(nil, p0)
		assert.Nil(t, err)

		p1 := &Reader{}
		n, err := DecodeCompact(b, p1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
	})

	// Skip unknown map field
	t.Run("map", func(t *testing.T) {
		type Outer struct {
			M map[string]int32 `frugal:"1,required,map<string:i32>"`
		}
		type Reader struct{}

		p0 := &Outer{M: map[string]int32{"x": 1, "y": 2}}
		b, err := AppendCompact(nil, p0)
		assert.Nil(t, err)

		p1 := &Reader{}
		n, err := DecodeCompact(b, p1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
	})

	// Collect unknown fields back and verify round-trip
	t.Run("unknown_fields_preserved", func(t *testing.T) {
		type Inner struct {
			X int32  `frugal:"1,required,i32"`
			Y string `frugal:"2,required,string"`
		}
		type Outer struct {
			F1 *Inner `frugal:"1,optional"`
		}
		type Reader struct {
			_unknownFields []byte
		}

		p0 := &Outer{F1: &Inner{X: 42, Y: "hello"}}
		b, err := AppendCompact(nil, p0)
		assert.Nil(t, err)

		p1 := &Reader{}
		n, err := DecodeCompact(b, p1)
		assert.Nil(t, err)
		assert.Equal(t, len(b), n)
		assert.True(t, len(p1._unknownFields) > 0)
	})
}

func assertSizeLimitCompact(t *testing.T, err error) {
	t.Helper()
	var pe *thrift.ProtocolException
	if !errors.As(err, &pe) {
		t.Fatalf("expected *thrift.ProtocolException, got %v", err)
	}
	assert.Equal(t, int32(thrift.SIZE_LIMIT), pe.TypeID())
}

func TestDecodeCompactNegativeSize(t *testing.T) {
	// string with negative length should produce errNegativeSize
	t.Run("string", func(t *testing.T) {
		type S struct {
			F string `frugal:"1,required,string"`
		}
		// field header (0x18 = delta:1, ctBINARY) + varint that overflows int to negative
		b := []byte{0x18, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
		_, err := DecodeCompact(b, &S{})
		assert.True(t, errors.Is(err, errNegativeSize))
	})

	// list with negative element count (raw varint, not zigzag)
	t.Run("list", func(t *testing.T) {
		type S struct {
			L []int32 `frugal:"1,required,list<i32>"`
		}
		// 0x19 = field 1, ctLIST. 0xF5 = extended size,ctI32. Then huge varint.
		b := []byte{0x19, 0xF5, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x00}
		_, err := DecodeCompact(b, &S{})
		assert.True(t, errors.Is(err, errNegativeSize))
	})

	// map with negative entry count
	t.Run("map", func(t *testing.T) {
		type S struct {
			M map[int32]int32 `frugal:"1,required,map<i32:i32>"`
		}
		b := []byte{0x18, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
		_, err := DecodeCompact(b, &S{})
		assert.True(t, errors.Is(err, errNegativeSize))
	})
}
