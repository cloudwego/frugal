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
    `github.com/cloudwego/frugal/internal/atm`
)

/** Function Prototype
 *
 *      func(buf *[]byte, p unsafe.Pointer, rs *RuntimeState) (err error)
 */

/** Register Allocations
 *
 *      P3      Current Working Pointer
 *      P4      Output Buffer Pointer
 *      P5      Runtime State Pointer
 *      P6      Error Type Pointer
 *      P7      Error Value Pointer
 *
 *      R5      Output Buffer Length
 *      R6      Output Buffer Capacity
 *      R7      State Index
 */

const (
    WP = atm.P3
    RP = atm.P4
    RS = atm.P5
    ET = atm.P6
    EP = atm.P7
)

const (
    RL = atm.R5
    RC = atm.R6
    ST = atm.R7
)

func Translate(s Program) atm.Program {
    p := atm.CreateProgramBuilder()
    prologue  (p)
    program   (p, s)
    epilogue  (p)
    moreSpace (p)
    return p.Build()
}

func program(p *atm.ProgramBuilder, s Program) {
    for i, v := range s {
        p.Mark(i)
        translators[v.Op()](p, v)
    }
}

func prologue(p *atm.ProgramBuilder) {
    p.MOV   (atm.Zr, ST)                // ST <=  0
    p.MOVRP (atm.Zr, ET)                // ET <=  0
    p.MOVRP (atm.Zr, EP)                // EP <=  0
    p.IB    (8, atm.R1)                 // R1 <=  8
    p.LP    (atm.SP, atm.P0)            // P0 <= *SP
    p.LP    (atm.P0, RP)                // RP <= *P0
    p.ADDP  (atm.P0, atm.R1, atm.P0)    // P0 <=  P0 + R1
    p.LQ    (atm.P0, RL)                // RL <= *P0
    p.ADDP  (atm.P0, atm.R1, atm.P0)    // P0 <=  P0 + R1
    p.LQ    (atm.P0, RC)                // RC <= *P0
    p.ADDP  (atm.SP, atm.R1, atm.P0)    // P0 <=  SP + R1
    p.LP    (atm.P0, WP)                // WP <= *P0
    p.ADDP  (atm.P0, atm.R1, atm.P0)    // P0 <=  P0 + R1
    p.LP    (atm.P0, RS)                // RS <= *P0
}

func epilogue(p *atm.ProgramBuilder) {
    p.IB    (24, atm.R1)                //  R1 <= 24
    p.ADDP  (atm.SP, atm.R1, atm.P0)    //  P0 <= SP + R1
    p.SP    (ET, atm.P0)                // *P0 <= ET
    p.IB    (8, atm.R1)                 //  R1 <= 8
    p.ADDP  (atm.P0, atm.R1, atm.P0)    //  P0 <= P0 + R1
    p.SP    (EP, atm.P0)                // *P0 <= EP
    p.RET   ()                          // return
}

func moreSpace(p *atm.ProgramBuilder) {

}

func checkSize(p *atm.ProgramBuilder, n int) {

}

var translators = [...]func(*atm.ProgramBuilder, Instr) {
    OP_i8             : translate_OP_i8,
    OP_i16            : translate_OP_i16,
    OP_i32            : translate_OP_i32,
    OP_i64            : translate_OP_i64,
    OP_double         : translate_OP_double,
    OP_binary         : translate_OP_binary,
    OP_bool           : translate_OP_bool,
    OP_goto           : translate_OP_goto,
    OP_offset         : translate_OP_offset,
    OP_follow         : translate_OP_follow,
    OP_map_begin      : translate_OP_map_begin,
    OP_map_check_key  : translate_OP_map_check_key,
    OP_map_value_next : translate_OP_map_value_next,
    OP_list_begin     : translate_OP_list_begin,
    OP_list_advance   : translate_OP_list_advance,
    OP_field_stop     : translate_OP_field_stop,
    OP_field_begin    : translate_OP_field_begin,
    OP_defer          : translate_OP_defer,
    OP_save           : translate_OP_save,
    OP_drop           : translate_OP_drop,
}

func translate_OP_i8(p *atm.ProgramBuilder, _ Instr) {

}

func translate_OP_i16(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_i32(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_i64(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_double(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_binary(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_bool(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_goto(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_offset(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_follow(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_begin(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_check_key(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_value_next(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_list_begin(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_list_advance(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_field_stop(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_field_begin(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_defer(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_save(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_drop(p *atm.ProgramBuilder, v Instr) {

}
