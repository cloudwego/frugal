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
    `github.com/cloudwego/frugal/internal/binary/defs`
    `github.com/cloudwego/frugal/internal/rt`
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
    UR = atm.R1
)

const (
    LB_eof      = "_eof"
    LB_halt     = "_halt"
    LB_type     = "_type"
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
    p.Label (LB_type)                   // _type:
    p.GCALL (error_type).               // GCALL error_type:
      A0    (UR).                       //     e        <= UR
      A1    (TR).                       //     t        <= TR
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
    OP_ctr_load          : translate_OP_ctr_load,
    OP_ctr_decr          : translate_OP_ctr_decr,
    OP_ctr_is_zero       : translate_OP_ctr_is_zero,
    OP_map_alloc         : translate_OP_map_alloc,
    OP_map_set_i8        : translate_OP_map_set_i8,
    OP_map_set_i16       : translate_OP_map_set_i16,
    OP_map_set_i32       : translate_OP_map_set_i32,
    OP_map_set_i64       : translate_OP_map_set_i64,
    OP_map_set_str       : translate_OP_map_set_str,
    OP_map_set_pointer   : translate_OP_map_set_pointer,
    OP_list_alloc        : translate_OP_list_alloc,
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
        case 1  : p.LB(IP, TR);                  p.SB(TR, WP); p.SUBI(IL, 1, IL); p.ADDPI(IP, 1, IP)    // *WP <= *IP++
        case 2  : p.LW(IP, TR); p.SWAPW(TR, TR); p.SW(TR, WP); p.SUBI(IL, 2, IL); p.ADDPI(IP, 2, IP)    // *WP <= bswap16(*IP++)
        case 4  : p.LL(IP, TR); p.SWAPL(TR, TR); p.SL(TR, WP); p.SUBI(IL, 4, IL); p.ADDPI(IP, 4, IP)    // *WP <= bswap32(*IP++)
        case 8  : p.LQ(IP, TR); p.SWAPQ(TR, TR); p.SQ(TR, WP); p.SUBI(IL, 8, IL); p.ADDPI(IP, 8, IP)    // *WP <= bswap64(*IP++)
        default : panic("can only convert 1, 2, 4 or 8 bytes at a time")
    }
}

func translate_OP_str(p *atm.Builder, _ Instr) {
    p.LQ    (IP, TR)                    //  TR <= *IP
    p.SUBI  (IL, 4, IL)                 //  IL <=  IL - 4
    p.ADDPI (IP, 4, IP)                 //  IP <=  IP + 4
    p.BLTU  (IL, TR, LB_eof)            //  if IL < TR then GOTO _eof
    p.SP    (atm.Pn, WP)                // *WP <=  nil
    p.BEQ   (TR, atm.Rz, "_empty_{n}")  //  if TR == 0 then GOTO _empty_{n}
    p.SP    (IP, WP)                    // *WP <=  IP
    p.SUB   (IL, TR, IL)                //  IL <=  IL - TR
    p.ADDP  (IP, TR, IP)                //  IP <=  IP + TR
    p.Label ("_empty_{n}")              // _empty_{n}:
    p.ADDPI (WP, 8, TP)                 //  TP <=  WP + 8
    p.SQ    (TR, TP)                    // *TP <=  TR
}

func translate_OP_bin(p *atm.Builder, v Instr) {
    translate_OP_str(p, v)
    p.ADDPI (TP, 8, TP)                 //  TP <= TP + 8
    p.SQ    (TR, TP)                    // *TP <= TR
}

func translate_OP_size(p *atm.Builder, v Instr) {
    p.IQ    (v.Iv, TR)                  // TR <= v.Iv
    p.BLTU  (IL, TR, LB_eof)            // if IL < TR then GOTO _eof
}

func translate_OP_type(p *atm.Builder, v Instr) {
    p.LB    (IP, TR)                    // TR <= *IP
    p.IB    (int8(v.Iv), UR)            // UR <=  v.Iv
    p.BNE   (TR, UR, LB_type)           // if TR != UR then GOTO _type
    p.SUBI  (IL, 1, IL)                 // IL <=  IL - 1
    p.ADDPI (IP, 1, IP)                 // IP <=  IP + 1
}

