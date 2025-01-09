//go:build !frugal_jit

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

package frugal

import (
	"reflect"

	"github.com/cloudwego/gopkg/protocol/thrift"
)

const nojit = true

func jitEncodedSize(val interface{}) int {
	panic("not supported")
}

func jitEncodeObject(buf []byte, w thrift.NocopyWriter, val interface{}) (int, error) {
	panic("not supported")
}

func jitDecodeObject(buf []byte, val interface{}) (int, error) {
	panic("not supported")
}

// Pretouch compiles vt ahead-of-time to avoid JIT compilation on-the-fly, in
// order to reduce the first-hit latency.
func Pretouch(vt reflect.Type, options ...Option) error {
	return nil // do not panic, legacy code may still use the func
}
