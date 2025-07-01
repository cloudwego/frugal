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
	"github.com/cloudwego/frugal/internal/opts"
)

// Option is the property setter function for opts.Options.
type Option func(*opts.Options)

// NoJIT ...
//
// Deprecated: JIT is deprecated
func NoJIT(v bool) {}

// WithMaxInlineDepth ...
//
// Deprecated: JIT is deprecated
func WithMaxInlineDepth(depth int) Option {
	return func(o *opts.Options) {}
}

// WithMaxInlineILSize ...
//
// Deprecated: JIT is deprecated
func WithMaxInlineILSize(size int) Option {
	return func(o *opts.Options) {}
}

// WithMaxPretouchDepth ...
//
// Deprecated: JIT is deprecated
func WithMaxPretouchDepth(depth int) Option {
	return func(o *opts.Options) {}
}

// SetMaxInlineDepth ...
//
// Deprecated: JIT is deprecated
func SetMaxInlineDepth(depth int) int {
	return depth
}

// SetMaxInlineILSize ...
//
// Deprecated: JIT is deprecated
func SetMaxInlineILSize(size int) int {
	return size
}
