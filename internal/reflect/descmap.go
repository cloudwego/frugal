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

import "sync/atomic"

// mapStructDesc represents a read-lock-free hashmap for *structDesc like sync.Map.
// it's NOT designed for writes.
// Each slot is an atomic.Pointer so that Set only reallocs the target slot.
type mapStructDesc struct {
	slots [mapStructDescBuckets + 1]atomic.Pointer[[]mapStructDescItem]
}

// XXX: fixed size to make it simple,
// we may not so many structs that need to rehash it
const mapStructDescBuckets = 0xffff

type mapStructDescItem struct {
	abiType uintptr
	sd      *structDesc
}

func newMapStructDesc() *mapStructDesc {
	return &mapStructDesc{}
}

// Get ...
func (m *mapStructDesc) Get(abiType uintptr) *structDesc {
	slot := m.slots[abiType&mapStructDescBuckets].Load()
	if slot == nil {
		return nil
	}
	for i := range *slot {
		if (*slot)[i].abiType == abiType {
			return (*slot)[i].sd
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
	bk := abiType & mapStructDescBuckets
	var old []mapStructDescItem
	if p := m.slots[bk].Load(); p != nil {
		old = *p
	}
	// alloc cap=len+1 upfront so append below won't realloc.
	items := make([]mapStructDescItem, len(old), len(old)+1)
	copy(items, old)
	for i := range items {
		if items[i].abiType == abiType {
			items[i].sd = sd
			m.slots[bk].Store(&items)
			return
		}
	}
	items = append(items, mapStructDescItem{abiType: abiType, sd: sd})
	m.slots[bk].Store(&items)
}
