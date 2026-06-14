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
	"sort"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/frugal"
	"github.com/cloudwego/frugal/internal/defs"
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

// compactVarint is a compact protocol varint decoder.
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

// buildCompactValue mirrors buildvalue for compact protocol.  The t
// parameter is unused (type is embedded in the compact field header).  It
// is kept for signature compatibility with buildvalue.
func buildCompactValue(t defs.Tag, v []byte, i int) (int, interface{}) {
	ret := make(map[uint16]interface{})
	var lastId uint16
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
		ni, val := buildCompactScalar(v, i, ft)
		i += ni
		ret[lastId] = val
	}
	return i, ret
}

func buildCompactScalar(v []byte, i int, ft uint8) (int, interface{}) {
	switch ft {
	case 0x01:
		return 0, true
	case 0x02:
		return 0, false
	case 0x03:
		return 1, int8(v[i])
	case 0x04:
		n, ni := compactVarint(v[i:])
		return ni, int16(int32((n >> 1) ^ -(n & 1)))
	case 0x05:
		n, ni := compactVarint(v[i:])
		return ni, int32((n >> 1) ^ -(n & 1))
	case 0x06:
		n, ni := compactVarint(v[i:])
		return ni, int64((n >> 1) ^ -(n & 1))
	case 0x07:
		if i+8 > len(v) {
			return 0, float64(0)
		}
		var u uint64
		for j := 0; j < 8; j++ {
			u |= uint64(v[i+j]) << (8 * uint(j))
		}
		return 8, math.Float64frombits(u)
	case 0x08:
		l64, ni := compactVarint(v[i:])
		l := int(l64)
		if l < 0 || i+ni+l > len(v) {
			return len(v) - i, "[truncated string]"
		}
		return ni + l, string(v[i+ni : i+ni+l])
	case 0x0C:
		ni, val := buildCompactValue(0, v, i)
		return ni - i, val
	case 0x09, 0x0A:
		ni, val := buildCompactList(v, i)
		return ni - i, val
	case 0x0B:
		ni, val := buildCompactMap(v, i)
		return ni - i, val
	}
	return 0, nil
}

func buildCompactList(v []byte, i int) (int, interface{}) {
	sn := v[i] >> 4
	et := v[i] & 0x0F
	i++
	l := int(sn)
	if sn == 0xF {
		n, ni := compactVarint(v[i:])
		l = int(n)
		i += ni
	}
	elems := make([]interface{}, 0, l)
	for j := 0; j < l; j++ {
		ni, val := buildCompactScalar(v, i, et)
		i += ni
		elems = append(elems, val)
	}
	return i, elems
}

func buildCompactMap(v []byte, i int) (int, interface{}) {
	n, ni := compactVarint(v[i:])
	l := int(n)
	i += ni
	type pair struct {
		K interface{}
		V interface{}
	}
	var kt, vt uint8
	if l > 0 {
		kt = v[i] >> 4
		vt = v[i] & 0x0F
		i++
	}
	entries := make([]pair, 0, l)
	for j := 0; j < l; j++ {
		nk, kv := buildCompactScalar(v, i, kt)
		i += nk
		nv, vv := buildCompactScalar(v, i, vt)
		i += nv
		entries = append(entries, pair{K: kv, V: vv})
	}
	sort.Slice(entries, func(a, b int) bool {
		return dumpval(entries[a].K) < dumpval(entries[b].K)
	})
	return i, entries
}
