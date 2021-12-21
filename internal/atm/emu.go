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
    Gr [6]uint64
    Pr [7]unsafe.Pointer
    Ar [8]Value
    Rv [8]Value
}

func LoadProgram(p Program) (e *Emulator) {
    e = newEmulator()
    e.PC = p.Head
    return
}

func (self *Emulator) trap() {
    println("****** DEBUGGER BREAK ******")
    println("Current State:", self.String())
    runtime.Breakpoint()
}

func (self *Emulator) Ru(i int) uint64         { return self.Rv[i].U }
func (self *Emulator) Rp(i int) unsafe.Pointer { return self.Rv[i].P }

func (self *Emulator) Au(i int, v uint64)         *Emulator { self.Ar[i].U = v; return self }
func (self *Emulator) Ap(i int, v unsafe.Pointer) *Emulator { self.Ar[i].P = v; return self }

func (self *Emulator) Run() {
    var p *Instr
    var q *Instr
    var v uint64

    /* run until end */
    for self.PC != nil {
        p, self.PC = self.PC, self.PC.Ln
        self.Gr[Rz], self.Pr[Pn] = 0, nil

        /* main switch on OpCode */
        switch p.Op {
            default       : return
            case OP_nop   : break
            case OP_ip    : self.Pr[p.Pd] = p.Pr
            case OP_lb    : self.Gr[p.Rx] = uint64(*(*int8)(self.Pr[p.Ps]))
            case OP_lw    : self.Gr[p.Rx] = uint64(*(*int16)(self.Pr[p.Ps]))
            case OP_ll    : self.Gr[p.Rx] = uint64(*(*int32)(self.Pr[p.Ps]))
            case OP_lq    : self.Gr[p.Rx] = uint64(*(*int64)(self.Pr[p.Ps]))
            case OP_lp    : self.Pr[p.Pd] = *(*unsafe.Pointer)(self.Pr[p.Ps])
            case OP_sb    : *(*int8)(self.Pr[p.Pd]) = int8(self.Gr[p.Rx])
            case OP_sw    : *(*int16)(self.Pr[p.Pd]) = int16(self.Gr[p.Rx])
            case OP_sl    : *(*int32)(self.Pr[p.Pd]) = int32(self.Gr[p.Rx])
            case OP_sq    : *(*int64)(self.Pr[p.Pd]) = int64(self.Gr[p.Rx])
            case OP_sp    : *(*unsafe.Pointer)(self.Pr[p.Pd]) = self.Pr[p.Ps]
            case OP_ldaq  : self.Gr[p.Rx] = self.Ar[p.Iv].U
            case OP_ldap  : self.Pr[p.Pd] = self.Ar[p.Iv].P
            case OP_strq  : self.Rv[p.Iv].U = self.Gr[p.Rx]
            case OP_strp  : self.Rv[p.Iv].P = self.Pr[p.Ps]
            case OP_addp  : self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) + uintptr(self.Gr[p.Rx]))
            case OP_subp  : self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) - uintptr(self.Gr[p.Rx]))
            case OP_addpi : self.Pr[p.Pd] = unsafe.Pointer(uintptr(self.Pr[p.Ps]) + uintptr(p.Iv))
            case OP_add   : self.Gr[p.Rz] = self.Gr[p.Rx] + self.Gr[p.Ry]
            case OP_sub   : self.Gr[p.Rz] = self.Gr[p.Rx] - self.Gr[p.Ry]
            case OP_addi  : self.Gr[p.Ry] = self.Gr[p.Rx] + uint64(p.Iv)
            case OP_muli  : self.Gr[p.Ry] = self.Gr[p.Rx] * uint64(p.Iv)
            case OP_andi  : self.Gr[p.Ry] = self.Gr[p.Rx] & uint64(p.Iv)
            case OP_xori  : self.Gr[p.Ry] = self.Gr[p.Rx] ^ uint64(p.Iv)
            case OP_shri  : self.Gr[p.Ry] = self.Gr[p.Rx] >> p.Iv
            case OP_sbiti : self.Gr[p.Ry] = self.Gr[p.Rx] | (1 << p.Iv)
            case OP_swapw : self.Gr[p.Ry] = uint64(bits.ReverseBytes16(uint16(self.Gr[p.Rx])))
            case OP_swapl : self.Gr[p.Ry] = uint64(bits.ReverseBytes32(uint32(self.Gr[p.Rx])))
            case OP_swapq : self.Gr[p.Ry] = bits.ReverseBytes64(self.Gr[p.Rx])
            case OP_beq   : if       self.Gr[p.Rx]  ==       self.Gr[p.Ry]  { self.PC = p.Br }
            case OP_bne   : if       self.Gr[p.Rx]  !=       self.Gr[p.Ry]  { self.PC = p.Br }
            case OP_blt   : if int64(self.Gr[p.Rx]) <  int64(self.Gr[p.Ry]) { self.PC = p.Br }
            case OP_bltu  : if       self.Gr[p.Rx]  <        self.Gr[p.Ry]  { self.PC = p.Br }
            case OP_bgeu  : if       self.Gr[p.Rx]  >=       self.Gr[p.Ry]  { self.PC = p.Br }
            case OP_beqn  : if       self.Pr[p.Ps]  ==                 nil  { self.PC = p.Br }
            case OP_bnen  : if       self.Pr[p.Ps]  !=                 nil  { self.PC = p.Br }
            case OP_jal   : self.Pr[p.Pd], self.PC = unsafe.Pointer(self.PC), p.Br
            case OP_bzero : memclrNoHeapPointers(self.Pr[p.Pd], uintptr(p.Iv))
            case OP_bcopy : memmove(self.Pr[p.Pd], self.Pr[p.Ps], uintptr(self.Gr[p.Rx]))
            case OP_halt  : self.PC = nil
            case OP_break : self.trap()

            /* bit test and set */
            case OP_bts: {
                x := self.Gr[p.Rx]
                y := self.Gr[p.Ry]

                /* test and set the bit */
                if self.Gr[p.Ry] |= 1 << x; y & (1 << x) == 0 {
                    self.Gr[p.Rz] = 0
                } else {
                    self.Gr[p.Rz] = 1
                }
            }

            /* table switch */
            case OP_bsw: {
                if v = self.Gr[p.Rx]; v < uint64(p.Iv) {
                    if q = *(**Instr)(unsafe.Pointer(uintptr(p.Pr) + uintptr(v) * 8)); q != nil {
                        self.PC = q
                    }
                }
            }

            /* call to C / Go / Go interface functions */
            case OP_ccall: fallthrough
            case OP_gcall: fallthrough
            case OP_icall: {
                if p.Iv < 0 || p.Iv >= int64(len(invokeTab)) {
                    panic("invalid function ID")
                } else {
                    invokeTab[p.Iv].Call(self, p)
                }
            }
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

/** State Dumping **/

const _F_emulator = `Emulator {
    pc  (%p)%s
    r0  %#x
    r1  %#x
    r2  %#x
    r3  %#x
    r4  %#x
    r5  %#x
   ----
    p0  %p
    p1  %p
    p2  %p
    p3  %p
    p4  %p
    p5  %p
    p6  %p
}`

func (self *Emulator) String() string {
    return fmt.Sprintf(
        _F_emulator,
        self.PC,
        self.PC.disassemble(nil),
        self.Gr[0],
        self.Gr[1],
        self.Gr[2],
        self.Gr[3],
        self.Gr[4],
        self.Gr[5],
        self.Pr[0],
        self.Pr[1],
        self.Pr[2],
        self.Pr[3],
        self.Pr[4],
        self.Pr[5],
        self.Pr[6],
    )
}
