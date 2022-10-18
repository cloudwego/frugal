/*
 * Copyright 2022 ByteDance Inc.
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

package ssa

import (
    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/atm/abi`
    `github.com/cloudwego/frugal/internal/rt`
)

var _AbiClobbersC = []x86_64.Register64 {
    x86_64.RAX,
    x86_64.RCX,
    x86_64.RDX,
    x86_64.RSI,
    x86_64.RDI,
    x86_64.R8,
    x86_64.R9,
    x86_64.R10,
    x86_64.R11,
}

var _AbiClobbersGo = []x86_64.Register64 {
    x86_64.RAX,
    x86_64.RCX,
    x86_64.RDX,
    x86_64.RBX,
    x86_64.RSI,
    x86_64.RDI,
    x86_64.R8,
    x86_64.R9,
    x86_64.R10,
    x86_64.R11,
    x86_64.R12,
    x86_64.R13,
    x86_64.R14,
    x86_64.R15,
}

var _NativeArgsOrder = [...]x86_64.Register64 {
    x86_64.RDI,
    x86_64.RSI,
    x86_64.RDX,
    x86_64.RCX,
    x86_64.R8,
    x86_64.R9,
}

func (ABILowering) abiCallFunc(cfg *CFG, bb *BasicBlock, p *IrCallFunc) {
    argc := len(p.In)
    retc := len(p.Out)

    /* check argument & return value count */
    if argc != len(p.Func.Args) || retc != len(p.Func.Rets) {
        panic("abi: gcall argument count mismatch: " + p.String())
    }

    /* register buffer */
    argv := make([]Reg, 0, argc)
    retv := make([]Reg, 0, retc)
    clob := make([]Reg, 0, len(_AbiClobbersGo))
    rmap := make(map[x86_64.Register64]bool, len(_AbiClobbersGo))

    /* add all arch registers */
    for _, r := range _AbiClobbersGo {
        rmap[r] = true
    }

    /* store each argument */
    for i, r := range p.In {
        if v := p.Func.Args[i]; !v.InRegister {
            mm := v.Mem
            bb.Ins = append(bb.Ins, IrArchStoreStack(r, mm, IrSlotCall))
        } else {
            rr := IrSetArch(cfg.CreateRegister(r.Ptr()), v.Reg)
            bb.Ins, argv = append(bb.Ins, IrArchCopy(rr, r)), append(argv, rr)
        }
    }

    /* convert each return register */
    for i, r := range p.Out {
        if v := p.Func.Rets[i]; v.InRegister && r.Kind() != K_zero {
            retv = append(retv, IrSetArch(cfg.CreateRegister(r.Ptr()), v.Reg))
            delete(rmap, v.Reg)
        }
    }

    /* exclude return values from clobbering list (they are implied) */
    for _, r := range _AbiClobbersGo {
        if rmap[r] {
            clob = append(clob, IrSetArch(cfg.CreateRegister(false), r))
        }
    }

    /* add the call instruction */
    bb.Ins = append(bb.Ins, &IrAMD64_CALL_reg {
        Fn   : p.R,
        In   : argv,
        Out  : retv,
        Clob : clob,
    })

    /* declare clobber list if any */
    if len(clob) != 0 {
        bb.Ins = append(bb.Ins, IrMarkClobber(clob...))
    }

    /* load each return value */
    for i, r := range p.Out {
        if r.Kind() != K_zero {
            if v := p.Func.Rets[i]; v.InRegister {
                bb.Ins = append(bb.Ins, IrArchCopy(r, retv[i]))
            } else {
                bb.Ins = append(bb.Ins, IrArchLoadStack(r, v.Mem, IrSlotCall))
            }
        }
    }
}

