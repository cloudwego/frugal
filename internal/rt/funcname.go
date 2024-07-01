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

package rt

import (
	"fmt"
	"runtime"
	"unsafe"
)

func FuncName(p unsafe.Pointer) string {
	if fn := runtime.FuncForPC(uintptr(p)); fn == nil {
		return "???"
	} else if fp := fn.Entry(); fp == uintptr(p) {
		return fn.Name()
	} else {
		return fmt.Sprintf("%s+%#x", fn.Name(), uintptr(p)-fp)
	}
}
