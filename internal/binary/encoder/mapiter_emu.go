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
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm`
    `github.com/cloudwego/frugal/internal/rt`
)

func getiter(e *atm.Emulator, p *atm.Instr, fn string) (it *rt.GoMapIterator) {
    if p.An != 1 || p.Rn != 0 || (p.Ai[0] & atm.ArgPointer) == 0 {
        panic("invalid " + fn + " call")
    } else {
        return (*rt.GoMapIterator)(e.Pr[p.Ai[0] & atm.ArgMask])
    }
}

func emu_gcall_mapiternext(e *atm.Emulator, p *atm.Instr) {
    mapiternext(getiter(e, p, "mapiternext"))
}


func emu_gcall_MapEndIterator(e *atm.Emulator, p *atm.Instr) {
    MapEndIterator(getiter(e, p, "MapEndIterator"))
}

func emu_gcall_MapBeginIterator(e *atm.Emulator, p *atm.Instr) {
    var v0 uint8
    var v1 uint8
    var v2 uint8

    /* check for arguments and return values */
    if (p.An != 2 || p.Rn != 1) ||
       (p.Ai[0] & atm.ArgPointer) == 0 ||
       (p.Ai[1] & atm.ArgPointer) == 0 ||
       (p.Rv[0] & atm.ArgPointer) == 0 {
        panic("invalid MapBeginIterator call")
    }

    /* extract the arguments and return value index */
    v0 = p.Ai[0] & atm.ArgMask
    v1 = p.Ai[1] & atm.ArgMask
    v2 = p.Rv[0] & atm.ArgMask

    /* call the function */
    e.Pr[v2] = unsafe.Pointer(MapBeginIterator(
        (*rt.GoMapType) (e.Pr[v0]),
        (*rt.GoMap)     (e.Pr[v1]),
    ))
}

func init() {
    atm.RegisterGCall(mapiternext      , emu_gcall_mapiternext)
    atm.RegisterGCall(MapEndIterator   , emu_gcall_MapEndIterator)
    atm.RegisterGCall(MapBeginIterator , emu_gcall_MapBeginIterator)
}
