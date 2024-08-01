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

package loader

import (
	"sync/atomic"
	"unsafe"

	"github.com/cloudwego/frugal/internal/utils"
)

const (
	_PCDATA_UnsafePoint       = 0
	_PCDATA_StackMapIndex     = 1
	_PCDATA_UnsafePointUnsafe = -2
)

//go:linkname lastmoduledatap runtime.lastmoduledatap
//goland:noinspection GoUnusedGlobalVariable
var lastmoduledatap unsafe.Pointer

//go:linkname moduledataverify1 runtime.moduledataverify1
func moduledataverify1(_ *_ModuleData)

var (
	/* retains local reference of all modules to bypass gc */
	modList0 = utils.ListNode{} // all frugal _ModuleData
	modList1 = utils.ListNode{} // all runtime.moduledata
)

func toZigzag(v int) int {
	return (v << 1) ^ (v >> 31)
}

func encodeFirst(v int) []byte {
	return encodeValue(v + 1)
}

func encodeValue(v int) []byte {
	return encodeVariant(toZigzag(v))
}

func encodeVariant(v int) []byte {
	var u int
	var r []byte

	/* split every 7 bits */
	for v > 127 {
		u = v & 0x7f
		v = v >> 7
		r = append(r, byte(u)|0x80)
	}

	/* check for last one */
	if v == 0 {
		return r
	}

	/* add the last one */
	r = append(r, byte(v))
	return r
}

func registerModule(p *_ModuleData) {
	mod := asRuntimeModuleData(p)
	modList0.Prepend(unsafe.Pointer(p))
	modList1.Prepend(mod)
	registerModuleLockFree(&lastmoduledatap, mod, rtModuleDataFields["next"].off)
}

func registerModuleLockFree(tail *unsafe.Pointer, mod unsafe.Pointer, nextoff uintptr) {
	// oldmod := tail
	// tail = mod
	// oldmod.next = mod
	for {
		oldmod := atomic.LoadPointer(tail)
		if atomic.CompareAndSwapPointer(tail, oldmod, mod) {
			p := unsafe.Add(oldmod, nextoff) // &oldmod.next
			atomic.StorePointer((*unsafe.Pointer)(p), mod)
			break
		}
	}
}
