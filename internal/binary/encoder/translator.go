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
    `sync/atomic`

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
)

/** Function Prototype
 *
 *      func(buf *[]byte, p unsafe.Pointer, rs *RuntimeState) (err error)
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
    ET = atm.P6
    EP = atm.P7
)

const (
    RL = atm.R5
    RC = atm.R6
    ST = atm.R7
)

const (
    TR = atm.R4
    TP = atm.P2
)

const (
    LB_error      = "_error"
    LB_more_space = "_more_space"
)

func Translate(s Program) atm.Program {
    p := atm.CreateProgramBuilder()
    prologue  (p)
    program   (p, s)
    epilogue  (p)
    moreSpace (p)
    return p.Build()
}

func program(p *atm.ProgramBuilder, s Program) {
    for i, v := range s {
        p.Mark(i)
        translators[v.Op()](p, v)
    }
}

func prologue(p *atm.ProgramBuilder) {
    p.MOV   (atm.R0, ST)                // ST <=  0
    p.LDAP  (0, atm.P0)                 // P0 <=  a0
    p.LDAP  (1, WP)                     // WP <=  a1
    p.LDAP  (2, RS)                     // RS <=  a2
    p.IB    (8, atm.R1)                 // R1 <=  8
    p.LP    (atm.P0, RP)                // RP <= *P0
    p.ADDP  (atm.P0, atm.R1, atm.P0)    // P0 <=  P0 + R1
    p.LQ    (atm.P0, RL)                // RL <= *P0
    p.ADDP  (atm.P0, atm.R1, atm.P0)    // P0 <=  P0 + R1
    p.LQ    (atm.P0, RC)                // RC <= *P0
}

func epilogue(p *atm.ProgramBuilder) {
    p.MOVRP (atm.R0, ET)                // ET <= 0
    p.MOVRP (atm.R0, EP)                // EP <= 0
    p.Label (LB_error)                  // _error:
    p.STRP  (ET, 0)                     // r0 <= ET
    p.STRP  (EP, 1)                     // r1 <= EP
    p.RET   ()                          // return
}

var (
    byteType    = rt.UnpackEface(byte(0)).Type
    moreSpaceId = uint64(0)
)

func nextId() string {
    return fmt.Sprintf("_next_%x", atomic.AddUint64(&moreSpaceId, 1))
}

func moreSpace(p *atm.ProgramBuilder) {
    p.Label (LB_more_space)             // _more_space:
    p.IB    (2, TR)                     //  TR <= 2
    p.IP    (byteType, TP)              //  TP <= byteType
    p.MUL   (RC, TR, TR)                //  TR <= RC * TR
    p.CALL  (growslice).                //  CALL growslice:
      A0    (TP).                       //      et      <= TP
      A1    (RP).                       //      old.ptr <= RP
      A2    (RL).                       //      old.len <= RL
      A3    (RC).                       //      old.cap <= RC
      A4    (TR).                       //      cap     <= TR
      R0    (RP).                       //      RET.ptr => RP
      R1    (RL).                       //      RET.len => RL
      R2    (RC)                        //      RET.cap => RC
    p.LDAP  (0, TP)                     //  TP <= arg[0]
    p.IB    (8, TR)                     //  R1 <= 8
    p.SP    (RP, TP)                    // *TP <= RP
    p.ADDP  (TP, TR, TP)                //  TP <= TP + R1
    p.SQ    (RL, TP)                    // *TP <= RL
    p.ADDP  (TP, TR, TP)                //  TP <= TP + R1
    p.SQ    (RC, TP)                    // *TP <= RC
    p.JALR  (atm.LR, atm.Pn)            //  PC <= LR
}

func checkSize(p *atm.ProgramBuilder, n int) {
    s := nextId()
    p.IQ    (int64(n), TR)              // TR <= n
    p.ADD   (RL, TR, TR)                // TR <= RL + TR
    p.BGEU  (RC, TR, s)                 // if RC >= TR then PC <= &s
    p.JAL   (LB_more_space, atm.LR)     // JAL _more_space
    p.Label (s)
}

var translators = [...]func(*atm.ProgramBuilder, Instr) {

}
