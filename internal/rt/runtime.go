/*
 * Copyright 2021 ByteDance Inc.
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
    `reflect`
    `unsafe`
)

const (
    MaxFastMap = 128
)

const (
    F_direct    = 1 << 5
    F_kind_mask = (1 << 5) - 1
)

var (
    reflectRtypeItab = findReflectRtypeItab()
)

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
    Str        int32
    PtrToSelf  int32
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

func (self *GoType) IsIndirect() bool {
    return (self.KindFlags & F_direct) == 0
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

type GoItab struct {
    it unsafe.Pointer
    vt *GoType
    hv uint32
    _  [4]byte
    fn [1]uintptr
}

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

func (self GoSlice) Set(i int, v byte) {
    *(*byte)(unsafe.Pointer(uintptr(self.Ptr) + uintptr(i))) = v
}

type GoSliceType struct {
    GoType
    Elem *GoType
}

func MapType(t *GoType) *GoMapType {
    if t.Kind() != reflect.Map {
        panic("t is not a map")
    } else {
        return (*GoMapType)(unsafe.Pointer(t))
    }
}

func FuncAddr(f interface{}) unsafe.Pointer {
    if vv := UnpackEface(f); vv.Type.Kind() != reflect.Func {
        panic("f is not a function")
    } else {
        return *(*unsafe.Pointer)(vv.Value)
    }
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
