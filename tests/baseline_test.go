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
    `io/ioutil`
    `math/rand`
    `testing`
    `time`

    `github.com/apache/thrift/lib/go/thrift`
    `github.com/cloudwego/frugal`
    vanilla_baseline `github.com/cloudwego/frugal/testdata/baseline`
    kitex_baseline `github.com/cloudwego/frugal/testdata/kitex_gen/baseline`
    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
)

func init() {
    rand.Seed(time.Now().UnixNano())
}

func dumpval(v interface{}) {
    c := spew.NewDefaultConfig()
    c.DisableMethods = true
    c.Dump(v)
}

func loaddata(t require.TestingT, v thrift.TStruct) int {
    buf, err := ioutil.ReadFile("testdata/object.bin")
    require.NoError(t, err)
    mm := thrift.NewTMemoryBuffer()
    _, err = mm.Write(buf)
    require.NoError(t, err)
    err = v.Read(thrift.NewTBinaryProtocolTransport(mm))
    require.NoError(t, err)
    return len(buf)
}

func TestMarshal(t *testing.T) {
    var v vanilla_baseline.Nesting2
    rand.Seed(time.Now().UnixNano())
    GenValue(&v)
    nb := frugal.EncodedSize(v)
    buf := make([]byte, nb)
    _, err := frugal.EncodeObject(buf, nil, v)
    require.NoError(t, err)
    spew.Dump(buf)
    dumpval(v)
}

func TestMarshalUnmarshal(t *testing.T) {
    var v vanilla_baseline.Nesting2
    loaddata(t, &v)
    nb := frugal.EncodedSize(v)
    println("Estimated Size:", nb)
    buf := make([]byte, nb)
    _, err := frugal.EncodeObject(buf, nil, v)
    require.NoError(t, err)
}

func BenchmarkMarshalVanilla(b *testing.B) {
    var v vanilla_baseline.Nesting2
    mm := thrift.NewTMemoryBuffer()
    b.SetBytes(int64(loaddata(b, &v)))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mm.Reset()
        _ = v.Write(thrift.NewTBinaryProtocolTransport(mm))
    }
}

func BenchmarkMarshalKitexFast(b *testing.B) {
    var v kitex_baseline.Nesting2
    b.SetBytes(int64(loaddata(b, &v)))
    b.ResetTimer()
    buf := make([]byte, v.BLength())
    for i := 0; i < b.N; i++ {
        _ = v.FastWriteNocopy(buf, nil)
    }
}

func BenchmarkMarshalKitexFastWithLength(b *testing.B) {
    var v kitex_baseline.Nesting2
    b.SetBytes(int64(loaddata(b, &v)))
    b.ResetTimer()
    buf := make([]byte, v.BLength())
    for i := 0; i < b.N; i++ {
        v.BLength()
        _ = v.FastWriteNocopy(buf, nil)
    }
}

func BenchmarkMarshalFrugal(b *testing.B) {
    var v vanilla_baseline.Nesting2
    b.SetBytes(int64(loaddata(b, &v)))
    b.ResetTimer()
    buf := make([]byte, frugal.EncodedSize(v))
    for i := 0; i < b.N; i++ {
        _, _ = frugal.EncodeObject(buf, nil, v)
    }
}

func BenchmarkMarshalFrugalWithLength(b *testing.B) {
    var v vanilla_baseline.Nesting2
    b.SetBytes(int64(loaddata(b, &v)))
    b.ResetTimer()
    buf := make([]byte, frugal.EncodedSize(v))
    for i := 0; i < b.N; i++ {
        frugal.EncodedSize(v)
        _, _ = frugal.EncodeObject(buf, nil, v)
    }
}