func translate_OP_seek(p *atm.Builder, v Instr) {
    p.ADDPI (WP, v.Iv, WP)              // WP <= WP + v.Iv
}

func translate_OP_deref(p *atm.Builder, v Instr) {
    p.LQ    (WP, TR)                    //  TR <= *WP
    p.BNE   (TR, atm.Rz, "_skip_{n}")   //  if TR != 0 then GOTO _skip_{n}
    p.IB    (1, UR)                     //  UR <= 1
    p.IP    (v.Vt, TP)                  //  TP <= v.Vt
    p.IQ    (int64(v.Vt.Size), TR)      //  TR <= v.Vt.Size
    p.GCALL (mallocgc).                 //  GCALL mallocgc:
      A0    (TR).                       //      size     <= TR
      A1    (TP).                       //      typ      <= TP
      A2    (UR).                       //      needzero <= UR
      R0    (TP)                        //      ret      => TP
    p.SP    (TP, WP)                    // *WP <= TP
    p.Label ("_skip_{n}")               // _skip_{n}:
    p.LP    (WP, WP)                    //  WP <= *WP
}

func translate_OP_ctr_load(p *atm.Builder, _ Instr) {
    p.LL    (IP, TR)                    //  TR <= *IP
    p.SUBI  (IL, 4, IL)                 //  IL <=  IL - 4
    p.ADDPI (IP, 4, IP)                 //  IP <=  IP + 4
    p.SQ    (TR, RS)                    // *RS <=  TR
}

func translate_OP_ctr_decr(p *atm.Builder, _ Instr) {
    p.LQ    (RS, TR)                    //  TR <= *RS
    p.SUBI  (TR, 1, TR)                 //  TR <=  TR - 1
    p.SQ    (TR, RS)                    // *RS <=  TR
}

func translate_OP_ctr_is_zero(p *atm.Builder, v Instr) {
    p.LQ    (RS, TR)                    // TR <= *RS
    p.BEQ   (TR, atm.Rz, p.At(v.To))    // if TR == 0 then GOTO @v.To
}

func translate_OP_map_alloc(p *atm.Builder, v Instr) {
    p.LQ    (RS, TR)                    //  TR <= *RS
    p.LP    (WP, TP)                    //  TP <= *WP
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (makemap).                  //  GCALL makemap:
      A0    (ET).                       //      t    <= ET
      A1    (TR).                       //      hint <= TR
      A2    (TP).                       //      h    <= TP
      R0    (TP)                        //      ret  => TP
    p.SP    (TP, WP)                    // *WP <=  TP
    p.ADDPI (RS, NbSize, EP)            //  EP <=  RS + NbSize
    p.SP    (TP, EP)                    // *EP <=  TP
}

func translate_OP_map_set_i8(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            // TP <=  RS + NbSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.IP    (v.Vt, ET)                  // ET <=  v.Vt
    p.GCALL (mapassign).                // GCALL mapassign:
      A0    (ET).                       //     t   <= ET
      A1    (TP).                       //     h   <= TP
      A2    (IP).                       //     key <= IP
      R0    (WP)                        //     ret => WP
    p.SUBI  (IL, 1, IL)                 // IL <=  IL - 1
    p.ADDPI (IP, 1, IP)                 // IP <=  IP + 1
}

func translate_OP_map_set_i16(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            //  TP <=  RS + NbSize
    p.LP    (TP, EP)                    //  EP <= *TP
    p.ADDPI (TP, WpSize, TP)            //  TP <=  TP + WpSize
    p.LW    (IP, TR)                    //  TR <= *IP
    p.SUBI  (IL, 2, IL)                 //  IL <=  IL - 2
    p.ADDPI (IP, 2, IP)                 //  IP <=  IP + 2
    p.SWAPW (TR, TR)                    //  TR <=  bswap16(TR)
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (mapassign).                // GCALL mapassign:
      A0    (ET).                       //     t   <= ET
      A1    (EP).                       //     h   <= EP
      A2    (TP).                       //     key <= TP
      R0    (WP)                        //     ret => WP
}

