/*
 * Copyright 2022 ByteDance Inc.
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
    `testing`
)

type FooIface interface {
    FooMethod()
}

type FooValue struct {
    X int
}

func (self FooValue) FooMethod() {
    println(self.X)
}

type FooPointer struct {
    X int
}

func (self *FooPointer) FooMethod() {
    println(self.X)
}

func TestIface_Invoke(t *testing.T) {
    var v FooIface
    v = FooValue{X: 100}
    v.FooMethod()
    v = &FooPointer{X: 200}
    v.FooMethod()
}
