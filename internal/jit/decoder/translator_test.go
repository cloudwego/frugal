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

package decoder

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type TranslatorTestStruct struct {
	A bool                             `frugal:"0,default,bool"`
	B int8                             `frugal:"1,required,i8"`
	C float64                          `frugal:"2,default,double"`
	D int16                            `frugal:"3,default,i16"`
	E int32                            `frugal:"4,default,i32"`
	F int64                            `frugal:"5,default,i64"`
	G string                           `frugal:"6,default,string"`
	H []byte                           `frugal:"7,default,binary"`
	I []int32                          `frugal:"8,required,list<i32>"`
	J map[string]string                `frugal:"9,default,map<string:string>"`
	K map[string]*TranslatorTestStruct `frugal:"65,required,map<string:TranslatorTestStruct>"`
}

func TestTranslator_Translate(t *testing.T) {
	var v TranslatorTestStruct
	p, err := CreateCompiler().Compile(reflect.TypeOf(v))
	require.NoError(t, err)
	tr := Translate(p)
	println(tr.Disassemble())
}

func TestTranslator_NoCopyString(t *testing.T) {
	var v NoCopyStringTestStruct
	p, err := CreateCompiler().Compile(reflect.TypeOf(v))
	require.NoError(t, err)
	tr := Translate(p)
	println(tr.Disassemble())
}
