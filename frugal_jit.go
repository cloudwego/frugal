//go:build !go1.24 && amd64 && !windows

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

package frugal

import (
	"os"
	"reflect"
	"strconv"
	"sync"

	"github.com/cloudwego/frugal/internal/jit/decoder"
	"github.com/cloudwego/frugal/internal/jit/encoder"
	"github.com/cloudwego/frugal/internal/jit/rt"
	"github.com/cloudwego/frugal/internal/jit/utils"
	"github.com/cloudwego/frugal/internal/opts"
	"github.com/cloudwego/gopkg/protocol/thrift"
)

var nojit bool

func init() {
	if v, err := strconv.ParseBool(os.Getenv("FRUGAL_NO_JIT")); err == nil {
		nojit = v
	}
}

func jitEncodedSize(val interface{}) int {
	return encoder.EncodedSize(val)
}

func jitEncodeObject(buf []byte, w thrift.NocopyWriter, val interface{}) (int, error) {
	return encoder.EncodeObject(buf, w, val)
}

func jitDecodeObject(buf []byte, val interface{}) (int, error) {
	return decoder.DecodeObject(buf, val)
}

type _Ty struct {
	d  int
	ty *rt.GoType
}

var (
	typool sync.Pool
)

func newty(ty *rt.GoType, d int) *_Ty {
	if v := typool.Get(); v == nil {
		return &_Ty{d, ty}
	} else {
		r := v.(*_Ty)
		r.d, r.ty = d, ty
		return r
	}
}

// Pretouch compiles vt ahead-of-time to avoid JIT compilation on-the-fly, in
// order to reduce the first-hit latency.
func Pretouch(vt reflect.Type, options ...Option) error {
	if nojit {
		return nil
	}
	d := 0
	o := opts.GetDefaultOptions()

	/* apply all the options */
	for _, fn := range options {
		fn(&o)
	}

	/* unpack the type */
	v := make(map[*rt.GoType]bool)
	t := rt.Dereference(rt.UnpackType(vt))

	/* add the root type */
	q := utils.NewQueue()
	q.Enqueue(newty(t, 1))

	/* BFS the type tree */
	for !q.Empty() {
		ty := q.Dequeue().(*_Ty)
		tv, err := decoder.Pretouch(ty.ty, o)

		/* also pretouch the encoder */
		if err == nil {
			err = encoder.Pretouch(ty.ty, o)
		}

		/* mark the type as been visited */
		d, v[ty.ty] = ty.d, true
		typool.Put(ty)

		/* check for errors */
		if err != nil {
			return err
		}

		/* check for cutoff conditions */
		if !o.CanPretouch(d) {
			continue
		}

		/* add all the not visited sub-types */
		for s := range tv {
			if t = rt.UnpackType(s); !v[t] {
				q.Enqueue(newty(t, d+1))
			}
		}
	}

	/* completed with no errors */
	return nil
}
