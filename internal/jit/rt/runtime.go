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

package rt

import (
	"reflect"
	"unsafe"
)

const (
	MaxFastMap = 128
)

const (
	T_uncommon = 1 << 0
)

const (
	F_direct    = 1 << 5
	F_kind_mask = (1 << 5) - 1
)

var (
	reflectRtypeItab = findReflectRtypeItab()
)

type (
	GoNameOffset int32
	GoTypeOffset int32
	GoTextOffset int32
)

func (self GoNameOffset) Resolve(vt *GoType) GoName {
	if self == -1 {
		return GoName{}
	} else {
		return resolveNameOff(unsafe.Pointer(vt), self)
	}
}

func (self GoTypeOffset) Resolve(vt *GoType) *GoType {
	if self == -1 {
		return nil
	} else {
		return resolveTypeOff(unsafe.Pointer(vt), self)
	}
}

func (self GoTextOffset) Resolve(vt *GoType) unsafe.Pointer {
	if self == -1 {
		return nil
	} else {
		return resolveTextOff(unsafe.Pointer(vt), self)
	}
}

type GoName struct {
	b *[1 << 16]byte
}

func (self GoName) read(p int) (i int, v int) {
	for i, v = 0, 0; ; i++ {
		if v += int(self.b[p+i]&0x7f) << (7 * i); self.b[p+i]&0x80 == 0 {
			return i + 1, v
		}
	}
}

func (self GoName) mkstr(i int, n int) (s string) {
	(*GoString)(unsafe.Pointer(&s)).Len = n
	(*GoString)(unsafe.Pointer(&s)).Ptr = unsafe.Pointer(&self.b[i])
	return
}

func (self GoName) Name() string {
	if self.b == nil {
		return ""
	} else if i, n := self.read(1); n == 0 {
		return ""
	} else {
		return self.mkstr(i+1, n)
	}
}

func (self GoName) IsExported() bool {
	return self.b != nil && self.b[0]&(1<<0) != 0
}

type GoType struct {
	Size       uintptr
	PtrData    uintptr
	Hash       uint32
	Flags      uint8
	Align      uint8
	FieldAlign uint8
	KindFlags  uint8
	Equal      func(unsafe.Pointer, unsafe.Pointer) bool
	GCData     *byte
	Str        GoNameOffset
	PtrToSelf  GoTypeOffset
}

func (self *GoType) mtab() []GoMethod {
	switch self.Kind() {
	case reflect.Ptr:
		return (*struct {
			GoPtrType
			GoUncommonType
		})(unsafe.Pointer(self)).Methods()
	case reflect.Struct:
		return (*struct {
			GoStructType
			GoUncommonType
		})(unsafe.Pointer(self)).Methods()
	default:
		return nil
	}
}

func (self *GoType) Kind() reflect.Kind {
	return reflect.Kind(self.KindFlags & F_kind_mask)
}

func (self *GoType) Pack() (t reflect.Type) {
	(*GoIface)(unsafe.Pointer(&t)).Itab = reflectRtypeItab
	(*GoIface)(unsafe.Pointer(&t)).Value = unsafe.Pointer(self)
	return
}

func (self *GoType) String() string {
	return self.Pack().String()
}

func (self *GoType) Methods() []GoMethod {
	if !self.IsUncommon() {
		return nil
	} else {
		return self.mtab()
	}
}

func (self *GoType) IsUncommon() bool {
	return (self.Flags & T_uncommon) != 0
}

func (self *GoType) IsIndirect() bool {
	return (self.KindFlags & F_direct) == 0
}

type GoPtrType struct {
	GoType
	Elem *GoType
}

type GoMapType struct {
	GoType
	Key        *GoType
	Elem       *GoType
	Bucket     *GoType
	Hasher     func(unsafe.Pointer, uintptr) uintptr
	KeySize    uint8
	ElemSize   uint8
	BucketSize uint16
	Flags      uint32
}

func (self *GoMapType) IsFastMap() bool {
	return self.Elem.Size <= MaxFastMap
}

type GoSliceType struct {
	GoType
	Elem *GoType
}

type GoStructType struct {
	GoType
	PkgPath GoName
	Fields  []GoStructField
}

type GoMethod struct {
	Name  GoNameOffset
	Type  GoTypeOffset
	IFunc GoTextOffset
	TFunc GoTextOffset
}

