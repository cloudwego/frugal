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

package tests

import (
    `math`
    `math/rand`
    `reflect`
)

func GenValue(v interface{}) {
    vv := reflect.ValueOf(v)
    if vv.Kind() != reflect.Ptr {
        panic("v is not a pointer")
    }
    genvalue(vv.Elem())
}

func mkint(v uint64) int64 {
    return int64((v << 63) | (v >> 1))
}

func genvalue(v reflect.Value) {
    switch v.Kind() {
        case reflect.Bool    : v.SetBool(rand.Float64() >= 0.5)
        case reflect.Int8    : fallthrough
        case reflect.Int16   : fallthrough
        case reflect.Int32   : fallthrough
        case reflect.Int64   : v.SetInt(mkint(rand.Uint64()))
        case reflect.Float64 : v.SetFloat(rand.NormFloat64())
        case reflect.Map     : genmap(v)
        case reflect.Ptr     : genptr(v)
        case reflect.Slice   : genslice(v)
        case reflect.String  : v.SetString(genstring())
        case reflect.Struct  : genstruct(v)
        default              : panic("unsupported type for thrift: " + v.Type().String())
    }
}

func genptr(v reflect.Value) {
    t := v.Type()
    if rand.Float64() < 0.5 {
        v.Set(reflect.Zero(t))
        return
    }
    v.Set(reflect.New(t.Elem()))
    genvalue(v.Elem())
}

func genmap(v reflect.Value) {
    t := v.Type()
    if rand.Float64() < 0.5 {
        v.Set(reflect.Zero(t))
        return
    }
    n := 1
    v.Set(reflect.MakeMap(t))
    for i := 0; i < n; i++ {
        k := reflect.New(t.Key()).Elem()
        e := reflect.New(t.Elem()).Elem()
        genvalue(k)
        genvalue(e)
        v.SetMapIndex(k, e)
    }
}

func genslice(v reflect.Value) {
    t := v.Type()
    if rand.Float64() < 0.5 {
        v.Set(reflect.Zero(t))
        return
    }
    if t.Elem().Kind() == reflect.Uint8 {
        b := make([]byte, rand.Intn(16))
        _, _ = rand.Read(b)
        v.SetBytes(b)
        return
    }
    n := rand.Intn(2)
    v.Set(reflect.MakeSlice(t, n, n))
    for i := 0; i < n; i++ {
        vv := v.Index(i)
        dup := true
        for dup {
            dup = false
            genvalue(vv)
            if vv.CanInterface() {
                a := vv.Interface()
                for j := 0; j < i; j++ {
                    b := v.Index(j).Interface()
                    if reflect.DeepEqual(a, b) {
                        dup = true
                        break
                    }
                }
            }
        }
    }
}

func genstruct(v reflect.Value) {
    for i := 0; i < v.NumField(); i++ {
        genvalue(v.Field(i))
    }
}

func genstring() string {
    n := rand.Intn(16)
    c := make([]rune, n)
    for i := range c {
        f := math.Abs(rand.NormFloat64() * 64 + 32)
        if f > 0x10ffff {
            f = 0x10ffff
        }
        c[i] = rune(f)
    }
    return string(c)
}
