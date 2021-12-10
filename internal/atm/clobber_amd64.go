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
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/rt`
    `github.com/oleiade/lane`
    `golang.org/x/arch/x86/x86asm`
)

var branchTable = map[x86asm.Op]bool {
    x86asm.JA    : true,
    x86asm.JAE   : true,
    x86asm.JB    : true,
    x86asm.JBE   : true,
    x86asm.JCXZ  : true,
    x86asm.JE    : true,
    x86asm.JECXZ : true,
    x86asm.JG    : true,
    x86asm.JGE   : true,
    x86asm.JL    : true,
    x86asm.JLE   : true,
    x86asm.JMP   : true,
    x86asm.JNE   : true,
    x86asm.JNO   : true,
    x86asm.JNP   : true,
    x86asm.JNS   : true,
    x86asm.JO    : true,
    x86asm.JP    : true,
    x86asm.JRCXZ : true,
    x86asm.JS    : true,
}

var registerTable = map[x86asm.Reg]x86_64.Register64 {
    x86asm.AL   : RAX,
    x86asm.CL   : RCX,
    x86asm.DL   : RDX,
    x86asm.BL   : RBX,
    x86asm.AH   : RAX,
    x86asm.CH   : RCX,
    x86asm.DH   : RDX,
    x86asm.BH   : RBX,
    x86asm.SPB  : RSP,
    x86asm.BPB  : RBP,
    x86asm.SIB  : RSI,
    x86asm.DIB  : RDI,
    x86asm.R8B  : R8,
    x86asm.R9B  : R9,
    x86asm.R10B : R10,
    x86asm.R11B : R11,
    x86asm.R12B : R12,
    x86asm.R13B : R13,
    x86asm.R14B : R14,
    x86asm.R15B : R15,
    x86asm.AX   : RAX,
    x86asm.CX   : RCX,
    x86asm.DX   : RDX,
    x86asm.BX   : RBX,
    x86asm.SP   : RSP,
    x86asm.BP   : RBP,
    x86asm.SI   : RSI,
    x86asm.DI   : RDI,
    x86asm.R8W  : R8,
    x86asm.R9W  : R9,
    x86asm.R10W : R10,
    x86asm.R11W : R11,
    x86asm.R12W : R12,
    x86asm.R13W : R13,
    x86asm.R14W : R14,
    x86asm.R15W : R15,
    x86asm.EAX  : RAX,
    x86asm.ECX  : RCX,
    x86asm.EDX  : RDX,
    x86asm.EBX  : RBX,
    x86asm.ESP  : RSP,
    x86asm.EBP  : RBP,
    x86asm.ESI  : RSI,
    x86asm.EDI  : RDI,
    x86asm.R8L  : R8,
    x86asm.R9L  : R9,
    x86asm.R10L : R10,
    x86asm.R11L : R11,
    x86asm.R12L : R12,
    x86asm.R13L : R13,
    x86asm.R14L : R14,
    x86asm.R15L : R15,
    x86asm.RAX  : RAX,
    x86asm.RCX  : RCX,
    x86asm.RDX  : RDX,
    x86asm.RBX  : RBX,
    x86asm.RSP  : RSP,
    x86asm.RBP  : RBP,
    x86asm.RSI  : RSI,
    x86asm.RDI  : RDI,
    x86asm.R8   : R8,
    x86asm.R9   : R9,
    x86asm.R10  : R10,
    x86asm.R11  : R11,
    x86asm.R12  : R12,
    x86asm.R13  : R13,
    x86asm.R14  : R14,
    x86asm.R15  : R15,
}

type _InstrBlock struct {
    ret    bool
    size   uintptr
    entry  unsafe.Pointer
    links [2]*_InstrBlock
}

func newInstrBlock(entry unsafe.Pointer) *_InstrBlock {
    return &_InstrBlock{entry: entry}
}

func (self *_InstrBlock) pc() unsafe.Pointer {
    return unsafe.Pointer(uintptr(self.entry) + self.size)
}

func (self *_InstrBlock) code() []byte {
    return rt.BytesFrom(self.pc(), 15, 15)
}

func (self *_InstrBlock) commit(size int) {
    self.size += uintptr(size)
}

func resolveClobberSet(fn interface{}) map[x86_64.Register64]bool {
    buf := lane.NewQueue()
    ret := make(map[x86_64.Register64]bool)
    bmp := make(map[unsafe.Pointer]*_InstrBlock)

    /* build the CFG with BFS */
    for buf.Enqueue(newInstrBlock(rt.FuncAddr(fn))); !buf.Empty(); {
        val := buf.Dequeue()
        cfg := val.(*_InstrBlock)

        /* parse every instruction in the block */
        for !cfg.ret {
            var err error
            var ins x86asm.Inst

            /* decode one instruction */
            if ins, err = x86asm.Decode(cfg.code(), 64); err != nil {
                panic(err)
            } else {
                cfg.commit(ins.Len)
            }

            /* calling to other functions, cannot analyze */
            if ins.Op == x86asm.CALL {
                return nil
            }

            /* simple algorithm: every write to register is treated as clobbering */
            if ins.Op == x86asm.MOV {
                if reg, ok := ins.Args[0].(x86asm.Reg); ok {
                    if rr, rok := registerTable[reg]; rok && !freeRegisters[rr] {
                        ret[rr] = true
                    }
                }
            }

            /* check for returns */
            if ins.Op == x86asm.RET {
                cfg.ret = true
                break
            }

            /* check for branches */
            if !branchTable[ins.Op] {
                continue
            }

            /* calculate branch address */
            links := [2]unsafe.Pointer {
                cfg.pc(),
                unsafe.Pointer(uintptr(cfg.pc()) + uintptr(ins.Args[0].(x86asm.Rel))),
            }

            /* link the next blocks */
            for i := 0; i < 2; i++ {
                if cfg.links[i] = bmp[links[i]]; cfg.links[i] == nil {
                    cfg.links[i] = newInstrBlock(links[i])
                    bmp[links[i]] = cfg.links[i]
                }
            }

            /* add the branches if not returned, if either one returns, mark the block returned */
            for i := 0; i < 2; i++ {
                if cfg.links[i].ret {
                    cfg.ret = true
                } else {
                    buf.Enqueue(cfg.links[i])
                }
            }
        }
    }

    /* all done */
    return ret
}
