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
    `math/bits`
    `runtime`
    `unsafe`
)

type Value struct {
    U uint64
    P unsafe.Pointer
}

type Emulator struct {
    PC *Instr
    Gr [9]uint64
    Pr [10]unsafe.Pointer
    Ar [10]Value
    Rv [8]Value
    Ln bool
}

func LoadProgram(p Program) (e *Emulator) {
    e = newEmulator()
    e.PC = p.Head
    return
}

var dispatchTab = [...]func(e *Emulator, p *Instr) {
    OP_nop   : (*Emulator).emu_OP_nop,
    OP_ib    : (*Emulator).emu_OP_ib,
    OP_iw    : (*Emulator).emu_OP_iw,
    OP_il    : (*Emulator).emu_OP_il,
    OP_iq    : (*Emulator).emu_OP_iq,
    OP_ip    : (*Emulator).emu_OP_ip,
    OP_lb    : (*Emulator).emu_OP_lb,
    OP_lw    : (*Emulator).emu_OP_lw,
    OP_ll    : (*Emulator).emu_OP_ll,
    OP_lq    : (*Emulator).emu_OP_lq,
    OP_lp    : (*Emulator).emu_OP_lp,
    OP_sb    : (*Emulator).emu_OP_sb,
    OP_sw    : (*Emulator).emu_OP_sw,
    OP_sl    : (*Emulator).emu_OP_sl,
    OP_sq    : (*Emulator).emu_OP_sq,
    OP_sp    : (*Emulator).emu_OP_sp,
    OP_mov   : (*Emulator).emu_OP_mov,
    OP_movp  : (*Emulator).emu_OP_movp,
    OP_ldaq  : (*Emulator).emu_OP_ldaq,
    OP_ldap  : (*Emulator).emu_OP_ldap,
    OP_strq  : (*Emulator).emu_OP_strq,
    OP_strp  : (*Emulator).emu_OP_strp,
    OP_addp  : (*Emulator).emu_OP_addp,
    OP_subp  : (*Emulator).emu_OP_subp,
    OP_addpi : (*Emulator).emu_OP_addpi,
    OP_subpi : (*Emulator).emu_OP_subpi,
    OP_add   : (*Emulator).emu_OP_add,
    OP_sub   : (*Emulator).emu_OP_sub,
    OP_mul   : (*Emulator).emu_OP_mul,
    OP_addi  : (*Emulator).emu_OP_addi,
    OP_subi  : (*Emulator).emu_OP_subi,
    OP_muli  : (*Emulator).emu_OP_muli,
    OP_swapw : (*Emulator).emu_OP_swapw,
    OP_swapl : (*Emulator).emu_OP_swapl,
    OP_swapq : (*Emulator).emu_OP_swapq,
    OP_beq   : (*Emulator).emu_OP_beq,
    OP_bne   : (*Emulator).emu_OP_bne,
    OP_blt   : (*Emulator).emu_OP_blt,
    OP_bge   : (*Emulator).emu_OP_bge,
    OP_bltu  : (*Emulator).emu_OP_bltu,
    OP_bgeu  : (*Emulator).emu_OP_bgeu,
    OP_jal   : (*Emulator).emu_OP_jal,
    OP_jalr  : (*Emulator).emu_OP_jalr,
    OP_halt  : (*Emulator).emu_OP_halt,
    OP_ccall : (*Emulator).emu_OP_ccall,
    OP_gcall : (*Emulator).emu_OP_gcall,
}

//go:nosplit
func (self *Emulator) emu_OP_nop(_ *Instr) {
    /* no operation */
}

