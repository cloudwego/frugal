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

	"github.com/cloudwego/frugal/internal/defs"
	"github.com/stretchr/testify/require"
)

func TestStructDescFieldProperties(t *testing.T) {
	rv := reflect.ValueOf(&TestTypes{})
	desc, err := getOrcreateStructDesc(rv)
	require.NoError(t, err)

	// Field 1: FBool - bool, required
	f1 := desc.GetField(1)
	require.NotNil(t, f1)
	require.Equal(t, tBOOL, f1.Type.T)
	require.Equal(t, defs.Required, f1.Spec)
	require.Equal(t, 1, f1.Type.FixedSize)

	// Field 2: FByte - int8, default
	f2 := desc.GetField(2)
	require.NotNil(t, f2)
	require.Equal(t, tBYTE, f2.Type.T)
	require.Equal(t, defs.Default, f2.Spec)
	require.Equal(t, 1, f2.Type.FixedSize)

	// Field 3: I8 - int8, default
	f3 := desc.GetField(3)
	require.NotNil(t, f3)
	require.Equal(t, tBYTE, f3.Type.T)
	require.Equal(t, defs.Default, f3.Spec)

	// Field 4: I16 - int16, default
	f4 := desc.GetField(4)
	require.NotNil(t, f4)
	require.Equal(t, tI16, f4.Type.T)
	require.Equal(t, defs.Default, f4.Spec)
	require.Equal(t, 2, f4.Type.FixedSize)

	// Field 5: I32 - int32, default
	f5 := desc.GetField(5)
	require.NotNil(t, f5)
	require.Equal(t, tI32, f5.Type.T)
	require.Equal(t, defs.Default, f5.Spec)
	require.Equal(t, 4, f5.Type.FixedSize)

	// Field 6: I64 - int64, default
	f6 := desc.GetField(6)
	require.NotNil(t, f6)
	require.Equal(t, tI64, f6.Type.T)
	require.Equal(t, defs.Default, f6.Spec)
	require.Equal(t, 8, f6.Type.FixedSize)

	// Field 7: Double - float64, default
	f7 := desc.GetField(7)
	require.NotNil(t, f7)
	require.Equal(t, tDOUBLE, f7.Type.T)
	require.Equal(t, defs.Default, f7.Spec)
	require.Equal(t, 8, f7.Type.FixedSize)

	// Field 8: String_ - string, default
	f8 := desc.GetField(8)
	require.NotNil(t, f8)
	require.Equal(t, tSTRING, f8.Type.T)
	require.Equal(t, defs.Default, f8.Spec)
	require.Equal(t, 0, f8.Type.FixedSize)

	// Field 9: Binary - []byte, default (represented as tSTRING in wire format)
	f9 := desc.GetField(9)
	require.NotNil(t, f9)
	require.Equal(t, tSTRING, f9.Type.T)
	require.Equal(t, defs.Default, f9.Spec)
	require.Equal(t, 0, f9.Type.FixedSize)

	// Field 10: Enum - Numberz, default
	f10 := desc.GetField(10)
	require.NotNil(t, f10)
	require.Equal(t, tENUM, f10.Type.T)
	require.Equal(t, defs.Default, f10.Spec)

	// Field 11: UID - UserID (int64), default
	f11 := desc.GetField(11)
	require.NotNil(t, f11)
	require.Equal(t, tI64, f11.Type.T)
	require.Equal(t, defs.Default, f11.Spec)

	// Field 12: S - *Msg, default
	f12 := desc.GetField(12)
	require.NotNil(t, f12)
	require.Equal(t, tSTRUCT, f12.Type.T)
	require.Equal(t, defs.Default, f12.Spec)
	require.True(t, f12.Type.IsPointer)
	require.NotNil(t, f12.Type.Sd)

	// Field 20: M0 - map[int32]int32, required
	f20 := desc.GetField(20)
	require.NotNil(t, f20)
	require.Equal(t, tMAP, f20.Type.T)
	require.Equal(t, defs.Required, f20.Spec)
	require.NotNil(t, f20.Type.K)
	require.Equal(t, tI32, f20.Type.K.T)
	require.NotNil(t, f20.Type.V)
	require.Equal(t, tI32, f20.Type.V.T)

	// Field 21: M1 - map[int32]string, default
	f21 := desc.GetField(21)
	require.NotNil(t, f21)
	require.Equal(t, tMAP, f21.Type.T)
	require.Equal(t, defs.Default, f21.Spec)
	require.Equal(t, tI32, f21.Type.K.T)
	require.Equal(t, tSTRING, f21.Type.V.T)

	// Field 22: M2 - map[int32]*Msg, default
	f22 := desc.GetField(22)
	require.NotNil(t, f22)
	require.Equal(t, tMAP, f22.Type.T)
	require.Equal(t, tI32, f22.Type.K.T)
	require.Equal(t, tSTRUCT, f22.Type.V.T)
	require.True(t, f22.Type.V.IsPointer)

	// Field 23: M3 - map[string]*Msg, default
	f23 := desc.GetField(23)
	require.NotNil(t, f23)
	require.Equal(t, tMAP, f23.Type.T)
	require.Equal(t, tSTRING, f23.Type.K.T)
	require.Equal(t, tSTRUCT, f23.Type.V.T)

	// Field 30: L0 - []int32, required
	f30 := desc.GetField(30)
	require.NotNil(t, f30)
	require.Equal(t, tLIST, f30.Type.T)
	require.Equal(t, defs.Required, f30.Spec)
	require.Equal(t, tI32, f30.Type.V.T)

	// Field 31: L1 - []string, default
	f31 := desc.GetField(31)
	require.NotNil(t, f31)
	require.Equal(t, tLIST, f31.Type.T)
	require.Equal(t, defs.Default, f31.Spec)
	require.Equal(t, tSTRING, f31.Type.V.T)

	// Field 32: L2 - []*Msg, default
	f32 := desc.GetField(32)
	require.NotNil(t, f32)
	require.Equal(t, tLIST, f32.Type.T)
	require.Equal(t, tSTRUCT, f32.Type.V.T)
	require.True(t, f32.Type.V.IsPointer)

	// Field 40: S0 - []int32, required (set)
	f40 := desc.GetField(40)
	require.NotNil(t, f40)
	require.Equal(t, tSET, f40.Type.T)
	require.Equal(t, defs.Required, f40.Spec)
	require.Equal(t, tI32, f40.Type.V.T)

	// Field 41: S1 - []string, default (set)
	f41 := desc.GetField(41)
	require.NotNil(t, f41)
	require.Equal(t, tSET, f41.Type.T)
	require.Equal(t, defs.Default, f41.Spec)
	require.Equal(t, tSTRING, f41.Type.V.T)

	// Field 50: LM - []map[int32]int32, default (list of maps)
	f50 := desc.GetField(50)
	require.NotNil(t, f50)
	require.Equal(t, tLIST, f50.Type.T)
	require.Equal(t, defs.Default, f50.Spec)
	require.Equal(t, tMAP, f50.Type.V.T)
	// Check nested map types
	require.Equal(t, tI32, f50.Type.V.K.T)
	require.Equal(t, tI32, f50.Type.V.V.T)

	// Field 60: ML - map[int32][]int32, default (map with list values)
	f60 := desc.GetField(60)
	require.NotNil(t, f60)
	require.Equal(t, tMAP, f60.Type.T)
	require.Equal(t, defs.Default, f60.Spec)
	require.Equal(t, tI32, f60.Type.K.T)
	require.Equal(t, tLIST, f60.Type.V.T)
	// Check nested list element type
	require.Equal(t, tI32, f60.Type.V.V.T)

	// Field 61: MS - map[int32][]int32, default (map with set values)
	f61 := desc.GetField(61)
	require.NotNil(t, f61)
	require.Equal(t, tMAP, f61.Type.T)
	require.Equal(t, defs.Default, f61.Spec)
	require.Equal(t, tI32, f61.Type.K.T)
	require.Equal(t, tSET, f61.Type.V.T)
	// Check nested set element type
	require.Equal(t, tI32, f61.Type.V.V.T)
}