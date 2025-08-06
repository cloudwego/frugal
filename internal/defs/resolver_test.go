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

func TestLookupStructTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      reflect.StructTag
		expected []string
		ok       bool
	}{
		{
			name:     "frugal and thrift tag",
			tag:      `frugal:"1,required,string" thrift:"fieldName,2,required"`,
			expected: []string{"1", "required", "string"},
			ok:       true,
		},
		{
			name:     "frugal tag with spaces",
			tag:      `frugal:"1, required , string "`,
			expected: []string{"1", "required", "string"},
			ok:       true,
		},
		{
			name:     "thrift tag ignores field name",
			tag:      `thrift:"fieldName,1,required"`,
			expected: []string{"1", "required"},
			ok:       true,
		},
		{
			name:     "thrift tag with only field name and id",
			tag:      `thrift:"fieldName,1"`,
			expected: []string{"1"},
			ok:       true,
		},
		{
			name:     "no tag found",
			tag:      `json:"field_name"`,
			expected: nil,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := lookupStructTag(tt.tag)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.expected, result)
		})
	}
}

type ThriftTagFields struct {
	WithRequiredness    string `thrift:"field1,1,required"`
	WithoutRequiredness string `thrift:"field2,2"`
	OptionalField       string `thrift:"field3,3,optional"`
}

type DuplicateIDFields struct {
	Field1 string `frugal:"1,default"`
	Field2 string `frugal:"1,default"`
}

func TestResolveFields_ThriftTagRequiredness(t *testing.T) {
	var vv ThriftTagFields
	ret, err := ResolveFields(reflect.TypeOf(vv))
	require.NoError(t, err)
	require.Len(t, ret, 3)

	require.Equal(t, Required, ret[0].Spec)
	require.Equal(t, Default, ret[1].Spec)
	require.Equal(t, Optional, ret[2].Spec)
}

func TestResolveFields_DuplicateID(t *testing.T) {
	var vv DuplicateIDFields
	_, err := ResolveFields(reflect.TypeOf(vv))
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicated field ID 1")
}
