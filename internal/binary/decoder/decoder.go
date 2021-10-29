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

package decoder

import (
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
    `github.com/cloudwego/frugal/internal/utils`
)

type Decoder func (
    buf []byte,
    p   unsafe.Pointer,
    rs  *RuntimeState,
    st  int,
) (int, error)

var (
    programCache = utils.CreateProgramCache()
)

func decode(vt *rt.GoType, buf []byte, p unsafe.Pointer, rs *RuntimeState, st int) (int, error) {
    if dec, err := resolve(vt); err != nil {
        return 0, err
    } else {
        return dec(buf, p, rs, st)
    }
}

func resolve(vt *rt.GoType) (Decoder, error) {
    if val := programCache.Get(vt); val != nil {
        return val.(Decoder), nil
    } else if ret, err := programCache.Compute(vt, compile); err == nil {
        return ret.(Decoder), nil
    } else {
        return nil, err
    }
}

func compile(vt *rt.GoType) (interface{}, error) {
    if Link == nil {
        panic("no linker available for decoder")
    } else if pp, err := CreateCompiler().Compile(vt.Pack()); err != nil {
        return nil, err
    } else {
        return Link(Translate(pp)), nil
    }
}
