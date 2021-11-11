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

package iovec

import (
    `bytes`
)

// SimpleIoVec provides a very basic implementation of IoVec using bytes.Buffer.
type SimpleIoVec struct {
    bytes.Buffer
}

func (self *SimpleIoVec) Put(v []byte) {
    _, _ = self.Write(v)
}

func (self *SimpleIoVec) Cat(v []byte, w []byte) {
    self.Put(v)
    self.Put(w)
}

func (self *SimpleIoVec) Add(n int, v []byte) []byte {
    self.Put(v)
    self.Grow(n)
    return self.Bytes()[self.Len():]
}