//go:nosplit
func (self *Emulator) emu_OP_ib(p *Instr) {
    self.Gr[p.Rx] = atoi8(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_iw(p *Instr) {
    self.Gr[p.Rx] = atoi16(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_il(p *Instr) {
    self.Gr[p.Rx] = atoi32(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_iq(p *Instr) {
    self.Gr[p.Rx] = atoi64(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_ip(p *Instr) {
    self.Pr[p.Pd] = p.Pr
}

//go:nosplit
func (self *Emulator) emu_OP_lb(p *Instr) {
    self.Gr[p.Rx] = uint64(*(*int8)(self.Pr[p.Ps]))
}

//go:nosplit
func (self *Emulator) emu_OP_lw(p *Instr) {
    self.Gr[p.Rx] = uint64(*(*int16)(self.Pr[p.Ps]))
}

//go:nosplit
func (self *Emulator) emu_OP_ll(p *Instr) {
    self.Gr[p.Rx] = uint64(*(*int32)(self.Pr[p.Ps]))
}

//go:nosplit
func (self *Emulator) emu_OP_lq(p *Instr) {
    self.Gr[p.Rx] = uint64(*(*int64)(self.Pr[p.Ps]))
}

//go:nosplit
func (self *Emulator) emu_OP_lp(p *Instr) {
    self.Pr[p.Pd] = *(*unsafe.Pointer)(self.Pr[p.Ps])
}

//go:nosplit
func (self *Emulator) emu_OP_sb(p *Instr) {
    *(*int8)(self.Pr[p.Pd]) = int8(self.Gr[p.Rx])
}

//go:nosplit
func (self *Emulator) emu_OP_sw(p *Instr) {
    *(*int16)(self.Pr[p.Pd]) = int16(self.Gr[p.Rx])
}

//go:nosplit
func (self *Emulator) emu_OP_sl(p *Instr) {
    *(*int32)(self.Pr[p.Pd]) = int32(self.Gr[p.Rx])
}

//go:nosplit
func (self *Emulator) emu_OP_sq(p *Instr) {
    *(*int64)(self.Pr[p.Pd]) = int64(self.Gr[p.Rx])
}

//go:nosplit
func (self *Emulator) emu_OP_sp(p *Instr) {
    *(*unsafe.Pointer)(self.Pr[p.Pd]) = self.Pr[p.Ps]
}

//go:nosplit
func (self *Emulator) emu_OP_mov(p *Instr) {
    self.Gr[p.Ry] = self.Gr[p.Rx]
}

//go:nosplit
func (self *Emulator) emu_OP_movp(p *Instr) {
    self.Pr[p.Pd] = self.Pr[p.Ps]
}

//go:nosplit
func (self *Emulator) emu_OP_ldaq(p *Instr) {
    self.Gr[p.Rx] = self.Ar[atoi64(p.Ai)].U
}

//go:nosplit
func (self *Emulator) emu_OP_ldap(p *Instr) {
    self.Pr[p.Pd] = self.Ar[atoi64(p.Ai)].P
}

//go:nosplit
func (self *Emulator) emu_OP_strq(p *Instr) {
    self.Rv[atoi64(p.Ai)].U = self.Gr[p.Rx]
}

//go:nosplit
func (self *Emulator) emu_OP_strp(p *Instr) {
    self.Rv[atoi64(p.Ai)].P = self.Pr[p.Ps]
}

//go:nosplit
func (self *Emulator) emu_OP_addp(p *Instr) {
    self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) + uintptr(self.Gr[p.Rx]))
}

//go:nosplit
func (self *Emulator) emu_OP_subp(p *Instr) {
    self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) - uintptr(self.Gr[p.Rx]))
}

//go:nosplit
func (self *Emulator) emu_OP_addpi(p *Instr) {
    self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) + uintptr(atoi64(p.Ai)))
}

//go:nosplit
func (self *Emulator) emu_OP_subpi(p *Instr) {
    self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) - uintptr(atoi64(p.Ai)))
}

//go:nosplit
func (self *Emulator) emu_OP_add(p *Instr) {
    self.Gr[p.Rz] = self.Gr[p.Rx] + self.Gr[p.Ry]
}

//go:nosplit
func (self *Emulator) emu_OP_sub(p *Instr) {
    self.Gr[p.Rz] = self.Gr[p.Rx] - self.Gr[p.Ry]
}

//go:nosplit
func (self *Emulator) emu_OP_mul(p *Instr) {
    self.Gr[p.Rz] = self.Gr[p.Rx] * self.Gr[p.Ry]
}

