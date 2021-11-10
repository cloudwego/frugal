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
    `unsafe`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/cloudwego/frugal/internal/utils`
)

/** Function Prototype
 *
 *      func(iov IoVec, p unsafe.Pointer, rs *RuntimeState, st int) (err error)
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
    UR = atm.R1
)

const (
    LB_halt     = "_halt"
    LB_error    = "_error"
    LB_overflow = "_overflow"
)

var (
    _F_encode   func(vt *rt.GoType, iov frugal.IoVec, p unsafe.Pointer, rs *RuntimeState, st int) error
    _E_overflow error
)

func init() {
    _F_encode   = encode
    _E_overflow = fmt.Errorf("frugal: encoder stack overflow")
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
        translators[v.Op()](p, v)
    }
}

func prologue(p *atm.Builder) {
    p.MOV   (atm.Rz, RC)                // RC <= 0
    p.MOV   (atm.Rz, RL)                // RL <= 0
    p.MOVP  (atm.Pn, RP)                // RP <= nil
    p.LDAP  (2, WP)                     // WP <= a2
    p.LDAP  (3, RS)                     // RS <= a3
    p.LDAQ  (4, ST)                     // ST <= a4
    p.ADDP  (RS, ST, RS)                // RS <= RS + ST
}

func epilogue(p *atm.Builder) {
    p.Label (LB_halt)                   // _halt:
    p.BEQ   (RL, atm.Rz, "_nobuf")      // if RL == 0 then GOTO _nobuf
    p.LDAP  (0, ET)                     // ET <= a0
    p.LDAP  (1, EP)                     // EP <= a1
    p.ICALL (ET, EP, utils.IoVecPut).   // ICALL IoVec.Put:
      A0    (RP).                       //     v.ptr  <= RP
      A1    (RL).                       //     v.len  <= RL
      A2    (RC)                        //     v.cap  <= RC
    p.Label ("_nobuf")                  // _nobuf:
    p.MOVP  (atm.Pn, ET)                // ET <= nil
    p.MOVP  (atm.Pn, EP)                // EP <= nil
    p.Label (LB_error)                  // _error:
    p.STRP  (ET, 0)                     // r0 <= ET
    p.STRP  (EP, 1)                     // r1 <= EP
    p.HALT  ()                          // HALT
}

var translators = [256]func(*atm.Builder, Instr) {
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
    OP_map_next    : translate_OP_map_next,
    OP_map_value   : translate_OP_map_value,
    OP_map_begin   : translate_OP_map_begin,
    OP_map_if_end  : translate_OP_map_if_end,
    OP_list_decr   : translate_OP_list_decr,
    OP_list_begin  : translate_OP_list_begin,
    OP_list_if_end : translate_OP_list_if_end,
    OP_goto        : translate_OP_goto,
    OP_if_nil      : translate_OP_if_nil,
    OP_make_state  : translate_OP_make_state,
    OP_drop_state  : translate_OP_drop_state,
    OP_halt        : translate_OP_halt,
}

func translate_OP_byte(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 1, RL)                 //  RL <= RL + 1
    p.IB    (int8(v.Iv()), TR)          //  TR <= v.Iv()
    p.SB    (TR, TP)                    // *TP <= TR
}

func translate_OP_word(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 2, RL)                 //  RL <= RL + 2
    p.IW    (bswap16(v.Iv()), TR)       //  TR <= bswap16(v.Iv())
    p.SW    (TR, TP)                    // *TP <= TR
}

func translate_OP_long(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <= RL + 4
    p.IL    (bswap32(v.Iv()), TR)       //  TR <= bswap32(v.Iv())
    p.SL    (TR, TP)                    // *TP <= TR
}

func translate_OP_quad(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 8, RL)                 //  RL <= RL + 8
    p.IQ    (bswap64(v.Iv()), TR)       //  TR <= bswap64(v.Iv())
    p.SQ    (TR, TP)                    // *TP <= TR
}

func translate_OP_size(p *atm.Builder, v Instr) {
    p.ADDI  (RL, v.Iv(), TR)            // TR <= RL + v.Iv()
    p.BGEU  (RC, TR, "_noalloc_{n}")    // if RC >= TR then GOTO _noalloc_{n}
    p.IQ    (v.Iv(), TR)                // TR <= v.Iv()
    p.LDAP  (0, ET)                     // ET <= a0
    p.LDAP  (1, EP)                     // EP <= a1
    p.ICALL (ET, EP, utils.IoVecAdd).   // ECALL IoVec.Add:
      A0    (TR).                       //     n       <= TR
      A1    (RP).                       //     v.ptr   <= RP
      A2    (RL).                       //     v.len   <= RL
      A3    (RC).                       //     v.cap   <= RC
      R0    (RP).                       //     ret.ptr => RP
      R1    (RL).                       //     ret.len => RL
      R2    (RC)                        //     ret.cap => RC
    p.Label ("_noalloc_{n}")            // _noalloc_{n}:
}

func translate_OP_sint(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                // TP <= RP + RL
    p.ADDI  (RL, v.Iv(), RL)            // RL <= RL + v.Iv()

    /* check for copy size */
    switch v.Iv() {
        case 1  : p.LB(WP, TR);                  p.SB(TR, TP)   // *TP <= *WP
        case 2  : p.LW(WP, TR); p.SWAPW(TR, TR); p.SW(TR, TP)   // *TP <= bswap16(*WP)
        case 4  : p.LL(WP, TR); p.SWAPL(TR, TR); p.SL(TR, TP)   // *TP <= bswap32(*WP)
        case 8  : p.LQ(WP, TR); p.SWAPQ(TR, TR); p.SQ(TR, TP)   // *TP <= bswap64(*WP)
        default : panic("can only convert 1, 2, 4 or 8 bytes at a time")
    }
}

func translate_OP_vstr(p *atm.Builder, _ Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <=  RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <=  RL + 4
    p.ADDPI (WP, 8, EP)                 //  EP <=  WP + 8
    p.LQ    (EP, TR)                    //  TR <= *EP
    p.SWAPL (TR, UR)                    //  UR <=  bswap32(TR)
    p.SL    (UR, TP)                    // *TP <=  UR
    p.LP    (WP, TP)                    //  TP <= *WP
    p.LDAP  (0, ET)                     //  ET <=  a0
    p.LDAP  (1, EP)                     //  EP <=  a1
    p.ICALL (ET, EP, utils.IoVecCat).   //  ICALL IoVec.Cat:
      A0    (RP).                       //     v.ptr <= RP
      A1    (RL).                       //     v.len <= RL
      A2    (RC).                       //     v.cap <= RC
      A3    (TP).                       //     w.ptr <= TP
      A4    (TR).                       //     w.len <= TR
      A5    (TR)                        //     w.cap <= TR
    p.MOV   (atm.Rz, RC)                //  RC <=  0
    p.MOV   (atm.Rz, RL)                //  RL <=  0
    p.MOVP  (atm.Pn, RP)                //  RP <=  nil
}

func translate_OP_seek(p *atm.Builder, v Instr) {
    p.ADDPI (WP, v.Iv(), WP)            // WP <= WP + v.Iv()
}

func translate_OP_deref(p *atm.Builder, _ Instr) {
    p.LP    (WP, WP)                    // WP <= *WP
}

func translate_OP_defer(p *atm.Builder, v Instr) {
    p.BEQ   (RL, atm.Rz, "_empty_{n}")  // if RL == 0 then GOTO _empty_{n}
    p.LDAP  (0, ET)                     // ET <= a0
    p.LDAP  (1, EP)                     // EP <= a1
    p.ICALL (ET, EP, utils.IoVecPut).   // ICALL IoVec.Put:
      A0    (RP).                       //     v.ptr  <= RP
      A1    (RL).                       //     v.len  <= RL
      A2    (RC)                        //     v.cap  <= RC
    p.MOV   (atm.Rz, RC)                // RC <= 0
    p.MOV   (atm.Rz, RL)                // RL <= 0
    p.MOVP  (atm.Pn, RP)                // RP <= nil
    p.Label ("_empty_{n}")              // _empty_{n}:
    p.LDAP  (0, ET)                     // ET <= a0
    p.LDAP  (1, EP)                     // EP <= a1
    p.IP    (v.Vt(), TP)                // TP <= v.Vt()
    p.SUBP  (RS, ST, RS)                // RS <= RS - ST
    p.GCALL (_F_encode).                // GCALL encode:
      A0    (TP).                       //     vt       <= TP
      A1    (ET).                       //     iov.itab <= ET
      A2    (EP).                       //     iov.data <= EP
      A3    (WP).                       //     p        <= WP
      A4    (RS).                       //     rs       <= RS
      A5    (ST).                       //     st       <= ST
      R0    (ET).                       //     err.type => ET
      R1    (EP)                        //     err.data => EP
    p.ADDP  (RS, ST, RS)                // RS <= RS + ST
}

func translate_OP_map_end(p *atm.Builder, _ Instr) {
    p.SUBPI (RS, MiWpSize, TP)          //  TP <=  RS - MiWpSize
    p.LP    (TP, EP)                    //  EP <= *TP
    p.SP    (atm.Pn, TP)                // *TP <=  nil
    p.GCALL (MapEndIterator).A0(EP)     //  GCALL MapEndIterator(it: EP)
}

func translate_OP_map_key(p *atm.Builder, _ Instr) {
    p.SUBPI (RS, MiWpSize, TP)          // TP <=  RS - MiWpSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.LP    (TP, WP)                    // WP <= *TP
}

func translate_OP_map_next(p *atm.Builder, _ Instr) {
    p.SUBPI (RS, MiWpSize, TP)          // TP <=  RS - MiWpSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.GCALL (mapiternext).A0(TP)        // GCALL mapiternext(it: TP)
}

func translate_OP_map_value(p *atm.Builder, _ Instr) {
    p.SUBPI (RS, MiWpSize, TP)          // TP <=  RS - MiWpSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.ADDPI (TP, 8, TP)                 // TP <=  TP + 8
    p.LP    (TP, WP)                    // WP <= *TP
}

func translate_OP_map_begin(p *atm.Builder, v Instr) {
    p.LP    (WP, EP)                    //  EP <= *WP
    p.ADDP  (RP, RL, TP)                //  TP <=  RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <=  RL + 4
    p.LQ    (EP, TR)                    //  TR <= *EP
    p.SWAPL (TR, TR)                    //  TR <=  bswap32(TR)
    p.SL    (TR, TP)                    // *TP <=  TR
    p.IP    (v.Vt(), ET)                //  ET <=  v.Vt()
    p.GCALL (MapBeginIterator).         //  GCALL MapBeginIterator:
      A0    (ET).                       //      vt <= ET
      A1    (EP).                       //      vp <= EP
      R0    (TP)                        //      it => TP
    p.SUBPI (RS, MiWpSize, EP)          //  EP <=  RS - MiWpSize
    p.SP    (TP, EP)                    // *EP <=  TP
}

func translate_OP_map_if_end(p *atm.Builder, v Instr) {
    p.SUBPI (RS, MiWpSize, TP)          // TP <=  RS - MiWpSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.LQ    (TP, TR)                    // TR <= *TP
    p.BEQ   (TR, atm.Rz, p.At(v.To()))  // if TR == 0 then GOTO @v.To()
}

func translate_OP_list_decr(p *atm.Builder, _ Instr) {
    p.SUBPI (RS, StateSize, TP)         //  TP <=  RS - StateSize
    p.LQ    (TP, TR)                    //  TR <= *TP
    p.SUBI  (TR, 1, TR)                 //  TR <=  TR - 1
    p.SQ    (TR, TP)                    // *TP <=  TR
}

func translate_OP_list_begin(p *atm.Builder, _ Instr) {
    p.ADDPI (WP, 8, TP)                 //  TP <=  WP + 8
    p.LQ    (TP, TR)                    //  TR <= *TP
    p.SUBPI (RS, StateSize, TP)         //  TP <=  RS - StateSize
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.LP    (WP, WP)                    //  WP <=  WP
    p.ADDP  (RP, RL, TP)                //  TP <=  RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <=  RL + 4
    p.SWAPL (TR, TR)                    //  TR <=  bswap32(TR)
    p.SL    (TR, TP)                    // *TP <=  TR
}

func translate_OP_list_if_end(p *atm.Builder, v Instr) {
    p.SUBPI (RS, StateSize, TP)         // TP <=  RS - StateSize
    p.LQ    (TP, TR)                    // TR <= *TP
    p.BEQ   (TR, atm.Rz, p.At(v.To()))  // if TR == 0 then GOTO @v.To()
}

func translate_OP_goto(p *atm.Builder, v Instr) {
    p.JAL   (p.At(v.To()), atm.Pn)      // GOTO @v.To()
}

func translate_OP_if_nil(p *atm.Builder, v Instr) {
    p.LQ    (WP, TR)                    // TR <= *WP
    p.BEQ   (TR, atm.Rz, p.At(v.To()))  // if TR == 0 then GOTO @v.To()
}

func translate_OP_make_state(p *atm.Builder, _ Instr) {
    p.IQ    (StateMax, TR)              //  TR <= StateMax
    p.BGEU  (ST, TR, LB_overflow)       //  if ST >= TR then GOTO _overflow
    p.ADDPI (RS, LnMiSize, RS)          //  RS <= RS + LnMiSize
    p.SP    (WP, RS)                    // *RS <= WP
    p.ADDPI (RS, WpSize, RS)            //  RS <= RS + WpSize
    p.ADDI  (ST, StateSize, ST)         //  ST <= ST + StateSize
}

func translate_OP_drop_state(p *atm.Builder, _ Instr) {
    p.SUBI  (ST, StateSize, ST)         //  ST <=  ST - StateSize
    p.SUBPI (RS, WpSize, RS)            //  RS <=  RS - WpSize
    p.LP    (RS, WP)                    //  WP <= *RS
    p.SP    (atm.Pn, RS)                // *RS <=  nil
    p.SUBPI (RS, MiSize, RS)            //  RS <=  RS - MiSize
    p.SP    (atm.Pn, RS)                // *RS <=  nil
    p.SUBPI (RS, LnSize, RS)            //  RS <=  RS - LnSize
    p.SQ    (atm.Rz, RS)                // *RS <=  0
}

func translate_OP_halt(p *atm.Builder, _ Instr) {
    p.JAL   (LB_halt, atm.Pn)           // GOTO _halt
}
