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

package decoder

import (
    `unsafe`

    `github.com/cloudwego/frugal/internal/binary/defs`
    `github.com/cloudwego/frugal/internal/rt`
)

const (
    NbOffset = int64(unsafe.Offsetof(StateItem{}.Nb))
    MpOffset = int64(unsafe.Offsetof(StateItem{}.Mp))
    WpOffset = int64(unsafe.Offsetof(StateItem{}.Wp))
    FmOffset = int64(unsafe.Offsetof(StateItem{}.Fm))
)

const (
    PrOffset = int64(unsafe.Offsetof(RuntimeState{}.Pr))
    IvOffset = int64(unsafe.Offsetof(RuntimeState{}.Iv))
)

const (
    StateMax  = (defs.MaxStack - 1) * StateSize
    StateSize = int64(unsafe.Sizeof(StateItem{}))
)

type StateItem struct {
    Nb uint64
    Mp *rt.GoMap
    Wp unsafe.Pointer
    Fm FieldBitmap
}

type RuntimeState struct {
    St [defs.MaxStack]StateItem     // Must be the first field.
    Pr unsafe.Pointer               // Pointer spill space, used for non-fast string or pointer map access.
    Iv uint64                       // Integer spill space, used for non-fast string map access.
}