func (ABILowering) abiCallNative(cfg *CFG, bb *BasicBlock, p *IrCallNative) {
    retv := Rz
    argc := len(p.In)

    /* check for argument count */
    if argc > len(_NativeArgsOrder) {
        panic("abi: too many native arguments: " + p.String())
    }

    /* register buffers */
    argv := make([]Reg, 0, argc)
    clob := make([]Reg, 0, len(_AbiClobbersC))
    rmap := make(map[x86_64.Register64]bool, len(_AbiClobbersC))

    /* add all arch registers */
    for _, r := range _AbiClobbersC {
        rmap[r] = true
    }

    /* convert each argument */
    for i, r := range p.In {
        argv = append(argv, IrSetArch(cfg.CreateRegister(r.Ptr()), _NativeArgsOrder[i]))
        bb.Ins = append(bb.Ins, IrArchCopy(argv[i], r))
    }

    /* allocate register for return value if needed */
    if p.Out.Kind() != K_zero {
        retv = IrSetArch(cfg.CreateRegister(p.Out.Ptr()), x86_64.RAX)
        delete(rmap, x86_64.RAX)
    }

    /* exclude return values from clobbering list (they are implied) */
    for _, r := range _AbiClobbersC {
        if rmap[r] {
            clob = append(clob, IrSetArch(cfg.CreateRegister(false), r))
        }
    }

    /* add the call instruction */
    bb.Ins = append(bb.Ins, &IrAMD64_CALL_reg {
        Fn   : p.R,
        In   : argv,
        Out  : []Reg { retv },
        Clob : clob,
    })

    /* declare clobber list if any */
    if len(clob) != 0 {
        bb.Ins = append(bb.Ins, IrMarkClobber(clob...))
    }

    /* copy the return value if needed */
    if p.Out.Kind() != K_zero {
        bb.Ins = append(bb.Ins, IrArchCopy(p.Out, retv))
    }
}

func (ABILowering) abiCallMethod(cfg *CFG, bb *BasicBlock, p *IrCallMethod) {
    argc := len(p.In) + 1
    retc := len(p.Out)

    /* check argument & return value count */
    if argc != len(p.Func.Args) || retc != len(p.Func.Rets) {
        panic("abi: icall argument count mismatch: " + p.String())
    }

    /* register buffer */
    argv := make([]Reg, 0, argc)
    retv := make([]Reg, 0, retc)
    clob := make([]Reg, 0, len(_AbiClobbersGo))
    rmap := make(map[x86_64.Register64]bool, len(_AbiClobbersGo))

    /* add all arch registers */
    for _, r := range _AbiClobbersGo {
        rmap[r] = true
    }

    /* store the receiver */
    if rx := p.Func.Args[0]; !rx.InRegister {
        mm := p.Func.Args[0].Mem
        bb.Ins = append(bb.Ins, IrArchStoreStack(p.V, mm, IrSlotCall))
    } else {
        rr := IrSetArch(cfg.CreateRegister(p.V.Ptr()), rx.Reg)
        bb.Ins, argv = append(bb.Ins, IrArchCopy(rr, p.V)), append(argv, rr)
    }

    /* store each argument */
    for i, r := range p.In {
        if v := p.Func.Args[i+1]; !v.InRegister {
            mm := v.Mem
            bb.Ins = append(bb.Ins, IrArchStoreStack(r, mm, IrSlotCall))
        } else {
            rr := IrSetArch(cfg.CreateRegister(r.Ptr()), v.Reg)
            bb.Ins, argv = append(bb.Ins, IrArchCopy(rr, r)), append(argv, rr)
        }
    }

    /* convert each return register */
    for i, r := range p.Out {
        if v := p.Func.Rets[i]; v.InRegister && r.Kind() != K_zero {
            retv = append(retv, IrSetArch(cfg.CreateRegister(r.Ptr()), v.Reg))
            delete(rmap, v.Reg)
        }
    }

    /* exclude return values from clobbering list (they are implied) */
    for _, r := range _AbiClobbersGo {
        if rmap[r] {
            clob = append(clob, IrSetArch(cfg.CreateRegister(false), r))
        }
    }

    /* add the call instruction */
    bb.Ins = append(bb.Ins, &IrAMD64_CALL_mem {
        Fn   : Ptr(p.T, int32(rt.GoItabFuncBase) + int32(p.Slot) * abi.PtrSize),
        In   : argv,
        Out  : retv,
        Clob : clob,
    })

    /* declare clobber list if any */
    if len(clob) != 0 {
        bb.Ins = append(bb.Ins, IrMarkClobber(clob...))
    }

    /* load each return value */
    for i, r := range p.Out {
        if r.Kind() != K_zero {
            if v := p.Func.Rets[i]; v.InRegister {
                bb.Ins = append(bb.Ins, IrArchCopy(r, retv[i]))
            } else {
                bb.Ins = append(bb.Ins, IrArchLoadStack(r, v.Mem, IrSlotCall))
            }
        }
    }
}