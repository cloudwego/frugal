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

package atm

import (
    `fmt`
    `runtime`
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

type OpCode byte

const (
    OP_nop OpCode = iota    // no operation
    OP_ib                   //  i8(Im) -> Rx
    OP_iw                   // i16(Im) -> Rx
    OP_il                   // i32(Im) -> Rx
    OP_iq                   // i64(Im) -> Rx
    OP_ip                   // ptr(Pr) -> Pd
    OP_lb                   // i64(*(* i8)Ps) -> Rx
    OP_lw                   // i64(*(*i16)Ps) -> Rx
    OP_ll                   // i64(*(*i32)Ps) -> Rx
    OP_lq                   //     *(*i64)Ps  -> Rx
    OP_lp                   //     *(*ptr)Ps  -> Pd
    OP_sb                   //  i8(Rx) -> *(*i8)Pd
    OP_sw                   // i16(Rx) -> *(*i16)Pd
    OP_sl                   // i32(Rx) -> *(*i32)Pd
    OP_sq                   //     Rx  -> *(*i64)Pd
    OP_sp                   //     Ps  -> *(*ptr)Pd
    OP_mov                  // Rx -> Ry
    OP_movp                 // Ps -> Pd
    OP_ldaq                 // arg[Im] -> Rx
    OP_ldap                 // arg[Im] -> Pd
    OP_strq                 // Rx -> ret[Im]
    OP_strp                 // Ps -> ret[Im]
    OP_addp                 // Ps + Rx -> Pd
    OP_subp                 // Ps - Rx -> Pd
    OP_addpi                // Ps + Im -> Pd
    OP_subpi                // Ps - Im -> Pd
    OP_sub                  // Rx - Ry -> Rz
    OP_addi                 // Rx + Im -> Ry
    OP_subi                 // Rx - Im -> Ry
    OP_muli                 // Rx * Im -> Ry
    OP_andi                 // Rx & Im -> Ry
    OP_xori                 // Rx ^ Im -> Ry
    OP_sbiti                // Rx | (1 << Im) -> Ry
    OP_swapw                // bswap16(Rx) -> Ry
    OP_swapl                // bswap32(Rx) -> Ry
    OP_swapq                // bswap64(Rx) -> Ry
    OP_beq                  // if (Rx == Ry) Br.PC -> PC
    OP_bne                  // if (Rx != Ry) Br.PC -> PC
    OP_blt                  // if (Rx <  Ry) Br.PC -> PC
    OP_bltu                 // if (u(Rx) <  u(Ry)) Br.PC -> PC
    OP_bgeu                 // if (u(Rx) >= u(Ry)) Br.PC -> PC
    OP_bsw                  // if (u(Rx) <  u(An)) Sw[u(Rx)].PC -> PC
    OP_jal                  // PC -> Pd; Br.PC -> PC
    OP_ccall                // call external C functions
    OP_gcall                // call external Go functions
    OP_icall                // call external Go iface methods
    OP_halt                 // halt the emulator
    OP_break                // trigger a debugger breakpoint
)

type Instr struct {
    Op OpCode
    Rx GenericRegister
    Ry GenericRegister
    Rz GenericRegister
    Ps PointerRegister
    Pd PointerRegister
    An uint8
    Rn uint8
    Ar [8]uint8
    Rr [8]uint8
    Iv int64
    Pr unsafe.Pointer
    Br *Instr
    Ln *Instr
}

func (self *Instr) iv(v int64)           *Instr { self.Iv = v; return self }
func (self *Instr) pr(v unsafe.Pointer)  *Instr { self.Pr = v; return self }
func (self *Instr) rx(v GenericRegister) *Instr { self.Rx = v; return self }
func (self *Instr) ry(v GenericRegister) *Instr { self.Ry = v; return self }
func (self *Instr) rz(v GenericRegister) *Instr { self.Rz = v; return self }
func (self *Instr) ps(v PointerRegister) *Instr { self.Ps = v; return self }
func (self *Instr) pd(v PointerRegister) *Instr { self.Pd = v; return self }

func (self *Instr) A0(v Register) *Instr { self.An, self.Ar[0] = 1, v.A(); return self }
func (self *Instr) A1(v Register) *Instr { self.An, self.Ar[1] = 2, v.A(); return self }
func (self *Instr) A2(v Register) *Instr { self.An, self.Ar[2] = 3, v.A(); return self }
func (self *Instr) A3(v Register) *Instr { self.An, self.Ar[3] = 4, v.A(); return self }
func (self *Instr) A4(v Register) *Instr { self.An, self.Ar[4] = 5, v.A(); return self }
func (self *Instr) A5(v Register) *Instr { self.An, self.Ar[5] = 6, v.A(); return self }
func (self *Instr) A6(v Register) *Instr { self.An, self.Ar[6] = 7, v.A(); return self }
func (self *Instr) A7(v Register) *Instr { self.An, self.Ar[7] = 8, v.A(); return self }

func (self *Instr) R0(v Register) *Instr { self.Rn, self.Rr[0] = 1, v.A(); return self }
func (self *Instr) R1(v Register) *Instr { self.Rn, self.Rr[1] = 2, v.A(); return self }
func (self *Instr) R2(v Register) *Instr { self.Rn, self.Rr[2] = 3, v.A(); return self }
func (self *Instr) R3(v Register) *Instr { self.Rn, self.Rr[3] = 4, v.A(); return self }
func (self *Instr) R4(v Register) *Instr { self.Rn, self.Rr[4] = 5, v.A(); return self }
func (self *Instr) R5(v Register) *Instr { self.Rn, self.Rr[5] = 6, v.A(); return self }
func (self *Instr) R6(v Register) *Instr { self.Rn, self.Rr[6] = 7, v.A(); return self }
func (self *Instr) R7(v Register) *Instr { self.Rn, self.Rr[7] = 8, v.A(); return self }

func (self *Instr) Sw() (p []*Instr) {
    (*rt.GoSlice)(unsafe.Pointer(&p)).Ptr = self.Pr
    (*rt.GoSlice)(unsafe.Pointer(&p)).Len = int(self.Iv)
    (*rt.GoSlice)(unsafe.Pointer(&p)).Cap = int(self.Iv)
    return
}

func (self *Instr) isBranch() bool {
    return self.Op >= OP_beq && self.Op <= OP_jal
}

func (self *Instr) formatFunc() string {
    return fmt.Sprintf("%s[*%p]", runtime.FuncForPC(uintptr(self.Pr)).Name(), self.Pr)
}

func (self *Instr) formatCalls() string {
    args := make([]string, self.An)
    rets := make([]string, self.Rn)

    /* add arguments */
    for i := uint8(0); i < self.An; i++ {
        if v := self.Ar[i]; (v & ArgPointer) == 0 {
            args[i] = "%" + GenericRegister(v & ArgMask).String()
        } else {
            args[i] = "%" + PointerRegister(v & ArgMask).String()
        }
    }

    /* add return values */
    for i := uint8(0); i < self.Rn; i++ {
        if v := self.Rr[i]; (v & ArgPointer) == 0 {
            rets[i] = "%" + GenericRegister(v & ArgMask).String()
        } else {
            rets[i] = "%" + PointerRegister(v & ArgMask).String()
        }
    }

    /* compose the result */
    return fmt.Sprintf(
        "{%s}, {%s}",
        strings.Join(args, ", "),
        strings.Join(rets, ", "),
    )
}

func (self *Instr) formatTable(refs map[*Instr]string) string {
    tab := self.Sw()
    ret := make([]string, 0, self.Iv)

    /* empty table */
    if self.Iv == 0 {
        return "{}"
    }

    /* format every label */
    for i, lb := range tab {
        if lb != nil {
            ret = append(ret, fmt.Sprintf("\t%4ccase %d: %s\n", ' ', i, refs[lb]))
        }
    }

    /* join them together */
    return fmt.Sprintf(
        "{\n%s\t}",
        strings.Join(ret, ""),
    )
}

func (self *Instr) formatMethod() string {
    vt := (*rt.GoType)(self.Pr).Pack()
    return fmt.Sprintf("%s.%s", vt, vt.Method(int(self.Iv)).Name)
}

func (self *Instr) disassemble(refs map[*Instr]string) string {
    switch self.Op {
        case OP_nop   : return "nop"
        case OP_ib    : return fmt.Sprintf("ib      $%d, %%%s", self.Iv, self.Rx)
        case OP_iw    : return fmt.Sprintf("iw      $%d, %%%s", self.Iv, self.Rx)
        case OP_il    : return fmt.Sprintf("il      $%d, %%%s", self.Iv, self.Rx)
        case OP_iq    : return fmt.Sprintf("iq      $%d, %%%s", self.Iv, self.Rx)
        case OP_ip    : return fmt.Sprintf("ip      $%p, %%%s", self.Pr, self.Pd)
        case OP_lb    : return fmt.Sprintf("lb      %%%s, %%%s", self.Ps, self.Rx)
        case OP_lw    : return fmt.Sprintf("lw      %%%s, %%%s", self.Ps, self.Rx)
        case OP_ll    : return fmt.Sprintf("ll      %%%s, %%%s", self.Ps, self.Rx)
        case OP_lq    : return fmt.Sprintf("lq      %%%s, %%%s", self.Ps, self.Rx)
        case OP_lp    : return fmt.Sprintf("lp      %%%s, %%%s", self.Ps, self.Pd)
        case OP_sb    : return fmt.Sprintf("sb      %%%s, %%%s", self.Rx, self.Pd)
        case OP_sw    : return fmt.Sprintf("sw      %%%s, %%%s", self.Rx, self.Pd)
        case OP_sl    : return fmt.Sprintf("sl      %%%s, %%%s", self.Rx, self.Pd)
        case OP_sq    : return fmt.Sprintf("sq      %%%s, %%%s", self.Rx, self.Pd)
        case OP_sp    : return fmt.Sprintf("sp      %%%s, %%%s", self.Ps, self.Pd)
        case OP_mov   : return fmt.Sprintf("mov     %%%s, %%%s", self.Rx, self.Ry)
        case OP_movp  : return fmt.Sprintf("mov     %%%s, %%%s", self.Ps, self.Pd)
        case OP_ldaq  : return fmt.Sprintf("lda     $%d, %%%s", self.Iv, self.Rx)
        case OP_ldap  : return fmt.Sprintf("lda     $%d, %%%s", self.Iv, self.Pd)
        case OP_strq  : return fmt.Sprintf("str     %%%s, $%d", self.Rx, self.Iv)
        case OP_strp  : return fmt.Sprintf("str     %%%s, $%d", self.Ps, self.Iv)
        case OP_addp  : return fmt.Sprintf("add     %%%s, %%%s, %%%s", self.Ps, self.Rx, self.Pd)
        case OP_subp  : return fmt.Sprintf("sub     %%%s, %%%s, %%%s", self.Ps, self.Rx, self.Pd)
        case OP_addpi : return fmt.Sprintf("add     %%%s, %d, %%%s", self.Ps, self.Iv, self.Pd)
        case OP_subpi : return fmt.Sprintf("sub     %%%s, %d, %%%s", self.Ps, self.Iv, self.Pd)
        case OP_sub   : return fmt.Sprintf("sub     %%%s, %%%s, %%%s", self.Rx, self.Ry, self.Rz)
        case OP_addi  : return fmt.Sprintf("add     %%%s, %d, %%%s", self.Rx, self.Iv, self.Ry)
        case OP_subi  : return fmt.Sprintf("sub     %%%s, %d, %%%s", self.Rx, self.Iv, self.Ry)
        case OP_muli  : return fmt.Sprintf("mul     %%%s, %d, %%%s", self.Rx, self.Iv, self.Ry)
        case OP_andi  : return fmt.Sprintf("and     %%%s, %d, %%%s", self.Rx, self.Iv, self.Ry)
        case OP_xori  : return fmt.Sprintf("xor     %%%s, %d, %%%s", self.Rx, self.Iv, self.Ry)
        case OP_sbiti : return fmt.Sprintf("sbit    %%%s, %d, %%%s", self.Rx, self.Iv, self.Ry)
        case OP_swapw : return fmt.Sprintf("swapw   %%%s, %%%s", self.Rx, self.Ry)
        case OP_swapl : return fmt.Sprintf("swapl   %%%s, %%%s", self.Rx, self.Ry)
        case OP_swapq : return fmt.Sprintf("swapq   %%%s, %%%s", self.Rx, self.Ry)
        case OP_beq   : return fmt.Sprintf("beq     %%%s, %%%s, %s", self.Rx, self.Ry, refs[self.Br])
        case OP_bne   : return fmt.Sprintf("bne     %%%s, %%%s, %s", self.Rx, self.Ry, refs[self.Br])
        case OP_blt   : return fmt.Sprintf("blt     %%%s, %%%s, %s", self.Rx, self.Ry, refs[self.Br])
        case OP_bltu  : return fmt.Sprintf("bltu    %%%s, %%%s, %s", self.Rx, self.Ry, refs[self.Br])
        case OP_bgeu  : return fmt.Sprintf("bgeu    %%%s, %%%s, %s", self.Rx, self.Ry, refs[self.Br])
        case OP_bsw   : return fmt.Sprintf("bsw     %%%s, %s", self.Rx, self.formatTable(refs))
        case OP_jal   : return fmt.Sprintf("jal     %s, %%%s", refs[self.Br], self.Pd)
        case OP_ccall : return fmt.Sprintf("ccall   %s, %s", self.formatFunc(), self.formatCalls())
        case OP_gcall : return fmt.Sprintf("gcall   %s, %s", self.formatFunc(), self.formatCalls())
        case OP_icall : return fmt.Sprintf("icall   %s, {%%%s, %%%s}, %s", self.formatMethod(), self.Ps, self.Pd, self.formatCalls())
        case OP_halt  : return "halt"
        case OP_break : return "break"
        default       : panic(fmt.Sprintf("invalid OpCode: 0x%02x", self.Op))
    }
}
