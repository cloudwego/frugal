/*
 * Copyright 2026 CloudWeGo Authors
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

package reflect

import (
	"strings"
	"testing"

	"github.com/cloudwego/frugal/internal/assert"
)

func checkEncodingRoundTrip(t *testing.T, src, dst any) {
	t.Helper()
	b, err := Append(nil, src)
	assert.Nil(t, err)
	assert.Equal(t, len(b), EncodedSize(src))
	n, err := Decode(b, dst)
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)
	assert.DeepEqual(t, src, dst)
}

func TestEncodedSizeValueStruct(t *testing.T) {
	type Child struct {
		S string `frugal:"1,default,string"`
	}
	type Empty struct{}
	type S struct {
		C  Child            `frugal:"1,default,Child"`
		CL []Child          `frugal:"2,default,list<Child>"`
		CM map[string]Child `frugal:"3,default,map<string:Child>"`
		E  Empty            `frugal:"4,default,Empty"`
		EL []Empty          `frugal:"5,default,list<Empty>"`
		EM map[string]Empty `frugal:"6,default,map<string:Empty>"`
	}
	checkEncodingRoundTrip(t, &S{
		C:  Child{S: "field"},
		CL: []Child{{S: "first"}, {S: "second"}},
		CM: map[string]Child{"key": {S: "value"}},
		EL: []Empty{{}, {}},
		EM: map[string]Empty{"key": {}},
	}, &S{})
}

func TestRejectContainerPointerFields(t *testing.T) {
	for _, v := range []any{
		&struct {
			V *[]int32 `frugal:"1,optional,list<i32>"`
		}{},
		&struct {
			V *[]int32 `frugal:"1,optional,set<i32>"`
		}{},
		&struct {
			V *map[string]int32 `frugal:"1,optional,map<string:i32>"`
		}{},
	} {
		// run twice to make sure failed descriptors are not cached
		for i := 0; i < 2; i++ {
			_, err := Append(nil, v)
			assert.True(t, err != nil && strings.Contains(err.Error(), "pointers to lists, sets or maps are not supported"), err)
		}
	}
}

type cycleA struct {
	B *cycleB `frugal:"1,optional,cycleB"`
}

type cycleB struct {
	A   *cycleA   `frugal:"1,optional,cycleA"`
	Bad *cycleBad `frugal:"2,optional,cycleBad"`
}

type cycleBad struct {
	L *[]int32 `frugal:"1,optional,list<i32>"`
}

type cycleRoot struct {
	B *cycleB `frugal:"1,optional,cycleB"`
}

type cycleOther struct {
	A *cycleA `frugal:"1,optional,cycleA"`
}

func TestRejectContainerPointerInCycle(t *testing.T) {
	// cycleA is prefetched before cycleB fails; rollback must clear both tType.Sd and prefetchStructDescCache
	data := []byte{byte(tSTRUCT), 0, 1, byte(tSTRUCT), 0, 1, byte(tSTOP), byte(tSTOP), byte(tSTOP)}
	for i := 0; i < 2; i++ {
		for _, v := range []any{&cycleRoot{}, &cycleOther{}} {
			_, err := Decode(data, v)
			assert.True(t, err != nil && strings.Contains(err.Error(), "pointers to lists, sets or maps are not supported"), v, err)
		}
	}
}

func TestRejectNestedPointerField(t *testing.T) {
	type Child struct {
		P **int32 `frugal:"1,optional,i32"`
	}
	type Root struct {
		C *Child `frugal:"1,optional,Child"`
	}
	for i := 0; i < 2; i++ {
		_, err := Decode([]byte{byte(tSTOP)}, &Root{})
		assert.True(t, err != nil && strings.Contains(err.Error(), "nested pointer is not allowed"), err)
	}
}

func TestOptionalPointerFields(t *testing.T) {
	type Child struct {
		S string `frugal:"1,default,string"`
	}
	type Binary []byte
	type S struct {
		List   []int32          `frugal:"1,optional,list<i32>"`
		Set    []int32          `frugal:"2,optional,set<i32>"`
		Map    map[string]int32 `frugal:"3,optional,map<string:i32>"`
		Binary []byte           `frugal:"4,optional,binary"`
		Child  *Child           `frugal:"5,optional,Child"`
		Scalar *int32           `frugal:"6,optional,i32"`
		PtrBin *[]byte          `frugal:"7,optional,binary"`
		Named  *Binary          `frugal:"8,optional,binary,nocopy"`
	}
	scalar := int32(42)
	ptrBin, named, empty := []byte("pointer"), Binary("named"), []byte{}
	var got S
	checkEncodingRoundTrip(t, &S{
		List: []int32{1, 2}, Set: []int32{3, 4}, Map: map[string]int32{"key": 5},
		Binary: []byte("value"), Child: &Child{S: "child"}, Scalar: &scalar,
		PtrBin: &ptrBin, Named: &named,
	}, &got)
	// DeepEqual ignores slice cap; decoding as string would leave cap at 0
	assert.Equal(t, len(*got.PtrBin), cap(*got.PtrBin))
	assert.Equal(t, len(*got.Named), cap(*got.Named))
	checkEncodingRoundTrip(t, &S{PtrBin: &empty}, &S{})
	checkEncodingRoundTrip(t, &S{}, &S{})
}
