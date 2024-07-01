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

package ssa

import (
	"fmt"
	"unsafe"
)

type _ConstData struct {
	i bool
	v int64
	c Constness
	p unsafe.Pointer
}

func (self _ConstData) String() string {
	if self.i {
		return fmt.Sprintf("(i64) %d", self.v)
	} else {
		return fmt.Sprintf("(%s ptr) %p", self.c, self.p)
	}
}

func constint(v int64) _ConstData {
	return _ConstData{
		v: v,
		i: true,
	}
}

func constptr(p unsafe.Pointer, cc Constness) _ConstData {
	return _ConstData{
		p: p,
		c: cc,
		i: false,
	}
}
