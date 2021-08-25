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
    `sync`

    `github.com/cloudwego/frugal/internal/rt`
)

var (
    instrPool          sync.Pool
    emulatorPool       sync.Pool
    programBuilderPool sync.Pool
)

func newInstr(op OpCode) *Instr {
    if v := instrPool.Get(); v == nil {
        return allocInstr(op)
    } else {
        return resetInstr(op, v.(*Instr))
    }
}

func freeInstr(p *Instr) {
    instrPool.Put(p)
}

func allocInstr(op OpCode) (p *Instr) {
    p = new(Instr)
    p.Op = op
    return
}

func resetInstr(op OpCode, p *Instr) *Instr {
    *p = Instr{Op: op}
    return p
}

func newEmulator() *Emulator {
    if v := emulatorPool.Get(); v == nil {
        return new(Emulator)
    } else {
        return resetEmulator(v.(*Emulator))
    }
}

func freeEmulator(p *Emulator) {
    emulatorPool.Put(p)
}

func resetEmulator(p *Emulator) *Emulator {
    *p = Emulator{}
    return p
}

func newProgramBuilder() *ProgramBuilder {
    if v := programBuilderPool.Get(); v == nil {
        return allocProgramBuilder()
    } else {
        return resetProgramBuilder(v.(*ProgramBuilder))
    }
}

func freeProgramBuilder(p *ProgramBuilder) {
    programBuilderPool.Put(p)
}

func allocProgramBuilder() (p *ProgramBuilder) {
    p       = new(ProgramBuilder)
    p.refs  = make(map[string]*Instr, 64)
    p.pends = make(map[string][]*Instr, 64)
    return
}

func resetProgramBuilder(p *ProgramBuilder) *ProgramBuilder {
    p.i    = 0
    p.head = nil
    p.tail = nil
    rt.MapClear(p.refs)
    rt.MapClear(p.pends)
    return p
}
