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
    `testing`

    `github.com/brianvoe/gofakeit`
    gofakeit_v6 `github.com/brianvoe/gofakeit/v6`
    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/binary/defs`
    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
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
    var ret int
    var err error
    s := &MyTypeTest{}
    gofakeit.Struct(s)
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
}

type MyNumberZ int64

type FakerTestType struct {
    Enum1 *MyNumberZ `frugal:"0,optional,MyNumberZ"`
}

func TestFakerV6(t *testing.T) {
    var got []byte
    var length int
    s := &FakerTestType{}
    _ = gofakeit_v6.Struct(s)
    spew.Dump(s)
    length = frugal.EncodedSize(s)
    got = make([]byte, length)
    _, err := frugal.EncodeObject(got, nil, s)
    require.NoError(t, err)
    spew.Dump(got)
    gotS := &FakerTestType{}
    _, err = frugal.DecodeObject(got, gotS)
    require.NoError(t, err)
}