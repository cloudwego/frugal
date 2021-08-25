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
    `github.com/cloudwego/frugal/internal/utils`
)

/** Function Prototype
 *
 *      func(iov IoVec, p unsafe.Pointer, rs *RuntimeState) (err error)
 */

/** Register Allocations
 *
 *      P1      IoVec Itab Pointer
 *      P2      IoVec Value Pointer
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
    VT = atm.P1
    VP = atm.P2
    WP = atm.P3
    RP = atm.P4
    RS = atm.P5
    ET = atm.P6     // may also be used as a temporary pointer register
    EP = atm.P7     // may also be used as a temporary pointer register
)

const (
    RL = atm.R5
    RC = atm.R6
    ST = atm.R7
)

const (
    TP = atm.P0
    TR = atm.R0
)

const (
    LB_error = "_error"
)

func Translate(s Program) atm.Program {
    p := atm.CreateProgramBuilder()
    prologue (p)
    program  (p, s)
    epilogue (p)
    return p.Build()
}

func program(p *atm.ProgramBuilder, s Program) {
    for i, v := range s {
        p.Mark(i)
        translators[v.Op()](p, v)
    }
}

func prologue(p *atm.ProgramBuilder) {
    p.MOV   (atm.Rz, ST)                // ST <= 0
    p.MOV   (atm.Rz, RC)                // RC <= 0
    p.MOV   (atm.Rz, RL)                // RL <= 0
    p.MOVP  (atm.Pn, RP)                // RP <= nil
    p.LDAP  (0, VT)                     // VT <= a0
    p.LDAP  (1, VP)                     // VP <= a1
    p.LDAP  (2, WP)                     // WP <= a2
    p.LDAP  (3, RS)                     // RS <= a3
}

func epilogue(p *atm.ProgramBuilder) {
    p.BEQ   (RL, atm.Rz, "_nobuf")      // if RL == 0 then GOTO _nobuf
    p.GCALL (utils.IoVecPut).           // GCALL IoVecPut:
      A0    (VT).                       //     p.itab <= VT
      A1    (VP).                       //     p.data <= VP
      A2    (RP).                       //     v.ptr  <= RP
      A3    (RL).                       //     v.len  <= RL
      A4    (RC)                        //     v.cap  <= RC
    p.Label ("_nobuf")                  // _nobuf:
    p.MOVRP (atm.Rz, ET)                // ET <= 0
    p.MOVRP (atm.Rz, EP)                // EP <= 0
    p.Label (LB_error)                  // _error:
    p.STRP  (ET, 0)                     // r0 <= ET
    p.STRP  (EP, 1)                     // r1 <= EP
    p.RET   ()                          // return
}

var translators = [...]func(*atm.ProgramBuilder, Instr) {
    OP_byte        : translate_OP_byte,
    OP_word        : translate_OP_word,
    OP_long        : translate_OP_long,
    OP_quad        : translate_OP_quad,
    OP_size        : translate_OP_size,
    OP_sint        : translate_OP_sint,
    OP_vstr        : translate_OP_vstr,
    OP_seek        : translate_OP_seek,
    OP_deref       : translate_OP_deref,
    OP_defer       : translate_OP_defer,
    OP_map_end     : translate_OP_map_end,
    OP_map_key     : translate_OP_map_key,
    OP_map_value   : translate_OP_map_value,
    OP_map_begin   : translate_OP_map_begin,
    OP_map_if_end  : translate_OP_map_if_end,
    OP_list_end    : translate_OP_list_end,
    OP_list_next   : translate_OP_list_next,
    OP_list_begin  : translate_OP_list_begin,
    OP_list_if_end : translate_OP_list_if_end,
    OP_goto        : translate_OP_goto,
    OP_if_nil      : translate_OP_if_nil,
}

func translate_OP_byte(p *atm.ProgramBuilder, v Instr) {
    p.IB    (1, TR)                     //  TR <= 1
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADD   (RL, TR, RL)                //  RL <= RL + TR
    p.IB    (int8(v.Iv()), TR)          //  TR <= v.Iv()
    p.SB    (TR, TP)                    // *TP <= TR
}

func translate_OP_word(p *atm.ProgramBuilder, v Instr) {
    p.IB    (2, TR)                     //  TR <= 2
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADD   (RL, TR, RL)                //  RL <= RL + TR
    p.IW    (int16(v.Iv()), TR)         //  TR <= v.Iv()
    p.SWAP2 (TR, TR)                    //  TR <= bswap16(TR)
    p.SW    (TR, TP)                    // *TP <= TR
}

func translate_OP_long(p *atm.ProgramBuilder, v Instr) {
    p.IB    (4, TR)                     //  TR <= 4
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADD   (RL, TR, RL)                //  RL <= RL + TR
    p.IL    (int32(v.Iv()), TR)         //  TR <= v.Iv()
    p.SWAP4 (TR, TR)                    //  TR <= bswap32(TR)
    p.SL    (TR, TP)                    // *TP <= TR
}

func translate_OP_quad(p *atm.ProgramBuilder, v Instr) {
    p.IB    (8, TR)                     //  TR <= 8
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADD   (RL, TR, RL)                //  RL <= RL + TR
    p.IQ    (v.Iv(), TR)                //  TR <= v.Iv()
    p.SWAP8 (TR, TR)                    //  TR <= bswap64(TR)
    p.SQ    (TR, TP)                    // *TP <= TR
}

func translate_OP_size(p *atm.ProgramBuilder, v Instr) {
    p.IQ    (v.Iv(), TR)                // TR <= v.Iv()
    p.GCALL (utils.IoVecAdd).           // GCALL IoVecAdd:
      A0    (VT).                       //     p.itab  <= VT
      A1    (VP).                       //     p.data  <= VP
      A2    (TR).                       //     n       <= TR
      A3    (RP).                       //     v.ptr   <= RP
      A4    (RL).                       //     v.len   <= RL
      A5    (RC).                       //     v.cap   <= RC
      R0    (RP).                       //     ret.ptr => RP
      R1    (RL).                       //     ret.len => RL
      R2    (RC)                        //     ret.cap => RC
}

func translate_OP_sint(p *atm.ProgramBuilder, v Instr) {
    p.IQ    (v.Iv(), TR)                // TR <= v.Iv()
    p.ADDP  (RP, RL, TP)                // TP <= RP + RL
    p.ADD   (RL, TR, RL)                // RL <= RL + TR

    /* check for copy size */
    switch v.Iv() {
        case 1  : p.LB(WP, TR);                  p.SB(TR, TP)   // *TP <= *WP
        case 2  : p.LW(WP, TR); p.SWAP2(TR, TR); p.SW(TR, TP)   // *TP <= bswap16(*WP)
        case 4  : p.LL(WP, TR); p.SWAP4(TR, TR); p.SL(TR, TP)   // *TP <= bswap32(*WP)
        case 8  : p.LQ(WP, TR); p.SWAP8(TR, TR); p.SQ(TR, TP)   // *TP <= bswap64(*WP)
        default : panic("can only convert 1, 2, 4 or 8 bytes at a time")
    }
}

