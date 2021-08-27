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
    OP_sint
    OP_vstr
    OP_seek
    OP_deref
    OP_defer
    OP_map_end
    OP_map_key
    OP_map_next
    OP_map_value
    OP_map_begin
    OP_map_if_end
    OP_list_end
    OP_list_exit
    OP_list_next
    OP_list_begin
    OP_list_if_end
    OP_goto
    OP_if_nil
)

var _OpNames = [256]string {
    OP_byte        : "byte",
    OP_word        : "word",
    OP_long        : "long",
    OP_quad        : "long",
    OP_size        : "size",
    OP_sint        : "sint",
    OP_vstr        : "vstr",
    OP_seek        : "seek",
    OP_deref       : "deref",
    OP_defer       : "defer",
    OP_map_end     : "map_end",
    OP_map_key     : "map_key",
    OP_map_next    : "map_next",
    OP_map_value   : "map_value",
    OP_map_begin   : "map_begin",
    OP_map_if_end  : "map_if_end",
    OP_list_end    : "list_end",
    OP_list_exit   : "list_exit",
    OP_list_next   : "list_next",
    OP_list_begin  : "list_begin",
    OP_list_if_end : "list_if_end",
    OP_goto        : "goto",
    OP_if_nil      : "if_nil",
}

var _OpBranches = [256]bool {
    OP_goto        : true,
    OP_if_nil      : true,
    OP_map_if_end  : true,
    OP_list_if_end : true,
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
