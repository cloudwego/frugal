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

package assert

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

// mockT captures Fatalf calls without stopping execution.
type mockT struct {
	testing.TB // embed to satisfy interface; only Helper/Fatalf are called
	msg        string
}

func (m *mockT) Helper()                   {}
func (m *mockT) Fatalf(f string, a ...any) { m.msg = fmt.Sprintf(f, a...) }
func (m *mockT) failed() bool              { return m.msg != "" }

func TestEqual(t *testing.T) {
	m := &mockT{}
	Equal(m, 1, 1)
	True(t, !m.failed())

	Equal(m, 1, 2)
	True(t, m.failed())
}

func TestBytesEqual(t *testing.T) {
	m := &mockT{}

	BytesEqual(m, []byte("abc"), []byte("abc"))
	True(t, !m.failed())

	// length diff
	BytesEqual(m, []byte("ab"), []byte("abc"))
	True(t, strings.Contains(m.msg, "length"))

	// content diff
	m.msg = ""
	BytesEqual(m, []byte{0x01, 0x02}, []byte{0x01, 0x03})
	True(t, strings.Contains(m.msg, "[1]"))
	True(t, strings.Contains(m.msg, "0x02"))
	True(t, strings.Contains(m.msg, "0x03"))
}

func TestDeepEqual(t *testing.T) {
	m := &mockT{}

	DeepEqual(m, []int{1, 2}, []int{1, 2})
	True(t, !m.failed())

	DeepEqual(m, []int{1, 2}, []int{1, 3})
	True(t, m.failed())
}

func TestTrue(t *testing.T) {
	m := &mockT{}

	True(m, true)
	True(t, !m.failed())

	True(m, false)
	True(t, m.failed())
}

func TestNil(t *testing.T) {
	m := &mockT{}

	// untyped nil
	Nil(m, nil)
	True(t, !m.failed())

	// typed nil pointer
	var p *int
	Nil(m, p)
	True(t, !m.failed())

	// typed nil slice
	var s []byte
	Nil(m, s)
	True(t, !m.failed())

	// typed nil map
	var mp map[string]int
	Nil(m, mp)
	True(t, !m.failed())

	// unsafe.Pointer nil
	Nil(m, unsafe.Pointer(nil))
	True(t, !m.failed())

	// non-nil value should fail
	x := 42
	Nil(m, &x)
	True(t, m.failed())
}

func TestFmtInputs(t *testing.T) {
	m := &mockT{}

	Equal(m, 1, 2, "context", "info")
	True(t, strings.Contains(m.msg, "context"))
}
