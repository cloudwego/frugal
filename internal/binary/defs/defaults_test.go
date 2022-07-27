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

package defs

import (
    `reflect`
    `testing`
    `unsafe`

    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
)

type CtorTestStruct struct {
    X int
    Y int
    Z int
}

func (self *CtorTestStruct) InitDefault() {
    *self = CtorTestStruct {
        X: 1,
        Y: 2,
        Z: 3,
    }
}

func TestDefaults_Resolve(t *testing.T) {
    fp := (*CtorTestStruct).InitDefault
    fa := **(**unsafe.Pointer)(unsafe.Pointer(&fp))
    spew.Dump(fa)
    fn, err := GetDefaultInitializer(reflect.TypeOf(CtorTestStruct{}))
    require.NoError(t, err)
    spew.Dump(fn)
    require.Equal(t, fa, fn)
}
