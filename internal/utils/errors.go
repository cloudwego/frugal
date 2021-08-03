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

package utils

import (
    `fmt`
    `reflect`

    `github.com/cloudwego/frugal`
)

func EType(vt reflect.Type) frugal.TypeError {
    return frugal.TypeError {
        Type: vt,
    }
}

func ESyntax(pos int, src string, reason string) frugal.SyntaxError {
    return frugal.SyntaxError {
        Pos    : pos,
        Src    : src,
        Reason : reason,
    }
}

func ESetList(pos int, src string, vt reflect.Type) frugal.SyntaxError {
    return ESyntax(pos, src, fmt.Sprintf(
        `ambiguous type between set<%s> and list<%s>, please specify in the "frugal" tag`,
        vt.Name(),
        vt.Name(),
    ))
}

func ENotSupp(vt reflect.Type, alt string) frugal.TypeError {
    return frugal.TypeError {
        Type: vt,
        Note: fmt.Sprintf("Thrift does not support %s, use %s instead", vt.String(), alt),
    }
}
