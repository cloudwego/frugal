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

package defs

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

type NoCopyStringFields struct {
	NormalString         string `frugal:"1,default,string"`
	NoCopyString         string `frugal:"2,default,string,nocopy"`
	TypelessString       string `frugal:"3,default"`
	TypelessString2      string `frugal:"4,default,"`
	NoCopyTypelessString string `frugal:"5,default,,nocopy"`
}

func TestResolver_StringOptions(t *testing.T) {
	var vv NoCopyStringFields
	ret, err := ResolveFields(reflect.TypeOf(vv))
	require.NoError(t, err)
	spew.Config.SortKeys = true
	spew.Config.DisablePointerMethods = true
	spew.Dump(ret)
}