type GoStructField struct {
	Name        GoName
	Type        *GoType
	OffsetEmbed uintptr
}

type GoUncommonType struct {
	PkgPath      int32
	NumMethod    uint16
	NumExported  uint16
	MethodOffset uint32
	_            uint32
}

func (self *GoUncommonType) mbuf() unsafe.Pointer {
	return unsafe.Pointer(uintptr(unsafe.Pointer(self)) + uintptr(self.MethodOffset))
}

func (self *GoUncommonType) Methods() []GoMethod {
	if self.NumMethod == 0 {
		return nil
	} else {
		return (*[1 << 16]GoMethod)(self.mbuf())[:self.NumMethod:self.NumMethod]
	}
}

type GoItab struct {
	it unsafe.Pointer
	vt *GoType
	hv uint32
	_  [4]byte
	fn [1]uintptr
}

const (
	GoItabFuncBase = unsafe.Offsetof(GoItab{}.fn)
)

type GoIface struct {
	Itab  *GoItab
	Value unsafe.Pointer
}

type GoEface struct {
	Type  *GoType
	Value unsafe.Pointer
}

type GoSlice struct {
	Ptr unsafe.Pointer
	Len int
	Cap int
}

func (self GoSlice) Set(i int, v byte) {
	*(*byte)(unsafe.Pointer(uintptr(self.Ptr) + uintptr(i))) = v
}

type GoString struct {
	Ptr unsafe.Pointer
	Len int
}

type GoMap struct {
	Count      int
	Flags      uint8
	B          uint8
	Overflow   uint16
	Hash0      uint32
	Buckets    unsafe.Pointer
	OldBuckets unsafe.Pointer
	Evacuate   uintptr
	Extra      unsafe.Pointer
}

type GoMapIterator struct {
	K           unsafe.Pointer
	V           unsafe.Pointer
	T           *GoMapType
	H           *GoMap
	Buckets     unsafe.Pointer
	Bptr        *unsafe.Pointer
	Overflow    *[]unsafe.Pointer
	OldOverflow *[]unsafe.Pointer
	StartBucket uintptr
	Offset      uint8
	Wrapped     bool
	B           uint8
	I           uint8
	Bucket      uintptr
	CheckBucket uintptr
}

func (self *GoMapIterator) Next() bool {
	mapiternext(self)
	return self.K != nil
}

func IsPtr(t *GoType) bool {
	return t.Kind() == reflect.Ptr || t.Kind() == reflect.UnsafePointer
}

func PtrElem(t *GoType) *GoType {
	if t.Kind() != reflect.Ptr {
		panic("t is not a ptr")
	} else {
		return (*GoPtrType)(unsafe.Pointer(t)).Elem
	}
}

func MapType(t *GoType) *GoMapType {
	if t.Kind() != reflect.Map {
		panic("t is not a map")
	} else {
		return (*GoMapType)(unsafe.Pointer(t))
	}
}

func Dereference(t *GoType) *GoType {
	for t.Kind() == reflect.Ptr {
		t = PtrElem(t)
	}
	return t
}

func FuncAddr(f interface{}) unsafe.Pointer {
	if vv := UnpackEface(f); vv.Type.Kind() != reflect.Func {
		panic("f is not a function")
	} else {
		return *(*unsafe.Pointer)(vv.Value)
	}
}

func BytesFrom(p unsafe.Pointer, n int, c int) (r []byte) {
	(*GoSlice)(unsafe.Pointer(&r)).Ptr = p
	(*GoSlice)(unsafe.Pointer(&r)).Len = n
	(*GoSlice)(unsafe.Pointer(&r)).Cap = c
	return
}

func StringPtr(s string) unsafe.Pointer {
	return (*GoString)(unsafe.Pointer(&s)).Ptr
}

func StringFrom(p unsafe.Pointer, n int) (r string) {
	(*GoString)(unsafe.Pointer(&r)).Ptr = p
	(*GoString)(unsafe.Pointer(&r)).Len = n
	return
}

func UnpackType(t reflect.Type) *GoType {
	return (*GoType)((*GoIface)(unsafe.Pointer(&t)).Value)
}

func UnpackEface(v interface{}) GoEface {
	return *(*GoEface)(unsafe.Pointer(&v))
}

func findReflectRtypeItab() *GoItab {
	v := reflect.TypeOf(struct{}{})
	return (*GoIface)(unsafe.Pointer(&v)).Itab
}
