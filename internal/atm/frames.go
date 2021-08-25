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
    `unsafe`

    `github.com/cloudwego/frugal/internal/defs`
    `github.com/cloudwego/frugal/internal/rt`
)

var (
    frameCache sync.Map
)

type Frame struct {
    t *rt.GoType
    p sync.Pool
}

func (self *Frame) newBuffer() Buffer {
    if v := self.p.Get(); v != nil {
        return Buffer{v.(unsafe.Pointer)}
    } else {
        return Buffer{rt.MallocGC(self.t.Size, self.t, true)}
    }
}

func (self *Frame) freeBuffer(p Buffer) {
    self.p.Put(p.m)
}

type Buffer struct {
    m unsafe.Pointer
}

func (self Buffer) u(i int) *uint64 {
    return (*uint64)(unsafe.Pointer(uintptr(self.m) + uintptr(i) * defs.PtrSize))
}

func (self Buffer) p(i int) *unsafe.Pointer {
    return (*unsafe.Pointer)(unsafe.Pointer(uintptr(self.m) + uintptr(i) * defs.PtrSize))
}

type BitVector struct {
    bits uint32
    data []byte
}

func (self *BitVector) ensure() {
    if self.bits % 8 == 0 {
        self.data = append(self.data, 0)
    }
}

func (self *BitVector) append(v uint8) {
    self.ensure()
    self.data[self.bits / 8] |= v << (self.bits % 8)
    self.bits++
}

func (self *BitVector) addValue(off int) {
    for self.bits < uint32(off / defs.PtrSize) {
        self.append(0)
    }
}

func (self *BitVector) addPointer(off int) {
    self.addValue(off)
    self.append(1)
}

func newFrame(p *Instr) *Frame {
    vt := new(rt.GoType)
    bv := new(BitVector)

    /* add the arguments */
    for i := 0; i < p.An; i++ {
        if (p.Ai[i] & ArgPointer) != 0 {
            bv.addPointer(i * defs.PtrSize)
        }
    }

    /* add the return values */
    for i := 0; i < p.Rn; i++ {
        if (p.Rv[i] & ArgPointer) != 0 {
            bv.addPointer((p.An + i) * defs.PtrSize)
        }
    }

    /* set GC data if any */
    if bv.bits != 0 {
        vt.GCData = &bv.data[0]
    }

    /* other type attributes */
    vt.Size    = uintptr((p.An + p.Rn) * defs.PtrSize)
    vt.Align   = defs.PtrSize
    vt.PtrData = uintptr(bv.bits) * defs.PtrSize

    /* construct the frame */
    return &Frame {
        t: vt,
        p: sync.Pool{},
    }
}

func findFrame(p *Instr) *Frame {
    var ok bool
    var vv interface{}

    /* try direct loading */
    if vv, ok = frameCache.Load(p); ok {
        return vv.(*Frame)
    }

    /* not found, create a new frame */
    vv, _ = frameCache.LoadOrStore(p, newFrame(p))
    return vv.(*Frame)
}
