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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendStruct(t *testing.T) {
	type EnumType int64
	type TestStruct struct {
		F11 *int8 `frugal:"11,optional,i8"`
		F12 *bool `frugal:"12,optional,bool"`

		F2 *int16    `frugal:"2,optional,i16"`
		F3 *int32    `frugal:"3,optional,i32"`
		F4 *int64    `frugal:"4,optional,i64"`
		F5 *EnumType `frugal:"5,optional,EnumType"`
		F6 *string   `frugal:"6,optional,string"`
	}

	p0 := &TestStruct{
		F11: P(int8(1)),
		F12: P(false),
		F2:  P(int16(2)),
		F3:  P(int32(3)),
		F4:  P(int64(4)),
		F5:  P(EnumType(5)),
		F6:  P("6"),
	}

	b, err := Append(nil, p0)
	require.NoError(t, err)
	p1 := &TestStruct{}
	_, err = Decode(b, p1)
	require.NoError(t, err)
	require.Equal(t, p0, p1)
}
