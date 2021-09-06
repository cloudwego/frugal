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
    `reflect`
    `testing`
    `time`
    `unsafe`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/internal/atm`
    `github.com/davecgh/go-spew/spew`
    `github.com/stretchr/testify/require`
)

type TranslatorTestStruct struct {
    A bool              `frugal:"0,default,bool"`
    B int8              `frugal:"1,default,i8"`
    C float64           `frugal:"2,default,double"`
    D int16             `frugal:"3,default,i16"`
    E int32             `frugal:"4,default,i32"`
    F int64             `frugal:"5,default,i64"`
    G string            `frugal:"6,default,string"`
    H []byte            `frugal:"7,default,binary"`
    I []int32           `frugal:"8,default,list<i32>"`
    J map[string]string `frugal:"9,default,map<string:string>"`
}

func itab_SimpleIoVec() unsafe.Pointer {
    var v frugal.IoVec = (*frugal.SimpleIoVec)(nil)
    return *(*unsafe.Pointer)(unsafe.Pointer(&v))
}

func TestTranslator_Translate(t *testing.T) {
    v := &TranslatorTestStruct {
        A: true,
        B: 0x12,
        C: 12.34,
        D: 0x3456,
        E: 0x12345678,
        F: 0x66778899aabbccdd,
        G: "hello, world",
        H: []byte("testbytebuffer"),
        I: []int32{0x11223344, 0x55667788, 3, 4, 5},
        J: map[string]string{"asdf": "qwer", "zxcv": "hjkl"},
    }
    p, err := CreateCompiler().Compile(reflect.TypeOf(v).Elem())
    require.NoError(t, err)
    tr := Translate(p)
    println(tr.Disassemble())
    rs := new(RuntimeState)
    iov := new(frugal.SimpleIoVec)
    emu := atm.LoadProgram(tr)
    emu.Ap(0, itab_SimpleIoVec())
    emu.Ap(1, unsafe.Pointer(iov))
    emu.Ap(2, unsafe.Pointer(v))
    emu.Ap(3, unsafe.Pointer(rs))
    emu.Au(4, 0)
    t0 := time.Now()
    emu.Run()
    dt := time.Since(t0)
    emu.Free()
    r0 := emu.Rp(0)
    r1 := emu.Rp(1)
    err = *(*error)(unsafe.Pointer(&[2]unsafe.Pointer{r0, r1}))
    require.NoError(t, err)
    spew.Dump(iov.Buffer.Bytes())
    println("Emulator takes " + dt.String() + " to finish.")
}
