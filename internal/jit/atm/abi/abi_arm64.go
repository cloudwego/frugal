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

package abi

import (
	"reflect"
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/rt"
)

// NOTE: this file is temporary and is used only for emu on arm
// TODO: delete this file after frugal supports arm

type ARM64ABI struct{}

func ArchCreateABI() *ARM64ABI {
	return &ARM64ABI{}
}

func (self *ARM64ABI) RegisterMethod(id int, mt rt.Method) int {
	return mt.Id
}

func (self *ARM64ABI) RegisterFunction(id int, fn interface{}) (fp unsafe.Pointer) {
	vv := rt.UnpackEface(fn)
	vt := vv.Type.Pack()

	/* must be a function */
	if vt.Kind() != reflect.Func {
		panic("fn is not a function")
	}

	return *(*unsafe.Pointer)(vv.Value)
}
