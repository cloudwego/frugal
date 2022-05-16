// +build go1.15,!go1.16

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

package loader

import (
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

type _Func struct {
    entry       uintptr
    nameoff     int32
    args        int32
    deferreturn uint32
    pcsp        int32
    pcfile      int32
    pcln        int32
    npcdata     int32
    funcID      uint8
    _           [2]int8
    nfuncdata   uint8
    pcdata      [2]uint32
    argptrs     uintptr
    localptrs   uintptr
}

type _FuncTab struct {
    entry   uintptr
    funcoff uintptr
}

type _BitVector struct {
    n        int32
    bytedata *uint8
}

type _ModuleData struct {
    pclntable             []byte
    ftab                  []_FuncTab
    filetab               []uint32
    findfunctab           *_FindFuncBucket
    minpc, maxpc          uintptr
    text, etext           uintptr
    noptrdata, enoptrdata uintptr
    data, edata           uintptr
    bss, ebss             uintptr
    noptrbss, enoptrbss   uintptr
    end, gcdata, gcbss    uintptr
    types, etypes         uintptr
    textsectmap           [][3]uintptr
    typelinks             []int32
    itablinks             []*rt.GoItab
    ptab                  [][2]int32
    pluginpath            string
    pkghashes             []byte
    modulename            string
    modulehashes          []byte
    hasmain               uint8
    gcdatamask, gcbssmask _BitVector
    typemap               map[int32]*rt.GoType
    bad                   bool
    next                  *_ModuleData
}

type _FindFuncBucket struct {
    idx        uint32
    subbuckets [16]byte
}

const minfunc = 16                 // minimum function size
const pcbucketsize = 256 * minfunc // size of bucket in the pc->func lookup table

var (
	emptyByte  byte
    bucketList []*_FindFuncBucket
)

func registerFunction(name string, pc uintptr, size uintptr, frame rt.Frame) {
    minpc := pc
    maxpc := pc + size
    ffunc := make([]_FindFuncBucket, size / pcbucketsize + 1)

    /* initialize the find function buckets */
    for i := range ffunc {
        ffunc[i].idx = 1
    }

    /* pin the find function bucket */
    pfunc := &ffunc[0]
    bucketList = append(bucketList, pfunc)

    /* build the PC & line table */
    pclnt := []byte {
        0xfb, 0xff, 0xff, 0xff,     // magic   : 0xfffffffb
        0,                          // pad1    : 0
        0,                          // pad2    : 0
        1,                          // minLC   : 1
        4 << (^uintptr(0) >> 63),   // ptrSize : 4 << (^uintptr(0) >> 63)
    }

    /* add the function name */
    noff := len(pclnt)
    pclnt = append(append(pclnt, name...), 0)

    /* add PCDATA */
    pcsp := len(pclnt)
    pbase := uintptr(0)
    sbase := uintptr(0)

    /* define the PC-SP ranges */
    for i, r := range frame.SpTab {
        nb := r.Nb
        ds := int(r.Sp - sbase)

        /* check for remaining size */
        if nb == 0 {
            if i == len(frame.SpTab) - 1 {
                nb = size - pbase
            } else {
                panic("invalid PC-SP tab")
            }
        }

        /* check for the first entry */
        if i == 0 {
            pclnt = append(pclnt, encodeFirst(ds)...)
        } else {
            pclnt = append(pclnt, encodeValue(ds)...)
        }

        /* encode the length */
        sbase = r.Sp
        pbase = pbase + nb
        pclnt = append(pclnt, encodeVariant(int(nb))...)
    }

    /* add the final zero */
    args := frame.ArgSize
    pclnt = append(pclnt, 0)

    /* function entry */
    fn := _Func {
        entry     : pc,
        nameoff   : int32(noff),
        args      : int32(args),
        pcsp      : int32(pcsp),
        npcdata   : 2,
        nfuncdata : 2,
        argptrs   : frame.ArgPtrs.Pin(),
        localptrs : frame.LocalPtrs.Pin(),
    }

    /* mark the entire function as a single line of code */
    fn.pcln = int32(len(pclnt))
    fn.pcfile = int32(len(pclnt))
    pclnt = append(pclnt, encodeFirst(1)...)
    pclnt = append(pclnt, encodeVariant(int(size))...)
    pclnt = append(pclnt, 0)

    /* set the entire function to use stack map 0 */
    fn.pcdata[_PCDATA_StackMapIndex] = uint32(len(pclnt))
    pclnt = append(pclnt, encodeFirst(0)...)
    pclnt = append(pclnt, encodeVariant(int(size))...)
    pclnt = append(pclnt, 0)

    /* mark the entire function as unsafe to async-preempt */
    fn.pcdata[_PCDATA_UnsafePoint] = uint32(len(pclnt))
    pclnt = append(pclnt, encodeFirst(_PCDATA_UnsafePointUnsafe)...)
    pclnt = append(pclnt, encodeVariant(int(size))...)
    pclnt = append(pclnt, 0)

    /* align the func to 8 bytes */
    if p := len(pclnt) % 8; p != 0 {
        pclnt = append(pclnt, make([]byte, 8 - p)...)
    }

    /* add the function descriptor */
    foff := len(pclnt)
    pclnt = append(pclnt, (*(*[unsafe.Sizeof(_Func{})]byte)(unsafe.Pointer(&fn)))[:]...)

    /* function table */
    tab := []_FuncTab {
        {entry: pc, funcoff: uintptr(foff)},
        {entry: pc, funcoff: uintptr(foff)},
        {entry: maxpc},
    }

    /* add the file name */
    soff := uint32(len(pclnt))
    pclnt = append(pclnt, "(jit-generated)\x00"...)

    /* module data */
    mod := &_ModuleData {
        pclntable   : pclnt,
        ftab        : tab,
        filetab     : []uint32{0, soff},
        findfunctab : pfunc,
        minpc       : minpc,
        maxpc       : maxpc,
        modulename  : name,
        gcdata      : uintptr(unsafe.Pointer(&emptyByte)), 
        gcbss       : uintptr(unsafe.Pointer(&emptyByte)),
    }

    /* verify and register the new module */
    moduledataverify1(mod)
    registerModule(mod)
}