func translate_OP_map_set_i32(p *atm.Builder, v Instr) {
    if rt.MapType(v.Vt).Elem.Size > MaxFastMap {
        translate_OP_map_set_i32_safe(p, v)
    } else {
        translate_OP_map_set_i32_fast(p, v)
    }
}

func translate_OP_map_set_i32_fast(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            // TP <=  RS + NbSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.LL    (IP, TR)                    // TR <= *IP
    p.SWAPL (TR, TR)                    // TR <=  bswap32(TR)
    p.SUBI  (IL, 4, IL)                 // IL <=  IL - 4
    p.ADDPI (IP, 4, IP)                 // IP <=  IP + 4
    p.IP    (v.Vt, ET)                  // ET <=  v.Vt
    p.GCALL (mapassign_fast32).         // GCALL mapassign_fast32:
      A0    (ET).                       //     t   <= ET
      A1    (TP).                       //     h   <= TP
      A2    (TR).                       //     key <= TR
      R0    (WP)                        //     ret => WP
}

func translate_OP_map_set_i32_safe(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            //  TP <=  RS + NbSize
    p.LP    (TP, EP)                    //  EP <= *TP
    p.ADDPI (TP, WpSize, TP)            //  TP <=  TP + WpSize
    p.LL    (IP, TR)                    //  TR <= *IP
    p.SUBI  (IL, 4, IL)                 //  IL <=  IL - 4
    p.ADDPI (IP, 4, IP)                 //  IP <=  IP + 4
    p.SWAPL (TR, TR)                    //  TR <=  bswap32(TR)
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (mapassign).                // GCALL mapassign:
      A0    (ET).                       //     t   <= ET
      A1    (EP).                       //     h   <= EP
      A2    (TP).                       //     key <= TP
      R0    (WP)                        //     ret => WP
}

func translate_OP_map_set_i64(p *atm.Builder, v Instr) {
    if rt.MapType(v.Vt).Elem.Size > MaxFastMap {
        translate_OP_map_set_i64_safe(p, v)
    } else {
        translate_OP_map_set_i64_fast(p, v)
    }
}

func translate_OP_map_set_i64_fast(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            // TP <=  RS + NbSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.LQ    (IP, TR)                    // TR <= *IP
    p.SWAPQ (TR, TR)                    // TR <=  bswap64(TR)
    p.SUBI  (IL, 8, IL)                 // IL <=  IL - 8
    p.ADDPI (IP, 8, IP)                 // IP <=  IP + 8
    p.IP    (v.Vt, ET)                  // ET <=  v.Vt
    p.GCALL (mapassign_fast64).         // GCALL mapassign_fast64:
      A0    (ET).                       //     t   <= ET
      A1    (TP).                       //     h   <= TP
      A2    (TR).                       //     key <= TR
      R0    (WP)                        //     ret => WP
}

func translate_OP_map_set_i64_safe(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            //  TP <=  RS + NbSize
    p.LP    (TP, EP)                    //  EP <= *TP
    p.ADDPI (TP, WpSize, TP)            //  TP <=  TP + WpSize
    p.LQ    (IP, TR)                    //  TR <= *IP
    p.SUBI  (IL, 8, IL)                 //  IL <=  IL - 8
    p.ADDPI (IP, 8, IP)                 //  IP <=  IP + 8
    p.SWAPQ (TR, TR)                    //  TR <=  bswap64(TR)
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (mapassign).                // GCALL mapassign:
      A0    (ET).                       //     t   <= ET
      A1    (EP).                       //     h   <= EP
      A2    (TP).                       //     key <= TP
      R0    (WP)                        //     ret => WP
}

func translate_OP_map_set_str(p *atm.Builder, v Instr) {
    if rt.MapType(v.Vt).Elem.Size > MaxFastMap {
        translate_OP_map_set_str_safe(p, v)
    } else {
        translate_OP_map_set_str_fast(p, v)
    }
}

