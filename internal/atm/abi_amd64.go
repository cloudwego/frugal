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
    `fmt`
    `reflect`
    `strings`
    `unsafe`

    `github.com/chenzhuoyu/iasm/x86_64`
    `github.com/cloudwego/frugal/internal/rt`
)

type Parameter struct {
    Mem        uintptr
    Reg        x86_64.Register64
    InRegister bool
}

func mkReg(reg x86_64.Register64) (p Parameter) {
    p.Reg = reg
    p.InRegister = true
    return
}

func mkStack(mem uintptr) (p Parameter) {
    p.Mem = mem
    p.InRegister = false
    return
}

func (self Parameter) String() string {
    if self.InRegister {
        return fmt.Sprintf("%%%s", self.Reg)
    } else {
        return fmt.Sprintf("%d(%%rsp)", self.Mem)
    }
}

type FunctionLayout struct {
    Id   int
    Sp   uintptr
    Args []Parameter
    Rets []Parameter
}

func (self *FunctionLayout) String() string {
    if self.Id < 0 {
        return fmt.Sprintf("{func,%s}", self.formatFn())
    } else {
        return fmt.Sprintf("{meth/%d,%s}", self.Id, self.formatFn())
    }
}

func (self *FunctionLayout) formatFn() string {
    return fmt.Sprintf("$%#x,(%s),(%s)", self.Sp, self.formatSeq(self.Args), self.formatSeq(self.Rets))
}

func (self *FunctionLayout) formatSeq(v []Parameter) string {
    nb := len(v)
    mm := make([]string, len(v))

    /* convert each part */
    for i := 0; i < nb; i++ {
        mm[i] = v[i].String()
    }

    /* join them together */
    return strings.Join(mm, ",")
}

type AMD64ABI struct {
    FnTab map[int]*FunctionLayout
}

func ArchCreateABI() *AMD64ABI {
    return &AMD64ABI {
        FnTab: make(map[int]*FunctionLayout),
    }
}

func (self *AMD64ABI) RegisterMethod(id int, mt rt.Method) int {
    self.FnTab[id] = self.LayoutFunc(mt.Id, mt.Vt.Pack().Method(mt.Id).Type)
    return mt.Id
}

func (self *AMD64ABI) RegisterFunction(id int, fn interface{}) (fp unsafe.Pointer) {
    vv := rt.UnpackEface(fn)
    vt := vv.Type.Pack()

    /* must be a function */
    if vt.Kind() != reflect.Func {
        panic("fn is not a function")
    }

    /* layout the function, and get the real function address */
    self.FnTab[id] = self.LayoutFunc(-1, vt)
    return *(*unsafe.Pointer)(vv.Value)
}
