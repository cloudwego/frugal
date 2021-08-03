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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either exPsess or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package atm

type OpCode byte

const (
    OP_ib OpCode = iota     // byte(Im) -> Rx
    OP_iw                   // word(Im) -> Rx
    OP_il                   // long(Im) -> Rx
    OP_iq                   // quad(Im) -> Rx
    OP_ldb                  // *(*byte)Ps -> Rx
    OP_ldw                  // *(*word)Ps -> Rx
    OP_ldl                  // *(*long)Ps -> Rx
    OP_ldq                  // *(*quad)Ps -> Rx
    OP_stb                  // Rx -> *(*byte)Pd
    OP_stw                  // Rx -> *(*word)Pd
    OP_stl                  // Rx -> *(*long)Pd
    OP_stq                  // Rx -> *(*quad)Pd
    OP_movps                // Ps -> Rx
    OP_movpd                // Rx -> Pd
    OP_addp                 // Ps + Rx -> Pd
    OP_subp                 // Ps - Rx -> Pd
    OP_add                  // Rx + Ry -> Rz
    OP_sub                  // Rx - Ry -> Rz
    OP_mul                  // Rx * Ry -> Rz
    OP_div                  // Rx / Ry -> Rz
    OP_mod                  // Rx % Ry -> Rz
    OP_and                  // Rx & Ry -> Rz
    OP_or                   // Rx | Ry -> Rz
    OP_xor                  // Rx ^ Ry -> Rz
    OP_shl                  // Rx << Ry -> Rz
    OP_shr                  // Rx >> Ry -> Rz
    OP_inv                  // ~Rx -> Ry
    OP_je                   // if (Rx == Ry) Br.PC -> PC
    OP_jne                  // if (Rx != Ry) Br.PC -> PC
    OP_jg                   // if (signed(Rx) >  signed(Ry)) Br.PC -> PC
    OP_jge                  // if (signed(Rx) >= signed(Ry)) Br.PC -> PC
    OP_jl                   // if (signed(Rx) <  signed(Ry)) Br.PC -> PC
    OP_jle                  // if (signed(Rx) <= signed(Ry)) Br.PC -> PC
    OP_ja                   // if (unsigned(Rx) >  unsigned(Ry)) Br.PC -> PC
    OP_jae                  // if (unsigned(Rx) >= unsigned(Ry)) Br.PC -> PC
    OP_jb                   // if (unsigned(Rx) <  unsigned(Ry)) Br.PC -> PC
    OP_jbe                  // if (unsigned(Rx) <= unsigned(Ry)) Br.PC -> PC
    OP_jmp                  // Br.PC -> PC
    OP_jsr                  // PC -> P7; Br.PC -> PC
    OP_rts                  // P7 -> PC
    OP_call                 // call(Rx)
    OP_ret                  // return Rx;
    OP_yield                // save_state(); PC -> P7; return Rx; load_state()
)

type Instr struct {
    PC int
    Op OpCode
    Im uint64
    Br *Instr
    Rx GenericRegister
    Ry GenericRegister
    Rz GenericRegister
    Ps PointerRegister
    Pd PointerRegister
}