func translate_OP_map_set_str_fast(p *atm.Builder, v Instr) {
    p.LQ    (IP, TR)                    // TR <= *IP
    p.SUBI  (IL, 4, IL)                 // IL <=  IL - 4
    p.ADDPI (IP, 4, IP)                 // IP <=  IP + 4
    p.BLTU  (IL, TR, LB_eof)            // if IL < TR then GOTO _eof
    p.MOVP  (atm.Pn, EP)                // EP <=  nil
    p.BEQ   (TR, atm.Rz, "_empty_{n}")  // if TR == 0 then GOTO _empty_{n}
    p.MOVP  (IP, EP)                    // EP <=  IP
    p.SUB   (IL, TR, IL)                // IL <=  IL - TR
    p.ADDP  (IP, TR, IP)                // IP <=  IP + TR
    p.Label ("_empty_{n}")              // _empty_{n}:
    p.ADDPI (RS, NbSize, TP)            // TP <=  RS + NbSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.IP    (v.Vt, ET)                  // ET <=  v.Vt
    p.GCALL (mapassign_faststr).        // GCALL mapassign_faststr:
      A0    (ET).                       //     t     <= ET
      A1    (TP).                       //     h     <= TP
      A2    (EP).                       //     s.ptr <= EP
      A3    (TR).                       //     s.len <= TR
      R0    (WP)                        //     ret   => WP
}

func translate_OP_map_set_str_safe(p *atm.Builder, v Instr) {
    p.LQ    (IP, TR)                    //  TR <= *IP
    p.SUBI  (IL, 4, IL)                 //  IL <=  IL - 4
    p.ADDPI (IP, 4, IP)                 //  IP <=  IP + 4
    p.BLTU  (IL, TR, LB_eof)            //  if IL < TR then GOTO _eof
    p.SUBP  (RS, ST, TP)                //  TP <=  RS - ST
    p.ADDPI (TP, StateCap, TP)          //  TP <=  TP + StateCap
    p.ADDPI (TP, defs.PtrSize, EP)      //  EP <=  TP + defs.PtrSize
    p.SP    (atm.Pn, TP)                // *TP <=  nil
    p.SQ    (TR, EP)                    // *EP <=  TR
    p.BEQ   (TR, atm.Rz, "_empty_{n}")  //  if TR == 0 then GOTO _empty_{n}
    p.SP    (IP, TP)                    // *TP <=  IP
    p.SUB   (IL, TR, IL)                //  IL <=  IL - TR
    p.ADDP  (IP, TR, IP)                //  IP <=  IP + TR
    p.Label ("_empty_{n}")              // _empty_{n}:
    p.ADDPI (RS, NbSize, EP)            //  EP <=  RS + NbSize
    p.LP    (EP, EP)                    //  EP <= *EP
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (mapassign).                //  GCALL mapassign:
      A0    (ET).                       //      t   <= ET
      A1    (EP).                       //      h   <= EP
      A2    (TP).                       //      key <= TP
      R0    (WP)                        //      ret => WP
    p.SP    (atm.Pn, TP)                // *TP <=  nil
}

func translate_OP_map_set_pointer(p *atm.Builder, v Instr) {
    if rt.MapType(v.Vt).Elem.Size > MaxFastMap {
        translate_OP_map_set_pointer_safe(p, v)
    } else {
        translate_OP_map_set_pointer_fast(p, v)
    }
}

func translate_OP_map_set_pointer_fast(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            // TP <=  RS + NbSize
    p.LP    (TP, TP)                    // TP <= *TP
    p.IP    (v.Vt, ET)                  // ET <=  v.Vt
    p.GCALL (mapassign_fast64ptr).      // GCALL mapassign_fast64ptr:
      A0    (ET).                       //     t   <= ET
      A1    (TP).                       //     h   <= TP
      A2    (WP).                       //     key <= WP
      R0    (WP)                        //     ret => WP
}

