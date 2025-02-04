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

package frugal

import (
	"fmt"

	"github.com/cloudwego/frugal/internal/opts"
)

// Option is the property setter function for opts.Options.
type Option func(*opts.Options)

// NoJIT disables JIT encoder and decoder explicitly.
//
// This function will be deprecated along with the JIT implementation in the future.
//
// Deprecated: JIT is disabled by default.
func NoJIT(v bool) {
	println(`
frugal.NoJIT is deprecated. JIT is disabled by default since v0.2.4.
If you still want to use JIT, please specify frugal_jit tag for go build to enable it.

Please do note that:
The JIT implementation is no longer maintained,
DO NOT submit any issues or pull requests.
`)
}

const (
	_MinILSize = 1024
)

// WithMaxInlineDepth sets the maximum inlining depth for the JIT compiler.
//
// Increasing of this option makes the compiler inline more aggressively, which
// gives better runtime performance at the cost of a longer compilation time,
// and vice versa.
//
// Set this option to "0" disables this limit, which means inlining everything.
//
// The default value of this option is "2".
//
// Deprecated: only for JIT
func WithMaxInlineDepth(depth int) Option {
	if depth < 0 {
		panic(fmt.Sprintf("frugal: invalid inline depth: %d", depth))
	} else {
		return func(o *opts.Options) { o.MaxInlineDepth = depth }
	}
}

// WithMaxInlineILSize sets the maximum IL instruction count before not inlining.
//
// Increasing of this option makes the compiler inline more aggressively, which
// lead to more memory consumptions but better runtime performance, and vice
// versa.
//
// Set this option to "0" disables this limit, which means unlimited inlining
// IL buffer.
//
// The default value of this option is "50000".
//
// Deprecated: only for JIT
func WithMaxInlineILSize(size int) Option {
	if size != 0 && size < _MinILSize {
		panic(fmt.Sprintf("frugal: invalid inline IL size: %d", size))
	} else {
		return func(o *opts.Options) { o.MaxInlineILSize = size }
	}
}

// WithMaxPretouchDepth controls how deep the compiler goes to compile
// indirectly referenced types.
//
// Larger depth means more types will be pre-compiled when pretouching,
// which lead to longer compilation time, but lower runtime JIT latency,
// and vice versa. You might want to tune this value to strike a balance
// between compilation time and runtime performance.
//
// The default value "0" means unlimited, which basically turns Frugal into
// an AOT compiler.
//
// This option is only available when performing pretouch, otherwise it is
// ignored and do not have any effect.
//
// Deprecated: only for JIT
func WithMaxPretouchDepth(depth int) Option {
	if depth < 0 {
		panic(fmt.Sprintf("frugal: invalid pretouch depth: %d", depth))
	} else {
		return func(o *opts.Options) { o.MaxPretouchDepth = depth }
	}
}

// SetMaxInlineDepth sets the default maximum inlining depth for all types from
// now on.
//
// This value can also be configured with the `FRUGAL_MAX_INLINE_DEPTH`
// environment variable.
//
// The default value of this option is "2".
//
// Returns the old opts.MaxInlineDepth value.
//
// Deprecated: only for JIT
func SetMaxInlineDepth(depth int) int {
	depth, opts.MaxInlineDepth = opts.MaxInlineDepth, depth
	return depth
}

// SetMaxInlineILSize sets the default maximum inlining IL instructions for all
// types from now on.
//
// This value can also be configured with the `FRUGAL_MAX_INLINE_IL_SIZE`
// environment variable.
//
// The default value of this option is "50000".
//
// Returns the old opts.MaxInlineILSize value.
//
// Deprecated: only for JIT
func SetMaxInlineILSize(size int) int {
	size, opts.MaxInlineILSize = opts.MaxInlineILSize, size
	return size
}
