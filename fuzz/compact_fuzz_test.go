// Copyright 2022 CloudWeGo Authors
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

package fuzz

import (
	"reflect"
	"testing"

	"github.com/cloudwego/frugal"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

func FuzzCompactSimple(f *testing.F) {
	type tr struct {
		B   bool   `frugal:"1,required"`
		I32 int32  `frugal:"2,required"`
		S   string `frugal:"3,required"`
	}
	seeds := []tr{
		{true, 1, "hello"},
		{false, -1, ""},
		{true, 2147483647, "world"},
	}
	for _, s := range seeds {
		n := frugal.EncodedSizeCompact(&s)
		b := make([]byte, n)
		frugal.EncodeObjectCompact(b, nil, &s)
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		p := &tr{}
		n, _ := frugal.DecodeObjectCompact(data, p)
		_ = n
		_ = p.B
		_ = p.I32
		_ = p.S
	})
}

func FuzzCompact(f *testing.F) {
	ct := &CompilerTest{
		I: &CompilerTestSubStruct{Y: &CompilerTestSubStruct{}},
	}
	n := frugal.EncodedSizeCompact(ct)
	buf := make([]byte, n)
	_, err := frugal.EncodeObjectCompact(buf, nil, ct)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(buf)

	f.Fuzz(func(t *testing.T, data []byte) {
		for i := thrift.BOOL; i < thrift.UTF16; i++ {
			typ, length, err := Check(data, thrift.TType(i))
			if err != nil {
				continue
			}
			if length != len(data) {
				continue
			}
			rt, err := fuzzDynamicStruct(typ)
			if err != nil {
				t.Fatal(err)
			}
			object := reflect.New(rt).Interface()

			_, err = frugal.DecodeObjectCompact(data, object)
			if err != nil {
				continue
			}

			n := frugal.EncodedSizeCompact(object)
			cbuf := make([]byte, n)
			_, err = frugal.EncodeObjectCompact(cbuf, nil, object)
			if err != nil {
				PrintStructTag(rt)
				t.Fatal(err)
			}

			object2 := reflect.New(rt).Interface()
			_, err = frugal.DecodeObjectCompact(cbuf, object2)
			if err != nil {
				PrintStructTag(rt)
				t.Fatal(err)
			}

			if !reflect.DeepEqual(object, object2) {
				t.Errorf("Compact round-trip mismatch for type %d", i)
			}
		}
	})
}
