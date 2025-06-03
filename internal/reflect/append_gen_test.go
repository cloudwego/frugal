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
	"flag"
	"fmt"
	"strings"
)

var (
	gencode = flag.Bool("gencode", false, "generate list/map code for better performance")
)

var tOTHER = ttype(0xee) // must not in use, only for generating code

func init() {
	t2s[tOTHER] = "Other" // makes ttype2str work
}

func ttype2FuncType(t ttype) string {
	switch t {
	case tSTRUCT, tMAP, tSET, tLIST:
		t = tOTHER
	case tDOUBLE:
		t = tI64
	}
	return ttype2str(t)
}

var (
	defineErr = map[ttype]bool{tOTHER: true}
	defineStr = map[ttype]bool{tSTRING: true}
)

func getAppendCode(typ ttype, t, p string) string {
	t2c := map[ttype]string{
		tBYTE:   "if cap(b)-len(b) < 1 {\n\tb = gb.NewBuffer(b, 1)\n}\nb = append(b, *((*byte)({p})))",
		tI16:    "if cap(b)-len(b) < 2 {\n\tb = gb.NewBuffer(b, 2)\n}\nb = appendUint16(b, *((*uint16)({p})))",
		tI32:    "if cap(b)-len(b) < 4 {\n\tb = gb.NewBuffer(b, 4)\n}\nb = appendUint32(b, *((*uint32)({p})))",
		tI64:    "if cap(b)-len(b) < 8 {\n\tb = gb.NewBuffer(b, 8)\n}\nb = appendUint64(b, *((*uint64)({p})))",
		tDOUBLE: "if cap(b)-len(b) < 8 {\n\tb = gb.NewBuffer(b, 8)\n}\nb = appendUint64(b, *((*uint64)({p})))",
		tENUM:   "if cap(b)-len(b) < 8 {\n\tb = gb.NewBuffer(b, 8)\n}\nb = appendUint32(b, uint32(*((*int64)({p}))))",
		tSTRING: `s = *((*string)({p}))
				if len(s) < nocopyWriteThreshold {
					if cap(b)-len(b) < len(s)+4 {
						b = gb.NewBuffer(b, len(s)+4)
					}
					b = appendUint32(b, uint32(len(s)))
					b = append(b, s...)
				} else {
					if cap(b)-len(b) < 4 {
						b = gb.NewBuffer(b, 4)
					}
					b = appendUint32(b, uint32(len(s)))
					b = gb.WriteDirect(b, unsafex.StringToBinary(s))
				}`,

		// tSTRUCT, tMAP, tSET, tLIST -> tOTHER
		tOTHER: `if {t}.IsPointer {
		b, err = {t}.AppendFunc({t}, b, *(*unsafe.Pointer)({p}), gb)
	} else {
		b, err = {t}.AppendFunc({t}, b, {p}, gb)
	}
	if err != nil {
		return b, err
}`,
	}
	s, ok := t2c[typ]
	if !ok {
		panic("type doesn't have code: " + ttype2str(typ))
	}
	s = strings.ReplaceAll(s, "{t}", t)
	s = strings.ReplaceAll(s, "{p}", p)
	return s
}

func codeWithLine(b []byte) string {
	p := &strings.Builder{}
	p.Grow(len(b) + 5*bytes.Count(b, []byte("\n")))

	n := 1
	i := 0
	fmt.Fprintf(p, "%4d ", n)
	for j := 0; j < len(b); j++ {
		if b[j] == '\n' {
			p.Write(b[i : j+1])
			i = j + 1
			n++
			fmt.Fprintf(p, "%4d ", n)
		}
	}
	if i < len(b) {
		p.Write(b[i:])
	}
	return p.String()
}
