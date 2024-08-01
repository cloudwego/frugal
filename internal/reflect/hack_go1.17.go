//go:build go1.17 && !go1.18

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
	"reflect"
	"unsafe"
)

type hackMapIter struct {
	m reflect.Value
	// it's a pointer before go1.18,
	// it causes allocation when calling m.MapRange & the 1st it.Next()
	hitter *hitter
}

func (iter *hackMapIter) initialized() bool { return iter.hitter != nil }

func (iter *hackMapIter) Next() (unsafe.Pointer, unsafe.Pointer) {
	mapiternext(unsafe.Pointer(iter.hitter))
	return iter.hitter.k, iter.hitter.v
}
