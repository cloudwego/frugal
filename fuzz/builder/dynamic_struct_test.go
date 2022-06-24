// Copyright 2022 ByteDance Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"reflect"
	"testing"

	"github.com/cloudwego/frugal"
	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/semantic"
	"github.com/simon0-o/gofakeit/v6"
)

var StructThrift = `
struct Bar {
	1: required list<Bar> a,
	2: optional list<Bar> b,
	3: required map<string,Bar> c,
	4: optional map<string,Bar> d,
	5: required set<Bar> e,
	6: optional set<Bar> f,
}

struct Foo {
	1: required Bar a,
	2: optional Bar b,
	3: required string c,
	4: optional string d,
	5: required binary e,
	6: optional binary f,
	7: required bool g,
	8: optional bool h,
	9: required double i,
	10: optional double j,
	11: required i64 k,
	12: optional i64 l,
	13: required i32 m,
	14: optional i32 n,
	15: required i16 o,
	16: optional i16 p,
	17: required i8 q,
	18: optional i8 r,
	19: required map<string,i64> s,
	20: optional map<string,i64> t,
	21: required set<string> u,
	22: optional set<string> v,
	23: required list<string> w,
	24: optional list<string> x,
}
`

func TestBuildStructFromThrift(t *testing.T) {
	tree, err := parser.ParseString("root.thrift", StructThrift)
	if err != nil {
		t.Fatal(err)
	}
	checker := semantic.NewChecker(semantic.Options{FixWarnings: true})
	_, err = checker.CheckAll(tree)
	if err != nil {
		t.Fatal(err)
	}
	err = semantic.ResolveSymbols(tree)
	if err != nil {
		t.Fatal(err)
	}
	builder := NewStructBuilder()
	for _, st := range tree.GetStructLikes() {
		ts, err := builder.buildStructFromAST(tree, st)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%#v", reflect.New(ts.Type).Interface())
	}
}

type MyTest struct {
	Field1 []int64          "frugal:\"1,required,list<i64>\""
	Field2 []int64          "frugal:\"2,optional,list<i64>\""
	Field3 map[string]int64 "frugal:\"3,required,map<string:i64>\""
	Field4 map[string]int64 "frugal:\"4,optional,map<string:i64>\""
	Field5 []int64          "frugal:\"5,required,set<i64>\""
	Field6 []int64          "frugal:\"6,optional,set<i64>\""
}

func TestMyTest(t *testing.T) {
	mt := &MyTest{}
	gofakeit.Struct(mt)
	t.Log(frugal.EncodedSize(mt))
}
