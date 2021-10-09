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

    `github.com/cloudwego/frugal/internal/atm`
)

/** Function Prototype
 *
 *      func(buf []byte, p unsafe.Pointer, rs *RuntimeState, st int) (err error)
 */

/** Register Allocations
 *
 *      P3      Input Buffer Pointer
 *      P4      Current Working Pointer
 *      P5      Runtime State Pointer
 *      P6      Error Type Pointer
 *      P7      Error Value Pointer
 *
 *      R6      Input Buffer Length
 *      R7      State Index
 */

const (
    IP = atm.P3
    WP = atm.P4
    RS = atm.P5
    ET = atm.P6     // may also be used as a temporary pointer register
    EP = atm.P7     // may also be used as a temporary pointer register
)

const (
    IL = atm.R6
    ST = atm.R7
)

const (
    TP = atm.P0
    TR = atm.R0
)

const (
    LB_eof      = "_eof"
    LB_halt     = "_halt"
    LB_error    = "_error"
    LB_overflow = "_overflow"
)

var (
    _E_overflow error
)

func init() {
    _E_overflow = fmt.Errorf("frugal: decoder stack overflow")
}

func Translate(s Program) atm.Program {
    p := atm.CreateBuilder()
    prologue (p)
    program  (p, s)
    epilogue (p)
    errors   (p)
    return p.Build()
}

func errors(p *atm.Builder) {
    p.Label (LB_eof)                    // _eof:
    p.SUB   (TR, IL, TR)                // TR <= TR - IL
    p.GCALL (error_eof).                // GCALL error_eof:
      A0    (TR).                       //     n        <= TR
      R0    (ET).                       //     ret.itab => ET
      R1    (EP)                        //     ret.data => EP
    p.JAL   (LB_error, atm.Pn)          // GOTO _error
    p.Label (LB_overflow)               // _overflow:
    p.IP    (&_E_overflow, TP)          // TP <= &_E_overflow
    p.LP    (TP, ET)                    // ET <= *TP
    p.ADDPI (TP, 8, TP)                 // TP <=  TP + 8
    p.LP    (TP, EP)                    // EP <= *TP
    p.JAL   (LB_error, atm.Pn)          // GOTO _error
}

func program(p *atm.Builder, s Program) {
    for i, v := range s {
        p.Mark(i)
        translators[v.Op](p, v)
    }
}

func prologue(p *atm.Builder) {
    p.LDAP  (0, IP)                     // IP <= a0
    p.LDAQ  (1, IL)                     // IL <= a1
    p.LDAP  (3, WP)                     // WP <= a3
    p.LDAP  (4, RS)                     // RS <= a4
    p.LDAQ  (5, ST)                     // ST <= a5
    p.ADDP  (RS, ST, RS)                // RS <= RS + ST
}

func epilogue(p *atm.Builder) {
    p.Label (LB_halt)                   // _halt:
    p.MOVP  (atm.Pn, ET)                // ET <= nil
    p.MOVP  (atm.Pn, EP)                // EP <= nil
    p.Label (LB_error)                  // _error:
    p.STRP  (ET, 0)                     // r0 <= ET
    p.STRP  (EP, 1)                     // r1 <= EP
    p.HALT  ()                          // HALT
}

