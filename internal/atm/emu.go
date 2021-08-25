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
    `unsafe`

    `github.com/cloudwego/frugal/internal/defs`
    `github.com/cloudwego/frugal/internal/rt`
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

//goland:noinspection GoVetUnsafePointer
var dispatchTab = [...]func(e *Emulator, p *Instr) {
    OP_nop   : func(_ *Emulator, _ *Instr) {},
    OP_ib    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = atoi8(p.Ai) },
    OP_iw    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = atoi16(p.Ai) },
    OP_il    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = atoi32(p.Ai) },
    OP_iq    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = atoi64(p.Ai) },
    OP_ip    : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = p.Pr },
    OP_lb    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = uint64(*(*int8 )(e.Pr[p.Ps])) },
    OP_lw    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = uint64(*(*int16)(e.Pr[p.Ps])) },
    OP_ll    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = uint64(*(*int32)(e.Pr[p.Ps])) },
    OP_lq    : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = uint64(*(*int64)(e.Pr[p.Ps])) },
    OP_lp    : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = *(*unsafe.Pointer)(e.Pr[p.Ps]) },
    OP_sb    : func(e *Emulator, p *Instr) { *(*int8 )(e.Pr[p.Pd]) = int8(e.Gr[p.Rx]) },
    OP_sw    : func(e *Emulator, p *Instr) { *(*int16)(e.Pr[p.Pd]) = int16(e.Gr[p.Rx]) },
    OP_sl    : func(e *Emulator, p *Instr) { *(*int32)(e.Pr[p.Pd]) = int32(e.Gr[p.Rx]) },
    OP_sq    : func(e *Emulator, p *Instr) { *(*int64)(e.Pr[p.Pd]) = int64(e.Gr[p.Rx]) },
    OP_sp    : func(e *Emulator, p *Instr) { *(*unsafe.Pointer)(e.Pr[p.Pd]) = e.Pr[p.Ps] },
    OP_mov   : func(e *Emulator, p *Instr) { e.Gr[p.Ry] = e.Gr[p.Rx] },
    OP_movp  : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = e.Pr[p.Ps] },
    OP_movpr : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = ptou64(e.Pr[p.Ps]) },
    OP_movrp : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = u64top(e.Gr[p.Rx]) },
    OP_ldaq  : func(e *Emulator, p *Instr) { e.Gr[p.Rx] = e.Ar[atoi64(p.Ai)].U },
    OP_ldap  : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = e.Ar[atoi64(p.Ai)].P },
    OP_strq  : func(e *Emulator, p *Instr) { e.Rv[atoi64(p.Ai)].U = e.Gr[p.Rx] },
    OP_strp  : func(e *Emulator, p *Instr) { e.Rv[atoi64(p.Ai)].P = e.Pr[p.Ps] },
    OP_addp  : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = unsafe.Pointer(uintptr(e.Pr[p.Ps]) + uintptr(e.Gr[p.Rx])) },
    OP_subp  : func(e *Emulator, p *Instr) { e.Pr[p.Pd] = unsafe.Pointer(uintptr(e.Pr[p.Ps]) - uintptr(e.Gr[p.Rx])) },
    OP_add   : func(e *Emulator, p *Instr) { e.Gr[p.Rz] = e.Gr[p.Rx] + e.Gr[p.Ry] },
    OP_sub   : func(e *Emulator, p *Instr) { e.Gr[p.Rz] = e.Gr[p.Rx] - e.Gr[p.Ry] },
    OP_mul   : func(e *Emulator, p *Instr) { e.Gr[p.Rz] = e.Gr[p.Rx] * e.Gr[p.Ry] },
    OP_swap2 : func(e *Emulator, p *Instr) { e.Gr[p.Ry] = uint64(bits.ReverseBytes16(uint16(e.Gr[p.Rx]))) },
    OP_swap4 : func(e *Emulator, p *Instr) { e.Gr[p.Ry] = uint64(bits.ReverseBytes32(uint32(e.Gr[p.Rx]))) },
    OP_swap8 : func(e *Emulator, p *Instr) { e.Gr[p.Ry] = bits.ReverseBytes64(e.Gr[p.Rx]) },
    OP_beq   : func(e *Emulator, p *Instr) { if e.Gr[p.Rx] == e.Gr[p.Ry] { e.PC, e.Ln = p.Ln, false } },
    OP_bne   : func(e *Emulator, p *Instr) { if e.Gr[p.Rx] != e.Gr[p.Ry] { e.PC, e.Ln = p.Ln, false } },
    OP_blt   : func(e *Emulator, p *Instr) { if int64(e.Gr[p.Rx]) < int64(e.Gr[p.Ry]) { e.PC, e.Ln = p.Ln, false } },
    OP_bge   : func(e *Emulator, p *Instr) { if int64(e.Gr[p.Rx]) >= int64(e.Gr[p.Ry]) { e.PC, e.Ln = p.Ln, false } },
    OP_bltu  : func(e *Emulator, p *Instr) { if e.Gr[p.Rx] < e.Gr[p.Ry] { e.PC, e.Ln = p.Ln, false } },
    OP_bgeu  : func(e *Emulator, p *Instr) { if e.Gr[p.Rx] >= e.Gr[p.Ry] { e.PC, e.Ln = p.Ln, false } },
    OP_jal   : func(e *Emulator, p *Instr) { e.Pr[p.Pd], e.PC, e.Ln = unsafe.Pointer(e.PC.Ln), p.Br, false },
    OP_jalr  : func(e *Emulator, p *Instr) { e.Pr[p.Pd], e.PC, e.Ln = unsafe.Pointer(e.PC.Ln), (*Instr)(e.Pr[p.Ps]), false },
    OP_halt  : func(e *Emulator, p *Instr) { e.PC, e.Ln = nil, false },
    OP_ccall : (*Emulator).ccall,
    OP_gcall : (*Emulator).gcall,
}

func (self *Emulator) ccall(_ *Instr) {
    // TODO: implement C function call
    panic("ccall: not implemented")
}

func (self *Emulator) gcall(p *Instr) {
    ff := findFrame(p)
    fb := ff.newBuffer()
    self.setargs(p, fb)
    rt.ReflectCall(ff.t, p.Pr, fb.m, ff.t.Size, p.An * defs.PtrSize)
    self.getrets(p, fb)
    ff.freeBuffer(fb)
}

func (self *Emulator) setargs(p *Instr, ff Buffer) {
    for i := 0; i < p.An; i++ {
        if (p.Ai[i] & ArgPointer) == 0 {
            *ff.u(i) = self.Gr[p.Ai[i] & ArgMask]
        } else {
            *ff.p(i) = self.Pr[p.Ai[i] & ArgMask]
        }
    }
}

func (self *Emulator) getrets(p *Instr, ff Buffer) {
    for i := 0; i < p.Rn; i++ {
        if (p.Rv[i] & ArgPointer) == 0 {
            self.Gr[p.Rv[i] & ArgMask] = *ff.u(p.An + i)
        } else {
            self.Pr[p.Rv[i] & ArgMask] = *ff.p(p.An + i)
        }
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
