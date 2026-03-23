// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package assert provides minimal test assertion helpers.
// All functions stop the test immediately on failure (t.Fatalf).
//
// Design: keep this package small — avoid adding new functions unless
// there is a clear need that cannot be covered by the existing ones.
// Prefer: Equal for comparable types, BytesEqual for []byte,
// DeepEqual for structs/maps/slices, True for boolean conditions,
// and Nil for nil checks.
package assert

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// Equal asserts that two comparable values are equal using ==.
func Equal[T comparable](t testing.TB, expected, actual T, a ...any) {
	if expected != actual {
		t.Helper()
		t.Fatalf("expected %#v, got %#v%s", expected, actual, fmtInputs(a))
	}
}

// BytesEqual asserts that two byte slices are equal.
// On failure it reports the length mismatch or the first differing byte,
// avoiding logging raw byte slices which can be large and unreadable.
func BytesEqual(t testing.TB, expected, actual []byte, a ...any) {
	if bytes.Equal(expected, actual) {
		return
	}
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("bytes differ in length: expected %d, got %d%s", len(expected), len(actual), fmtInputs(a))
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("bytes differ at [%d]: expected 0x%02x, got 0x%02x%s", i, expected[i], actual[i], fmtInputs(a))
			return
		}
	}
}

// DeepEqual asserts that two values are deeply equal using reflect.DeepEqual.
// Use for structs, maps, and non-byte slices. Use BytesEqual for []byte.
func DeepEqual(t testing.TB, expected, actual any, a ...any) {
	if !reflect.DeepEqual(expected, actual) {
		t.Helper()
		t.Fatalf("expected %#v, got %#v%s", expected, actual, fmtInputs(a))
	}
}

// True asserts that cond is true.
func True(t testing.TB, cond bool, a ...any) {
	if !cond {
		t.Helper()
		t.Fatalf("expected true%s", fmtInputs(a))
	}
}

// Nil asserts that v is nil. Uses reflect to handle typed nils (e.g. (*T)(nil)).
func Nil(t testing.TB, v any, a ...any) {
	if !isNil(v) {
		t.Helper()
		t.Fatalf("expected nil, got: %v%s", v, fmtInputs(a))
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	case reflect.UnsafePointer:
		return rv.Pointer() == 0
	}
	return false
}

func fmtInputs(a []any) string {
	if len(a) == 0 {
		return ""
	}
	return ": " + fmt.Sprint(a...)
}