func translate_OP_vstr(p *atm.ProgramBuilder, _ Instr) {
    p.IB    (8, TR)                     // TR <=  8
    p.LP    (WP, TP)                    // TP <= *WP
    p.ADDP  (WP, TR, EP)                // EP <=  WP + TR
    p.LQ    (EP, TR)                    // TR <= *EP
    p.GCALL (utils.IoVecCat).           // GCALL IoVecCat:
      A0    (VT).                       //     p.itab <= VT
      A1    (VP).                       //     p.data <= VP
      A2    (RP).                       //     v.ptr  <= RP
      A3    (RL).                       //     v.len  <= RL
      A4    (RC).                       //     v.cap  <= RC
      A5    (TP).                       //     w.ptr  <= TP
      A6    (TR).                       //     w.len  <= TR
      A7    (TR)                        //     w.cap  <= TR
    p.MOV   (atm.Rz, RC)                // RC <=  0
    p.MOV   (atm.Rz, RL)                // RL <=  0
    p.MOVP  (atm.Pn, RP)                // RP <=  nil
}

func translate_OP_seek(p *atm.ProgramBuilder, v Instr) {
    p.IQ    (v.Iv(), TR)                // TR <= v.Iv()
    p.ADDP  (WP, TR, WP)                // WP <= WP + TR
}

func translate_OP_deref(p *atm.ProgramBuilder, _ Instr) {
    p.LP    (WP, WP)                    // WP <= *WP
}

func translate_OP_defer(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_end(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_key(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_value(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_begin(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_map_if_end(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_list_end(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_list_next(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_list_begin(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_list_if_end(p *atm.ProgramBuilder, v Instr) {

}

func translate_OP_goto(p *atm.ProgramBuilder, v Instr) {
    p.JALI  (v.Iv(), atm.Pn)            // GOTO @v.Iv()
}

func translate_OP_if_nil(p *atm.ProgramBuilder, v Instr) {
    p.LQ    (WP, TR)                    // TR <= *WP
    p.BNE   (TR, atm.Rz, "_nz_{n}")     // if TR != 0 then GOTO _nz_{n}
    p.JALI  (v.Iv(), atm.Pn)            // JALI v.Iv()
    p.Label ("_nz_{n}")                 // _nz_{n}:
}
