//go:build go1.16 && !go1.18

/*
 * Copyright 2022 CloudWeGo Authors
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
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/rt"
	"github.com/cloudwego/frugal/internal/jit/utils"
)

type _Func struct {
	entry       uintptr
	nameoff     int32
	args        int32
	deferreturn uint32
	pcsp        uint32
	pcfile      uint32
	pcln        uint32
	npcdata     uint32
	cuOffset    uint32
	funcID      uint8
	_           [2]byte
	nfuncdata   uint8
	pcdata      [2]uint32
	argptrs     uintptr
	localptrs   uintptr
}

type _FuncTab struct {
	entry   uintptr
	funcoff uintptr
}

type _PCHeader struct {
	magic          uint32
	pad1, pad2     uint8
	minLC          uint8
	ptrSize        uint8
	nfunc          int
	nfiles         uint
	funcnameOffset uintptr
	cuOffset       uintptr
	filetabOffset  uintptr
	pctabOffset    uintptr
	pclnOffset     uintptr
}

type _BitVector struct {
	n        int32
	bytedata *uint8
}

type _ModuleData struct {
	pcHeader              *_PCHeader
	funcnametab           []byte
	cutab                 []uint32
	filetab               []byte
	pctab                 []byte
	pclntable             []_Func
	ftab                  []_FuncTab
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
	itablinks             []unsafe.Pointer
	ptab                  [][2]int32
	pluginpath            string
	pkghashes             []struct{}
	modulename            string
	modulehashes          []struct{}
	hasmain               uint8
	gcdatamask, gcbssmask _BitVector
	typemap               map[int32]unsafe.Pointer
	bad                   bool
	next                  *_ModuleData
}

type _FindFuncBucket struct {
	idx        uint32
	subbuckets [16]byte
}

const minfunc = 16                 // minimum function size
const pcbucketsize = 256 * minfunc // size of bucket in the pc->func lookup table

var modHeader = &_PCHeader{
	magic:   0xfffffffa,
	minLC:   1,
	nfunc:   1,
	ptrSize: 4 << (^uintptr(0) >> 63),
}

var (
	emptyByte byte

	/* retain local reference of all buckets to bypass gc */
	bucketList = &utils.ListNode{}
)

func registerFunction(name string, pc uintptr, size uintptr, frame rt.Frame) {
	var pbase uintptr
	var sbase uintptr

	/* PC ranges */
	minpc := pc
	maxpc := pc + size
	pctab := make([]byte, 1)
	ffunc := make([]_FindFuncBucket, size/pcbucketsize+1)

	/* initialize the find function buckets */
	for i := range ffunc {
		ffunc[i].idx = 1
	}

	/* define the PC-SP ranges */
	for i, r := range frame.SpTab {
		nb := r.Nb
		ds := int(r.Sp - sbase)

		/* check for remaining size */
		if nb == 0 {
			if i == len(frame.SpTab)-1 {
				nb = size - pbase
			} else {
				panic("invalid PC-SP tab")
			}
		}

		/* check for the first entry */
		if i == 0 {
			pctab = append(pctab, encodeFirst(ds)...)
		} else {
			pctab = append(pctab, encodeValue(ds)...)
		}

		/* encode the length */
		sbase = r.Sp
		pbase = pbase + nb
		pctab = append(pctab, encodeVariant(int(nb))...)
	}

	/* pin the find function bucket */
	ftab := &ffunc[0]
	pctab = append(pctab, 0)
	bucketList.Prepend(unsafe.Pointer(ftab))

	/* function entry */
	fn := _Func{
		entry:     pc,
		nameoff:   1,
		args:      int32(frame.ArgSize),
		pcsp:      1,
		npcdata:   2,
		nfuncdata: 2,
		cuOffset:  1,
		argptrs:   frame.ArgPtrs.Pin(),
		localptrs: frame.LocalPtrs.Pin(),
	}

	/* mark the entire function as a single line of code */
	fn.pcln = uint32(len(pctab))
	fn.pcfile = uint32(len(pctab))
	pctab = append(pctab, encodeFirst(1)...)
	pctab = append(pctab, encodeVariant(int(size))...)
	pctab = append(pctab, 0)

	/* set the entire function to use stack map 0 */
	fn.pcdata[_PCDATA_StackMapIndex] = uint32(len(pctab))
	pctab = append(pctab, encodeFirst(0)...)
	pctab = append(pctab, encodeVariant(int(size))...)
	pctab = append(pctab, 0)

	/* mark the entire function as unsafe to async-preempt */
	fn.pcdata[_PCDATA_UnsafePoint] = uint32(len(pctab))
	pctab = append(pctab, encodeFirst(_PCDATA_UnsafePointUnsafe)...)
	pctab = append(pctab, encodeVariant(int(size))...)
	pctab = append(pctab, 0)

	/* function table */
	tab := []_FuncTab{
		{entry: pc},
		{entry: pc},
		{entry: maxpc},
	}

	/* module data */
	mod := &_ModuleData{
		pcHeader:    modHeader,
		funcnametab: append(append([]byte{0}, name...), 0),
		cutab:       []uint32{0, 0, 1},
		filetab:     []byte("\x00(jit-generated)\x00"),
		pctab:       pctab,
		pclntable:   []_Func{fn},
		ftab:        tab,
		findfunctab: ftab,
		minpc:       minpc,
		maxpc:       maxpc,
		modulename:  name,
		gcdata:      uintptr(unsafe.Pointer(&emptyByte)),
		gcbss:       uintptr(unsafe.Pointer(&emptyByte)),
	}

	/* verify and register the new module */
	moduledataverify1(mod)
	registerModule(mod)
}
