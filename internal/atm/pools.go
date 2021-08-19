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
    programPool        sync.Pool
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

func newProgram(n int) Program {
    if v := programPool.Get(); v == nil {
        return make(Program, 0, n)
    } else {
        return resetProgram(v.(Program), n)
    }
}

func freeProgram(p Program) {
    programPool.Put(p)
}

func resetProgram(p Program, n int) Program {
    if cap(p) >= n {
        return p[:0]
    } else {
        return resizeProgram(p[:0], n)
    }
}

func resizeProgram(p Program, n int) Program {
    rt.GrowSlice(&p, n)
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
