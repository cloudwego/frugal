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

package encoder

import (
	"github.com/cloudwego/frugal/internal/jit/atm/hir"
	"github.com/cloudwego/frugal/internal/jit/rt"
)

func emu_gcall_mapiternext(ctx hir.CallContext) {
	if !ctx.Verify("*", "") {
		panic("invalid mapiternext call")
	} else {
		mapiternext((*rt.GoMapIterator)(ctx.Ap(0)))
	}
}

func emu_gcall_mapiterstart(ctx hir.CallContext) {
	if !ctx.Verify("***", "") {
		panic("invalid mapiterstart call")
	} else {
		mapiterstart((*rt.GoMapType)(ctx.Ap(0)), (*rt.GoMap)(ctx.Ap(1)), (*rt.GoMapIterator)(ctx.Ap(2)))
	}
}
