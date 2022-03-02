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
    `reflect`
    `testing`

    `github.com/stretchr/testify/require`
)

type CompilerTest struct {
    A bool                   `frugal:"0,default,bool"`
    B int8                   `frugal:"1,default,i8"`
    C float64                `frugal:"2,default,double"`
    D int16                  `frugal:"3,default,i16"`
    E int32                  `frugal:"4,default,i32"`
    F int64                  `frugal:"5,default,i64"`
    G string                 `frugal:"6,default,string"`
    H CompilerTestSubStruct  `frugal:"7,default,CompilerTestSubStruct"`
    I *CompilerTestSubStruct `frugal:"8,default,CompilerTestSubStruct"`
    J map[string]int         `frugal:"9,default,map<string:int>"`
    K []string               `frugal:"10,default,set<string>"`
    L []string               `frugal:"11,default,list<string>"`
    M []byte                 `frugal:"12,default,binary"`
    N []int8                 `frugal:"13,default,set<i8>"`
    O []int8                 `frugal:"14,default,list<i8>"`
}

type CompilerTestSubStruct struct {
    X int                    `frugal:"0,default,i64"`
    Y *CompilerTestSubStruct `frugal:"1,default,CompilerTestSubStruct"`
}

func TestCompiler_Compile(t *testing.T) {
    p, err := CreateCompiler().Compile(reflect.TypeOf(CompilerTest{}))
    require.NoError(t, err)
    println(p.Disassemble())
}