//go:nosplit
func (self *Emulator) emu_OP_addi(p *Instr) {
    self.Gr[p.Ry] = self.Gr[p.Rx] + atoi64(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_subi(p *Instr) {
    self.Gr[p.Ry] = self.Gr[p.Rx] - atoi64(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_muli(p *Instr) {
    self.Gr[p.Ry] = self.Gr[p.Rx] * atoi64(p.Ai)
}

//go:nosplit
func (self *Emulator) emu_OP_swapw(p *Instr) {
    self.Gr[p.Ry] = uint64(bits.ReverseBytes16(uint16(self.Gr[p.Rx])))
}

//go:nosplit
func (self *Emulator) emu_OP_swapl(p *Instr) {
    self.Gr[p.Ry] = uint64(bits.ReverseBytes32(uint32(self.Gr[p.Rx])))
}

//go:nosplit
func (self *Emulator) emu_OP_swapq(p *Instr) {
    self.Gr[p.Ry] = bits.ReverseBytes64(self.Gr[p.Rx])
}

//go:nosplit
func (self *Emulator) emu_OP_beq(p *Instr) {
    if self.Gr[p.Rx] == self.Gr[p.Ry] {
        self.PC = p.Br
        self.Ln = false
    }
}

//go:nosplit
func (self *Emulator) emu_OP_bne(p *Instr) {
    if self.Gr[p.Rx] != self.Gr[p.Ry] {
        self.PC = p.Br
        self.Ln = false
    }
}

//go:nosplit
func (self *Emulator) emu_OP_blt(p *Instr) {
    if int64(self.Gr[p.Rx]) < int64(self.Gr[p.Ry]) {
        self.PC = p.Br
        self.Ln = false
    }
}

//go:nosplit
func (self *Emulator) emu_OP_bge(p *Instr) {
    if int64(self.Gr[p.Rx]) >= int64(self.Gr[p.Ry]) {
        self.PC = p.Br
        self.Ln = false
    }
}

//go:nosplit
func (self *Emulator) emu_OP_bltu(p *Instr) {
    if self.Gr[p.Rx] < self.Gr[p.Ry] {
        self.PC = p.Br
        self.Ln = false
    }
}

//go:nosplit
func (self *Emulator) emu_OP_bgeu(p *Instr) {
    if self.Gr[p.Rx] >= self.Gr[p.Ry] {
        self.PC = p.Br
        self.Ln = false
    }
}

//go:nosplit
func (self *Emulator) emu_OP_jal(p *Instr) {
    self.Pr[p.Pd] = unsafe.Pointer(self.PC.Ln)
    self.PC       = p.Br
    self.Ln       = false
}

//go:nosplit
func (self *Emulator) emu_OP_jalr(p *Instr) {
    self.Pr[p.Pd] = unsafe.Pointer(self.PC.Ln)
    self.PC       = (*Instr)(self.Pr[p.Ps])
    self.Ln       = false
}

//go:nosplit
func (self *Emulator) emu_OP_halt(_ *Instr) {
    self.PC = nil
    self.Ln = false
}

//go:nosplit
func (self *Emulator) emu_OP_ccall(p *Instr) {
    if proxy := ccallTab[p.Pr]; proxy != nil {
        proxy(self, p)
    } else {
        panic(fmt.Sprintf("ccall: function not registered: *%p", p.Pr))
    }
}

//go:nosplit
func (self *Emulator) emu_OP_gcall(p *Instr) {
    if proxy := gcallTab[p.Pr]; proxy != nil {
        proxy(self, p)
    } else {
        panic(fmt.Sprintf("gcall: function not registered: %s(*%p)", runtime.FuncForPC(uintptr(p.Pr)).Name(), p.Pr))
    }
}

func (self *Emulator) Ru(i int) uint64         { return self.Rv[i].U }
func (self *Emulator) Rp(i int) unsafe.Pointer { return self.Rv[i].P }

func (self *Emulator) Au(i int, v uint64)         *Emulator { self.Ar[i].U = v; return self }
func (self *Emulator) Ap(i int, v unsafe.Pointer) *Emulator { self.Ar[i].P = v; return self }

func (self *Emulator) Run() {
    var ip *Instr
    var fn func(e *Emulator, p *Instr)

    /* run until end */
    for self.PC != nil {
        ip = self.PC
        fn = dispatchTab[ip.Op]

        /* move cold path outside of the loop */
        if fn == nil {
            break
        }

        /* clear certain registers every cycle */
        self.Ln = true
        self.Gr[Rz] = 0
        self.Pr[Pn] = nil

        /* execute and advance the PC if needed */
        if fn(self, ip); self.Ln {
            self.PC = self.PC.Ln
        }
    }

    /* check for exceptions */
    if self.PC != nil {
        panic(fmt.Sprintf("illegal OpCode: %#02x", self.PC.Op))
    }
}

func (self *Emulator) Free() {
    freeEmulator(self)
}
