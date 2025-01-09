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

	"github.com/cloudwego/frugal/internal/reflect"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

// EncodedSize measures the encoded size of val.
func EncodedSize(val interface{}) int {
	if nojit {
		return reflect.EncodedSize(val)
	}
	return jitEncodedSize(val)
}

// EncodeObject serializes val into buf with Thrift Binary Protocol, with optional Zero-Copy thrift.NocopyWriter.
// buf must be large enough to contain the entire serialization result.
func EncodeObject(buf []byte, w thrift.NocopyWriter, val interface{}) (int, error) {
	if nojit {
		ret, err := reflect.Append(buf[:0], val)
		if len(ret) > len(buf) {
			return 0, fmt.Errorf("index out of range [%d] with length %d.\n"+
				"Please make sure the input will not be changed after calling EncodedSize or during EncodeObject(concurrency issues).",
				len(ret), len(buf))
		}
		return len(ret), err
	}
	return jitEncodeObject(buf, w, val)
}

// DecodeObject deserializes buf into val with Thrift Binary Protocol.
func DecodeObject(buf []byte, val interface{}) (int, error) {
	if nojit {
		return reflect.Decode(buf, val)
	}
	return jitDecodeObject(buf, val)
}
