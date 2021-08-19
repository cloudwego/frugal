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

package atm

import (
    `unsafe`
)

func i8toa(v int8) (r [8]uint8) {
    r[0] = uint8(v)
    return
}

func i16toa(v int16) (r [8]uint8) {
    *(*int16)(unsafe.Pointer(&r)) = v
    return
}

func i32toa(v int32) (r [8]uint8) {
    *(*int32)(unsafe.Pointer(&r)) = v
    return
}

func i64toa(v int64) (r [8]uint8) {
    *(*int64)(unsafe.Pointer(&r)) = v
    return
}
