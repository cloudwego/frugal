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

package encoder

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/cloudwego/frugal/internal/jit/rt"
	"github.com/cloudwego/frugal/internal/jit/utils"
	"github.com/cloudwego/frugal/internal/opts"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

type Encoder func(
	buf unsafe.Pointer,
	len int,
	mem thrift.NocopyWriter,
	p unsafe.Pointer,
	rs *RuntimeState,
	st int,
) (int, error)

var (
	MissCount uint64 = 0
	TypeCount uint64 = 0
)

var programCache = utils.CreateProgramCache()

func encode(vt *rt.GoType, buf unsafe.Pointer, len int, mem thrift.NocopyWriter, p unsafe.Pointer, rs *RuntimeState, st int) (int, error) {
	if enc, err := resolve(vt); err != nil {
		return -1, err
	} else {
		return enc(buf, len, mem, p, rs, st)
	}
}

func resolve(vt *rt.GoType) (Encoder, error) {
	var err error
	var val interface{}

	/* fast-path: type is cached */
	if val = programCache.Get(vt); val != nil {
		return val.(Encoder), nil
	}

	/* record the cache miss, and compile the type */
	atomic.AddUint64(&MissCount, 1)
	val, err = programCache.Compute(vt, compile)
	/* check for errors */ if err != nil {
		return nil, err
	}

	/* record the successful compilation */
	atomic.AddUint64(&TypeCount, 1)
	return val.(Encoder), nil
}

func compile(vt *rt.GoType) (interface{}, error) {
	if pp, err := CreateCompiler().CompileAndFree(vt.Pack()); err != nil {
		return nil, err
	} else {
		return Link(Translate(pp)), nil
	}
}

func mkcompile(opts opts.Options) func(*rt.GoType) (interface{}, error) {
	return func(vt *rt.GoType) (interface{}, error) {
		if pp, err := CreateCompiler().Apply(opts).CompileAndFree(vt.Pack()); err != nil {
			return nil, err
		} else {
			return Link(Translate(pp)), nil
		}
	}
}

func Pretouch(vt *rt.GoType, opts opts.Options) error {
	if programCache.Get(vt) != nil {
		return nil
	} else if _, err := programCache.Compute(vt, mkcompile(opts)); err != nil {
		return err
	} else {
		atomic.AddUint64(&TypeCount, 1)
		return nil
	}
}

func EncodedSize(val interface{}) int {
	if ret, err := EncodeObject(nil, nil, val); err != nil {
		panic(fmt.Errorf("frugal: cannot measure encoded size: %w", err))
	} else {
		return ret
	}
}

func EncodeObject(buf []byte, mem thrift.NocopyWriter, val interface{}) (ret int, err error) {
	rst := newRuntimeState()
	efv := rt.UnpackEface(val)
	out := (*rt.GoSlice)(unsafe.Pointer(&buf))

	if mem == nil {
		// starting from go1.22,
		// even though `mem`==nil, it may equal to (0x0, 0xc0000fe1e0).
		// it keeps original data pointer with itab = nil.
		// this would cause JIT panic when we only use data pointer to call its methods.
		// updating `mem` to nil, (0x0, 0xc0000fe1e0) -> (0x0, 0x0), is a quick fix for this case.
		mem = nil
	}

	/* check for indirect types */
	if efv.Type.IsIndirect() {
		ret, err = encode(efv.Type, out.Ptr, out.Len, mem, efv.Value, rst, 0)
	} else {
		/* avoid an extra mallocgc which is expensive for small objects */
		rst.Val = efv.Value
		ret, err = encode(efv.Type, out.Ptr, out.Len, mem, unsafe.Pointer(&rst.Val), rst, 0)
		/* remove reference to avoid leak since rst will be reused */
		rst.Val = nil
	}

	/* return the state into pool */
	freeRuntimeState(rst)
	return
}
