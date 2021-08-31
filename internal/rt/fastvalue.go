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

func (self GoSlice) Set(i int, v byte) {
    *(*byte)(unsafe.Pointer(uintptr(self.Ptr) + uintptr(i))) = v
}

type GoSliceType struct {
    GoType
    Elem *GoType
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
