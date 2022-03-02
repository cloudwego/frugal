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

package main

import (
    `fmt`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/testdata/kitex_gen/baseline`
)

var V int

var Obj []byte

func init() {
    obj := new(baseline.Simple)
    buf := make([]byte, frugal.EncodedSize(obj))
    _, err := frugal.EncodeObject(buf, nil, obj)
    if err != nil {
        panic(err)
    }
    Obj = buf
}

func F() { fmt.Printf("Hello, number %d\n", V) }

func Marshal(val interface{}) ([]byte, error) {
    buf := make([]byte, frugal.EncodedSize(val))
    _, err := frugal.EncodeObject(buf, nil, val)
    return buf, err
}
