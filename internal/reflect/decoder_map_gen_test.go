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
	"github.com/stretchr/testify/require"
)

const decoderMapFileName = "decoder_map_gen.go"

func TestGenDecoderMapCode(t *testing.T) {
	if *gencode {
		genDecoderMapCode(t, decoderMapFileName)
		return
	}

	x := thrift.BinaryProtocol{}

	// testdata
	const dataLen = 3
	bools := []bool{true, false, true}
	i08s := []int8{-1, 0, 1<<7 - 1}
	i16s := []int16{-1, 0, 1<<15 - 1}
	i32s := []int32{-1, 0, 1<<31 - 1}
	i64s := []int64{-1, 0, 1<<63 - 1}
	enums := []int64{-1, 0, 1<<31 - 1}
	ss := []string{"", "h", "hello"}
	bb := [][]byte{{0, 0, 0}, {1, 1, 1}, {255, 255, 255}}

	testdata := map[ttype]any{
		tBOOL:   bools,
		tBYTE:   i08s,
		tI16:    i16s,
		tI32:    i32s,
		tI64:    i64s,
		tENUM:   enums,
		tSTRING: ss,
		tBINARY: bb,
	}

	appendBool := func(i int, b []byte) []byte {
		return x.AppendBool(b, bools[i])
	}
	appendI08 := func(i int, b []byte) []byte {
		return x.AppendByte(b, i08s[i])
	}
	appendI16 := func(i int, b []byte) []byte {
		return x.AppendI16(b, i16s[i])
	}
	appendI32 := func(i int, b []byte) []byte {
		return x.AppendI32(b, i32s[i])
	}
	appendI64 := func(i int, b []byte) []byte {
		return x.AppendI64(b, i64s[i])
	}
	appendEnum := func(i int, b []byte) []byte {
		return x.AppendI32(b, int32(enums[i]))
	}
	appendStr := func(i int, b []byte) []byte {
		return x.AppendString(b, ss[i])
	}
	appendBytes := func(i int, b []byte) []byte {
		return x.AppendBinary(b, bb[i])
	}

	appendfuncs := map[ttype]func(i int, b []byte) []byte{
		tBOOL:   appendBool,
		tBYTE:   appendI08,
		tI16:    appendI16,
		tI32:    appendI32,
		tI64:    appendI64,
		tENUM:   appendEnum,
		tSTRING: appendStr,
		tBINARY: appendBytes,
	}

	testKeyTypes := []ttype{
		tBOOL, tBYTE, tI16, tI32, tI64, tENUM, tSTRING,
	}
	testVauleTypes := []ttype{
		tBOOL, tBYTE, tI16, tI32, tI64, tENUM, tSTRING, tBINARY,
	}

	d := decoderPool.Get().(*tDecoder)
	defer decoderPool.Put(d)

	for _, k := range testKeyTypes {
		for _, v := range testVauleTypes {
			typ := &tType{
				T: tMAP,
				K: &tType{T: k, WT: ttype2wiretype(k)},
				V: &tType{T: v, WT: ttype2wiretype(v)},
			}

			// input bytes
			b := make([]byte, 0, 1024)
			b = x.AppendMapBegin(b, thrift.TType(typ.K.WT), thrift.TType(typ.V.WT), dataLen)
			for i := 0; i < dataLen; i++ {
				b = appendfuncs[k](i, b)
				b = appendfuncs[v](i, b)
			}

			// decode
			var p unsafe.Pointer
			n, err := mapDecodeFuncs[decodeFuncKey{k, v}](d, typ, b, unsafe.Pointer(&p), 0)
			require.NoError(t, err)
			require.Equal(t, len(b), n)

			expect := slice2map(testdata[k], testdata[v])
			got := rvWithPtr(reflect.MakeMap(expect.Type()), p)
			require.Equal(t, expect.Interface(), got.Interface())
		}
	}
}

func slice2map(keys, values any) reflect.Value {
	kk := reflect.ValueOf(keys)
	vv := reflect.ValueOf(values)
	if kk.Kind() != reflect.Slice || vv.Kind() != reflect.Slice {
		panic("not slice")
	}
	if kk.Len() != vv.Len() {
		panic("len not equal")
	}
	mt := reflect.MapOf(kk.Type().Elem(), vv.Type().Elem())
	mv := reflect.MakeMapWithSize(mt, kk.Len())
	for i := 0; i < kk.Len(); i++ {
		mv.SetMapIndex(kk.Index(i), vv.Index(i))
	}
	return mv
}

