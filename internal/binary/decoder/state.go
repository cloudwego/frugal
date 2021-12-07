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
)

const (
    WpOffset = int64(unsafe.Offsetof(StateItem{}.Wp))
    FmOffset = int64(unsafe.Offsetof(StateItem{}.Fm))
    VuOffset = int64(unsafe.Offsetof(StateItem{}.Vu))
)

const (
    StateMax  = StateCap - StateSize
    StateCap  = defs.MaxStack * StateSize
    StateSize = int64(unsafe.Sizeof(StateItem{}))
)

// StateItem is the runtime state item.
// The translator knows the layout and size of this struct, so please keep in sync with it.
type StateItem struct {
    Nb uint64
    Wp unsafe.Pointer
    Fm FieldBitmap
    Vu uint64
}

type RuntimeState struct {
    St [defs.MaxStack]StateItem     // Must be the first field.
    Pr unsafe.Pointer               // Pointer spill space, used for non-fast string or pointer map access.
    Iv uint64                       // Integer spill space, used for non-fast string map access.
}
