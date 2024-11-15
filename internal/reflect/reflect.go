/*
 * Copyright 2024 CloudWeGo Authors
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

package reflect

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

func EncodedSize(v interface{}) int {
	panicIfHackErr()
	rv := reflect.ValueOf(v)
	sd := getStructDesc(rv) // copy get and create funcs here for inlining
	if sd == nil {
		var err error
		sd, err = createStructDesc(rv)
		if err != nil {
			panic(fmt.Sprintf("unexpected err when parse fields: %s", err))
		}
	}
	// get underlying pointer
	var p unsafe.Pointer
	if rv.Kind() == reflect.Struct {
		// unaddressable, need to copy to heap, and then get the ptr
		prv := sd.rvPool.Get().(*reflect.Value)
		defer sd.rvPool.Put(prv)
		(*prv).Elem().Set(rv)
		p = (*rvtype)(unsafe.Pointer(prv)).ptr // like `rvPtr` without copy
	} else {
		// we doesn't support multilevel Pointer like **struct
		// it checks in createStructDesc
		p = rvPtr(rv)
	}

	t := &tType{Sd: sd}
	n, err := t.EncodedSize(p)
	if err != nil {
		panic(fmt.Sprintf("unexpected err: %s", err))
	}
	return n
}

func Append(b []byte, v interface{}) ([]byte, error) {
	panicIfHackErr()

	var err error
	rv := reflect.ValueOf(v)
	sd := getStructDesc(rv) // copy get and create funcs here for inlining
	if sd == nil {
		sd, err = createStructDesc(rv)
		if err != nil {
			return b, err
		}
	}
	// get underlying pointer
	var p unsafe.Pointer
	if rv.Kind() == reflect.Struct {
		// unaddressable, need to copy to heap, and then get the ptr
		prv := sd.rvPool.Get().(*reflect.Value)
		defer sd.rvPool.Put(prv)
		(*prv).Elem().Set(rv)
		p = (*rvtype)(unsafe.Pointer(prv)).ptr // like `rvPtr` without copy
	} else {
		// we doesn't support multilevel Pointer like **struct
		// it checks in createStructDesc
		p = rvPtr(rv)
	}
	return appendStruct(&tType{Sd: sd}, b, p)
}

func Decode(b []byte, v interface{}) (int, error) {
	panicIfHackErr()
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return 0, errors.New("not a pointer")
	}
	if rv.IsNil() {
		return 0, errors.New("can't decode nil pointer")
	}
	if rv.Elem().Kind() != reflect.Struct {
		return 0, errors.New("not a pointer to a struct")
	}
	sd, err := getOrcreateStructDesc(rv)
	if err != nil {
		return 0, err
	}
	d := decoderPool.Get().(*tDecoder)
	n, err := d.Decode(b, rv.UnsafePointer(), sd, maxDepthLimit)
	decoderPool.Put(d)
	return n, err
}
