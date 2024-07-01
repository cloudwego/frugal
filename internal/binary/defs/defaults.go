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
	"fmt"
	"reflect"
	"unsafe"
)

type DefaultInitializer interface {
	InitDefault()
}

func GetDefaultInitializer(vt reflect.Type) (unsafe.Pointer, error) {
	var ok bool
	var mt reflect.Method

	/* element type and pointer type */
	et := vt
	pt := reflect.PtrTo(vt)

	/* dereference the type */
	for et.Kind() == reflect.Ptr {
		pt = et
		et = et.Elem()
	}

	/* find the default initializer method */
	if mt, ok = et.MethodByName("InitDefault"); ok {
		return nil, fmt.Errorf("implementation of `InitDefault()` must have a pointer receiver: %s", mt.Type)
	} else if mt, ok = pt.MethodByName("InitDefault"); !ok {
		return nil, nil
	} else if mt.Type.NumIn() != 1 || mt.Type.NumOut() != 0 {
		return nil, fmt.Errorf("invalid implementation of `InitDefault()`: %s", mt.Type)
	} else {
		return *(*[2]*unsafe.Pointer)(unsafe.Pointer(&mt.Func))[1], nil
	}
}
