//go:build go1.21

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

package rtx

import (
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/rt"
)

//go:linkname writeBarrier runtime.writeBarrier
var writeBarrier uintptr

//go:nosplit
//go:linkname gcWriteBarrier2 runtime.gcWriteBarrier2
func gcWriteBarrier2()

func gcWriteBarrier() {
	// obsoleted in go1.21+, but it's referenced by ssa and we're not going to update
	// ssa at the moment, we just leave an empty function here
}

var (
	V_pWriteBarrier   = unsafe.Pointer(&writeBarrier)
	F_gcWriteBarrier  = rt.FuncAddr(gcWriteBarrier) // referenced by ssa (but not used)
	F_gcWriteBarrier2 = rt.FuncAddr(gcWriteBarrier2)
)
