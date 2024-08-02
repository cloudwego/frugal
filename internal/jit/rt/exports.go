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
	"unsafe"
)

//go:noescape
//go:linkname mapclear runtime.mapclear
//goland:noinspection GoUnusedParameter
func mapclear(t *GoType, h unsafe.Pointer)

//go:noescape
//go:linkname mapiternext runtime.mapiternext
//goland:noinspection GoUnusedParameter
func mapiternext(it *GoMapIterator)

//go:noescape
//go:linkname resolveNameOff runtime.resolveNameOff
//goland:noinspection GoUnusedParameter
func resolveNameOff(p unsafe.Pointer, off GoNameOffset) GoName

//go:noescape
//go:linkname resolveTypeOff runtime.resolveTypeOff
//goland:noinspection GoUnusedParameter
func resolveTypeOff(p unsafe.Pointer, off GoTypeOffset) *GoType

//go:noescape
//go:linkname resolveTextOff reflect.resolveTextOff
//goland:noinspection GoUnusedParameter
func resolveTextOff(p unsafe.Pointer, off GoTextOffset) unsafe.Pointer

//go:nosplit
func MapClear(m interface{}) {
	v := UnpackEface(m)
	mapclear(v.Type, v.Value)
}
