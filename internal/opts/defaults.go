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

package opts

import (
	"os"
	"strconv"
)

const (
	_DefaultMaxInlineDepth  = 2     // cutoff at 2 levels of inlining
	_DefaultMaxInlineILSize = 50000 // cutoff at 50k of IL instructions
)

var (
	MaxInlineDepth  = parseOrDefault("FRUGAL_MAX_INLINE_DEPTH", _DefaultMaxInlineDepth, 1)
	MaxInlineILSize = parseOrDefault("FRUGAL_MAX_INLINE_IL_SIZE", _DefaultMaxInlineILSize, 256)
)

func parseOrDefault(key string, def int, min int) int {
	if env := os.Getenv(key); env == "" {
		return def
	} else if val, err := strconv.ParseUint(env, 0, 64); err != nil {
		panic("frugal: invalid value for " + key)
	} else if ret := int(val); ret <= min {
		panic("frugal: value too small for " + key)
	} else {
		return ret
	}
}
