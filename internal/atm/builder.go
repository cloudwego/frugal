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
    `reflect`
    `strconv`
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

const (
    _LB_jump_pc = "_jump_pc_"
)

type Builder struct {
    i     int
    head  *Instr
    tail  *Instr
    refs  map[string]*Instr
    pends map[string][]*Instr
}

func CreateBuilder() *Builder {
    return newBuilder()
}

func (self *Builder) add(ins *Instr) *Instr {
    self.push(ins)
    return ins
}

func (self *Builder) jmp(p *Instr, to string) *Instr {
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

func (self *Builder) push(ins *Instr) {
    if self.head == nil {
        self.head = ins
        self.tail = ins
    } else {
        self.tail.Ln = ins
        self.tail    = ins
    }
}

func (self *Builder) Mark(pc int) {
    self.i++
    self.Label(_LB_jump_pc + strconv.Itoa(pc))
}

func (self *Builder) Label(to string) {
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

func (self *Builder) Build() (r Program) {
    var n int
    var p *Instr
    var q *Instr

    /* check for unresolved labels */
    for key := range self.pends {
        panic("labels are not fully resolved: " + key)
    }

    /* adjust jumps to point at actual instructions */
    for p = self.head; p != nil; p = p.Ln {
        if p.isBranch() {
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
        return
    }

    /* remove all the NOPs, there should be no jumps pointing to any NOPs */
    for p = self.head; p != nil; p, n = p.Ln, n + 1 {
        for p.Ln != nil && p.Ln.Op == OP_nop {
            q = p.Ln
            p.Ln = q.Ln
            freeInstr(q)
        }
    }

    /* the Builder's life-time ends here */
    freeBuilder(self)
    return Program{self.head}
}

func (self *Builder) NOP() *Instr {
    return self.add(newInstr(OP_nop))
}

func (self *Builder) IB(v int8, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_ib).ai(i8toa(v)).rx(rx))
}

func (self *Builder) IW(v int16, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_iw).ai(i16toa(v)).rx(rx))
}

func (self *Builder) IL(v int32, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_il).ai(i32toa(v)).rx(rx))
}

func (self *Builder) IQ(v int64, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_iq).ai(i64toa(v)).rx(rx))
}

func (self *Builder) IP(v interface{}, pd PointerRegister) *Instr {
    if vv := rt.UnpackEface(v); vv.Type.Kind() != reflect.Ptr {
        panic("v is not a pointer")
    } else {
        return self.add(newInstr(OP_ip).pr(vv.Value).pd(pd))
    }
}

func (self *Builder) LB(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_lb).ps(ps).rx(rx))
}

func (self *Builder) LW(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_lw).ps(ps).rx(rx))
}

func (self *Builder) LL(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_ll).ps(ps).rx(rx))
}

func (self *Builder) LQ(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_lq).ps(ps).rx(rx))
}

func (self *Builder) LP(ps PointerRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_lp).ps(ps).pd(pd))
}

func (self *Builder) SB(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sb).rx(rx).pd(pd))
}

func (self *Builder) SW(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sw).rx(rx).pd(pd))
}

func (self *Builder) SL(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sl).rx(rx).pd(pd))
}

func (self *Builder) SQ(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sq).rx(rx).pd(pd))
}

func (self *Builder) SP(ps PointerRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_sp).ps(ps).pd(pd))
}

func (self *Builder) MOV(rx GenericRegister, ry GenericRegister) *Instr {
    return self.add(newInstr(OP_mov).rx(rx).ry(ry))
}

func (self *Builder) MOVP(ps PointerRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_movp).ps(ps).pd(pd))
}

func (self *Builder) MOVPR(ps PointerRegister, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_movpr).ps(ps).rx(rx))
}

func (self *Builder) MOVRP(rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_movrp).rx(rx).pd(pd))
}

func (self *Builder) LDAQ(id int, rx GenericRegister) *Instr {
    return self.add(newInstr(OP_ldaq).ai(i64toa(int64(id))).rx(rx))
}

func (self *Builder) LDAP(id int, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_ldap).ai(i64toa(int64(id))).pd(pd))
}

func (self *Builder) STRQ(rx GenericRegister, id int) *Instr {
    return self.add(newInstr(OP_strq).rx(rx).ai(i64toa(int64(id))))
}

func (self *Builder) STRP(ps PointerRegister, id int) *Instr {
    return self.add(newInstr(OP_strp).ps(ps).ai(i64toa(int64(id))))
}

func (self *Builder) ADDP(ps PointerRegister, rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_addp).ps(ps).rx(rx).pd(pd))
}

func (self *Builder) SUBP(ps PointerRegister, rx GenericRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_subp).ps(ps).rx(rx).pd(pd))
}

func (self *Builder) ADD(rx GenericRegister, ry GenericRegister, rz GenericRegister) *Instr {
    return self.add(newInstr(OP_add).rx(rx).ry(ry).rz(rz))
}

func (self *Builder) SUB(rx GenericRegister, ry GenericRegister, rz GenericRegister) *Instr {
    return self.add(newInstr(OP_sub).rx(rx).ry(ry).rz(rz))
}

func (self *Builder) MUL(rx GenericRegister, ry GenericRegister, rz GenericRegister) *Instr {
    return self.add(newInstr(OP_mul).rx(rx).ry(ry).rz(rz))
}

func (self *Builder) SWAPW(rx GenericRegister, ry GenericRegister) *Instr {
    return self.add(newInstr(OP_swapw).rx(rx).ry(ry))
}

func (self *Builder) SWAPL(rx GenericRegister, ry GenericRegister) *Instr {
    return self.add(newInstr(OP_swapl).rx(rx).ry(ry))
}

func (self *Builder) SWAPQ(rx GenericRegister, ry GenericRegister) *Instr {
    return self.add(newInstr(OP_swapq).rx(rx).ry(ry))
}

func (self *Builder) BEQ(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_beq).rx(rx).ry(ry), to)
}

func (self *Builder) BNE(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bne).rx(rx).ry(ry), to)
}

func (self *Builder) BLT(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_blt).rx(rx).ry(ry), to)
}

func (self *Builder) BGE(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bge).rx(rx).ry(ry), to)
}

func (self *Builder) BLTU(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bltu).rx(rx).ry(ry), to)
}

func (self *Builder) BGEU(rx GenericRegister, ry GenericRegister, to string) *Instr {
    return self.jmp(newInstr(OP_bgeu).rx(rx).ry(ry), to)
}

func (self *Builder) JAL(to string, pd PointerRegister) *Instr {
    return self.jmp(newInstr(OP_jal).pd(pd), to)
}

func (self *Builder) JALI(to int, pd PointerRegister) *Instr {
    return self.jmp(newInstr(OP_jal).pd(pd), _LB_jump_pc + strconv.Itoa(to))
}

func (self *Builder) JALR(ps PointerRegister, pd PointerRegister) *Instr {
    return self.add(newInstr(OP_jalr).ps(ps).pd(pd))
}

func (self *Builder) HALT() *Instr {
    return self.add(newInstr(OP_halt))
}

func (self *Builder) CCALL(fn unsafe.Pointer) *Instr {
    return self.add(newInstr(OP_ccall).pr(fn))
}

func (self *Builder) GCALL(fn interface{}) *Instr {
    if fp := rt.FuncAddr(fn); gcallTab[fp] == nil {
        panic(fmt.Sprintf("gcall to unknown function: %p", fn))
    } else {
        return self.add(newInstr(OP_gcall).pr(rt.FuncAddr(fn)))
    }
}
