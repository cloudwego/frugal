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

type OpCode byte

const (
    OP_nop OpCode = iota    // no operation
    OP_ib                   //  i8(Im) -> Rx
    OP_iw                   // i16(Im) -> Rx
    OP_il                   // i32(Im) -> Rx
    OP_iq                   // i64(Im) -> Rx
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
    OP_movpr                // Ps -> Rx
    OP_movrp                // Rx -> Pd
    OP_addp                 // Ps + Rx -> Pd
    OP_subp                 // Ps - Rx -> Pd
    OP_add                  // Rx + Ry -> Rz
    OP_sub                  // Rx - Ry -> Rz
    OP_mul                  // Rx * Ry -> Rz
    OP_beq                  // if (Rx == Ry) Br.PC -> PC
    OP_bne                  // if (Rx != Ry) Br.PC -> PC
    OP_blt                  // if (signed(Rx) <  signed(Ry)) Br.PC -> PC
    OP_bge                  // if (signed(Rx) >= signed(Ry)) Br.PC -> PC
    OP_bltu                 // if (unsigned(Rx) <  unsigned(Ry)) Br.PC -> PC
    OP_bgeu                 // if (unsigned(Rx) >= unsigned(Ry)) Br.PC -> PC
    OP_jal                  // PC -> Rx; Br.PC -> PC
    OP_jalr                 // PC -> Rx; Ry -> PC
    OP_call_go              // call external Go functions
    OP_ret                  // return from function
)

type Instr struct {
    Op OpCode
    Rx GenericRegister
    Ry GenericRegister
    Rz GenericRegister
    Ps PointerRegister
    Pd PointerRegister
    Ar [2]uint8
    Ai [8]uint8
    An int
    Br *Instr
    Ln *Instr
}

func (self *Instr) rx(v GenericRegister) *Instr { self.Rx = v; return self }
func (self *Instr) ry(v GenericRegister) *Instr { self.Ry = v; return self }
func (self *Instr) rz(v GenericRegister) *Instr { self.Rz = v; return self }
func (self *Instr) ps(v PointerRegister) *Instr { self.Ps = v; return self }
func (self *Instr) pd(v PointerRegister) *Instr { self.Pd = v; return self }
func (self *Instr) ai(v [8]uint8)        *Instr { self.Ai = v; return self }

func (self *Instr) A0(v uint8) *Instr { self.An, self.Ar[0] =  1, v; return self }
func (self *Instr) A1(v uint8) *Instr { self.An, self.Ar[1] =  2, v; return self }
func (self *Instr) A2(v uint8) *Instr { self.An, self.Ai[0] =  3, v; return self }
func (self *Instr) A3(v uint8) *Instr { self.An, self.Ai[1] =  4, v; return self }
func (self *Instr) A4(v uint8) *Instr { self.An, self.Ai[2] =  5, v; return self }
func (self *Instr) A5(v uint8) *Instr { self.An, self.Ai[3] =  6, v; return self }
func (self *Instr) A6(v uint8) *Instr { self.An, self.Ai[4] =  7, v; return self }
func (self *Instr) A7(v uint8) *Instr { self.An, self.Ai[5] =  8, v; return self }
func (self *Instr) A8(v uint8) *Instr { self.An, self.Ai[6] =  9, v; return self }
func (self *Instr) A9(v uint8) *Instr { self.An, self.Ai[7] = 10, v; return self }

func (self *Instr) isLabelBranch() bool {
    return self.Op >= OP_beq && self.Op <= OP_jal
}
