/*
 * Copyright 2024 CloudWeGo Authors
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

// Code generated by thriftgo (0.3.13). DO NOT EDIT.

package reflect

import (
	"fmt"
)

type Numberz int64

const (
	Numberz_TEN Numberz = 10
)

func NumberzFromString(s string) (Numberz, error) {
	switch s {
	case "TEN":
		return Numberz_TEN, nil
	}
	return Numberz(0), fmt.Errorf("not a valid Numberz string")
}

func NumberzPtr(v Numberz) *Numberz { return &v }

type UserID = int64

type Msg struct {
	Message string `thrift:"message,1" frugal:"1,default,string" json:"message"`
	Type    int32  `thrift:"type,2" frugal:"2,default,i32" json:"type"`
}

func NewMsg() *Msg {
	return &Msg{}
}

func (p *Msg) InitDefault() {
}

type TestTypes struct {
	FBool   bool              `thrift:"FBool,1,required" frugal:"1,required,bool" json:"FBool"`
	FByte   int8              `thrift:"FByte,2" frugal:"2,default,byte" json:"FByte"`
	I8      int8              `thrift:"I8,3" frugal:"3,default,i8" json:"I8"`
	I16     int16             `thrift:"I16,4" frugal:"4,default,i16" json:"I16"`
	I32     int32             `thrift:"I32,5" frugal:"5,default,i32" json:"I32"`
	I64     int64             `thrift:"I64,6" frugal:"6,default,i64" json:"I64"`
	Double  float64           `thrift:"Double,7" frugal:"7,default,double" json:"Double"`
	String_ string            `thrift:"String,8" frugal:"8,default,string" json:"String"`
	Binary  []byte            `thrift:"Binary,9" frugal:"9,default,binary" json:"Binary"`
	Enum    Numberz           `thrift:"Enum,10" frugal:"10,default,Numberz" json:"Enum"`
	UID     UserID            `thrift:"UID,11" frugal:"11,default,i64" json:"UID"`
	S       *Msg              `thrift:"S,12" frugal:"12,default,Msg" json:"S"`
	M0      map[int32]int32   `thrift:"M0,20,required" frugal:"20,required,map<i32:i32>" json:"M0"`
	M1      map[int32]string  `thrift:"M1,21" frugal:"21,default,map<i32:string>" json:"M1"`
	M2      map[int32]*Msg    `thrift:"M2,22" frugal:"22,default,map<i32:Msg>" json:"M2"`
	M3      map[string]*Msg   `thrift:"M3,23" frugal:"23,default,map<string:Msg>" json:"M3"`
	L0      []int32           `thrift:"L0,30,required" frugal:"30,required,list<i32>" json:"L0"`
	L1      []string          `thrift:"L1,31" frugal:"31,default,list<string>" json:"L1"`
	L2      []*Msg            `thrift:"L2,32" frugal:"32,default,list<Msg>" json:"L2"`
	S0      []int32           `thrift:"S0,40,required" frugal:"40,required,set<i32>" json:"S0"`
	S1      []string          `thrift:"S1,41" frugal:"41,default,set<string>" json:"S1"`
	LM      []map[int32]int32 `thrift:"LM,50" frugal:"50,default,list<map<i32:i32>>" json:"LM"`
	ML      map[int32][]int32 `thrift:"ML,60" frugal:"60,default,map<i32:list<i32>>" json:"ML"`
}

func NewTestTypes() *TestTypes {
	return &TestTypes{}
}

func (p *TestTypes) InitDefault() {
}

type TestTypesOptional struct {
	FBool   *bool             `thrift:"FBool,1,optional" frugal:"1,optional,bool" json:"FBool,omitempty"`
	FByte   *int8             `thrift:"FByte,2,optional" frugal:"2,optional,byte" json:"FByte,omitempty"`
	I8      *int8             `thrift:"I8,3,optional" frugal:"3,optional,i8" json:"I8,omitempty"`
	I16     *int16            `thrift:"I16,4,optional" frugal:"4,optional,i16" json:"I16,omitempty"`
	I32     *int32            `thrift:"I32,5,optional" frugal:"5,optional,i32" json:"I32,omitempty"`
	I64     *int64            `thrift:"I64,6,optional" frugal:"6,optional,i64" json:"I64,omitempty"`
	Double  *float64          `thrift:"Double,7,optional" frugal:"7,optional,double" json:"Double,omitempty"`
	String_ *string           `thrift:"String,8,optional" frugal:"8,optional,string" json:"String,omitempty"`
	Binary  []byte            `thrift:"Binary,9,optional" frugal:"9,optional,binary" json:"Binary,omitempty"`
	Enum    *Numberz          `thrift:"Enum,10,optional" frugal:"10,optional,Numberz" json:"Enum,omitempty"`
	UID     *UserID           `thrift:"UID,11,optional" frugal:"11,optional,i64" json:"UID,omitempty"`
	S       *Msg              `thrift:"S,12,optional" frugal:"12,optional,Msg" json:"S,omitempty"`
	M0      map[int32]int32   `thrift:"M0,20,optional" frugal:"20,optional,map<i32:i32>" json:"M0,omitempty"`
	M1      map[int32]string  `thrift:"M1,21,optional" frugal:"21,optional,map<i32:string>" json:"M1,omitempty"`
	M2      map[int32]*Msg    `thrift:"M2,22,optional" frugal:"22,optional,map<i32:Msg>" json:"M2,omitempty"`
	M3      map[string]*Msg   `thrift:"M3,23,optional" frugal:"23,optional,map<string:Msg>" json:"M3,omitempty"`
	L0      []int32           `thrift:"L0,30,optional" frugal:"30,optional,list<i32>" json:"L0,omitempty"`
	L1      []string          `thrift:"L1,31,optional" frugal:"31,optional,list<string>" json:"L1,omitempty"`
	L2      []*Msg            `thrift:"L2,32,optional" frugal:"32,optional,list<Msg>" json:"L2,omitempty"`
	S0      []int32           `thrift:"S0,40,optional" frugal:"40,optional,set<i32>" json:"S0,omitempty"`
	S1      []string          `thrift:"S1,41,optional" frugal:"41,optional,set<string>" json:"S1,omitempty"`
	LM      []map[int32]int32 `thrift:"LM,50,optional" frugal:"50,optional,list<map<i32:i32>>" json:"LM,omitempty"`
	ML      map[int32][]int32 `thrift:"ML,60,optional" frugal:"60,optional,map<i32:list<i32>>" json:"ML,omitempty"`
}

func NewTestTypesOptional() *TestTypesOptional {
	return &TestTypesOptional{}
}

func (p *TestTypesOptional) InitDefault() {
}

type TestTypesWithDefault struct {
	FBool   bool    `thrift:"FBool,1,optional" frugal:"1,optional,bool" json:"FBool,omitempty"`
	FByte   int8    `thrift:"FByte,2,optional" frugal:"2,optional,byte" json:"FByte,omitempty"`
	I8      int8    `thrift:"I8,3,optional" frugal:"3,optional,i8" json:"I8,omitempty"`
	I6      int16   `thrift:"I6,4,optional" frugal:"4,optional,i16" json:"I6,omitempty"`
	I32     int32   `thrift:"I32,5,optional" frugal:"5,optional,i32" json:"I32,omitempty"`
	I64     int64   `thrift:"I64,6,optional" frugal:"6,optional,i64" json:"I64,omitempty"`
	Double  float64 `thrift:"Double,7,optional" frugal:"7,optional,double" json:"Double,omitempty"`
	String_ string  `thrift:"String,8,optional" frugal:"8,optional,string" json:"String,omitempty"`
	Binary  []byte  `thrift:"Binary,9,optional" frugal:"9,optional,binary" json:"Binary,omitempty"`
	Enum    Numberz `thrift:"Enum,10,optional" frugal:"10,optional,Numberz" json:"Enum,omitempty"`
	UID     UserID  `thrift:"UID,11,optional" frugal:"11,optional,i64" json:"UID,omitempty"`
	L0      []int32 `thrift:"L0,30,optional" frugal:"30,optional,list<i32>" json:"L0,omitempty"`
	S0      []int32 `thrift:"S0,40,optional" frugal:"40,optional,set<i32>" json:"S0,omitempty"`
}

func NewTestTypesWithDefault() *TestTypesWithDefault {
	return &TestTypesWithDefault{
		FBool:   true,
		FByte:   2,
		I8:      3,
		I6:      4,
		I32:     5,
		I64:     6,
		Double:  7.0,
		String_: "8",
		Binary:  []byte("8"),
		Enum:    10,
		UID:     11,
		L0: []int32{
			30,
		},
		S0: []int32{
			40,
		},
	}
}

func (p *TestTypesWithDefault) InitDefault() {
	p.FBool = true
	p.FByte = 2
	p.I8 = 3
	p.I6 = 4
	p.I32 = 5
	p.I64 = 6
	p.Double = 7.0
	p.String_ = "8"
	p.Binary = []byte("8")
	p.Enum = 10
	p.UID = 11
	p.L0 = []int32{
		30,
	}
	p.S0 = []int32{
		40,
	}
}

type TestTypesForBenchmark struct {
	B0   bool            `thrift:"B0,1,optional" frugal:"1,optional,bool" json:"B0,omitempty"`
	B1   *bool           `thrift:"B1,2,optional" frugal:"2,optional,bool" json:"B1,omitempty"`
	B2   bool            `thrift:"B2,3,required" frugal:"3,required,bool" json:"B2"`
	Str0 string          `thrift:"Str0,11,optional" frugal:"11,optional,string" json:"Str0,omitempty"`
	Str1 *string         `thrift:"Str1,12,optional" frugal:"12,optional,string" json:"Str1,omitempty"`
	Str2 string          `thrift:"Str2,13,required" frugal:"13,required,string" json:"Str2"`
	Str3 string          `thrift:"Str3,14,required" frugal:"14,required,string" json:"Str3"`
	Msg0 *Msg            `thrift:"Msg0,21,optional" frugal:"21,optional,Msg" json:"Msg0,omitempty"`
	Msg1 *Msg            `thrift:"Msg1,22,required" frugal:"22,required,Msg" json:"Msg1"`
	M0   map[int32]int32 `thrift:"M0,31,optional" frugal:"31,optional,map<i32:i32>" json:"M0,omitempty"`
	M1   map[string]*Msg `thrift:"M1,32,required" frugal:"32,required,map<string:Msg>" json:"M1"`
	L0   []int32         `thrift:"L0,41,optional" frugal:"41,optional,list<i32>" json:"L0,omitempty"`
	L1   []*Msg          `thrift:"L1,42,required" frugal:"42,required,list<Msg>" json:"L1"`
	Set0 []int32         `thrift:"Set0,51,optional" frugal:"51,optional,set<i32>" json:"Set0,omitempty"`
	Set1 []string        `thrift:"Set1,52,required" frugal:"52,required,set<string>" json:"Set1"`
}

func NewTestTypesForBenchmark() *TestTypesForBenchmark {
	return &TestTypesForBenchmark{
		B0:   true,
		Str0: "8",
		Str3: "9",
	}
}

func (p *TestTypesForBenchmark) InitDefault() {
	p.B0 = true
	p.Str0 = "8"
	p.Str3 = "9"
}
