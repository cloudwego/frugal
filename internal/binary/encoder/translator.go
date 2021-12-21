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
    `os`
    `reflect`

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/binary/defs`
    `github.com/cloudwego/frugal/internal/utils`
)

/** Function Prototype
 *
 *      func (
 *          buf unsafe.Pointer,
 *          len int,
 *          mem iov.BufferWriter,
 *          p   unsafe.Pointer,
 *          rs  *RuntimeState,
 *          st  int,
 *      ) (
 *          pos int,
 *          err error,
 *      )
 */

const (
    ARG_buf      = 0
    ARG_len      = 1
    ARG_mem_itab = 2
    ARG_mem_data = 3
    ARG_p        = 4
    ARG_rs       = 5
    ARG_st       = 6
)

const (
    RET_pos      = 0
    RET_err_itab = 1
    RET_err_data = 2
)

/** Register Allocations
 *
 *      P1      Current Working Pointer
 *      P2      Output Buffer Pointer
 *      P3      Runtime State Pointer
 *      P4      Error Type Pointer
 *      P5      Error Value Pointer
 *
 *      R2      Output Buffer Length
 *      R3      Output Buffer Capacity
 *      R4      State Index
 */

const (
    WP = atm.P1
    RP = atm.P2
    RS = atm.P3
    ET = atm.P4     // may also be used as a temporary pointer register
    EP = atm.P5     // may also be used as a temporary pointer register
)

const (
    RL = atm.R2
    RC = atm.R3
    ST = atm.R4
)

const (
    TP = atm.P0
    TR = atm.R0
    UR = atm.R1
)

const (
    LB_halt       = "_halt"
    LB_error      = "_error"
    LB_nomem      = "_nomem"
    LB_overflow   = "_overflow"
    LB_duplicated = "_duplicated"
)

var (
    _N_page       = int64(os.Getpagesize())
    _E_nomem      = fmt.Errorf("frugal: buffer is too small")
    _E_overflow   = fmt.Errorf("frugal: encoder stack overflow")
    _E_duplicated = fmt.Errorf("frugal: duplicated element within sets")
)

func Translate(s Program) atm.Program {
    p := atm.CreateBuilder()
    prologue (p)
    program  (p, s)
    epilogue (p)
    errors   (p)
    return p.Build()
}

func errors(p *atm.Builder) {
    p.Label (LB_nomem)                  // _nomem:
    p.MOV   (UR, RL)                    // RL <= UR
    p.IP    (&_E_nomem, TP)             // TP <= &_E_overflow
    p.JAL   ("_basic_error", atm.Pn)    // GOTO _basic_error
    p.Label (LB_overflow)               // _overflow:
    p.IP    (&_E_overflow, TP)          // TP <= &_E_overflow
    p.JAL   ("_basic_error", atm.Pn)    // GOTO _basic_error
    p.Label (LB_duplicated)             // _duplicated:
    p.IP    (&_E_duplicated, TP)        // TP <= &_E_duplicated
    p.Label ("_basic_error")            // _basic_error:
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
    p.LDAP  (ARG_buf, RP)               // RP <= ARG.buf
    p.LDAQ  (ARG_len, RC)               // RC <= ARG.len
    p.LDAP  (ARG_p, WP)                 // WP <= ARG.p
    p.LDAP  (ARG_rs, RS)                // RS <= ARG.rs
    p.LDAQ  (ARG_st, ST)                // ST <= ARG.st
    p.MOV   (atm.Rz, RL)                // RL <= 0
}

func epilogue(p *atm.Builder) {
    p.Label (LB_halt)                   // _halt:
    p.MOVP  (atm.Pn, ET)                // ET <= nil
    p.MOVP  (atm.Pn, EP)                // EP <= nil
    p.Label (LB_error)                  // _error:
    p.STRQ  (RL, RET_pos)               // RL => RET.pos
    p.STRP  (ET, RET_err_itab)          // ET => RET.err.itab
    p.STRP  (EP, RET_err_data)          // EP => RET.err.data
    p.HALT  ()                          // HALT
}

var translators = [256]func(*atm.Builder, Instr) {
    OP_size_check  : translate_OP_size_check,
    OP_size_const  : translate_OP_size_const,
    OP_size_dyn    : translate_OP_size_dyn,
    OP_size_map    : translate_OP_size_map,
    OP_size_defer  : translate_OP_size_defer,
    OP_byte        : translate_OP_byte,
    OP_word        : translate_OP_word,
    OP_long        : translate_OP_long,
    OP_quad        : translate_OP_quad,
    OP_sint        : translate_OP_sint,
    OP_length      : translate_OP_length,
    OP_memcpy_be   : translate_OP_memcpy_be,
    OP_seek        : translate_OP_seek,
    OP_deref       : translate_OP_deref,
    OP_defer       : translate_OP_defer,
    OP_map_len     : translate_OP_map_len,
    OP_map_end     : translate_OP_map_end,
    OP_map_key     : translate_OP_map_key,
    OP_map_next    : translate_OP_map_next,
    OP_map_value   : translate_OP_map_value,
    OP_map_begin   : translate_OP_map_begin,
    OP_map_if_end  : translate_OP_map_if_end,
    OP_list_decr   : translate_OP_list_decr,
    OP_list_begin  : translate_OP_list_begin,
    OP_list_if_end : translate_OP_list_if_end,
    OP_unique      : translate_OP_unique,
    OP_goto        : translate_OP_goto,
    OP_if_nil      : translate_OP_if_nil,
    OP_if_hasbuf   : translate_OP_if_hasbuf,
    OP_make_state  : translate_OP_make_state,
    OP_drop_state  : translate_OP_drop_state,
    OP_halt        : translate_OP_halt,
}

func translate_OP_size_check(p *atm.Builder, v Instr) {
    p.ADDI  (RL, v.Iv, UR)              // UR <= RL + v.Iv
    p.BLTU  (RC, UR, LB_nomem)          // if RC < UR then GOTO _nomem
}

func translate_OP_size_const(p *atm.Builder, v Instr) {
    p.ADDI  (RL, v.Iv, RL)              // RL <= RL + v.Iv
}

func translate_OP_size_dyn(p *atm.Builder, v Instr) {
    p.ADDPI (WP, int64(v.Uv), TP)       // TP <=  WP + v.Uv
    p.LQ    (TP, TR)                    // TR <= *TP
    p.MULI  (TR, v.Iv, TR)              // TR <=  TR * v.Iv
    p.ADD   (RL, TR, RL)                // RL <=  RL + TR
}

func translate_OP_size_map(p *atm.Builder, v Instr) {
    p.LP    (WP, TP)                    // TP <= *WP
    p.LQ    (TP, TR)                    // TR <= *TP
    p.MULI  (TR, v.Iv, TR)              // TR <=  TR * v.Iv
    p.ADD   (RL, TR, RL)                // RL <=  RL + TR
}

func translate_OP_size_defer(p *atm.Builder, v Instr) {
    p.IP    (v.Vt, TP)                  // TP <= v.Vt
    p.GCALL (F_encode).                 // GCALL encode:
      A0    (TP).                       //     vt       <= TP
      A1    (atm.Pn).                   //     buf      <= nil
      A2    (atm.Rz).                   //     len      <= 0
      A3    (atm.Pn).                   //     mem.itab <= nil
      A4    (atm.Pn).                   //     mem.data <= nil
      A5    (WP).                       //     p        <= WP
      A6    (RS).                       //     rs       <= RS
      A7    (ST).                       //     st       <= ST
      R0    (TR).                       //     pos      => TR
      R1    (ET).                       //     err.type => ET
      R2    (EP)                        //     err.data => EP
    p.BNEN  (ET, LB_error)              // if ET != nil then GOTO _error
    p.ADD   (RL, TR, RL)                // RL <= RL + TR
}

func translate_OP_byte(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 1, RL)                 //  RL <= RL + 1
    p.IB    (int8(v.Iv), TR)            //  TR <= v.Iv
    p.SB    (TR, TP)                    // *TP <= TR
}

func translate_OP_word(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 2, RL)                 //  RL <= RL + 2
    p.IW    (bswap16(v.Iv), TR)         //  TR <= bswap16(v.Iv)
    p.SW    (TR, TP)                    // *TP <= TR
}

func translate_OP_long(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <= RL + 4
    p.IL    (bswap32(v.Iv), TR)         //  TR <= bswap32(v.Iv)
    p.SL    (TR, TP)                    // *TP <= TR
}

func translate_OP_quad(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                //  TP <= RP + RL
    p.ADDI  (RL, 8, RL)                 //  RL <= RL + 8
    p.IQ    (bswap64(v.Iv), TR)         //  TR <= bswap64(v.Iv)
    p.SQ    (TR, TP)                    // *TP <= TR
}

func translate_OP_sint(p *atm.Builder, v Instr) {
    p.ADDP  (RP, RL, TP)                // TP <= RP + RL
    p.ADDI  (RL, v.Iv, RL)              // RL <= RL + v.Iv

    /* check for copy size */
    switch v.Iv {
        case 1  : p.LB(WP, TR);                  p.SB(TR, TP)   // *TP <= *WP
        case 2  : p.LW(WP, TR); p.SWAPW(TR, TR); p.SW(TR, TP)   // *TP <= bswap16(*WP)
        case 4  : p.LL(WP, TR); p.SWAPL(TR, TR); p.SL(TR, TP)   // *TP <= bswap32(*WP)
        case 8  : p.LQ(WP, TR); p.SWAPQ(TR, TR); p.SQ(TR, TP)   // *TP <= bswap64(*WP)
        default : panic("can only convert 1, 2, 4 or 8 bytes at a time")
    }
}

func translate_OP_length(p *atm.Builder, v Instr) {
    p.ADDPI (WP, v.Iv, TP)              //  TP <=  WP + v.Iv
    p.LL    (TP, TR)                    //  TR <= *TP
    p.SWAPL (TR, TR)                    //  TR <=  bswap32(TR)
    p.ADDP  (RP, RL, TP)                //  TP <=  RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <=  RL + 4
    p.SL    (TR, TP)                    // *TP <=  TR
}

func translate_OP_memcpy_1(p *atm.Builder) {
    p.IQ    (_N_page, UR)               // UR <= _N_page
    p.BGEU  (UR, TR, "_do_copy_{n}")    // if UR >= TR then GOTO _do_copy_{n}
    p.LDAP  (ARG_mem_itab, ET)          // ET <= ARG.mem.itab
    p.LDAP  (ARG_mem_data, EP)          // EP <= ARG.mem.data
    p.BEQN  (EP, "_do_copy_{n}")        // if EP == nil then GOTO _do_copy_{n}
    p.SUB   (RC, RL, UR)                // UR <= RC - RL
    p.ICALL (ET, EP, utils.FnWrite).    // ICALL BufferWriter.WriteDirect(ET:EP)
      A0    (TP).                       //     buf.ptr      <= TP
      A1    (TR).                       //     buf.len      <= TR
      A2    (TR).                       //     buf.cap      <= TR
      A3    (UR).                       //     remainingCap <= UR
      R0    (ET).                       //     err.itab     => ET
      R1    (EP)                        //     err.data     => EP
    p.BNEN  (ET, LB_error)              // if ET != nil then GOTO _error
    p.JAL   ("_done_{n}", atm.Pn)       // GOTO _done_{n}
    p.Label ("_do_copy_{n}")            // _do_copy_{n}:
    p.ADD   (RL, TR, UR)                // UR <= RL + TR
    p.BLTU  (RC, UR, LB_nomem)          // if RC < UR then GOTO _nomem
    p.ADDP  (RP, RL, EP)                // EP <= RP + RL
    p.MOV   (UR, RL)                    // RL <= UR
    p.BCOPY (TP, TR, EP)                // memcpy(EP, TP, TR)
    p.Label ("_done_{n}")               // _done_{n}:
}

func translate_OP_memcpy_be(p *atm.Builder, v Instr) {
    p.ADDPI (WP, int64(v.Uv), TP)       // TP <=  WP + v.Uv
    p.LQ    (TP, TR)                    // TR <= *TP
    p.BEQ   (TR, atm.Rz, "_done_{n}")   // if TR == 0 then GOTO _done_{n}
    p.LP    (WP, TP)                    // TP <= *WP

    /* special case: unit of a single byte */
    if v.Iv == 1 {
        translate_OP_memcpy_1(p)
        return
    }

    /* adjust the buffer length */
    p.MULI  (TR, v.Iv, UR)              // UR <= TR * v.Iv
    p.ADD   (RL, UR, UR)                // UR <= RL + UR
    p.BLTU  (RC, UR, LB_nomem)          // if RC < UR then GOTO _nomem
    p.ADDP  (RP, RL, EP)                // EP <= RP + RL
    p.MOV   (UR, RL)                    // RL <= UR
    p.Label ("_loop_{n}")               // _loop_{n}:
    p.BEQ   (TR, atm.Rz, "_done_{n}")   // if TR == 0 then GOTO _done_{n}

    /* load-swap-store sequence */
    switch v.Iv {
        case 2  : p.LW(TP, UR); p.SWAPW(UR, UR); p.SW(UR, EP)
        case 4  : p.LL(TP, UR); p.SWAPL(UR, UR); p.SL(UR, EP)
        case 8  : p.LQ(TP, UR); p.SWAPQ(UR, UR); p.SQ(UR, EP)
        default : panic("can only swap 2, 4 or 8 bytes at a time")
    }

    /* update loop counter */
    p.SUBI  (TR, 1, TR)                 // TR <= TR - 1
    p.ADDPI (TP, v.Iv, TP)              // TP <= TP + v.Iv
    p.ADDPI (EP, v.Iv, EP)              // RP <= RP + v.Iv
    p.JAL   ("_loop_{n}", atm.Pn)       // GOTO _loop_{n}
    p.Label ("_done_{n}")               // _done_{n}:
}

func translate_OP_seek(p *atm.Builder, v Instr) {
    p.ADDPI (WP, v.Iv, WP)              // WP <= WP + v.Iv
}

func translate_OP_deref(p *atm.Builder, _ Instr) {
    p.LP    (WP, WP)                    // WP <= *WP
}

func translate_OP_defer(p *atm.Builder, v Instr) {
    p.IP    (v.Vt, TP)                  // TP <= v.Vt
    p.LDAP  (ARG_mem_itab, ET)          // ET <= ARG.mem.itab
    p.LDAP  (ARG_mem_data, EP)          // EP <= ARG.mem.data
    p.SUB   (RC, RL, TR)                // TR <= RC - RL
    p.ADDP  (RP, RL, RP)                // RP <= RP + RL
    p.GCALL (F_encode).                 // GCALL encode:
      A0    (TP).                       //     vt       <= TP
      A1    (RP).                       //     buf      <= RP
      A2    (TR).                       //     len      <= TR
      A3    (ET).                       //     mem.itab <= ET
      A4    (EP).                       //     mem.data <= EP
      A5    (WP).                       //     p        <= WP
      A6    (RS).                       //     rs       <= RS
      A7    (ST).                       //     st       <= ST
      R0    (TR).                       //     pos      => TR
      R1    (ET).                       //     err.type => ET
      R2    (EP)                        //     err.data => EP
    p.BNEN  (ET, LB_error)              // if ET != nil then GOTO _error
    p.SUBP  (RP, RL, RP)                // RP <= RP - RL
    p.ADD   (RL, TR, RL)                // RL <= RL + TR
}

func translate_OP_map_len(p *atm.Builder, _ Instr) {
    p.LP    (WP, TP)                    //  TP <= *WP
    p.LL    (TP, TR)                    //  TR <= *TP
    p.SWAPL (TR, TR)                    //  TR <=  bswap32(TR)
    p.ADDP  (RP, RL, TP)                //  TP <=  RP + RL
    p.ADDI  (RL, 4, RL)                 //  RL <=  RL + 4
    p.SL    (TR, TP)                    // *TP <=  TR
}

func translate_OP_map_end(p *atm.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)                //  TP <=  RS + ST
    p.ADDPI (TP, MiOffset, TP)          //  TP <=  TP + MiOffset
    p.LP    (TP, EP)                    //  EP <= *TP
    p.SP    (atm.Pn, TP)                // *TP <=  nil
    p.GCALL (F_MapEndIterator).A0(EP)   //  GCALL MapEndIterator(it: EP)
}

func translate_OP_map_key(p *atm.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)                // TP <=  RS + ST
    p.ADDPI (TP, MiOffset, TP)          // TP <=  TP + MiOffset
    p.LP    (TP, TP)                    // TP <= *TP
    p.LP    (TP, WP)                    // WP <= *TP
}

func translate_OP_map_next(p *atm.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)                // TP <=  RS + ST
    p.ADDPI (TP, MiOffset, TP)          // TP <=  TP + MiOffset
    p.LP    (TP, TP)                    // TP <= *TP
    p.GCALL (F_mapiternext).A0(TP)      // GCALL mapiternext(it: TP)
}

func translate_OP_map_value(p *atm.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)                // TP <=  RS + ST
    p.ADDPI (TP, MiOffset, TP)          // TP <=  TP + MiOffset
    p.LP    (TP, TP)                    // TP <= *TP
    p.ADDPI (TP, 8, TP)                 // TP <=  TP + 8
    p.LP    (TP, WP)                    // WP <= *TP
}

func translate_OP_map_begin(p *atm.Builder, v Instr) {
    p.LP    (WP, EP)                    //  EP <= *WP
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (F_MapBeginIterator).       //  GCALL MapBeginIterator:
      A0    (ET).                       //      vt <= ET
      A1    (EP).                       //      vp <= EP
      R0    (TP)                        //      it => TP
    p.ADDP  (RS, ST, EP)                //  EP <=  RS + ST
    p.ADDPI (EP, MiOffset, EP)          //  EP <=  EP + MiOffset
    p.SP    (TP, EP)                    // *EP <=  TP
}

func translate_OP_map_if_end(p *atm.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)                // TP <=  RS + ST
    p.ADDPI (TP, MiOffset, TP)          // TP <=  TP + MiOffset
    p.LP    (TP, TP)                    // TP <= *TP
    p.LP    (TP, TP)                    // TP <= *TP
    p.BEQN  (TP, p.At(v.To))            // if TP == nil then GOTO @v.To
}

func translate_OP_list_decr(p *atm.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)                //  TP <=  RS + ST
    p.ADDPI (TP, LnOffset, TP)          //  TP <=  TP + LnOffset
    p.LQ    (TP, TR)                    //  TR <= *TP
    p.SUBI  (TR, 1, TR)                 //  TR <=  TR - 1
    p.SQ    (TR, TP)                    // *TP <=  TR
}

func translate_OP_list_begin(p *atm.Builder, _ Instr) {
    p.ADDPI (WP, atm.PtrSize, TP)       //  TP <=  WP + atm.PtrSize
    p.LQ    (TP, TR)                    //  TR <= *TP
    p.ADDP  (RS, ST, TP)                //  TP <=  RS + ST
    p.ADDPI (TP, LnOffset, TP)          //  TP <=  TP + LnOffset
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.LP    (WP, WP)                    //  WP <= *WP
}

func translate_OP_list_if_end(p *atm.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)                // TP <=  RS + ST
    p.ADDPI (TP, LnOffset, TP)          // TP <=  TP + LnOffset
    p.LQ    (TP, TR)                    // TR <= *TP
    p.BEQ   (TR, atm.Rz, p.At(v.To))    // if TR == 0 then GOTO @v.To
}

func translate_OP_unique(p *atm.Builder, v Instr) {
    switch v.Vt.Kind() {
        case reflect.Bool    : translate_OP_unique_b(p)
        case reflect.Int     : translate_OP_unique_int(p)
        case reflect.Int8    : translate_OP_unique_i8(p)
        case reflect.Int16   : translate_OP_unique_i16(p)
        case reflect.Int32   : translate_OP_unique_i32(p)
        case reflect.Int64   : translate_OP_unique_i64(p)
        case reflect.Float64 : translate_OP_unique_i64(p)
        case reflect.Map     : break
        case reflect.Ptr     : break
        case reflect.Slice   : break
        case reflect.String  : translate_OP_unique_str(p)
        case reflect.Struct  : break
        default              : panic("unique: invalid type: " + v.Vt.String())
    }
}

func translate_OP_unique_b(p *atm.Builder) {
    p.ADDPI (WP, atm.PtrSize, TP)       // TP <=  WP + atm.PtrSize
    p.LQ    (TP, TR)                    // TR <= *TP
    p.IB    (2, UR)                     // UR <=  2
    p.BLTU  (TR, UR, "_ok_{n}")         // if TR < UR then GOTO _ok_{n}
    p.BLTU  (UR, TR, LB_duplicated)     // if UR < TR then GOTO _duplicated
    p.LP    (WP, TP)                    // TP <= *WP
    p.LB    (TP, TR)                    // TR <= *TP
    p.ADDPI (TP, 1, TP)                 // TP <=  TP + 1
    p.LB    (TP, UR)                    // UR <= *TP
    p.BEQ   (TR, UR, LB_duplicated)     // if TR == UR then GOTO _duplicated
    p.Label ("_ok_{n}")                 // _ok_{n}:
}

func translate_OP_unique_i8(p *atm.Builder) {
    translate_OP_unique_small(p, BitmapMax8, Uint8Size, p.LB)
}

func translate_OP_unique_i16(p *atm.Builder) {
    translate_OP_unique_small(p, BitmapMax16, Uint16Size, p.LW)
}

func translate_OP_unique_small(p *atm.Builder, nb int64, dv int64, ld func(atm.PointerRegister, atm.GenericRegister) *atm.Instr) {
    p.ADDPI (WP, atm.PtrSize, TP)       //  TP <=  WP + atm.PtrSize
    p.LQ    (TP, TR)                    //  TR <= *TP
    p.IB    (2, UR)                     //  UR <=  2
    p.BLTU  (TR, UR, "_ok_{n}")         //  if TR < UR then GOTO _ok_{n}
    p.ADDPI (RS, TrOffset, ET)          //  ET <=  RS + TrOffset
    p.SQ    (ST, ET)                    // *ET <=  ST
    p.ADDPI (RS, BmOffset, ET)          //  ET <=  RS + BmOffset
    p.BZERO (nb, ET)                    //  memset(ET, 0, nb)
    p.LP    (WP, EP)                    //  EP <=  WP
    p.Label ("_loop_{n}")               // _loop_{n}:
    ld      (EP, ST)                    //  ST <= *EP
    p.SHRI  (ST, 3, UR)                 //  UR <=  ST >> 3
    p.ANDI  (ST, 0x3f, ST)              //  ST <=  ST & 0x3f
    p.ANDI  (UR, ^0x3f, UR)             //  UR <=  UR & ~0x3f
    p.ADDP  (ET, UR, TP)                //  TP <=  ET + UR
    p.LQ    (TP, UR)                    //  UR <= *TP
    p.BTS   (ST, UR, ST)                //  ST <=  test_and_set(&UR, ST)
    p.SQ    (UR, TP)                    // *TP <=  UR
    p.BNE   (ST, atm.Rz, LB_duplicated) //  if ST != 0 then GOTO _duplicated
    p.SUBI  (TR, 1, TR)                 //  TR <=  TR - 1
    p.BEQ   (TR, atm.Rz, "_done_{n}")   //  if TR == 0 then GOTO _done_{n}
    p.ADDPI (EP, dv, EP)                //  EP <=  EP + dv
    p.JAL   ("_loop_{n}", atm.Pn)       //  GOTO _loop_{n}
    p.Label ("_done_{n}")               // _done_{n}:
    p.ADDPI (RS, TrOffset, ET)          //  ET <=  RS + TrOffset
    p.LQ    (ET, ST)                    //  ST <= *ET
    p.Label ("_ok_{n}")                 // _ok_{n}:
}

func translate_OP_unique_i32(p *atm.Builder) {
    // TODO: implement OP_unique_32
}

func translate_OP_unique_i64(p *atm.Builder) {
    // TODO: implement OP_unique_64
}

func translate_OP_unique_int(p *atm.Builder) {
    switch defs.IntSize {
        case 4  : translate_OP_unique_i32(p)
        case 8  : translate_OP_unique_i64(p)
        default : panic("invalid int size")
    }
}

func translate_OP_unique_str(p *atm.Builder) {
    // TODO: implement OP_unique_str
}

func translate_OP_goto(p *atm.Builder, v Instr) {
    p.JAL   (p.At(v.To), atm.Pn)        // GOTO @v.To
}

func translate_OP_if_nil(p *atm.Builder, v Instr) {
    p.LP    (WP, TP)                    // TP <= *WP
    p.BEQN  (TP, p.At(v.To))            // if TP == nil then GOTO @v.To
}

func translate_OP_if_hasbuf(p *atm.Builder, v Instr) {
    p.BNEN  (RP, p.At(v.To))            // if RP != nil then GOTO @v.To
}

func translate_OP_make_state(p *atm.Builder, _ Instr) {
    p.IQ    (StateMax, TR)              //  TR <= StateMax
    p.BGEU  (ST, TR, LB_overflow)       //  if ST >= TR then GOTO _overflow
    p.ADDP  (RS, ST, TP)                //  TP <= RS + TP
    p.ADDPI (TP, WpOffset, TP)          //  TP <= TP + WpOffset
    p.SP    (WP, TP)                    // *TP <= WP
    p.ADDI  (ST, StateSize, ST)         //  ST <= ST + StateSize
}

func translate_OP_drop_state(p *atm.Builder, _ Instr) {
    p.SUBI  (ST, StateSize, ST)         //  ST <=  ST - StateSize
    p.ADDP  (RS, ST, TP)                //  TP <=  RS + ST
    p.ADDPI (TP, WpOffset, TP)          //  TP <=  TP + WpOffset
    p.LP    (TP, WP)                    //  WP <= *TP
    p.SP    (atm.Pn, TP)                // *TP <=  nil
}

func translate_OP_halt(p *atm.Builder, _ Instr) {
    p.JAL   (LB_halt, atm.Pn)           // GOTO _halt
}
