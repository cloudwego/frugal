/*
 * Copyright 2021 ByteDance Inc.
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
    _ `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

//go:noescape
//go:linkname mapiternext runtime.mapiternext
//goland:noinspection GoUnusedParameter
func mapiternext(it *rt.GoMapIterator)

//go:noescape
//go:linkname mapiterinit runtime.mapiterinit
//goland:noinspection GoUnusedParameter
func mapiterinit(t *rt.GoMapType, h *rt.GoMap, it *rt.GoMapIterator)

//go:nosplit
func MapEndIterator(it *rt.GoMapIterator) {
    freeIterator(it)
}

//go:nosplit
func MapBeginIterator(vt *rt.GoMapType, mm *rt.GoMap) (it *rt.GoMapIterator) {
    it = newIterator()
    mapiterinit(vt, mm, it)
    return
}
