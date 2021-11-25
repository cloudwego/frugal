// +build go1.17,!go1.18

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

/** Go Internal ABI implementation
 *
 *  This module implements the function layout algorithm described by the Go internal ABI.
 *  See https://github.com/golang/go/blob/master/src/cmd/compile/abi-internal.md for more info.
 */

package atm

import (
    `reflect`
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
)

const (
    _PS = 8 // pointer size
    _PA = 8 // pointer alignment
    _NI = 9 // number of integer registers
)

var (
    ptrType  = reflect.TypeOf(unsafe.Pointer(nil))
    regOrder = [_NI]x86_64.Register64{RAX, RBX, RCX, RDI, RSI, R8, R9, R10, R11}
)

type _StackAlloc struct {
    i int
    s uintptr
}

func (self *_StackAlloc) reg() (p Parameter) {
    p = mkReg(regOrder[self.i])
    self.i++
    return
}

func (self *_StackAlloc) spill(n uintptr, a int) uintptr {
    self.s = alignUp(self.s, a) + n
    return self.s
}

func (self *_StackAlloc) alloc(p []Parameter, vt reflect.Type) []Parameter {
    nc := 0
    nb := vt.Size()
    vk := vt.Kind()

    /* zero-sized objects are allocated on stack */
    if nb == 0 {
        return append(p, mkStack(self.s))
    }

    /* check for value type */
    switch vk {
        case reflect.Bool          : fallthrough
        case reflect.Int           : fallthrough
        case reflect.Int8          : fallthrough
        case reflect.Int16         : fallthrough
        case reflect.Int32         : fallthrough
        case reflect.Int64         : fallthrough
        case reflect.Uint          : fallthrough
        case reflect.Uint8         : fallthrough
        case reflect.Uint16        : fallthrough
        case reflect.Uint32        : fallthrough
        case reflect.Uint64        : fallthrough
        case reflect.Uintptr       : nc = 1
        case reflect.Float32       : fallthrough
        case reflect.Float64       : fallthrough
        case reflect.Complex64     : fallthrough
        case reflect.Complex128    : panic("abi: go117: not implemented: FP numbers")
        case reflect.Array         : panic("abi: go117: not implemented: arrays")
        case reflect.Chan          : fallthrough
        case reflect.Func          : fallthrough
        case reflect.Map           : fallthrough
        case reflect.Ptr           : fallthrough
        case reflect.UnsafePointer : nc = 1
        case reflect.Interface     : nc = 2
        case reflect.Slice         : nc = 3
        case reflect.String        : nc = 2
        case reflect.Struct        : panic("abi: go117: not implemented: structs")
    }

    /* if there are no more registers available, allocate on stack */
    if self.i + nc > _NI {
        return append(p, mkStack(self.s))
    }

    /* allocate all the register components */
    for i := 0; i < nc; i++ { p = append(p, self.reg()) }
    return p
}

func (self *AMD64ABI) layoutFunction(id int, ft reflect.Type) *FunctionLayout {
    var sa _StackAlloc
    var fn FunctionLayout

    /* allocate the receiver if any (interface call always uses pointer) */
    if id >= 0 {
        fn.Args = sa.alloc(fn.Args, ptrType)
    }

    /* assign every arguments */
    for i := 0; i < ft.NumIn(); i++ {
        fn.Args = sa.alloc(fn.Args, ft.In(i))
    }

    /* reset the register counter, and add a pointer alignment field */
    sa.i = 0
    sa.spill(0, _PA)

    /* assign every return value */
    for i := 0; i < ft.NumOut(); i++ {
        fn.Rets = sa.alloc(fn.Rets, ft.Out(i))
    }

    /* assign spill slots */
    for i := 0; i < len(fn.Args); i++ {
        if fn.Args[i].Tag == ByReg {
            fn.Args[i].Mem = sa.spill(_PS, _PA) - _PS
        }
    }

    /* add the final pointer alignment field */
    fn.Id = id
    fn.Sp = sa.spill(0, _PA)
    return &fn
}
