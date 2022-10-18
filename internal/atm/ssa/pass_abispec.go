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
    `fmt`
)

// ABILowering lowers ABI-specific instructions to machine specific instructions.
type ABILowering struct{}

func (self ABILowering) Apply(cfg *CFG) {
    rr := make([]Reg, len(cfg.Layout.Args))
    rb := rr[:0]

    /* map each argument load to register alias or stack load */
    for i, v := range cfg.Root.Ins {
        if iv, ok := v.(*IrLoadArg); !ok {
            break
        } else if iv.I < 0 || iv.I >= len(cfg.Layout.Args) {
            panic("abi: argument load out of bound: " + v.String())
        } else if rr[iv.I] != 0 {
            panic("abi: second load of the same argument: " + v.String())
        } else if a := cfg.Layout.Args[iv.I]; a.InRegister {
            rr[iv.I] = IrSetArch(Rz, a.Reg)
            cfg.Root.Ins[i] = &IrAlias { R: iv.R, V: rr[iv.I] }
        } else {
            rr[iv.I] = Rz
            cfg.Root.Ins[i] = IrArchLoadStack(iv.R, a.Mem, IrSlotArgs)
        }
    }

    /* extract all the registers */
    for _, r := range rr {
        if r != 0 && r != Rz {
            rb = append(rb, r)
        }
    }

    /* insert an entry point node to hold all the register arguments */
    if len(rb) != 0 {
        cfg.Root.Ins = append(
            []IrNode { &IrEntry { rb } },
            cfg.Root.Ins...
        )
    }

    /* lower the entire program */
    cfg.PostOrder().ForEach(func(bb *BasicBlock) {
        ins := bb.Ins
        bb.Ins = make([]IrNode, 0, len(ins))

        /* scan every instruction */
        for _, v := range ins {
            switch p := v.(type) {
                case *IrLoadArg    : panic(fmt.Sprintf("abi: argument load in the middle of CFG: bb_%d: %s", bb.Id, v))
                case *IrCallFunc   : self.abiCallFunc(cfg, bb, p)
                case *IrCallNative : self.abiCallNative(cfg, bb, p)
                case *IrCallMethod : self.abiCallMethod(cfg, bb, p)
                default            : bb.Ins = append(bb.Ins, p)
            }
        }

        /* scan the terminator */
        if p, ok := bb.Term.(*IrReturn); ok {
            rr = make([]Reg, len(p.R))
            rb = rr[:0]

            /* check for return value count */
            if len(p.R) != len(cfg.Layout.Rets) {
                panic("abi: return value store size mismatch: " + p.String())
            }

            /* copy return values */
            for i, rv := range p.R {
                if r := cfg.Layout.Rets[i]; r.InRegister {
                    rr[i] = IrSetArch(cfg.CreateRegister(rv.Ptr()), r.Reg)
                    bb.Ins = append(bb.Ins, IrArchCopy(rr[i], rv))
                } else {
                    rr[i] = Rz
                    bb.Ins = append(bb.Ins, IrArchStoreStack(rv, r.Mem, IrSlotArgs))
                }
            }

            /* extract all the registers */
            for _, r := range rr {
                if r != 0 && r != Rz {
                    rb = append(rb, r)
                }
            }

            /* replace the terminator */
            if len(rb) != 0 {
                bb.Term = IrArchReturn(rb)
            } else {
                bb.Term = IrArchReturn(nil)
            }
        }
    })
}
