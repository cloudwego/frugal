/*
 * Copyright 2021 ByteDance Inc.
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

//go:generate thriftgo -g go:support_frugal -o . baseline.thrift
package tests

import (
    `encoding/binary`
    `io/ioutil`
    `math`
    `math/rand`
    `reflect`
    `sort`
    `testing`
    `time`

    `github.com/apache/thrift/lib/go/thrift`
    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/binary/defs`
    vanilla_baseline `github.com/cloudwego/frugal/testdata/baseline`
    kitex_baseline `github.com/cloudwego/frugal/testdata/kitex_gen/baseline`
    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/assert`
    `github.com/stretchr/testify/require`
)

func init() {
    rand.Seed(time.Now().UnixNano())
}

func dumpval(v interface{}) string {
    c := spew.NewDefaultConfig()
    c.SortKeys = true
    c.SpewKeys = true
    c.DisableMethods = true
    c.DisablePointerAddresses = true
    return c.Sdump(v)
}

func loaddata(t require.TestingT, v thrift.TStruct) []byte {
    buf, err := ioutil.ReadFile("testdata/object.bin")
    require.NoError(t, err)
    if v != nil {
        mm := thrift.NewTMemoryBuffer()
        _, err = mm.Write(buf)
        require.NoError(t, err)
        err = v.Read(thrift.NewTBinaryProtocolTransport(mm))
        require.NoError(t, err)
    }
    return buf
}

func buildvalue(t defs.Tag, v []byte, i int) (int, interface{}) {
    switch t {
    case defs.T_bool:
        if i >= len(v) { panic("unexpected EOF") }
        return i + 1, v[i] != 0
    case defs.T_i8:
        if i >= len(v) { panic("unexpected EOF") }
        return i + 1, int8(v[i])
    case defs.T_double:
        if i >= len(v) - 7 { panic("unexpected EOF") }
        return i + 8, math.Float64frombits(binary.BigEndian.Uint64(v[i:]))
    case defs.T_i16:
        if i >= len(v) - 1 { panic("unexpected EOF") }
        return i + 2, int16(binary.BigEndian.Uint16(v[i:]))
    case defs.T_i32:
        if i >= len(v) - 3 { panic("unexpected EOF") }
        return i + 4, int32(binary.BigEndian.Uint32(v[i:]))
    case defs.T_i64:
        if i >= len(v) - 7 { panic("unexpected EOF") }
        return i + 8, int64(binary.BigEndian.Uint64(v[i:]))
    case defs.T_string:
        if i >= len(v) - 3 { panic("unexpected EOF") }
        nb := int(binary.BigEndian.Uint32(v[i:]))
        i += 4
        if i > len(v) - nb { panic("unexpected EOF") }
        return i + nb, string(v[i:i + nb])
    case defs.T_struct:
        ret := make(map[uint16]interface{})
        for i < len(v) {
            cc := defs.Tag(v[i])
            if cc == 0 {
                return i + 1, ret
            }
            if i >= len(v) - 2 {
                panic("unexpected EOF")
            }
            id := binary.BigEndian.Uint16(v[i + 1:])
            i, ret[id] = buildvalue(cc, v, i + 3)
        }
        panic("unexpected EOF")
    case defs.T_map:
        type Pair struct {
            K interface{}
            V interface{}
        }
        var ret []Pair
        if i >= len(v) - 5 { panic("unexpected EOF") }
        kt, vt, np := defs.Tag(v[i]), defs.Tag(v[i + 1]), binary.BigEndian.Uint32(v[i + 2:])
        i += 6
        for ; np > 0; np-- {
            var k interface{}
            var e interface{}
            i, k = buildvalue(kt, v, i)
            i, e = buildvalue(vt, v, i)
            ret = append(ret, Pair{K: k, V: e})
        }
        sort.Slice(ret, func(i, j int) bool {
            return dumpval(ret[i].K) < dumpval(ret[j].K)
        })
        return i, ret
    case defs.T_set:
        fallthrough
    case defs.T_list:
        var ret []interface{}
        if i >= len(v) - 4 { panic("unexpected EOF") }
        et, nv := defs.Tag(v[i]), binary.BigEndian.Uint32(v[i + 1:])
        i += 5
        for ; nv > 0; nv-- {
            var e interface{}
            i, e = buildvalue(et, v, i)
            ret = append(ret, e)
        }
        return i, ret
    default:
        panic("invalid type")
    }
}

func comparestruct(t require.TestingT, a []byte, b []byte) {
    require.Equal(t, len(a), len(b))
    _, x := buildvalue(defs.T_struct, a, 0)
    _, y := buildvalue(defs.T_struct, b, 0)
    require.True(t, reflect.DeepEqual(x, y))
}

func TestMarshalCompare(t *testing.T) {
    var v vanilla_baseline.Nesting2
    loaddata(t, &v)
    mm := thrift.NewTMemoryBuffer()
    err := v.Write(thrift.NewTBinaryProtocolTransport(mm))
    require.NoError(t, err)
    println("Expected Size :", mm.Len())
    nb := frugal.EncodedSize(v)
    println("Measured Size :", nb)
    require.Equal(t, mm.Len(), nb)
    buf := make([]byte, nb)
    ret, err := frugal.EncodeObject(buf, nil, v)
    require.NoError(t, err)
    println("Encoded Size  :", ret)
    require.Equal(t, nb, ret)
    buf = buf[:ret]
    comparestruct(t, mm.Bytes(), buf)
}

func TestUnmarshalCompare(t *testing.T) {
    var v vanilla_baseline.Nesting2
    var v1 vanilla_baseline.Nesting2
    var v2 vanilla_baseline.Nesting2
    loaddata(t, &v)
    nb := frugal.EncodedSize(v)
    println("Estimated Size:", nb)
    buf := make([]byte, nb)
    _, err := frugal.EncodeObject(buf, nil, v)
    require.NoError(t, err)
    mm := thrift.NewTMemoryBuffer()
    _, _ = mm.Write(buf)
    err = v1.Read(thrift.NewTBinaryProtocolTransport(mm))
    _, err = frugal.DecodeObject(buf, &v2)
    require.NoError(t, err)
    assert.Equal(t, dumpval(v1), dumpval(v2))
}

func BenchmarkMarshalVanilla(b *testing.B) {
    var v vanilla_baseline.Nesting2
    mm := thrift.NewTMemoryBuffer()
    b.SetBytes(int64(len(loaddata(b, &v))))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mm.Reset()
        _ = v.Write(thrift.NewTBinaryProtocolTransport(mm))
    }
}

func BenchmarkMarshalKitexFast(b *testing.B) {
    var v kitex_baseline.Nesting2
    b.SetBytes(int64(len(loaddata(b, &v))))
    b.ResetTimer()
    buf := make([]byte, v.BLength())
    for i := 0; i < b.N; i++ {
        _ = v.FastWriteNocopy(buf, nil)
    }
}

func BenchmarkMarshalKitexFastWithLength(b *testing.B) {
    var v kitex_baseline.Nesting2
    b.SetBytes(int64(len(loaddata(b, &v))))
    b.ResetTimer()
    buf := make([]byte, v.BLength())
    for i := 0; i < b.N; i++ {
        v.BLength()
        _ = v.FastWriteNocopy(buf, nil)
    }
}

func BenchmarkMarshalFrugal(b *testing.B) {
    var v vanilla_baseline.Nesting2
    b.SetBytes(int64(len(loaddata(b, &v))))
    b.ResetTimer()
    buf := make([]byte, frugal.EncodedSize(v))
    for i := 0; i < b.N; i++ {
        _, _ = frugal.EncodeObject(buf, nil, v)
    }
}

func BenchmarkMarshalFrugalWithLength(b *testing.B) {
    var v vanilla_baseline.Nesting2
    b.SetBytes(int64(len(loaddata(b, &v))))
    b.ResetTimer()
    buf := make([]byte, frugal.EncodedSize(v))
    for i := 0; i < b.N; i++ {
        frugal.EncodedSize(v)
        _, _ = frugal.EncodeObject(buf, nil, v)
    }
}

func BenchmarkUnmarshalVanilla(b *testing.B) {
    mm := thrift.NewTMemoryBuffer()
    buf := loaddata(b, nil)
    b.SetBytes(int64(len(buf)))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var v vanilla_baseline.Nesting2
        mm.Reset()
        _, _ = mm.Write(buf)
        _ = v.Read(thrift.NewTBinaryProtocolTransport(mm))
    }
}

func BenchmarkUnmarshalKitexFast(b *testing.B) {
    buf := loaddata(b, nil)
    b.SetBytes(int64(len(buf)))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var v kitex_baseline.Nesting2
        _, _ = v.FastRead(buf)
    }
}

func BenchmarkUnmarshalFrugal(b *testing.B) {
    var r vanilla_baseline.Nesting2
    buf := loaddata(b, nil)
    _, _ = frugal.DecodeObject(buf, &r)
    b.SetBytes(int64(len(buf)))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var v vanilla_baseline.Nesting2
        _, _ = frugal.DecodeObject(buf, &v)
    }
}
