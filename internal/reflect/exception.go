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
	"fmt"
)

var (
	errDepthLimitExceeded = &tProtocolException{t: thrift_DEPTH_LIMIT, m: "depth limit exceeded"}
	errInvalidData        = &tProtocolException{t: thrift_INVALID_DATA, m: "invalid data"}
)

// tProtocolException implements TProtocolException of apache thrift
type tProtocolException struct {
	t int
	m string
}

// consts from in github.com/apache/thrift@v0.13.0/lib/go/thrift
const (
	thrift_UNKNOWN_PROTOCOL_EXCEPTION = 0
	thrift_INVALID_DATA               = 1
	thrift_NEGATIVE_SIZE              = 2
	thrift_SIZE_LIMIT                 = 3
	thrift_BAD_VERSION                = 4
	thrift_NOT_IMPLEMENTED            = 5
	thrift_DEPTH_LIMIT                = 6
)

// TypeId implements apache thrift TProtocolException
func (t *tProtocolException) TypeId() int { return thrift_INVALID_DATA }

// TypeID implements kitex TypeID interface
func (t *tProtocolException) TypeID() int { return thrift_INVALID_DATA }

func (e *tProtocolException) String() string { return e.m }
func (e *tProtocolException) Error() string  { return e.m }

func newRequiredFieldNotSetException(name string) error {
	return &tProtocolException{
		t: thrift_INVALID_DATA,
		m: fmt.Sprintf("required field %q is not set", name),
	}
}

func newUnknownDataTypeException(t ttype) error {
	return &tProtocolException{
		t: thrift_INVALID_DATA,
		m: fmt.Sprintf("unknown data type %d", t),
	}
}
