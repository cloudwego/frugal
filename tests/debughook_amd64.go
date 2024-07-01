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

package tests

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/cloudwego/frugal/internal/rt"
)

//go:nosplit
//go:linkname rt_exit runtime.exit
func rt_exit(int)

//go:nosplit
func rt_exit_hook(r int) {
	if r != 0 {
		println("Non-zero exit code:", r)
		println("Now it's time to attach debugger, PID:", os.Getpid())
		for {
			_ = true
		}
	}
}

func mprotectpage(addr unsafe.Pointer, prot uintptr) {
	if _, _, err := syscall.RawSyscall(syscall.SYS_MPROTECT, uintptr(addr)&^4095, 4096, prot); err != 0 {
		panic(err)
	}
}

func init() {
	if os.Getenv("FRUGAL_DEBUGGER_HOOK") == "yes" {
		fp := rt.FuncAddr(rt_exit)
		to := rt.FuncAddr(rt_exit_hook)
		mprotectpage(fp, syscall.PROT_READ|syscall.PROT_WRITE)
		*(*[2]byte)(fp) = [2]byte{0x48, 0xba}
		*(*uintptr)(unsafe.Pointer(uintptr(fp) + 2)) = uintptr(to)
		*(*[2]byte)(unsafe.Pointer(uintptr(fp) + 10)) = [2]byte{0xff, 0xe2}
		mprotectpage(fp, syscall.PROT_READ|syscall.PROT_EXEC)
	}
}
