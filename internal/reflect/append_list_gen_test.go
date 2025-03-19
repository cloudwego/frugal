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

package reflect

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"reflect"
	"testing"
	"unsafe"

	"github.com/cloudwego/gopkg/protocol/thrift"
	"github.com/stretchr/testify/assert"
)

const appendListFileName = "append_list_gen.go"

func TestGenAppendListCode(t *testing.T) {
	if *gencode {
		genAppendListCode(t, appendListFileName)
		return
	}

	type EnumType int64

	type Msg struct {
		X int64 `frugal:"1,default,i64"`
		Y int64 `frugal:"2,default,i64"`
	}

	type TestStruct struct {
		L1 []int8     `frugal:"1,optional,list<i8>"`
		L2 []int16    `frugal:"2,optional,list<i16>"`
		L3 []int32    `frugal:"3,optional,list<i32>"`
		L4 []int64    `frugal:"4,optional,list<i64>"`
		L5 []EnumType `frugal:"5,optional,list<EnumType>"`
		L6 []string   `frugal:"6,optional,list<string>"`
		L7 []*Msg     `frugal:"7,optional,list<Msg>"`
	}

	p := &TestStruct{}
	desc, err := getOrcreateStructDesc(reflect.ValueOf(p))
	assert.NoError(t, err)

	x := thrift.BinaryProtocol{}

	// fix compile err when generating code
	appendList_tI08 := listAppendFuncs[tI08]
	appendList_tI16 := listAppendFuncs[tI16]
	appendList_tI32 := listAppendFuncs[tI32]
	appendList_tI64 := listAppendFuncs[tI64]
	appendList_tENUM := listAppendFuncs[tENUM]
	appendList_tSTRING := listAppendFuncs[tSTRING]
	appendList_tOTHER := listAppendFuncs[tSTRUCT]

	{ // appendList_tI08

		f := desc.GetField(1)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.I08, len(p.L1))
			for _, v := range p.L1 {
				xb = x.AppendByte(xb, v)
			}
			return xb
		}
		p.L1 = []int8{-1, 0, 1}
		b, err := appendList_tI08(f.Type, nil, unsafe.Pointer(&p.L1))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L1 = nil
		b, err = appendList_tI08(f.Type, nil, unsafe.Pointer(&p.L1))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}

	{ // appendList_tI16
		f := desc.GetField(2)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.I16, len(p.L2))
			for _, v := range p.L2 {
				xb = x.AppendI16(xb, v)
			}
			return xb
		}

		p.L2 = []int16{-1, 0, 1}
		b, err := appendList_tI16(f.Type, nil, unsafe.Pointer(&p.L2))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L2 = nil
		b, err = appendList_tI16(f.Type, nil, unsafe.Pointer(&p.L2))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}

	{ // appendList_tI32

		f := desc.GetField(3)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.I32, len(p.L3))
			for _, v := range p.L3 {
				xb = x.AppendI32(xb, v)
			}
			return xb
		}

		p.L3 = []int32{-1, 0, 1}
		b, err := appendList_tI32(f.Type, nil, unsafe.Pointer(&p.L3))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L3 = nil
		b, err = appendList_tI32(f.Type, nil, unsafe.Pointer(&p.L3))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}

	{ // appendList_tI64

		f := desc.GetField(4)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.I64, len(p.L4))
			for _, v := range p.L4 {
				xb = x.AppendI64(xb, v)
			}
			return xb
		}

		p.L4 = []int64{-1, 0, 1}
		b, err := appendList_tI64(f.Type, nil, unsafe.Pointer(&p.L4))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L4 = nil
		b, err = appendList_tI64(f.Type, nil, unsafe.Pointer(&p.L4))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}

	{ // appendList_tENUM

		f := desc.GetField(5)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.I32, len(p.L5))
			for _, v := range p.L5 {
				xb = x.AppendI32(xb, int32(v))
			}
			return xb
		}

		p.L5 = []EnumType{-1, 0, 1}
		b, err := appendList_tENUM(f.Type, nil, unsafe.Pointer(&p.L5))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L5 = nil
		b, err = appendList_tENUM(f.Type, nil, unsafe.Pointer(&p.L5))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}

	{ // appendList_tSTRING
		f := desc.GetField(6)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.STRING, len(p.L6))
			for _, v := range p.L6 {
				xb = x.AppendString(xb, v)
			}
			return xb
		}
		p.L6 = []string{"", "1", "hello"}
		b, err := appendList_tSTRING(f.Type, nil, unsafe.Pointer(&p.L6))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L6 = nil
		b, err = appendList_tSTRING(f.Type, nil, unsafe.Pointer(&p.L6))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}

	{ // appendList_tOTHER
		f := desc.GetField(7)
		expectfunc := func() []byte {
			xb := x.AppendListBegin(nil, thrift.STRUCT, len(p.L7))
			for _, v := range p.L7 {
				xb = x.AppendFieldBegin(xb, thrift.I64, 1)
				xb = x.AppendI64(xb, v.X)
				xb = x.AppendFieldBegin(xb, thrift.I64, 2)
				xb = x.AppendI64(xb, v.Y)
				xb = x.AppendFieldStop(xb)
			}
			return xb
		}
		p.L7 = []*Msg{{X: -1, Y: -1}, {X: 0, Y: 0}, {X: 1, Y: 1}}
		b, err := appendList_tOTHER(f.Type, nil, unsafe.Pointer(&p.L7))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)

		p.L7 = nil
		b, err = appendList_tOTHER(f.Type, nil, unsafe.Pointer(&p.L7))
		assert.NoError(t, err)
		assert.Equal(t, expectfunc(), b)
	}
}

