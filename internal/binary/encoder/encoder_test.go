/*
 * Copyright 2022 ByteDance Inc.
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

package encoder

import (
    `bytes`
    `encoding/base64`
    `testing`

    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
)

func TestEncoder_Encode(t *testing.T) {
    v := TranslatorTestStruct {
        A: true,
        B: 0x12,
        C: 12.34,
        D: 0x3456,
        E: 0x12345678,
        F: 0x66778899aabbccdd,
        G: "hello, world",
        H: []byte("testbytebuffer"),
        I: []int32{0x11223344, 0x55667788, 3, 4, 5},
        J: map[string]string{"asdf": "qwer", "zxcv": "hjkl"},
        K: map[string]*TranslatorTestStruct{
            "foo": {
                B: -1,
            },
        },
        L: &(&struct{ x bool }{true}).x,
        M: &(&struct{ x int8 }{0x1}).x,
        N: &(&struct{ x float64 }{0x12345678}).x,
        O: &(&struct{ x int16 }{0x1234}).x,
        P: &(&struct{ x int32 }{0x123456}).x,
        Q: &(&struct{ x int64 }{0x12345678}).x,
    }
    nb := EncodedSize(v)
    println("Estimated Size:", nb)
    buf := make([]byte, nb)
    ret, err := EncodeObject(buf, nil, v)
    if err != nil {
        println("Buffer Shortage:", ret - nb)
        require.NoError(t, err)
    }
    buf = buf[:ret]
    spew.Dump(buf)
    mm := bytes.NewBufferString("\x80\x01\x00\x01\x00\x00\x00\x01a\x00\x00\x00\x00")
    mm.Write(buf)
    println("Base64 Encoded Message:", base64.StdEncoding.EncodeToString(mm.Bytes()))
}

type StructSeekTest struct {
    H StructSeekTestSubStruct `frugal:"0,default,StructSeekTestSubStruct"`
    O []int8                  `frugal:"1,default,list<i8>"`
}

type StructSeekTestSubStruct struct {
    X int  `frugal:"0,default,i64"`
    Y *int `frugal:"1,optional,i64"`
}

func TestEncoder_StructSeek(t *testing.T) {
    c := StructSeekTest{O: []int8{-61}}
    buf := make([]byte, EncodedSize(c))
    _, _ = EncodeObject(buf, nil, c)
}

type ListI8UniqueOuter struct {
    Field0 *ListI8UniqueInner `frugal:"0,default,ListI8UniqueInner"`
}

type ListI8UniqueInner struct {
    Field12336 []int8 `frugal:"12336,default,set<i8>"`
}

func TestEncoder_ListI8Unique(t *testing.T) {
    want := &ListI8UniqueOuter{&ListI8UniqueInner{Field12336: []int8{-61, 67}}}
    buf := make([]byte, EncodedSize(want))
    _, err := EncodeObject(buf, nil, want)
    if err != nil {
        t.Fatal(err)
    }
}

type Foo struct {
    Message string `frugal:"1,required"`
}

type ContainerPointerElement struct {
    Data map[int64]*Foo `frugal:"1,required"`
}

func TestEncoder_ContainerPointerElement(t *testing.T) {
    want := &ContainerPointerElement{map[int64]*Foo{0x1234: nil}}
    nb := EncodedSize(want)
    println("encoded size =", nb)
    buf := make([]byte, nb)
    nx, err := EncodeObject(buf, nil, want)
    if err != nil {
        t.Fatal(err)
    }
    spew.Dump(buf[:nx])
    require.Equal(t, []byte{
        0x0d, 0x00, 0x01, 0x0a, 0x0c, 0x00, 0x00, 0x00, 0x01,   // field 1: map<i64, struct>, len = 1
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34,         //     key   = (i64) 0x1234
        0x00,                                                   //     value = (struct) {}
        0x00,                                                   // end
    }, buf[:nx])
}
