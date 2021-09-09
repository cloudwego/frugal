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

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/cloudwego/frugal/internal/utils`
)

type Encoder func (
    iov frugal.IoVec,
    p   unsafe.Pointer,
    rs  *RuntimeState,
    st  int,
) error

var (
    programCache = utils.CreateProgramCache()
)

func encode(vt *rt.GoType, iov frugal.IoVec, p unsafe.Pointer, rs *RuntimeState, st int) error {
    if enc, err := resolve(vt); err != nil {
        return err
    } else {
        return enc(iov, p, rs, st)
    }
}

func resolve(vt *rt.GoType) (Encoder, error) {
    if val := programCache.Get(vt); val != nil {
        return val.(Encoder), nil
    } else if ret, err := programCache.Compute(vt, compile); err == nil {
        return ret.(Encoder), nil
    } else {
        return nil, err
    }
}

func compile(vt *rt.GoType) (interface{}, error) {
    cc := CreateCompiler()
    hp, err := cc.Compile(vt.Pack())

    /* compile the type into High-Level IL, then
     * translate to Low-Leve IL, and then link the program */
    if cc.Free(); err != nil {
        return nil, err
    } else {
        return link(Translate(hp)), nil
    }
}
