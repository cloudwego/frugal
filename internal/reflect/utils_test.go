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
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestCheckUniqueness(t *testing.T) {
	{ // tBOOL
		typ := &tType{T: tBOOL, RT: reflect.TypeOf(bool(true))}
		vv := []bool{true, false}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []bool{true, true}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
	{ // tI08
		typ := &tType{T: tI08, RT: reflect.TypeOf(int8(0))}
		vv := []int8{1, 2}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []int8{1, 1}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
	{ // tI16
		typ := &tType{T: tI16, RT: reflect.TypeOf(int16(0))}
		vv := []int16{1, 2}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []int16{1, 1}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
	{ // tI32
		typ := &tType{T: tI32, RT: reflect.TypeOf(int32(0))}
		vv := []int32{1, 2}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []int32{1, 1}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
	{ // tI64
		typ := &tType{T: tI64, RT: reflect.TypeOf(int64(0))}
		vv := []int64{1, 2}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []int64{1, 1}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
	{ // tDOUBLE
		typ := &tType{T: tDOUBLE, RT: reflect.TypeOf(float64(0))}
		vv := []float64{1, 2}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []float64{1, 1}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
	{ // tSTRING
		typ := &tType{T: tSTRING, RT: reflect.TypeOf(string(""))}
		vv := []string{"1", "2"}
		assert.NoError(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
		vv = []string{"1", "1"}
		assert.Error(t, checkUniqueness(typ, (*sliceHeader)(unsafe.Pointer(&vv))))
	}
}
