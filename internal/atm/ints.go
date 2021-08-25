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

func atoi8(v [8]uint8) uint64 {
    return uint64(int8(v[0]))
}

func atoi16(v [8]uint8) uint64 {
    return uint64(*(*int16)(unsafe.Pointer(&v)))
}

func atoi32(v [8]uint8) uint64 {
    return uint64(*(*int32)(unsafe.Pointer(&v)))
}

func atoi64(v [8]uint8) uint64 {
    return *(*uint64)(unsafe.Pointer(&v))
}

func ptou64(p unsafe.Pointer) uint64 {
    return uint64(uintptr(p))
}

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

func u64top(v uint64) (p unsafe.Pointer) {
    *(*uint64)(unsafe.Pointer(&p)) = v
    return
}
