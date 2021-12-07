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
    `github.com/cloudwego/frugal/iovec`
    `github.com/cloudwego/frugal/testdata/baseline`
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

func TestMarshal(t *testing.T) {
    var m iovec.SimpleIoVec
    var v baseline.Nesting2
    rand.Seed(time.Now().UnixNano())
    GenValue(&v)
    err := frugal.EncodeObject(&m, v)
    require.NoError(t, err)
    spew.Dump(m.Bytes())
    dumpval(v)
}

func TestMarshalBaseline(t *testing.T) {
    var m iovec.SimpleIoVec
    var v baseline.Nesting2
    buf, err := ioutil.ReadFile("testdata/object.bin")
    require.NoError(t, err)
    mm := thrift.NewTMemoryBuffer()
    _, err = mm.Write(buf)
    require.NoError(t, err)
    err = v.Read(thrift.NewTBinaryProtocolTransport(mm))
    require.NoError(t, err)
    _ = frugal.EncodeObject(&m, v)
}

func BenchmarkMarshalVanilla(b *testing.B) {
    var v baseline.Nesting2
    buf, err := ioutil.ReadFile("testdata/object.bin")
    require.NoError(b, err)
    mm := thrift.NewTMemoryBuffer()
    _, err = mm.Write(buf)
    require.NoError(b, err)
    err = v.Read(thrift.NewTBinaryProtocolTransport(mm))
    require.NoError(b, err)
    b.SetBytes(int64(len(buf)))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mm.Reset()
        _ = v.Write(thrift.NewTBinaryProtocolTransport(mm))
    }
}

func BenchmarkMarshalFrugal(b *testing.B) {
    var m iovec.SimpleIoVec
    var v baseline.Nesting2
    buf, err := ioutil.ReadFile("testdata/object.bin")
    require.NoError(b, err)
    mm := thrift.NewTMemoryBuffer()
    _, err = mm.Write(buf)
    require.NoError(b, err)
    err = v.Read(thrift.NewTBinaryProtocolTransport(mm))
    require.NoError(b, err)
    b.SetBytes(int64(len(buf)))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.Reset()
        _ = frugal.EncodeObject(&m, v)
    }
}
