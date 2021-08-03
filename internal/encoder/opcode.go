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
    OP_i8 OpCode = iota
    OP_i16
    OP_i32
    OP_i64
    OP_double
    OP_binary
    OP_bool
    OP_index
    OP_goto
    OP_if_nil
    OP_map_begin
    OP_map_check_key
    OP_map_value_next
    OP_list_begin
    OP_list_advance
    OP_field_stop
    OP_field_begin
    OP_defer
    OP_save
    OP_load
    OP_drop
)

var _OpNames = [256]string {
    OP_i8             : "i8",
    OP_i16            : "i16",
    OP_i32            : "i32",
    OP_i64            : "i64",
    OP_double         : "double",
    OP_binary         : "binary",
    OP_bool           : "bool",
    OP_index          : "index",
    OP_goto           : "goto",
    OP_if_nil         : "if_nil",
    OP_map_begin      : "map_begin",
    OP_map_check_key  : "map_check_key",
    OP_map_value_next : "map_value_next",
    OP_list_begin     : "list_begin",
    OP_list_advance   : "list_advance",
    OP_field_stop     : "field_stop",
    OP_field_begin    : "field_begin",
    OP_defer          : "defer",
    OP_save           : "save",
    OP_load           : "load",
    OP_drop           : "drop",
}

var _OpBranches = [256]bool {
    OP_goto          : true,
    OP_if_nil        : true,
    OP_map_check_key : true,
    OP_list_advance  : true,
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