func genAppendListCode(t *testing.T, filename string) {
	f := &bytes.Buffer{}
	f.WriteString(appendListGenFileHeader)
	fm := func(format string, args ...any) {
		fmt.Fprintf(f, format, args...)
		fmt.Fprintln(f)
	}

	var listAppendElementCode = map[ttype]func(){
		tBYTE:   func() { fm("b = append(b, *(*byte)(vp))") },
		tI16:    func() { fm("b = appendUint16(b, *(*uint16)(vp))") },
		tI32:    func() { fm("b = appendUint32(b, *(*uint32)(vp))") },
		tI64:    func() { fm("b = appendUint64(b, *(*uint64)(vp))") },
		tDOUBLE: func() { fm("b = appendUint64(b, *(*uint64)(vp))") },
		tENUM:   func() { fm("b = appendUint32(b, uint32(*(*uint64)(vp)))") },
		tSTRING: func() {
			fm("s := *(*string)(vp)")
			fm("b = appendUint32(b, uint32(len(s)))")
			fm("b = append(b, s...)")
		},
		// tSTRUCT, tMAP, tSET, tLIST -> tOTHER
		tOTHER: func() {
			fm("var err error")
			fm("if t.IsPointer {")
			{
				fm("b, err = t.AppendFunc(t, b, *(*unsafe.Pointer)(vp))")
			}
			fm("} else {")
			{
				fm("b, err = t.AppendFunc(t, b, vp)")
			}
			fm("}")
			fm("if err != nil { return b, err }")
		},
	}

	// func init
	fm("func init() {")
	supportTypes := []ttype{
		tBYTE, tI16, tI32, tI64, tDOUBLE,
		tENUM, tSTRING, tSTRUCT, tMAP, tSET, tLIST,
	}
	for _, v := range supportTypes {
		fm("registerListAppendFunc(%s, %s)", t2s[v], appendListFuncName(v))
	}
	fm("}")

	// func appendList_XXX
	for _, v := range []ttype{tBYTE, tI16, tI32, tI64, tENUM, tSTRING, tOTHER} {
		fm("\nfunc %s(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {",
			appendListFuncName(v))
		fm("t = t.V")
		fm("b, n, vp := appendListHeader(t, b, p)")
		fmt.Fprintln(f, "if n == 0 { return b, nil }")

		fm("for i := uint32(0); i < n; i++ {")
		{
			fm("if i != 0 { vp = unsafe.Add(vp, t.Size) }")
			listAppendElementCode[v]()
		}
		fm("}")
		fm("return b, nil")
		fm("}")
	}

	fileb, err := format.Source(f.Bytes())
	if err != nil {
		t.Log(codeWithLine(f.Bytes()))
		t.Fatal(err)
	}
	err = os.WriteFile(filename, fileb, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("generated: %s", filename)
}

func appendListFuncName(t ttype) string {
	switch t {
	case tSTRUCT, tMAP, tSET, tLIST:
		t = tOTHER
	case tBOOL:
		t = tBYTE
	case tDOUBLE:
		t = tI64
	}
	return "appendList_" + ttype2str(t)
}

const appendListGenFileHeader = `/*
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

package reflect

import "unsafe"

// This File is generated by append_gen.sh. DO NOT EDIT.
// Template and code can be found in append_list_gen_test.go.

`
