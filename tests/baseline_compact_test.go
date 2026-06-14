/*
 * Copyright 2022 CloudWeGo Authors
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

package tests

import (
	"math"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/frugal"
	"github.com/cloudwego/frugal/tests/baseline"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compactProtocol(trans thrift.TTransport) thrift.TProtocol {
	return thrift.NewTCompactProtocolFactory().GetProtocol(trans)
}

func TestCompactMarshalEnum(t *testing.T) {
	v := EnumTestStruct{X: baseline.Enums_ValueC}
	n := frugal.EncodedSizeCompact(v)
	buf := make([]byte, n)
	ret, err := frugal.EncodeObjectCompact(buf, nil, v)
	require.NoError(t, err)
	require.Equal(t, n, ret)
	require.Equal(t, []byte{0x15, 0x04, 0x00}, buf)
	spew.Dump(buf)
}

func TestCompactMarshalCompare(t *testing.T) {
	var v baseline.Nesting2
	loaddata(t, &v)
	mm := thrift.NewTMemoryBuffer()
	err := v.Write(compactProtocol(mm))
	require.NoError(t, err)
	nb := frugal.EncodedSizeCompact(&v)
	require.Equal(t, mm.Len(), nb)
	buf := make([]byte, nb)
	ret, err := frugal.EncodeObjectCompact(buf, nil, v)
	require.NoError(t, err)
	require.Equal(t, nb, ret)
	buf = buf[:ret]
}

func TestCompactMarshalDefaultCompare(t *testing.T) {
	var v baseline.OptionalDefaultValues
	v.InitDefault()
	mm := thrift.NewTMemoryBuffer()
	err := v.Write(compactProtocol(mm))
	require.NoError(t, err)
	nb := frugal.EncodedSizeCompact(v)
	require.Equal(t, mm.Len(), nb)
	buf := make([]byte, nb)
	ret, err := frugal.EncodeObjectCompact(buf, nil, v)
	require.NoError(t, err)
	require.Equal(t, nb, ret)
	buf = buf[:ret]
}

func TestCompactUnmarshalEnum(t *testing.T) {
	var v EnumTestStruct
	v.X = baseline.Enums(1 << 32)
	buf := []byte{0x15, 0x04, 0x00}
	ret, err := frugal.DecodeObjectCompact(buf, &v)
	require.NoError(t, err)
	require.Equal(t, len(buf), ret)
	require.Equal(t, EnumTestStruct{X: baseline.Enums_ValueC}, v)
	spew.Dump(v)
}

func TestCompactUnmarshalCompare(t *testing.T) {
	var v baseline.Nesting2
	var v1 baseline.Nesting2
	var v2 baseline.Nesting2
	loaddata(t, &v)
	nb := frugal.EncodedSizeCompact(v)
	buf := make([]byte, nb)
	_, err := frugal.EncodeObjectCompact(buf, nil, v)
	require.NoError(t, err)
	mm := thrift.NewTMemoryBuffer()
	_, _ = mm.Write(buf)
	err = v1.Read(compactProtocol(mm))
	require.NoError(t, err)
	_, err = frugal.DecodeObjectCompact(buf, &v2)
	require.NoError(t, err)
	assert.Equal(t, dumpval(v1), dumpval(v2))
}

func TestCompactBaseline_SizeEstimation(t *testing.T) {
	v := baseline.Simple{
		ByteField:   math.MaxInt8,
		I64Field:    math.MaxInt64,
		DoubleField: math.MaxFloat64,
		I32Field:    math.MaxInt32,
		StringField: "hello",
		BinaryField: []byte{0xde, 0xad, 0xbe, 0xef},
	}
	nb := frugal.EncodedSizeCompact(&v)
	mm := thrift.NewTMemoryBuffer()
	err := v.Write(compactProtocol(mm))
	require.NoError(t, err)
	require.Equal(t, mm.Len(), nb)
	buf := make([]byte, nb)
	ret, err := frugal.EncodeObjectCompact(buf, nil, &v)
	require.NoError(t, err)
	require.Equal(t, nb, ret)
	require.Equal(t, mm.Bytes(), buf)
}

// buildCompactTree and compactVarint are debug helpers for TestCompactTypeSerdes.

func compactVarint(b []byte) (uint64, int) {
	var v uint64
	var s uint
	for i, c := range b {
		v |= uint64(c&0x7f) << s
		s += 7
		if c&0x80 == 0 {
			return v, i + 1
		}
	}
	return 0, len(b)
}

func buildCompactTree(v []byte) (int, map[uint16]interface{}) {
	ret := make(map[uint16]interface{})
	var lastId uint16
	i := 0
	for i < len(v) {
		h := v[i]
		ft := h & 0x0F
		delta := uint16(h >> 4)
		i++
		if ft == 0 && delta == 0 {
			break
		}
		if delta == 0 {
			if i+2 > len(v) {
				break
			}
			lastId = uint16(v[i])<<8 | uint16(v[i+1])
			i += 2
		} else {
			lastId += delta
		}
		switch {
		case ft <= 8:
			ni, val := buildCompactScalar(v[i:], ft)
			ret[lastId] = val
			i += ni
		case ft == 12:
			ni, nested := buildCompactTree(v[i-1:])
			ret[lastId] = nested
			i = (i - 1) + ni
		default:
			ret[lastId] = "[container]"
		}
	}
	return i, ret
}

func buildCompactScalar(b []byte, ft uint8) (int, interface{}) {
	switch ft {
	case 1:
		return 0, true
	case 2:
		return 0, false
	case 3:
		return 1, int8(b[0])
	case 4:
		n, ni := compactVarint(b)
		return ni, int16(int32((n >> 1) ^ -(n & 1)))
	case 5:
		n, ni := compactVarint(b)
		return ni, int32((n >> 1) ^ -(n & 1))
	case 6:
		n, ni := compactVarint(b)
		return ni, int64((n >> 1) ^ -(n & 1))
	case 7:
		if len(b) < 8 {
			return len(b), float64(0)
		}
		var u uint64
		for j := 0; j < 8; j++ {
			u |= uint64(b[j]) << (8 * uint(j))
		}
		return 8, math.Float64frombits(u)
	case 8:
		l64, ni := compactVarint(b)
		l := int(l64)
		if l < 0 || ni+l > len(b) {
			return len(b), "[truncated string]"
		}
		return ni + l, string(b[ni : ni+l])
	}
	return 0, nil
}
