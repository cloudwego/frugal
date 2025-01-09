//go:build frugal_jit

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

	"github.com/cloudwego/frugal/internal/jit"
	"github.com/cloudwego/frugal/internal/jit/decoder"
	"github.com/cloudwego/frugal/internal/jit/encoder"
	"github.com/cloudwego/frugal/internal/opts"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

const nojit = false

func jitEncodedSize(val interface{}) int {
	return encoder.EncodedSize(val)
}

func jitEncodeObject(buf []byte, w thrift.NocopyWriter, val interface{}) (int, error) {
	return encoder.EncodeObject(buf, w, val)
}

func jitDecodeObject(buf []byte, val interface{}) (int, error) {
	return decoder.DecodeObject(buf, val)
}

func Pretouch(vt reflect.Type, options ...Option) error {
	o := opts.GetDefaultOptions()
	for _, fn := range options {
		fn(&o)
	}
	return jit.Pretouch(vt, o)
}
