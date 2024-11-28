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
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

var hackErrMsg string

func init() {
	err := testhack()
	if err != nil {
		hackErrMsg = fmt.Sprintf("[BUG] Please upgrade frgual to latest version.\n"+
			"If the issue still exists kindly report to author.\n"+
			"Err: %s/%s %s", runtime.Version(), runtime.GOARCH, err)
	}
}

func panicIfHackErr() {
	if len(hackErrMsg) > 0 {
		panic(hackErrMsg)
	}
}

// this func should be called once to test compatibility with Go runtime
func testhack() error {
	{ // mapIter
		m := map[int]string{7: "hello"}
		rv := reflect.ValueOf(m)
		it := newMapIter(rv)
		kp, vp := it.Next()
		if *(*int)(kp) != 7 || *(*string)(vp) != "hello" {
			return errors.New("compatibility issue found: mapIter")
		}
	}

	{ // maplen
		m := map[int]string{}
		m[8] = "world"
		m[9] = "!"
		m[10] = "?"
		if maplen(reflect.ValueOf(m).UnsafePointer()) != 3 {
			return errors.New("compatibility issue found: maplen")
		}
	}

	{ // rvWithPtr
		m := map[int]string{7: "hello"}
		rv := reflect.NewAt(reflect.TypeOf(m), unsafe.Pointer(&m)).Elem()
		rv1 := rvWithPtr(rv, unsafe.Pointer(&m))
		if p0, p1 := rv.UnsafePointer(), rv1.UnsafePointer(); p0 != p1 {
			return fmt.Errorf("compatibility issue found: rvWithPtr %p -> %p m=%p", p0, p1, &m)
		}
		m1, ok := rv1.Interface().(map[int]string)
		if !ok || !reflect.DeepEqual(m, m1) {
			return errors.New("compatibility issue found: rvWithPtr (Interface())")
		}
	}

	{ // rvTypePtr, rtTypePtr
		m1 := map[int]string{}
		m2 := map[int]*string{}
		m3 := map[int]*string{}
		rv := reflect.New(reflect.TypeOf(m1)).Elem()

		if rvTypePtr(reflect.ValueOf(m1)) != rvTypePtr(rv) ||
			rvTypePtr(reflect.ValueOf(m2)) != rvTypePtr(reflect.ValueOf(m3)) ||
			rvTypePtr(reflect.ValueOf(m1)) == rvTypePtr(reflect.ValueOf(m2)) {
			return errors.New("compatibility issue found: rvTypePtr")
		}

		if rtTypePtr(reflect.TypeOf(m1)) != rtTypePtr(rv.Type()) ||
			rtTypePtr(reflect.TypeOf(m2)) != rtTypePtr(reflect.TypeOf(m3)) ||
			rtTypePtr(reflect.TypeOf(m1)) == rtTypePtr(reflect.TypeOf(m3)) {
			return errors.New("compatibility issue found: rtTypePtr")
		}

		if rtTypePtr(reflect.TypeOf(m1)) != rvTypePtr(rv) ||
			rtTypePtr(reflect.TypeOf(m2)) != rvTypePtr(reflect.ValueOf(m3)) {
			return errors.New("compatibility issue found: rtTypePtr<>rvTypePtr")
		}
	}

	{
		f := iFoo(&dog{"test"}) // init itab with iFoo
		f1 := f                 // copy itab to f1

		//  change f1 data pointer to d
		d := &dog{sound: "woof"}
		updateIface(unsafe.Pointer(&f1), unsafe.Pointer(d))

		//  f1 calls d.Foo()
		if f1.Foo() != d.sound {
			return fmt.Errorf("compatibility issue found: updateIface %s <> %s", f1.Foo(), d.sound)
		}

		// f remains unchanged
		if f.Foo() != "test" {
			return fmt.Errorf("compatibility issue found: updateIface %s <> %s", f.Foo(), "test")
		}
	}

	return nil
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

type hackMapIter struct {
	m      reflect.Value
	hitter hitter
}

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
	// use reflect.Next to initialize hitter
	// then we no need to bind mapiterinit, mapiternext
	m.MapIter.Next()
	p := (*hackMapIter)(unsafe.Pointer(&m.MapIter))
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

// updateIface updates the underlying data ptr of a iface.
// PLEASE MAKE SURE the iface.tab matches the data pointer
func updateIface(p, data unsafe.Pointer) {
	type iface struct {
		tab  uintptr
		data unsafe.Pointer
	}
	(*iface)(p).data = data
}

// rvTypePtr returns the abi.Type pointer of the given reflect.Value.
// It used by createOrGetStructDesc for mapping a struct type to *StructDesc,
// and also used when Malloc
func rvTypePtr(rv reflect.Value) uintptr {
	return (*rvtype)(unsafe.Pointer(&rv)).abiType
}

// rtTypePtr returns the abi.Type pointer of the given reflect.Type.
// *rtype of reflect pkg shares the same data struct with *abi.Type
func rtTypePtr(rt reflect.Type) uintptr {
	type iface struct {
		tab  uintptr
		data uintptr
	}
	return (*iface)(unsafe.Pointer(&rt)).data
}

// same as reflect.StringHeader
type stringHeader struct {
	Data uintptr
	Len  int
}

// same as reflect.SliceHeader
type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

// UnsafePointer ... for passing checkptr
// `p := unsafe.Pointer(h.Data)` is NOT allowed when testing with -race
func (h *sliceHeader) UnsafePointer() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(h))
}

var (
	emptyslice = make([]byte, 0)

	// for slice, Data should points to zerobase var in `runtime`
	// so that it can represent as []type{} instead of []type(nil)
	zerobase = ((*sliceHeader)(unsafe.Pointer(&emptyslice))).Data
)

func (h *sliceHeader) Zero() {
	h.Len = 0
	h.Cap = 0
	h.Data = zerobase
}

//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ unsafe.Pointer, needzero bool) unsafe.Pointer
