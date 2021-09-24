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

package decoder

import (
    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
)

func emu_ccall_StateClearBitmap(e *atm.Emulator, p *atm.Instr) {
    if p.An != 1 || p.Rn != 0 || (p.Av[0] & atm.ArgPointer) == 0 {
        panic("invalid StateClearBitmap call")
    } else {
        (*StateItem)(e.Pr[p.Av[0] & atm.ArgMask]).Fm = [MaxBitmap]uint8{}
    }
}

func init() {
    FnStateClearBitmap = rt.FuncAddr(func(){})
    atm.RegisterCCall(FnStateClearBitmap, emu_ccall_StateClearBitmap)
}
