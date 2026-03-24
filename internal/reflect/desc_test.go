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

	"github.com/cloudwego/frugal/internal/assert"
	"github.com/cloudwego/frugal/internal/defs"
)

func TestStructDescFieldProperties(t *testing.T) {
	rv := reflect.ValueOf(&TestTypes{})
	desc, err := getOrcreateStructDesc(rv)
	assert.Nil(t, err)

	// Field 1: FBool - bool, required
	f1 := desc.GetField(1)
	assert.True(t, f1 != nil)
	assert.Equal(t, tBOOL, f1.Type.T)
	assert.Equal(t, defs.Required, f1.Spec)
	assert.Equal(t, 1, f1.Type.FixedSize)

	// Field 2: FByte - int8, default
	f2 := desc.GetField(2)
	assert.True(t, f2 != nil)
	assert.Equal(t, tBYTE, f2.Type.T)
	assert.Equal(t, defs.Default, f2.Spec)
	assert.Equal(t, 1, f2.Type.FixedSize)

	// Field 3: I8 - int8, default
	f3 := desc.GetField(3)
	assert.True(t, f3 != nil)
	assert.Equal(t, tBYTE, f3.Type.T)
	assert.Equal(t, defs.Default, f3.Spec)

	// Field 4: I16 - int16, default
	f4 := desc.GetField(4)
	assert.True(t, f4 != nil)
	assert.Equal(t, tI16, f4.Type.T)
	assert.Equal(t, defs.Default, f4.Spec)
	assert.Equal(t, 2, f4.Type.FixedSize)

	// Field 5: I32 - int32, default
	f5 := desc.GetField(5)
	assert.True(t, f5 != nil)
	assert.Equal(t, tI32, f5.Type.T)
	assert.Equal(t, defs.Default, f5.Spec)
	assert.Equal(t, 4, f5.Type.FixedSize)

	// Field 6: I64 - int64, default
	f6 := desc.GetField(6)
	assert.True(t, f6 != nil)
	assert.Equal(t, tI64, f6.Type.T)
	assert.Equal(t, defs.Default, f6.Spec)
	assert.Equal(t, 8, f6.Type.FixedSize)

	// Field 7: Double - float64, default
	f7 := desc.GetField(7)
	assert.True(t, f7 != nil)
	assert.Equal(t, tDOUBLE, f7.Type.T)
	assert.Equal(t, defs.Default, f7.Spec)
	assert.Equal(t, 8, f7.Type.FixedSize)

	// Field 8: String_ - string, default
	f8 := desc.GetField(8)
	assert.True(t, f8 != nil)
	assert.Equal(t, tSTRING, f8.Type.T)
	assert.Equal(t, defs.Default, f8.Spec)
	assert.Equal(t, 0, f8.Type.FixedSize)

	// Field 9: Binary - []byte, default (represented as tSTRING in wire format)
	f9 := desc.GetField(9)
	assert.True(t, f9 != nil)
	assert.Equal(t, tSTRING, f9.Type.T)
	assert.Equal(t, defs.Default, f9.Spec)
	assert.Equal(t, 0, f9.Type.FixedSize)

	// Field 10: Enum - Numberz, default
	f10 := desc.GetField(10)
	assert.True(t, f10 != nil)
	assert.Equal(t, tENUM, f10.Type.T)
	assert.Equal(t, defs.Default, f10.Spec)

	// Field 11: UID - UserID (int64), default
	f11 := desc.GetField(11)
	assert.True(t, f11 != nil)
	assert.Equal(t, tI64, f11.Type.T)
	assert.Equal(t, defs.Default, f11.Spec)

	// Field 12: S - *Msg, default
	f12 := desc.GetField(12)
	assert.True(t, f12 != nil)
	assert.Equal(t, tSTRUCT, f12.Type.T)
	assert.Equal(t, defs.Default, f12.Spec)
	assert.True(t, f12.Type.IsPointer)
	assert.True(t, f12.Type.Sd != nil)

	// Field 20: M0 - map[int32]int32, required
	f20 := desc.GetField(20)
	assert.True(t, f20 != nil)
	assert.Equal(t, tMAP, f20.Type.T)
	assert.Equal(t, defs.Required, f20.Spec)
	assert.True(t, f20.Type.K != nil)
	assert.Equal(t, tI32, f20.Type.K.T)
	assert.True(t, f20.Type.V != nil)
	assert.Equal(t, tI32, f20.Type.V.T)

	// Field 21: M1 - map[int32]string, default
	f21 := desc.GetField(21)
	assert.True(t, f21 != nil)
	assert.Equal(t, tMAP, f21.Type.T)
	assert.Equal(t, defs.Default, f21.Spec)
	assert.Equal(t, tI32, f21.Type.K.T)
	assert.Equal(t, tSTRING, f21.Type.V.T)

	// Field 22: M2 - map[int32]*Msg, default
	f22 := desc.GetField(22)
	assert.True(t, f22 != nil)
	assert.Equal(t, tMAP, f22.Type.T)
	assert.Equal(t, tI32, f22.Type.K.T)
	assert.Equal(t, tSTRUCT, f22.Type.V.T)
	assert.True(t, f22.Type.V.IsPointer)

	// Field 23: M3 - map[string]*Msg, default
	f23 := desc.GetField(23)
	assert.True(t, f23 != nil)
	assert.Equal(t, tMAP, f23.Type.T)
	assert.Equal(t, tSTRING, f23.Type.K.T)
	assert.Equal(t, tSTRUCT, f23.Type.V.T)

	// Field 30: L0 - []int32, required
	f30 := desc.GetField(30)
	assert.True(t, f30 != nil)
	assert.Equal(t, tLIST, f30.Type.T)
	assert.Equal(t, defs.Required, f30.Spec)
	assert.Equal(t, tI32, f30.Type.V.T)

	// Field 31: L1 - []string, default
	f31 := desc.GetField(31)
	assert.True(t, f31 != nil)
	assert.Equal(t, tLIST, f31.Type.T)
	assert.Equal(t, defs.Default, f31.Spec)
	assert.Equal(t, tSTRING, f31.Type.V.T)

	// Field 32: L2 - []*Msg, default
	f32 := desc.GetField(32)
	assert.True(t, f32 != nil)
	assert.Equal(t, tLIST, f32.Type.T)
	assert.Equal(t, tSTRUCT, f32.Type.V.T)
	assert.True(t, f32.Type.V.IsPointer)

	// Field 40: S0 - []int32, required (set)
	f40 := desc.GetField(40)
	assert.True(t, f40 != nil)
	assert.Equal(t, tSET, f40.Type.T)
	assert.Equal(t, defs.Required, f40.Spec)
	assert.Equal(t, tI32, f40.Type.V.T)

	// Field 41: S1 - []string, default (set)
	f41 := desc.GetField(41)
	assert.True(t, f41 != nil)
	assert.Equal(t, tSET, f41.Type.T)
	assert.Equal(t, defs.Default, f41.Spec)
	assert.Equal(t, tSTRING, f41.Type.V.T)

	// Field 50: LM - []map[int32]int32, default (list of maps)
	f50 := desc.GetField(50)
	assert.True(t, f50 != nil)
	assert.Equal(t, tLIST, f50.Type.T)
	assert.Equal(t, defs.Default, f50.Spec)
	assert.Equal(t, tMAP, f50.Type.V.T)
	// Check nested map types
	assert.Equal(t, tI32, f50.Type.V.K.T)
	assert.Equal(t, tI32, f50.Type.V.V.T)

	// Field 60: ML - map[int32][]int32, default (map with list values)
	f60 := desc.GetField(60)
	assert.True(t, f60 != nil)
	assert.Equal(t, tMAP, f60.Type.T)
	assert.Equal(t, defs.Default, f60.Spec)
	assert.Equal(t, tI32, f60.Type.K.T)
	assert.Equal(t, tLIST, f60.Type.V.T)
	// Check nested list element type
	assert.Equal(t, tI32, f60.Type.V.V.T)

	// Field 61: MS - map[int32][]int32, default (map with set values)
	f61 := desc.GetField(61)
	assert.True(t, f61 != nil)
	assert.Equal(t, tMAP, f61.Type.T)
	assert.Equal(t, defs.Default, f61.Spec)
	assert.Equal(t, tI32, f61.Type.K.T)
	assert.Equal(t, tSET, f61.Type.V.T)
	// Check nested set element type
	assert.Equal(t, tI32, f61.Type.V.V.T)
}
