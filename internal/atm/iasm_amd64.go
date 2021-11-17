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
    `math`

    `github.com/chenzhuoyu/iasm/x86_64`
)

const (
    RAX = x86_64.RAX
    RCX = x86_64.RCX
    RDX = x86_64.RDX
    RBX = x86_64.RBX
    RSP = x86_64.RSP
    RBP = x86_64.RBP
    RSI = x86_64.RSI
    RDI = x86_64.RDI
    R8  = x86_64.R8
    R9  = x86_64.R9
    R10 = x86_64.R10
    R11 = x86_64.R11
    R12 = x86_64.R12
    R13 = x86_64.R13
    R14 = x86_64.R14
    R15 = x86_64.R15
)

const (
    NA x86_64.Register64 = 0xff
)

var defaultRegs = [16]x86_64.Register64 {
    NA, NA, NA, NA, NA, NA, NA, NA,
    NA, NA, NA, NA, NA, NA, NA, NA,
}

var allocationOrder = [14]x86_64.Register64 {
    R10, R11, R12, R13, R14, R15,   // reserved registers first
    RAX,                            // then the return value
    RBX,                            // then RBX
    R9, R8, RCX, RDX, RSI, RDI,     // then argument registers in reverse order
}

func Ptr(base x86_64.Register, disp int32) *x86_64.MemoryOperand {
    return x86_64.Ptr(base, disp)
}

func isInt32(v int64) bool {
    return v >= math.MinInt32 && v <= math.MaxUint32
}
