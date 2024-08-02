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

package decoder

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/atm/hir"
	"github.com/cloudwego/frugal/internal/jit/rt"
)

var (
	initFnLock  = new(sync.RWMutex)
	initFnCache = make(map[unsafe.Pointer]*hir.CallHandle)
)

func toInitFn(fp unsafe.Pointer) (fn func(unsafe.Pointer)) {
	*(*unsafe.Pointer)(unsafe.Pointer(&fn)) = unsafe.Pointer(&fp)
	return
}

func addInitFn(fp unsafe.Pointer) *hir.CallHandle {
	var ok bool
	var fn *hir.CallHandle

	/* check function cache */
	initFnLock.RLock()
	fn, ok = initFnCache[fp]
	initFnLock.RUnlock()

	/* exists, use the cached value */
	if ok {
		return fn
	}

	/* lock in write mode */
	initFnLock.Lock()
	defer initFnLock.Unlock()

	/* double check */
	if fn, ok = initFnCache[fp]; ok {
		return fn
	}

	/* still not exists, register a new function */
	fn = hir.RegisterGCall(toInitFn(fp), func(ctx hir.CallContext) {
		if !ctx.Verify("*", "") {
			panic(fmt.Sprintf("invalid %s call", rt.FuncName(fp)))
		} else {
			toInitFn(fp)(ctx.Ap(0))
		}
	})

	/* update the cache */
	initFnCache[fp] = fn
	return fn
}