var translators = [256]func(*atm.Builder, Instr) {
    OP_int               : translate_OP_int,
    OP_str               : translate_OP_str,
    OP_bin               : translate_OP_bin,
    OP_size              : translate_OP_size,
    OP_type              : translate_OP_type,
    OP_seek              : translate_OP_seek,
    OP_deref             : translate_OP_deref,
    OP_map_next          : translate_OP_map_next,
    OP_map_begin         : translate_OP_map_begin,
    OP_map_set_i8        : translate_OP_map_set_i8,
    OP_map_set_i16       : translate_OP_map_set_i16,
    OP_map_set_i32       : translate_OP_map_set_i32,
    OP_map_set_i64       : translate_OP_map_set_i64,
    OP_map_set_str       : translate_OP_map_set_str,
    OP_map_set_bool      : translate_OP_map_set_bool,
    OP_map_set_double    : translate_OP_map_set_double,
    OP_map_set_pointer   : translate_OP_map_set_pointer,
    OP_map_is_done       : translate_OP_map_is_done,
    OP_list_next         : translate_OP_list_next,
    OP_list_begin        : translate_OP_list_begin,
    OP_list_is_done      : translate_OP_list_is_done,
    OP_struct_skip       : translate_OP_struct_skip,
    OP_struct_ignore     : translate_OP_struct_ignore,
    OP_struct_bitmap     : translate_OP_struct_bitmap,
    OP_struct_switch     : translate_OP_struct_switch,
    OP_struct_require    : translate_OP_struct_require,
    OP_struct_is_stop    : translate_OP_struct_is_stop,
    OP_struct_read_tag   : translate_OP_struct_read_tag,
    OP_struct_mark_tag   : translate_OP_struct_mark_tag,
    OP_struct_check_type : translate_OP_struct_check_type,
    OP_make_state        : translate_OP_make_state,
    OP_drop_state        : translate_OP_drop_state,
    OP_construct         : translate_OP_construct,
    OP_defer             : translate_OP_defer,
    OP_goto              : translate_OP_goto,
    OP_halt              : translate_OP_halt,
}

func translate_OP_int(p *atm.Builder, v Instr) {
    switch v.Iv {
        case 1  : p.LB(IP, TR);                  p.SB(TR, WP); p.ADDPI(IP, 1, IP); p.SUBI(IL, 1, IL)   // *WP <= *TP++
        case 2  : p.LW(IP, TR); p.SWAPW(TR, TR); p.SW(TR, WP); p.ADDPI(IP, 2, IP); p.SUBI(IL, 2, IL)   // *WP <= bswap16(*TP++)
        case 4  : p.LL(IP, TR); p.SWAPL(TR, TR); p.SL(TR, WP); p.ADDPI(IP, 4, IP); p.SUBI(IL, 4, IL)   // *WP <= bswap32(*TP++)
        case 8  : p.LQ(IP, TR); p.SWAPQ(TR, TR); p.SQ(TR, WP); p.ADDPI(IP, 8, IP); p.SUBI(IL, 8, IL)   // *WP <= bswap64(*TP++)
        default : panic("can only convert 1, 2, 4 or 8 bytes at a time")
    }
}

func translate_OP_str(p *atm.Builder, _ Instr) {

}

func translate_OP_bin(p *atm.Builder, _ Instr) {

}

func translate_OP_size(p *atm.Builder, v Instr) {
    p.IQ    (v.Iv, TR)                  // TR <= v.Iv
    p.BLTU  (IL, TR, LB_eof)            // if IL < TR then GOTO _eof
}

func translate_OP_type(p *atm.Builder, _ Instr) {

}

func translate_OP_seek(p *atm.Builder, _ Instr) {

}

func translate_OP_deref(p *atm.Builder, _ Instr) {

}

func translate_OP_map_next(p *atm.Builder, _ Instr) {

}

func translate_OP_map_begin(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_i8(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_i16(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_i32(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_i64(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_str(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_bool(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_double(p *atm.Builder, _ Instr) {

}

func translate_OP_map_set_pointer(p *atm.Builder, _ Instr) {

}

func translate_OP_map_is_done(p *atm.Builder, _ Instr) {

}

func translate_OP_list_next(p *atm.Builder, _ Instr) {

}

func translate_OP_list_begin(p *atm.Builder, _ Instr) {

}

func translate_OP_list_is_done(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_skip(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_ignore(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_bitmap(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_switch(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_require(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_is_stop(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_read_tag(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_mark_tag(p *atm.Builder, _ Instr) {

}

func translate_OP_struct_check_type(p *atm.Builder, _ Instr) {

}

func translate_OP_make_state(p *atm.Builder, _ Instr) {

}

func translate_OP_drop_state(p *atm.Builder, _ Instr) {

}

func translate_OP_construct(p *atm.Builder, _ Instr) {

}

func translate_OP_defer(p *atm.Builder, _ Instr) {

}

func translate_OP_goto(p *atm.Builder, _ Instr) {

}

func translate_OP_halt(p *atm.Builder, _ Instr) {

}
