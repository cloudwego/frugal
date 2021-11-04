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

package decoder

import (
    `testing`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
)

func TestDecoder_Decode(t *testing.T) {
    var v TranslatorTestStruct
    rs := new(RuntimeState)
    buf := []byte {
        0x02, 0x00, 0x00, 0x01, 0x03, 0x00, 0x01, 0x12, 0x04, 0x00, 0x02, 0x40, 0x28, 0xae, 0x14, 0x7a,
        0xe1, 0x47, 0xae, 0x06, 0x00, 0x03, 0x34, 0x56, 0x08, 0x00, 0x04, 0x12, 0x34, 0x56, 0x78, 0x0a,
        0x00, 0x05, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0x0b, 0x00, 0x06, 0x00, 0x00, 0x00,
        0x0c, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x2c, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x0b, 0x00, 0x07,
        0x00, 0x00, 0x00, 0x0e, 0x74, 0x65, 0x73, 0x74, 0x62, 0x79, 0x74, 0x65, 0x62, 0x75, 0x66, 0x66,
        0x65, 0x72, 0x0f, 0x00, 0x08, 0x08, 0x00, 0x00, 0x00, 0x05, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66,
        0x77, 0x88, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x05, 0x0d, 0x00,
        0x09, 0x0b, 0x0b, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, 0x61, 0x73, 0x64, 0x66, 0x00,
        0x00, 0x00, 0x04, 0x71, 0x77, 0x65, 0x72, 0x00, 0x00, 0x00, 0x04, 0x7a, 0x78, 0x63, 0x76, 0x00,
        0x00, 0x00, 0x04, 0x68, 0x6a, 0x6b, 0x6c, 0x0d, 0x00, 0x41, 0x0b, 0x0c, 0x00, 0x00, 0x00, 0x01,
        0x00, 0x00, 0x00, 0x03, 0x66, 0x6f, 0x6f, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x01, 0xff, 0x04,
        0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x00, 0x03, 0x00, 0x00, 0x08,
        0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x0b, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x0b, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00, 0x0f,
        0x00, 0x08, 0x08, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x00, 0x09, 0x0b, 0x0b, 0x00, 0x00, 0x00, 0x00,
        0x0d, 0x00, 0x41, 0x0b, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    }
    pos, err := decode(rt.UnpackEface(v).Type, buf, 0, unsafe.Pointer(&v), rs, 0)
    require.NoError(t, err)
    require.Equal(t, len(buf), pos)
    require.Equal(t, TranslatorTestStruct {
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
    }, v)
    spew.Dump(v)
}