func translate_OP_map_set_pointer_safe(p *atm.Builder, v Instr) {
    p.ADDPI (RS, NbSize, TP)            //  TP <=  RS + NbSize
    p.LP    (TP, EP)                    //  EP <= *TP
    p.SUBP  (RS, ST, TP)                //  TP <=  RS - ST
    p.ADDPI (TP, StateCap, TP)          //  TP <=  TP + StateCap
    p.SP    (WP, TP)                    // *TP <=  WP
    p.IP    (v.Vt, ET)                  //  ET <=  v.Vt
    p.GCALL (mapassign).                //  GCALL mapassign:
      A0    (ET).                       //      t   <= ET
      A1    (EP).                       //      h   <= EP
      A2    (TP).                       //      key <= TP
      R0    (WP)                        //      ret => WP
    p.SP    (atm.Pn, TP)                // *TP <=  nil
}

func translate_OP_list_alloc(p *atm.Builder, v Instr) {
    p.LQ    (RS, TR)                    //  TR <= *RS
    p.ADDPI (WP, 8, TP)                 //  TP <=  WP + 8
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.ADDPI (TP, 8, TP)                 //  TP <=  TP + 8
    p.LQ    (TP, UR)                    //  UR <= *TP
    p.BGEU  (UR, TR, "_noalloc_{n}")    //  if UR >= TR then GOTO _noalloc_{n}
    p.SQ    (TR, TP)                    // *TP <=  TR
    p.IB    (1, UR)                     //  UR <=  1
    p.IP    (v.Vt, TP)                  //  TP <=  v.Vt
    p.MULI  (TR, int64(v.Vt.Size), TR)  //  TR <=  TR * v.Vt.Size
    p.GCALL (mallocgc).                 //  GCALL mallocgc:
      A0    (TR).                       //      size     <= TR
      A1    (TP).                       //      typ      <= TP
      A2    (UR).                       //      needzero <= UR
      R0    (TP)                        //      ret      => TP
    p.SP    (TP, WP)                    // *WP <= TP
    p.Label ("_noalloc_{n}")            // _noalloc_{n}:
    p.LP    (WP, WP)                    //  WP <= *WP
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
    p.IQ    (StateMax, TR)              //  TR <= StateMax
    p.BGEU  (ST, TR, LB_overflow)       //  if ST >= TR then GOTO _overflow
    p.ADDPI (RS, NbSize, RS)            //  RS <= RS + NbSize
    p.SP    (WP, RS)                    // *RS <= WP
    p.ADDPI (RS, WpFmSize, RS)          //  RS <= RS + WpFmSize
    p.ADDI  (ST, StateSize, ST)         //  ST <= ST + StateSize
}

func translate_OP_drop_state(p *atm.Builder, _ Instr) {
    p.SUBI  (ST, StateSize, ST)         //  ST <=  ST - StateSize
    p.SUBPI (RS, WpFmSize, RS)          //  RS <=  RS - WpFmSize
    p.LP    (RS, WP)                    //  WP <= *RS
    p.SP    (atm.Pn, RS)                // *RS <=  nil
    p.SUBPI (RS, NbSize, RS)            //  RS <=  RS - NbSize
    p.SQ    (atm.Rz, RS)                // *RS <=  0
}

func translate_OP_construct(p *atm.Builder, v Instr) {
    p.IB    (1, UR)                     //  UR <= 1
    p.IP    (v.Vt, TP)                  //  TP <= v.Vt
    p.IQ    (int64(v.Vt.Size), TR)      //  TR <= v.Vt.Size
    p.GCALL (mallocgc).                 //  GCALL mallocgc:
      A0    (TR).                       //      size     <= TR
      A1    (TP).                       //      typ      <= TP
      A2    (UR).                       //      needzero <= UR
      R0    (WP)                        //      ret      => WP
}

func translate_OP_defer(p *atm.Builder, _ Instr) {

}

func translate_OP_goto(p *atm.Builder, v Instr) {
    p.JAL   (p.At(v.To), atm.Pn)        // GOTO @v.To
}

func translate_OP_halt(p *atm.Builder, _ Instr) {
    p.JAL   (LB_halt, atm.Pn)           // GOTO _halt
}
