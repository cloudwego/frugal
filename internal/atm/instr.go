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
    OP_ib OpCode = iota     // byte(Im) -> Rx
    OP_iw                   // word(Im) -> Rx
    OP_il                   // long(Im) -> Rx
    OP_iq                   // quad(Im) -> Rx
    OP_ldb                  // *(*byte)Pr -> Rx
    OP_ldw                  // *(*word)Pr -> Rx
    OP_ldl                  // *(*long)Pr -> Rx
    OP_ldq                  // *(*quad)Pr -> Rx
    OP_stb                  // Rx -> *(*byte)Pr
    OP_stw                  // Rx -> *(*word)Pr
    OP_stl                  // Rx -> *(*long)Pr
    OP_stq                  // Rx -> *(*quad)Pr
    OP_movpr                // Pr -> Rx
    OP_movrp                // Rx -> Pr
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
    OP_je                   // if (Rx == Ry) PC + Im -> PC
    OP_jne                  // if (Rx != Ry) PC + Im -> PC
    OP_jg                   // if (signed(Rx) >  signed(Ry)) PC + Im -> PC
    OP_jge                  // if (signed(Rx) >= signed(Ry)) PC + Im -> PC
    OP_jl                   // if (signed(Rx) <  signed(Ry)) PC + Im -> PC
    OP_jle                  // if (signed(Rx) <= signed(Ry)) PC + Im -> PC
    OP_ja                   // if (unsigned(Rx) >  unsigned(Ry)) PC + Im -> PC
    OP_jae                  // if (unsigned(Rx) >= unsigned(Ry)) PC + Im -> PC
    OP_jb                   // if (unsigned(Rx) <  unsigned(Ry)) PC + Im -> PC
    OP_jbe                  // if (unsigned(Rx) <= unsigned(Ry)) PC + Im -> PC
    OP_jmp                  // PC + Rx -> PC
    OP_jsr                  // PC -> P7; PC + Im -> PC
    OP_rts                  // P7 -> PC
    OP_call                 // call(Rx)
    OP_ret                  // return Rx;
    OP_yield                // save_state(); PC -> P7; return Rx; load_state()
)

type Instr struct {
    Op OpCode
    Im uint64
    Rx GenericRegister
    Ry GenericRegister
    Rz GenericRegister
    Pr PointerRegister
}
