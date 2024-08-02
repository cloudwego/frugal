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

package abi

import (
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/rt"
)

type AbstractABI interface {
	RegisterMethod(id int, mt rt.Method) int
	RegisterFunction(id int, fn interface{}) unsafe.Pointer
}

var (
	ABI = ArchCreateABI()
)

const (
	PtrSize = 8 // pointer size
)

func alignUp(n uintptr, a int) uintptr {
	return (n + uintptr(a) - 1) &^ (uintptr(a) - 1)
}
