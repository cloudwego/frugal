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

package decoder

import (
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/rt"
	"github.com/cloudwego/frugal/internal/jit/utils"
	"github.com/cloudwego/frugal/internal/opts"
)

type Decoder func(
	buf unsafe.Pointer,
	nb int,
	i int,
	p unsafe.Pointer,
	rs *RuntimeState,
	st int,
) (int, error)

var (
	MissCount uint64 = 0
	TypeCount uint64 = 0
)

var (
	programCache = utils.CreateProgramCache()
)

func decode(vt *rt.GoType, buf unsafe.Pointer, nb int, i int, p unsafe.Pointer, rs *RuntimeState, st int) (int, error) {
	if dec, err := resolve(vt); err != nil {
		return 0, err
	} else {
		return dec(buf, nb, i, p, rs, st)
	}
}

func resolve(vt *rt.GoType) (Decoder, error) {
	var err error
	var val interface{}

	/* fast-path: type is cached */
	if val = programCache.Get(vt); val != nil {
		return val.(Decoder), nil
	}

	/* record the cache miss, and compile the type */
	atomic.AddUint64(&MissCount, 1)
	val, err = programCache.Compute(vt, compile)

	/* check for errors */
	if err != nil {
		return nil, err
	}

	/* record the successful compilation */
	atomic.AddUint64(&TypeCount, 1)
	return val.(Decoder), nil
}

func compile(vt *rt.GoType) (interface{}, error) {
	if pp, err := CreateCompiler().CompileAndFree(vt.Pack()); err != nil {
		return nil, err
	} else {
		return Link(Translate(pp)), nil
	}
}

func mkcompile(ty map[reflect.Type]struct{}, opts opts.Options) func(*rt.GoType) (interface{}, error) {
	return func(vt *rt.GoType) (interface{}, error) {
		cc := CreateCompiler()
		pp, err := cc.Apply(opts).Compile(vt.Pack())

		/* add all the deferred types */
		for t := range cc.d {
			ty[t] = struct{}{}
		}

		/* translate and link the program */
		if err != nil {
			return nil, err
		} else {
			return Link(Translate(pp)), nil
		}
	}
}

type DecodeError struct {
	vt *rt.GoType
}

func (self DecodeError) Error() string {
	if self.vt == nil {
		return "frugal: unmarshal to nil interface"
	} else if self.vt.Kind() == reflect.Ptr {
		return "frugal: unmarshal to nil " + self.vt.String()
	} else {
		return "frugal: unmarshal to non-pointer " + self.vt.String()
	}
}

func Pretouch(vt *rt.GoType, opts opts.Options) (map[reflect.Type]struct{}, error) {
	var err error
	var ret map[reflect.Type]struct{}

	/* check for cached types */
	if programCache.Get(vt) != nil {
		return nil, nil
	}

	/* compile & load the type */
	ret = make(map[reflect.Type]struct{})
	_, err = programCache.Compute(vt, mkcompile(ret, opts))

	/* check for errors */
	if err != nil {
		return nil, err
	}

	/* add the type count */
	atomic.AddUint64(&TypeCount, 1)
	return ret, nil
}

func DecodeObject(buf []byte, val interface{}) (ret int, err error) {
	vv := rt.UnpackEface(val)
	vt := vv.Type

	/* check for nil interface */
	if vt == nil || vv.Value == nil || vt.Kind() != reflect.Ptr {
		return 0, DecodeError{vt}
	}

	/* create a new runtime state */
	et := rt.PtrElem(vt)
	st := newRuntimeState()
	sl := (*rt.GoSlice)(unsafe.Pointer(&buf))

	/* call the encoder, and return the runtime state into pool */
	ret, err = decode(et, sl.Ptr, sl.Len, 0, vv.Value, st, 0)
	freeRuntimeState(st)
	return
}
