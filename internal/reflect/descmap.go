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
	"sync/atomic"
	"unsafe"
)

// mapFieldDesc represents a read-lock-free hashmap for *fieldDesc like sync.Map.
// it's NOT designed for writes.
type mapFieldDesc struct {
	p unsafe.Pointer // for atomic, point to hashtable
}

// XXX: fixed size to make it simple,
// we may not so many structs that need to rehash it
const mapFieldDescBuckets = 0xffff

type mapFieldDescItem struct {
	abiType uintptr
	fd      *fieldDesc
}

func newMapFieldDesc() *mapFieldDesc {
	m := &mapFieldDesc{}
	buckets := make([][]mapFieldDescItem, mapFieldDescBuckets+1) // [0] - [0xffff]
	atomic.StorePointer(&m.p, unsafe.Pointer(&buckets))
	return m
}

// Get ...
func (m *mapFieldDesc) Get(abiType uintptr) *fieldDesc {
	buckets := *(*[][]mapFieldDescItem)(atomic.LoadPointer(&m.p))
	dd := buckets[abiType&mapFieldDescBuckets]
	for i := range dd {
		if dd[i].abiType == abiType {
			return dd[i].fd
		}
	}
	return nil
}

// Set ...
// createOrGetFieldDesc will protect calling Set with lock
func (m *mapFieldDesc) Set(abiType uintptr, fd *fieldDesc) {
	if m.Get(abiType) == fd {
		return
	}
	oldBuckets := *(*[][]mapFieldDescItem)(atomic.LoadPointer(&m.p))
	newBuckets := make([][]mapFieldDescItem, mapFieldDescBuckets+1)
	copy(newBuckets, oldBuckets)
	bk := abiType & mapFieldDescBuckets
	newBuckets[bk] = append(newBuckets[bk], mapFieldDescItem{abiType: abiType, fd: fd})
	atomic.StorePointer(&m.p, unsafe.Pointer(&newBuckets))
}
