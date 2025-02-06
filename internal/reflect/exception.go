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

	"github.com/cloudwego/gopkg/protocol/thrift"
)

var (
	errDepthLimitExceeded = thrift.NewProtocolException(thrift.DEPTH_LIMIT, "depth limit exceeded")
	errNegativeSize       = thrift.NewProtocolException(thrift.NEGATIVE_SIZE, "negative size")
)

func newRequiredFieldNotSetException(name string) error {
	return thrift.NewProtocolException(
		thrift.INVALID_DATA,
		fmt.Sprintf("required field %q is not set", name),
	)
}

func newTypeMismatch(expect, got ttype) error {
	return thrift.NewProtocolException(
		thrift.INVALID_DATA,
		fmt.Sprintf("type mismatch. expect %s, got %s",
			ttype2str(expect), ttype2str(got)))
}

func newTypeMismatchKV(gotk, gotv, expectk, expectv ttype) error {
	return thrift.NewProtocolException(
		thrift.INVALID_DATA,
		fmt.Sprintf("type mismatch. got map[%s]%s, expect map[%s]%s",
			ttype2str(expectk), ttype2str(expectv), ttype2str(gotk), ttype2str(gotv)))
}
