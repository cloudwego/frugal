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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapStructDesc_GetEmpty(t *testing.T) {
	m := newMapStructDesc()
	assert.Nil(t, m.Get(1))
	assert.Nil(t, m.Get(0))
	assert.Nil(t, m.Get(0xffff))
}

func TestMapStructDesc_SetGet(t *testing.T) {
	m := newMapStructDesc()
	sd1 := &structDesc{}
	sd2 := &structDesc{}

	m.Set(1, sd1)
	assert.Equal(t, sd1, m.Get(1))
	assert.Nil(t, m.Get(2))

	m.Set(2, sd2)
	assert.Equal(t, sd1, m.Get(1))
	assert.Equal(t, sd2, m.Get(2))
}

func TestMapStructDesc_Update(t *testing.T) {
	m := newMapStructDesc()
	sd1 := &structDesc{}
	sd2 := &structDesc{}

	m.Set(1, sd1)
	assert.Equal(t, sd1, m.Get(1))

	// update same key with new value
	m.Set(1, sd2)
	assert.Equal(t, sd2, m.Get(1))

	// slot should have exactly 1 item, not 2
	bk := uintptr(1) & mapStructDescBuckets
	slot := m.slots[bk].Load()
	assert.Equal(t, 1, len(*slot))
}

func TestMapStructDesc_SetNoop(t *testing.T) {
	m := newMapStructDesc()
	sd := &structDesc{}

	m.Set(1, sd)
	// set same key+value again should be a noop
	m.Set(1, sd)

	bk := uintptr(1) & mapStructDescBuckets
	slot := m.slots[bk].Load()
	assert.Equal(t, 1, len(*slot))
}

func TestMapStructDesc_HashCollision(t *testing.T) {
	m := newMapStructDesc()
	sd1 := &structDesc{}
	sd2 := &structDesc{}

	// keys that map to the same bucket
	k1 := uintptr(1)
	k2 := k1 + mapStructDescBuckets + 1 // same bucket as k1

	m.Set(k1, sd1)
	m.Set(k2, sd2)

	assert.Equal(t, sd1, m.Get(k1))
	assert.Equal(t, sd2, m.Get(k2))

	// both in the same slot
	bk := k1 & mapStructDescBuckets
	slot := m.slots[bk].Load()
	assert.Equal(t, 2, len(*slot))
}

func TestMapStructDesc_UpdateWithCollision(t *testing.T) {
	m := newMapStructDesc()
	sd1 := &structDesc{}
	sd2 := &structDesc{}
	sd3 := &structDesc{}

	k1 := uintptr(1)
	k2 := k1 + mapStructDescBuckets + 1

	m.Set(k1, sd1)
	m.Set(k2, sd2)

	// update k1, should not affect k2
	m.Set(k1, sd3)
	assert.Equal(t, sd3, m.Get(k1))
	assert.Equal(t, sd2, m.Get(k2))

	bk := k1 & mapStructDescBuckets
	slot := m.slots[bk].Load()
	assert.Equal(t, 2, len(*slot))
}
