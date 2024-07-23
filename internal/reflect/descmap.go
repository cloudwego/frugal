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

// mapStructDesc represents a read-lock-free hashmap for *structDesc like sync.Map.
// it's NOT designed for writes.
type mapStructDesc struct {
	p unsafe.Pointer // for atomic, point to hashtable
}

// XXX: fixed size to make it simple,
// we may not so many structs that need to rehash it
const mapStructDescBuckets = 0xffff

type mapStructDescItem struct {
	abiType uintptr
	sd      *structDesc
}

func newMapStructDesc() *mapStructDesc {
	m := &mapStructDesc{}
	buckets := make([][]mapStructDescItem, mapStructDescBuckets+1) // [0] - [0xffff]
	atomic.StorePointer(&m.p, unsafe.Pointer(&buckets))
	return m
}

// Get ...
func (m *mapStructDesc) Get(abiType uintptr) *structDesc {
	buckets := *(*[][]mapStructDescItem)(atomic.LoadPointer(&m.p))
	dd := buckets[abiType&mapStructDescBuckets]
	for i := range dd {
		if dd[i].abiType == abiType {
			return dd[i].sd
		}
	}
	return nil
}

// Set ...
// createOrGetStructDesc will protect calling Set with lock
func (m *mapStructDesc) Set(abiType uintptr, sd *structDesc) {
	if m.Get(abiType) == sd {
		return
	}
	oldBuckets := *(*[][]mapStructDescItem)(atomic.LoadPointer(&m.p))
	newBuckets := make([][]mapStructDescItem, mapStructDescBuckets+1)
	copy(newBuckets, oldBuckets)
	bk := abiType & mapStructDescBuckets
	newBuckets[bk] = append(newBuckets[bk], mapStructDescItem{abiType: abiType, sd: sd})
	atomic.StorePointer(&m.p, unsafe.Pointer(&newBuckets))
}
