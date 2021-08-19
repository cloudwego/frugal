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
    `strconv`
    `strings`
)

const (
    _LB_jump_pc = "_jump_pc_"
)

type Program []*Instr

func (self Program) Free() {
    for _, v := range self { freeInstr(v) }
    freeProgram(self)
}

type ProgramBuilder struct {
    i     int
    head  *Instr
    tail  *Instr
    refs  map[string]*Instr
    pends map[string][]*Instr
}

func CreateProgramBuilder() *ProgramBuilder {
    return newProgramBuilder()
}

func (self *ProgramBuilder) add(ins *Instr) *Instr {
    self.push(ins)
    return ins
}

func (self *ProgramBuilder) jmp(p *Instr, to string) *Instr {
    var ok bool
    var lb *Instr

    /* placeholder substitution */
    if strings.Contains(to, "{n}") {
        to = strings.ReplaceAll(to, "{n}", strconv.Itoa(self.i))
    }

    /* check for backward jumps */
    if lb, ok = self.refs[to]; !ok {
        self.pends[to] = append(self.pends[to], p)
    }

    /* add to instruction buffer */
    p.Br = lb
    return self.add(p)
}

func (self *ProgramBuilder) push(ins *Instr) {
    if self.head == nil {
        self.head = ins
        self.tail = ins
    } else {
        self.tail.Ln = ins
        self.tail    = ins
    }
}

func (self *ProgramBuilder) Mark(pc int) {
    self.i++
    self.Label(_LB_jump_pc + strconv.Itoa(pc))
}

func (self *ProgramBuilder) Label(to string) {
    var p *Instr
    var v []*Instr

    /* placeholder substitution */
    if strings.Contains(to, "{n}") {
        to = strings.ReplaceAll(to, "{n}", strconv.Itoa(self.i))
    }

    /* check for duplications */
    if _, ok := self.refs[to]; ok {
        panic("label " + to + " has already been linked")
    }

    /* get the pending links */
    p = self.NOP()
    v = self.pends[to]

    /* patch all the pending jumps */
    for _, q := range v {
        q.Br = p
    }

    /* mark the label as resolved */
    self.refs[to] = p
    delete(self.pends, to)
}

func (self *ProgramBuilder) Build() (r []*Instr) {
    var n int
    var p *Instr

    /* check for unresolved labels */
    for key := range self.pends {
        panic("labels are not fully resolved: " + key)
    }

    /* adjust jumps to point at actual instructions */
    for p = self.head; p != nil; p = p.Ln {
        if p.isLabelBranch() {
            for p.Br.Ln != nil && p.Br.Op == OP_nop {
                p.Br = p.Br.Ln
            }
        }
    }

    /* remove NOPs at the front */
    for self.head != nil && self.head.Op == OP_nop {
        self.head = self.head.Ln
    }

    /* no instructions left, the program was composed entirely by NOPs */
    if self.head == nil {
        self.tail = nil
        return nil
    }

    /* remove all the NOPs, there should be no jumps pointing to any NOPs */
    for p = self.head; p != nil; p, n = p.Ln, n + 1 {
        for p.Ln != nil && p.Ln.Op == OP_nop {
            p.Ln = p.Ln.Ln
        }
    }

    /* allocate space for result */
    p = self.head
    r = newProgram(n)

    /* dump the instructions */
    for p != nil {
        r = append(r, p)
        p = p.Ln
    }

    /* the ProgramBuilder's life-time ends here */
    freeProgramBuilder(self)
    return
}

func (self *ProgramBuilder) NOP() *Instr {
    return self.add(newInstr(OP_nop))
}

func (self *ProgramBuilder) IB(v int8, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_ib).ai(i8toa(v)).rx(rx))
}

func (self *ProgramBuilder) IW(v int16, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_iw).ai(i16toa(v)).rx(rx))
}

func (self *ProgramBuilder) IL(v int32, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_il).ai(i32toa(v)).rx(rx))
}

func (self *ProgramBuilder) IQ(v int64, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_iq).ai(i64toa(v)).rx(rx))
}

func (self *ProgramBuilder) LB(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_lb).ps(ps).rx(rx))
}

func (self *ProgramBuilder) LW(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_lw).ps(ps).rx(rx))
}

func (self *ProgramBuilder) LL(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_ll).ps(ps).rx(rx))
}

func (self *ProgramBuilder) LQ(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_lq).ps(ps).rx(rx))
}

func (self *ProgramBuilder) LP(ps PointerRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_lp).ps(ps).pd(pd))
}

func (self *ProgramBuilder) SB(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sb).rx(rx).pd(pd))
}

func (self *ProgramBuilder) SW(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sw).rx(rx).pd(pd))
}

func (self *ProgramBuilder) SL(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sl).rx(rx).pd(pd))
}

func (self *ProgramBuilder) SQ(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sq).rx(rx).pd(pd))
}

func (self *ProgramBuilder) SP(ps PointerRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sp).ps(ps).pd(pd))
}

func (self *ProgramBuilder) MOV(rx GenericRegister, ry GenericRegister) *Instr {
    return self.add(newInstr(OP_mov).rx(rx).ry(ry))
}

func (self *ProgramBuilder) MOVPR(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_movpr).ps(ps).rx(rx))
}

func (self *ProgramBuilder) MOVRP(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_movrp).rx(rx).pd(pd))
}

func (self *ProgramBuilder) ADDP(ps PointerRegister, rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_addp).ps(ps).rx(rx).pd(pd))
}

func (self *ProgramBuilder) SUBP(ps PointerRegister, rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_subp).ps(ps).rx(rx).pd(pd))
}

func (self *ProgramBuilder) ADD(rx GenericRegister, ry GenericRegister, rz GenericRegister) *Instr {
    return self.add(newInstr(OP_add).rx(rx).ry(ry).rz(rz))
}

func (self *ProgramBuilder) SUB(rx GenericRegister, ry GenericRegister, rz GenericRegister) *Instr {
    return self.add(newInstr(OP_sub).rx(rx).ry(ry).rz(rz))
}

func (self *ProgramBuilder) MUL(rx GenericRegister, ry GenericRegister, rz GenericRegister) *Instr {
    return self.add(newInstr(OP_mul).rx(rx).ry(ry).rz(rz))
}

func (self *ProgramBuilder) BEQ(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_beq).rx(rx).ry(ry), to)
}

func (self *ProgramBuilder) BNE(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bne).rx(rx).ry(ry), to)
}

func (self *ProgramBuilder) BLT(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_blt).rx(rx).ry(ry), to)
}

func (self *ProgramBuilder) BGE(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bge).rx(rx).ry(ry), to)
}

func (self *ProgramBuilder) BLTU(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bltu).rx(rx).ry(ry), to)
}

func (self *ProgramBuilder) BGEU(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bgeu).rx(rx).ry(ry), to)
}

func (self *ProgramBuilder) JAL(rx GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_jal).rx(rx), to)
}

func (self *ProgramBuilder) JALI(rx GenericRegister, to int) *Instr {
    return self.jmp(newInstr(OP_jal).rx(rx), _LB_jump_pc + strconv.Itoa(to))
}

func (self *ProgramBuilder) JALR(rx GenericRegister, ry GenericRegister) *Instr {
    return self.add(newInstr(OP_jalr).rx(rx).ry(ry))
}

func (self *ProgramBuilder) CALL_GO() *Instr {
    return self.add(newInstr(OP_call_go))
}

func (self *ProgramBuilder) RET() *Instr {
    return self.add(newInstr(OP_ret))
}
