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
)

var ArchRegs = [...]x86_64.Register64 {
    x86_64.RAX,
    x86_64.RCX,
    x86_64.RDX,
    x86_64.RBX,
    x86_64.RSP,
    x86_64.RBP,
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

var ArchRegNames = map[x86_64.Register64]string {
    x86_64.RAX : "rax",
    x86_64.RCX : "rcx",
    x86_64.RDX : "rdx",
    x86_64.RBX : "rbx",
    x86_64.RSP : "rsp",
    x86_64.RBP : "rbp",
    x86_64.RSI : "rsi",
    x86_64.RDI : "rdi",
    x86_64.R8  : "r8",
    x86_64.R9  : "r9",
    x86_64.R10 : "r10",
    x86_64.R11 : "r11",
    x86_64.R12 : "r12",
    x86_64.R13 : "r13",
    x86_64.R14 : "r14",
    x86_64.R15 : "r15",
}