func genDecoderMapCode(t *testing.T, filename string) {
	f := &bytes.Buffer{}
	f.WriteString(decoderMapGenFileHeader)
	fm := func(format string, args ...any) {
		fmt.Fprintf(f, format, args...)
		fmt.Fprintln(f)
	}

	var2type := func(p string) string {
		if p == "k" {
			return "t.K"
		}
		return "t.V"
	}

	var decodeCodeSnippets = map[ttype]func(p string){
		tBOOL: func(p string) { fm("%s := b[i] != 0", p); fm("i += 1") },
		tBYTE: func(p string) { fm("%s := b[i]", p); fm("i += 1") },
		tI16:  func(p string) { fm("%s := decodeU16(b[i:])", p); fm("i += 2") },
		tI32:  func(p string) { fm("%s := decodeU32(b[i:])", p); fm("i += 4") },
		tI64:  func(p string) { fm("%s := decodeU64(b[i:])", p); fm("i += 8") },
		tENUM: func(p string) { fm("%s := decodeEnum(b[i:])", p); fm("i += 4") },
		tSTRING: func(p string) {
			fm("var %s string", p)
			fm("n, err = _decoderString(d, %s, b[i:], unsafe.Pointer(&%s), false)", var2type(p), p)
			fm("if err != nil { return i, err }")
			fm("i += n")
		},
		tBINARY: func(p string) {
			fm("var %s []byte", p)
			fm("n, err = _decoderString(d, %s, b[i:], unsafe.Pointer(&%s), false)", var2type(p), p)
			fm("if err != nil { return i, err }")
			fm("i += n")
		},
	}

	defineTmpVars := func(t ttype) bool {
		return t == tSTRING || t == tBINARY
	}

	supportedKeyTypes := []ttype{
		tBOOL, tBYTE, tI16, tI32, tI64, tDOUBLE, tENUM, tSTRING,
	}
	supportedVauleTypes := []ttype{
		tBOOL, tBYTE, tI16, tI32, tI64, tDOUBLE, tENUM, tSTRING, tBINARY,
	}
	t2go := map[ttype]string{
		tBOOL: "bool", tBYTE: "byte",
		tI16: "uint16", tI32: "uint32", tI64: "uint64", tDOUBLE: "uint64",
		tENUM: "int64", tSTRING: "string", tBINARY: "[]byte",
	}
	// func init
	fm("func init() {")
	for _, k := range supportedKeyTypes {
		for _, v := range supportedVauleTypes {
			fm("registerMapDecodeFunc(%s, %s, %s)", t2s[k], t2s[v], mapDecodeFuncName(k, v))
		}
	}
	fm("}")

	for _, k := range supportedKeyTypes {
		for _, v := range supportedVauleTypes {
			if k == tDOUBLE || v == tDOUBLE {
				continue // reuse tI64
			}
			maptype := "map[" + t2go[k] + "]" + t2go[v]
			fm("\nfunc %s(d *tDecoder, t *tType, b []byte, p unsafe.Pointer, maxdepth int) (int, error) {", mapDecodeFuncName(k, v))
			{
				fm("l, err := decodeMapHeader(t, b)")
				fm("if err != nil { return 0, err }")
				if defineTmpVars(k) || defineTmpVars(v) {
					fm("var n int")
				}
				fm("i, m := 6, make(%s, l)", maptype)
				fm("for j := 0; j < l; j++ {")
				{
					decodeCodeSnippets[k]("k")
					decodeCodeSnippets[v]("v")
					fm("m[k] = v")
				}
				fm("}")
				fm("*(*%s)(p) = m", maptype)
				fm("return i, nil")
			}
			fm("}")
		}
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

func mapDecodeFuncName(k, v ttype) string {
	if k == tDOUBLE {
		k = tI64
	}
	if v == tDOUBLE {
		v = tI64
	}
	return fmt.Sprintf("decodeMap_%s_%s", ttype2str(k), ttype2str(v))
}

const decoderMapGenFileHeader = `/*
 * Copyright 2025 CloudWeGo Authors
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
// Template and code can be found in decoder_map_gen_test.go.

`
