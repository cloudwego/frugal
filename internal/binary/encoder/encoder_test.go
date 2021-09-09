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

package encoder

import (
    `bytes`
    `encoding/base64`
    `testing`
    `unsafe`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/rt`
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
    }
    rs := new(RuntimeState)
    iov := new(frugal.SimpleIoVec)
    err := encode(rt.UnpackEface(v).Type, iov, unsafe.Pointer(&v), rs, 0)
    require.NoError(t, err)
    spew.Dump(iov.Bytes())
    mm := bytes.NewBufferString("\x80\x01\x00\x01\x00\x00\x00\x01a\x00\x00\x00\x00")
    mm.Write(iov.Bytes())
    println("Base64 Encoded Message:", base64.StdEncoding.EncodeToString(mm.Bytes()))
}
