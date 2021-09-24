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

package defs

import (
    `fmt`
    `reflect`
    `testing`
)

func TestTypes_Parsing(t *testing.T) {
    var v map[string][]reflect.SliceHeader
    tt := ParseType(reflect.TypeOf(v), "map<string:set<foo.SliceHeader>>")
    fmt.Println(tt)
}

func TestTypes_MapKeyType(t *testing.T) {
    var v map[*reflect.SliceHeader]int
    tt := ParseType(reflect.TypeOf(v), "map<foo.SliceHeader:i64>")
    fmt.Println(tt)
}
