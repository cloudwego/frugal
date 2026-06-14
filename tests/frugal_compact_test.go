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

package tests

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"

	"github.com/cloudwego/frugal"
	"github.com/davecgh/go-spew/spew"
)

func TestCompactTypeSerdes(t *testing.T) {
	f := fuzz.New()
	var ret int
	var err error
	s := &MyTypeTest{}
	f.Fuzz(s)
	got := make([]byte, frugal.EncodedSizeCompact(s))
	ret, err = frugal.EncodeObjectCompact(got, nil, s)
	require.NoError(t, err)
	require.Equal(t, len(got), ret)
	println("--------- COMPACT BYTES ---------")
	spew.Dump(got)
	_, vv := buildCompactTree(got)
	spew.Config.SortKeys = true
	println("--------- VALUE TREE ---------")
	spew.Dump(vv)
	println("--------- ORIGINAL VALUE ---------")
	spew.Dump(s)
	gotS := &MyTypeTest{}
	ret, err = frugal.DecodeObjectCompact(got, gotS)
	require.NoError(t, err)
	require.Equal(t, len(got), ret)
	println("--------- DECODED VALUE ---------")
	spew.Dump(gotS)
}

func TestCompactSignExtension(t *testing.T) {
	type compactSignExtTest struct {
		X MyNumberZ `frugal:"1,default,MyNumberZ"`
	}
	v := compactSignExtTest{X: MyNumberZ(-3948394)}
	n := frugal.EncodedSizeCompact(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObjectCompact(m, nil, &v)
	require.NoError(t, err)
	v = compactSignExtTest{}
	_, err = frugal.DecodeObjectCompact(m, &v)
	require.NoError(t, err)
	require.Equal(t, MyNumberZ(-3948394), v.X)
}

func TestCompactEnumKey(t *testing.T) {
	type compactEnumKeyTest struct {
		X map[MyNumberZ]int64 `frugal:"1,default,map<MyNumberZ:i64>"`
	}
	v := compactEnumKeyTest{X: map[MyNumberZ]int64{MyNumberZ(-3948394): -123}}
	n := frugal.EncodedSizeCompact(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObjectCompact(m, nil, &v)
	require.NoError(t, err)
	v = compactEnumKeyTest{}
	_, err = frugal.DecodeObjectCompact(m, &v)
	require.NoError(t, err)
	require.Equal(t, map[MyNumberZ]int64{MyNumberZ(-3948394): -123}, v.X)
}

func TestCompactListOfEnum(t *testing.T) {
	type compactListOfEnumTest struct {
		X []MyNumberZ `frugal:"1,default,list<MyNumberZ>"`
	}
	v := compactListOfEnumTest{X: []MyNumberZ{-3948394, 0, 1, 2, 3, 4, 5}}
	n := frugal.EncodedSizeCompact(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObjectCompact(m, nil, &v)
	require.NoError(t, err)
	v = compactListOfEnumTest{}
	_, err = frugal.DecodeObjectCompact(m, &v)
	require.NoError(t, err)
	require.Equal(t, []MyNumberZ{-3948394, 0, 1, 2, 3, 4, 5}, v.X)
}

func TestCompactFakerType(t *testing.T) {
	type compactFakerTestType struct {
		Enum1 *MyNumberZ `frugal:"1,optional,MyNumberZ"`
	}
	nz := MyNumberZ(42)
	v := compactFakerTestType{Enum1: &nz}
	n := frugal.EncodedSizeCompact(v)
	m := make([]byte, n)
	_, err := frugal.EncodeObjectCompact(m, nil, &v)
	require.NoError(t, err)
	v2 := compactFakerTestType{}
	_, err = frugal.DecodeObjectCompact(m, &v2)
	require.NoError(t, err)
	require.NotNil(t, v2.Enum1)
	require.Equal(t, MyNumberZ(42), *v2.Enum1)
}
