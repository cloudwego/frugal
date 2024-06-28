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
	"reflect"
	"sync"
	"unsafe"
)

var testhackOnce sync.Once

// this func should be called once to test compatibility with Go runtime
func testhack() {
	m := map[int]string{7: "hello"}
	rv := reflect.ValueOf(m)
	it := newMapIter(rv)
	kp, vp := it.Next()
	if *(*int)(kp) != 7 || *(*string)(vp) != "hello" {
		panic("compatibility issue found: mapIter")
	}

	m[8] = "world"
	m[9] = "!"
	m[10] = "?"
	if maplen(rvUnsafePointer(rv)) != 4 {
		panic("compatibility issue found: maplen")
	}

	rv1 := reflect.New(rv.Type()).Elem()
	// rv1 is indirect value, it doesn't like rv from eface.
	// so we need to get the addr of rvUnsafePointer
	tmpp := rvUnsafePointer(rv)
	rv1 = rvWithPtr(rv1, unsafe.Pointer(&tmpp))
	if p0, p1 := rvUnsafePointer(rv), rvUnsafePointer(rv1); p0 != p1 {
		panic(fmt.Sprintf("compatibility issue found: rvWithPtr %p -> %p", p0, p1))
	}
	m1, ok := rv1.Interface().(map[int]string)
	if !ok || !reflect.DeepEqual(m, m1) {
		panic("compatibility issue found: rvWithPtr (Interface())")
	}

	m2 := map[int]string{}
	m3 := map[int]*string{}
	m4 := map[int]*string{}

	if rvTypePtr(reflect.ValueOf(m1)) != rvTypePtr(reflect.ValueOf(m2)) ||
		rvTypePtr(reflect.ValueOf(m3)) != rvTypePtr(reflect.ValueOf(m4)) ||
		rvTypePtr(reflect.ValueOf(m2)) == rvTypePtr(reflect.ValueOf(m4)) {
		panic("compatibility issue found: rvTypePtr")
	}

	var f iFoo = &dog{"test"} // update itab
	if f.Foo() != "test" {
		panic("never goes here ...")
	}
	f1 := f // copy itab
	d := &dog{sound: "woof"}
	updateIface(unsafe.Pointer(&f1), unsafe.Pointer(d)) // update data pointer
	if f1.Foo() != d.sound {
		panic("compatibility issue found: updateIface")
	}
	if f.Foo() != "test" { // won't change, coz it's copied to f1
		panic("compatibility issue found: updateIface")
	}
}

// ↓↓↓ for testing updateIface

type iFoo interface{ Foo() string }

type dog struct{ sound string }

func (d *dog) Foo() string { return d.sound }

// ↑↑↑ for testing updateIface

type hitter struct {
	// k and v is always the 1st two fields of hitter
	// it will not be changed easily even though in the future
	k unsafe.Pointer
	v unsafe.Pointer
}

// see hack_go1.17.go, hack_go1.18.go
// type hackMapIter struct { ... }

// mapIter wraps reflect.MapIter for faster unsafe Next()
type mapIter struct {
	reflect.MapIter
}

// newMapIter creates reflect.MapIter for reflect.Value.
// for go1.17, go1.18, rv.MapRange() will cause one more allocation
// for >=go1.19, can use rv.MapRange() directly.
// see: https://github.com/golang/go/commit/c5edd5f616b4ee4bbaefdb1579c6078e7ed7e84e
// TODO: remove this func, and use mapIter{rv.MapRange()} when >=go1.19
func newMapIter(rv reflect.Value) mapIter {
	ret := mapIter{}
	(*hackMapIter)(unsafe.Pointer(&ret.MapIter)).m = rv
	return ret
}

func (m *mapIter) Next() (unsafe.Pointer, unsafe.Pointer) {
	p := (*hackMapIter)(unsafe.Pointer(&m.MapIter))
	if p.initialized() {
		return p.Next()
	}
	// use reflect.Next to initialize hitter
	// then we no need to bind mapiterinit
	m.MapIter.Next()
	return p.hitter.k, p.hitter.v
}

func maplen(p unsafe.Pointer) int {
	// XXX: race detector not working with this func
	type hmap struct {
		count int // count is the 1st field
	}
	return (*hmap)(p).count
}

type rvtype struct { // reflect.Value
	abiType uintptr
	ptr     unsafe.Pointer // data pointer
}

// rvWithPtr returns reflect.Value with the unsafe.Pointer.
// Same reflect.NewAt().Elem() without the cost of getting abi.Type
func rvWithPtr(rv reflect.Value, p unsafe.Pointer) reflect.Value {
	(*rvtype)(unsafe.Pointer(&rv)).ptr = p
	return rv
}

// rvPtr returns the underlying ptr of reflect.Value
func rvPtr(rv reflect.Value) unsafe.Pointer {
	return (*rvtype)(unsafe.Pointer(&rv)).ptr
}

type iface struct {
	tab  uintptr
	data unsafe.Pointer
}

// updateIface updates the underlying data ptr of a iface.
// PLEASE MAKE SURE the iface.tab matches the data pointer
func updateIface(p, data unsafe.Pointer) {
	(*iface)(p).data = data
}

// rvTypePtr returns the abi.Type pointer of the given reflect.Value.
// It used by createOrGetFieldDesc for mapping a struct type to *FieldDesc,
// and also used when Malloc
func rvTypePtr(rv reflect.Value) uintptr {
	return (*rvtype)(unsafe.Pointer(&rv)).abiType
}

// same as reflect.StringHeader with Data type is unsafe.Pointer
type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

// same as reflect.SliceHeader with Data type is unsafe.Pointer
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ unsafe.Pointer, needzero bool) unsafe.Pointer

//go:noescape
//go:linkname mapiternext runtime.mapiternext
func mapiternext(it unsafe.Pointer)
