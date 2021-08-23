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
    `fmt`
)

type OpCode uint8

const (
    OP_byte OpCode = iota
    OP_word
    OP_long
    OP_quad
    OP_size
    OP_copy
    OP_vstr
    OP_seek
    OP_deref
    OP_defer
    OP_map_end
    OP_map_key
    OP_map_value
    OP_map_begin
    OP_map_is_end
    OP_list_end
    OP_list_next
    OP_list_begin
    OP_list_is_end
    OP_goto
    OP_if_nil
    OP_if_true
)

var _OpNames = [256]string {
    OP_byte        : "byte",
    OP_word        : "word",
    OP_long        : "long",
    OP_quad        : "long",
    OP_size        : "size",
    OP_copy        : "copy",
    OP_vstr        : "vstr",
    OP_seek        : "seek",
    OP_deref       : "deref",
    OP_defer       : "defer",
    OP_map_end     : "map_end",
    OP_map_key     : "map_key",
    OP_map_value   : "map_value",
    OP_map_begin   : "map_begin",
    OP_map_is_end  : "map_is_end",
    OP_list_end    : "list_end",
    OP_list_next   : "list_next",
    OP_list_begin  : "list_begin",
    OP_list_is_end : "list_is_end",
    OP_goto        : "goto",
    OP_if_nil      : "if_nil",
    OP_if_true     : "if_true",
}

var _OpBranches = [256]bool {
    OP_goto    : true,
    OP_if_nil  : true,
    OP_if_true : true,
}

func (self OpCode) String() string {
    if _OpNames[self] != "" {
        return _OpNames[self]
    } else {
        return fmt.Sprintf("OpCode(%d)", self)
    }
}

func (self OpCode) isBranch() bool {
    return _OpBranches[self]
}
