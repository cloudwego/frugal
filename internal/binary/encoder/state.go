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
    `unsafe`

    `github.com/cloudwego/frugal/internal/binary/defs`
)

const (
    LnOffset = int64(unsafe.Offsetof(StateItem{}.Ln))
    MiOffset = int64(unsafe.Offsetof(StateItem{}.Mi))
    WpOffset = int64(unsafe.Offsetof(StateItem{}.Wp))
)

const (
    StateMax  = (defs.MaxStack - 1) * StateSize
    StateSize = int64(unsafe.Sizeof(StateItem{}))
)

type StateItem struct {
    Ln uintptr
    Mi unsafe.Pointer
    Wp unsafe.Pointer
}

type RuntimeState struct {
    St [defs.MaxStack]StateItem
}
