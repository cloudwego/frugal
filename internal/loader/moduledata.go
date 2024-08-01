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

package loader

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

type mdFieldInfo struct {
	off uintptr
	sz  uintptr
}

var (
	moduleDataType reflect.Type // runtime.moduledata

	rtModuleDataFields map[string]mdFieldInfo
	fgModuleDataFields map[string]mdFieldInfo
)

func moduledataPanic(reason string) {
	panic(" moduledata compatibility issue found: " + reason)
}

func searchStructForModuleData(t reflect.Type, depth int) bool {
	if depth == 0 {
		return false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	if t.Name() == "moduledata" && t.PkgPath() == "runtime" {
		moduleDataType = t
		return true
	}
	for i := 0; i < t.NumField(); i++ {
		if searchStructForModuleData(t.Field(i).Type, depth-1) {
			return true
		}
	}
	return false
}

func init() {
	// extract reflect.Type of runtime.moduledata from runtime.Frame
	t := reflect.TypeOf(runtime.Frame{})
	if !searchStructForModuleData(t, 3) {
		moduledataPanic("not found runtime.moduledata in runtime.Frame{}")
	}

	t = moduleDataType
	rtModuleDataFields = map[string]mdFieldInfo{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		rtModuleDataFields[f.Name] = mdFieldInfo{off: f.Offset, sz: f.Type.Size()}
	}

	t = reflect.TypeOf(_ModuleData{})
	fgModuleDataFields = map[string]mdFieldInfo{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fgModuleDataFields[f.Name] = mdFieldInfo{off: f.Offset, sz: f.Type.Size()}
	}

	for name, fi := range fgModuleDataFields {
		fi0, ok := rtModuleDataFields[name]
		if !ok {
			moduledataPanic(fmt.Sprintf("runtime.moduledata field %q gone?", name))
		}
		if fi0.sz != fi.sz {
			moduledataPanic(fmt.Sprintf("runtime.moduledata field %q type size mismatch.", name))
		}
	}
}

func asRuntimeModuleData(m *_ModuleData) unsafe.Pointer {
	b := make([]byte, moduleDataType.Size())
	dst := unsafe.Pointer(&b[0])
	src := unsafe.Pointer(m)
	for name, fsrc := range fgModuleDataFields {
		fdst := rtModuleDataFields[name]
		for i := uintptr(0); i < fsrc.sz; i++ { // copy by byte
			*(*byte)(unsafe.Add(dst, fdst.off+i)) = *(*byte)(unsafe.Add(src, fsrc.off+i))
		}
	}
	return dst
}
