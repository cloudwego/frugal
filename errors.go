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

package frugal

import (
    `fmt`
    `reflect`
)

// TypeError occures when encoding or decoding a type that Thrift cannot represent.
type TypeError struct {
    Note string
    Type reflect.Type
}

func (self TypeError) Error() string {
    if self.Note != "" {
        return fmt.Sprintf("TypeError(%s): %s", self.Type.String(), self.Note)
    } else {
        return fmt.Sprintf("TypeError(%s): not supported by Thrift", self.Type.String())
    }
}

// SyntaxError occures when failed to parse the thrift type defination.
type SyntaxError struct {
    Pos    int
    Src    string
    Reason string
}

func (self SyntaxError) Error() string {
    return fmt.Sprintf("Syntax error at position %d: %s", self.Pos, self.Reason)
}
