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
    `fmt`
)

type OpCode uint8

const (
    OP_int OpCode = iota
    OP_str
    OP_bin
    OP_size
    OP_type
    OP_index
    OP_deref
    OP_map_next
    OP_map_begin
    OP_map_set_i8
    OP_map_set_i16
    OP_map_set_i32
    OP_map_set_i64
    OP_map_set_str
    OP_map_set_bool
    OP_map_set_double
    OP_map_set_pointer
    OP_map_is_done
    OP_list_next
    OP_list_begin
    OP_list_is_done
    OP_struct_skip
    OP_struct_begin
    OP_struct_switch
    OP_struct_require
    OP_struct_is_stop
    OP_struct_read_tag
    OP_struct_check_type
    OP_make_state
    OP_drop_state
    OP_construct
    OP_defer
    OP_goto
    OP_halt
)

var _OpNames = [256]string {
    OP_int               : "int",
    OP_str               : "str",
    OP_bin               : "bin",
    OP_size              : "size",
    OP_type              : "type",
    OP_index             : "index",
    OP_deref             : "deref",
    OP_map_next          : "map_next",
    OP_map_begin         : "map_begin",
    OP_map_set_i8        : "map_set_i8",
    OP_map_set_i16       : "map_set_i16",
    OP_map_set_i32       : "map_set_i32",
    OP_map_set_i64       : "map_set_i64",
    OP_map_set_str       : "map_set_str",
    OP_map_set_bool      : "map_set_bool",
    OP_map_set_double    : "map_set_double",
    OP_map_set_pointer   : "map_set_pointer",
    OP_map_is_done       : "map_is_done",
    OP_list_next         : "list_next",
    OP_list_begin        : "list_begin",
    OP_list_is_done      : "list_is_done",
    OP_struct_skip       : "struct_skip",
    OP_struct_begin      : "struct_begin",
    OP_struct_switch     : "struct_switch",
    OP_struct_require    : "struct_require",
    OP_struct_is_stop    : "struct_is_stop",
    OP_struct_read_tag   : "struct_read_tag",
    OP_struct_check_type : "struct_check_type",
    OP_make_state        : "make_state",
    OP_drop_state        : "drop_state",
    OP_construct         : "construct",
    OP_defer             : "defer",
    OP_goto              : "goto",
    OP_halt              : "halt",
}

func (self OpCode) String() string {
    if _OpNames[self] != "" {
        return _OpNames[self]
    } else {
        return fmt.Sprintf("OpCode(%d)", self)
    }
}
