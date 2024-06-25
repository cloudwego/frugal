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

package tests

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"

	"github.com/cloudwego/frugal"
	"github.com/cloudwego/frugal/debug"
	"github.com/cloudwego/frugal/internal/binary/defs"
	"github.com/cloudwego/frugal/tests/kitex_gen/baseline"
)

type MyNode struct {
	Name string `thrift:"Name,1" frugal:"1,default,string" json:"Name"`
	ID   int32  `thrift:"ID,2" frugal:"2,default,i32" json:"ID"`
}

type MyTypeTest struct {
	Bool0   bool              `thrift:"Bool0,1" frugal:"1,default,bool" json:"memem"`
	Bool1   *bool             `thrift:"Bool1,2" frugal:"2,optional,bool" json:"Bool1,omitempty"`
	Byte0   int8              `thrift:"Byte0,3" frugal:"3,default,byte" json:"Byte0"`
	Byte1   *int8             `thrift:"Byte1,4" frugal:"4,optional,byte" json:"Byte1,omitempty"`
	I80     int8              `thrift:"I80,5" frugal:"5,default,i8" json:"I80"`
	I81     int8              `thrift:"I81,6" frugal:"6,optional,i8" json:"I81,omitempty"`
	Double0 float64           `thrift:"Double0,7" frugal:"7,default,double" json:"Double0"`
	Double1 *float64          `thrift:"Double1,8" frugal:"8,optional,double" json:"Double1,omitempty"`
	String0 string            `thrift:"String0,9" frugal:"9,default,string" json:"String0"`
	String1 *string           `thrift:"String1,10" frugal:"10,optional,string" json:"String1,omitempty"`
	Binary0 []byte            `thrift:"Binary0,11" frugal:"11,default,binary" json:"Binary0"`
	Binary1 []byte            `thrift:"Binary1,12" frugal:"12,optional,binary" json:"Binary1,omitempty"`
	Map0    map[string]string `thrift:"Map0,13" frugal:"13,default,map<string:string>" json:"Map0"`
	Map1    map[string]string `thrift:"Map1,14" frugal:"14,optional,map<string:string>" json:"Map1,omitempty"`
	Set0    []string          `thrift:"Set0,15" frugal:"15,default,list<string>" json:"Set0"`
	Set1    []string          `thrift:"Set1,16" frugal:"16,optional,list<string>" json:"Set1,omitempty"`
	List0   []string          `thrift:"list0,17" frugal:"17,default,list<string>" json:"list0"`
	List1   []string          `thrift:"List1,18" frugal:"18,optional,list<string>" json:"List1,omitempty"`
	I160    int16             `thrift:"I160,19" frugal:"19,default,i16" json:"I160"`
	I161    *int16            `thrift:"I161,20" frugal:"20,optional,i16" json:"I161,omitempty"`
	I320    int32             `thrift:"I320,21" frugal:"21,default,i32" json:"I320"`
	I321    int32             `thrift:"I321,22" frugal:"22,optional,i32" json:"I321,omitempty"`
	I640    int64             `thrift:"I640,23" frugal:"23,default,i64" json:"I640"`
	I641    *int64            `thrift:"I641,24" frugal:"24,optional,i64" json:"I641,omitempty"`
	Struct0 *MyNode           `thrift:"Struct0,25" frugal:"25,default,MyNode" json:"Struct0"`
}

func TestTypeSerdes(t *testing.T) {
	f := fuzz.New()
	var ret int
	var err error
	s := &MyTypeTest{}
	f.Fuzz(s)
	got := make([]byte, frugal.EncodedSize(s))
	ret, err = frugal.EncodeObject(got, nil, s)
	require.NoError(t, err)
	require.Equal(t, len(got), ret)
	println("--------- THRIFT BYTES ---------")
	spew.Dump(got)
	_, vv := buildvalue(defs.T_struct, got, 0)
	spew.Config.SortKeys = true
	println("--------- VALUE TREE ---------")
	spew.Dump(vv)
	gotS := &MyTypeTest{}
	ret, err = frugal.DecodeObject(got, gotS)
	require.NoError(t, err)
	require.Equal(t, len(got), ret)
	println("--------- ORIGINAL VALUE ---------")
	spew.Dump(s)
	println("--------- DECODED VALUE ---------")
	spew.Dump(gotS)
	spew.Dump(debug.GetStats())
}

type MyNumberZ int64

type FakerTestType struct {
	Enum1 *MyNumberZ `frugal:"0,optional,MyNumberZ"`
}

type SignExtTest struct {
	X MyNumberZ `frugal:"0,default,MyNumberZ"`
}

func TestSignExtension(t *testing.T) {
	v := SignExtTest{X: MyNumberZ(-3948394)}
	n := frugal.EncodedSize(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObject(m, nil, &v)
	require.NoError(t, err)
	spew.Dump(m)
	v = SignExtTest{}
	_, err = frugal.DecodeObject(m, &v)
	require.NoError(t, err)
	require.Equal(t, MyNumberZ(-3948394), v.X)
}

type EnumKeyTest struct {
	X map[MyNumberZ]int64 `frugal:"0,default,map<MyNumberZ:i64>"`
}

func TestEnumKey(t *testing.T) {
	v := EnumKeyTest{X: map[MyNumberZ]int64{MyNumberZ(-3948394): -123}}
	n := frugal.EncodedSize(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObject(m, nil, &v)
	require.NoError(t, err)
	spew.Dump(m)
	_, val := buildvalue(defs.T_struct, m, 0)
	spew.Dump(val)
	v = EnumKeyTest{}
	_, err = frugal.DecodeObject(m, &v)
	require.NoError(t, err)
	require.Equal(t, map[MyNumberZ]int64{MyNumberZ(-3948394): -123}, v.X)
}

type ListOfEnumTest struct {
	X []MyNumberZ `frugal:"0,default,list<MyNumberZ>"`
}

func TestListOfEnum(t *testing.T) {
	v := ListOfEnumTest{X: []MyNumberZ{-3948394, 0, 1, 2, 3, 4, 5}}
	n := frugal.EncodedSize(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObject(m, nil, &v)
	require.NoError(t, err)
	spew.Dump(m)
	_, val := buildvalue(defs.T_struct, m, 0)
	spew.Dump(val)
	v = ListOfEnumTest{}
	_, err = frugal.DecodeObject(m, &v)
	require.NoError(t, err)
	require.Equal(t, []MyNumberZ{-3948394, 0, 1, 2, 3, 4, 5}, v.X)
}

func TestPretouch(t *testing.T) {
	var v baseline.Nesting2
	s0 := debug.GetStats()
	err := frugal.Pretouch(reflect.TypeOf(v), frugal.WithMaxInlineDepth(1), frugal.WithMaxInlineILSize(0))
	require.NoError(t, err)
	spew.Dump(s0, debug.GetStats())
}

func TestSSACompile(t *testing.T) {
	var v baseline.Nesting2
	println(frugal.EncodedSize(v))
}
