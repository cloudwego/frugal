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
    `testing`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

func TestGCWB_FuncAddr(t *testing.T) {
    fp := rt.FuncAddr(gcWriteBarrier)
    disasm(uintptr(fp), *(*[]byte)(unsafe.Pointer(&rt.GoSlice {
        Ptr: fp,
        Len: 64,
        Cap: 64,
    })))
}

