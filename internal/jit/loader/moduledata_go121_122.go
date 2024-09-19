//go:build go1.21 && !go1.23

/*
 * Copyright 2023 ByteDance Inc.
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
)

type _Func struct {
	entryOff    uint32
	nameoff     int32
	args        int32
	deferreturn uint32
	pcsp        uint32
	pcfile      uint32
	pcln        uint32
	npcdata     uint32
	cuOffset    uint32
	startLine   int32
	funcID      uint8
	flag        uint8
	_           [1]byte
	nfuncdata   uint8
	pcdata      [2]uint32
	argptrs     uint32
	localptrs   uint32
}

type _ModuleData struct {
	pcHeader              *_PCHeader
	funcnametab           []byte
	cutab                 []uint32
	filetab               []byte
	pctab                 []byte
	pclntable             []byte
	ftab                  []_FuncTab
	findfunctab           uintptr
	minpc, maxpc          uintptr
	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	covctrs, ecovctrs     uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr
	rodata                uintptr
	gofunc                uintptr
	textsectmap           [][3]uintptr
	typelinks             []int32
	itablinks             []*rt.GoItab
	ptab                  [][2]int32
	pluginpath            string
	pkghashes             []struct{}
	// This slice records the initializing tasks that need to be
	// done to start up the program. It is built by the linker.
	inittasks             []unsafe.Pointer
	modulename            string
	modulehashes          []struct{}
	hasmain               uint8
	gcdatamask, gcbssmask _BitVector
	typemap               map[int32]*rt.GoType
	bad                   bool
	next                  *_ModuleData
}

const (
	_ModuleMagic = 0xfffffff1
)
