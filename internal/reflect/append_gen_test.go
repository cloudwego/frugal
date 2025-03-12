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
