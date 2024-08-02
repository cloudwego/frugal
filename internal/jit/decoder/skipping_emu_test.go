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
	"encoding/hex"
	"testing"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
	"github.com/cloudwego/frugal/internal/jit/rt"
)

func run_skipping_emu(t *testing.T, v []byte, exp int, tag defs.Tag) {
	var sb _skipbuf_t
	mm := *(*rt.GoSlice)(unsafe.Pointer(&v))
	rv := do_skip(&sb, mm.Ptr, mm.Len, tag)
	if rv != exp {
		if rv >= 0 && exp < 0 {
			t.Errorf("got %d while expecting error %d", rv, exp)
		} else if rv < 0 && exp >= 0 {
			t.Errorf("got unexpected error %d while expecting %d", rv, exp)
		} else if rv < 0 && exp < 0 {
			t.Errorf("got a wrong error %d while expecting error %d", rv, exp)
		} else {
			t.Errorf("got unexpected return value %d while expecting %d", rv, exp)
		}
		t.FailNow()
	}
}

func TestSkippingEmu_SkipPrimitives(t *testing.T) {
	run_skipping_emu(t, []byte{0}, 1, defs.T_bool)
	run_skipping_emu(t, []byte{0}, 1, defs.T_i8)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4, 5, 6, 7}, 8, defs.T_double)
	run_skipping_emu(t, []byte{0, 1}, 2, defs.T_i16)
	run_skipping_emu(t, []byte{0, 1, 2, 3}, 4, defs.T_i32)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4, 5, 6, 7}, 8, defs.T_i64)
}

func TestSkippingEmu_SkipPrimitivesButGotEOF(t *testing.T) {
	run_skipping_emu(t, []byte{}, EEOF, defs.T_bool)
	run_skipping_emu(t, []byte{}, EEOF, defs.T_i8)
	run_skipping_emu(t, []byte{}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0, 1}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0, 1, 2}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0, 1, 2, 3}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4, 5}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4, 5, 6}, EEOF, defs.T_double)
	run_skipping_emu(t, []byte{}, EEOF, defs.T_i16)
	run_skipping_emu(t, []byte{0}, EEOF, defs.T_i16)
	run_skipping_emu(t, []byte{}, EEOF, defs.T_i32)
	run_skipping_emu(t, []byte{0}, EEOF, defs.T_i32)
	run_skipping_emu(t, []byte{0, 1}, EEOF, defs.T_i32)
	run_skipping_emu(t, []byte{0, 1, 2}, EEOF, defs.T_i32)
	run_skipping_emu(t, []byte{}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0, 1}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0, 1, 2}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0, 1, 2, 3}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4, 5}, EEOF, defs.T_i64)
	run_skipping_emu(t, []byte{0, 1, 2, 3, 4, 5, 6}, EEOF, defs.T_i64)
}

func TestSkippingEmu_SkipStringsAndBinaries(t *testing.T) {
	run_skipping_emu(t, []byte{0, 0, 0, 0}, 4, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello, world"), 16, defs.T_string)
}

func TestSkippingEmu_SkipStringsAndBinariesButGotEOF(t *testing.T) {
	run_skipping_emu(t, []byte{}, EEOF, defs.T_string)
	run_skipping_emu(t, []byte{0}, EEOF, defs.T_string)
	run_skipping_emu(t, []byte{0, 0}, EEOF, defs.T_string)
	run_skipping_emu(t, []byte{0, 0, 0}, EEOF, defs.T_string)
	run_skipping_emu(t, []byte{0, 0, 0, 12}, EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0ch"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0che"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chel"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chell"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello,"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello, "), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello, w"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello, wo"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello, wor"), EEOF, defs.T_string)
	run_skipping_emu(t, []byte("\x00\x00\x00\x0chello, worl"), EEOF, defs.T_string)
}

func TestSkippingEmu_SkipStruct(t *testing.T) {
	run_skipping_emu(t, []byte{0}, 1, defs.T_struct)
	run_skipping_emu(t, []byte{2, 0, 0, 1, 0}, 5, defs.T_struct)
	run_skipping_emu(t, []byte{3, 0, 0, 1, 0}, 5, defs.T_struct)
	run_skipping_emu(t, []byte{3, 0, 0, 1, 6, 0, 1, 1, 2, 0}, 10, defs.T_struct)
	run_skipping_emu(t, []byte{4, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 0}, 12, defs.T_struct)
	run_skipping_emu(t, []byte{6, 0, 0, 1, 2, 0}, 6, defs.T_struct)
	run_skipping_emu(t, []byte{8, 0, 0, 1, 2, 3, 4, 0}, 8, defs.T_struct)
	run_skipping_emu(t, []byte{10, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 0}, 12, defs.T_struct)
	run_skipping_emu(t, []byte("\x0b\x00\x00\x00\x00\x00\x0chello, world\x00"), 20, defs.T_struct)
	run_skipping_emu(t, []byte{12, 0, 0, 0, 0}, 5, defs.T_struct)
	run_skipping_emu(t, []byte{12, 0, 0, 3, 0, 0, 1, 0, 0}, 9, defs.T_struct)
	run_skipping_emu(t, []byte{13, 0, 0, 3, 6, 0, 0, 0, 1, 1, 2, 3, 0}, 13, defs.T_struct)
	run_skipping_emu(t, []byte{14, 0, 0, 8, 0, 0, 0, 1, 1, 2, 3, 4, 0}, 13, defs.T_struct)
	run_skipping_emu(t, []byte{15, 0, 0, 6, 0, 0, 0, 2, 1, 2, 3, 4, 0}, 13, defs.T_struct)
}

func TestSkippingEmu_SkipStructButGotErrors(t *testing.T) {
	run_skipping_emu(t, []byte{}, EEOF, defs.T_struct)
	run_skipping_emu(t, []byte{2}, EEOF, defs.T_struct)
	run_skipping_emu(t, []byte{2, 0, 0}, EEOF, defs.T_struct)
	run_skipping_emu(t, []byte{9}, ETAG, defs.T_struct)
	run_skipping_emu(t, []byte{11}, EEOF, defs.T_struct)
	run_skipping_emu(t, []byte{11, 0, 0}, EEOF, defs.T_struct)
}

func TestSkippingEmu_SkipMapOfPrimitives(t *testing.T) {
	run_skipping_emu(t, []byte{3, 6, 0, 0, 0, 0}, 6, defs.T_map)
	run_skipping_emu(t, []byte{3, 6, 0, 0, 0, 3, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 15, defs.T_map)
	run_skipping_emu(t, []byte{6, 8, 0, 0, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4}, 18, defs.T_map)
}

func TestSkippingEmu_SkipMapWithNonPrimitiveKeys(t *testing.T) {
	run_skipping_emu(t, []byte{11, 8, 0, 0, 0, 0}, 6, defs.T_map)
	run_skipping_emu(t, []byte("\x0b\x08\x00\x00\x00\x01\x00\x00\x00\x04test\x01\x02\x03\x04"), 18, defs.T_map)
}

func TestSkippingEmu_SkipMapButGotErrors(t *testing.T) {
	run_skipping_emu(t, []byte{}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3, 6}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3, 6, 0}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3, 6, 0, 0}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3, 6, 0, 0, 0}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3, 6, 0, 0, 0, 1}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{3, 6, 0, 0, 0, 2, 1}, EEOF, defs.T_map)
	run_skipping_emu(t, []byte{9, 6, 0, 0, 0, 0}, ETAG, defs.T_map)
	run_skipping_emu(t, []byte{3, 9, 0, 0, 0, 3, 1, 2, 3, 4, 5, 6, 7, 8, 9}, ETAG, defs.T_map)
	run_skipping_emu(t, []byte{9, 9, 0, 0, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4}, ETAG, defs.T_map)
	run_skipping_emu(t, []byte("\x0b\x0b\x00\x00\x00\x01"), EEOF, defs.T_map)
}

func TestSkippingEmu_SkipSetOrListOfPrimitives(t *testing.T) {
	run_skipping_emu(t, []byte{2, 0, 0, 0, 16, 1, 2, 3, 4, 5, 6, 7, 8, 4, 3, 2, 1, 8, 7, 6, 5}, 21, defs.T_list)
	run_skipping_emu(t, []byte{3, 0, 0, 0, 16, 1, 2, 3, 4, 5, 6, 7, 8, 4, 3, 2, 1, 8, 7, 6, 5}, 21, defs.T_list)
	run_skipping_emu(t, []byte{4, 0, 0, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8, 4, 3, 2, 1, 8, 7, 6, 5}, 21, defs.T_list)
	run_skipping_emu(t, []byte{6, 0, 0, 0, 8, 1, 2, 3, 4, 5, 6, 7, 8, 4, 3, 2, 1, 8, 7, 6, 5}, 21, defs.T_list)
	run_skipping_emu(t, []byte{8, 0, 0, 0, 4, 1, 2, 3, 4, 5, 6, 7, 8, 4, 3, 2, 1, 8, 7, 6, 5}, 21, defs.T_list)
	run_skipping_emu(t, []byte{10, 0, 0, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8, 4, 3, 2, 1, 8, 7, 6, 5}, 21, defs.T_list)
}

func TestSkippingEmu_SkipSetOrListOfBinariesOrStrings(t *testing.T) {
	run_skipping_emu(t, []byte("\x0b\x00\x00\x00\x00"), 5, defs.T_list)
	run_skipping_emu(t, []byte("\x0b\x00\x00\x00\x01\x00\x00\x00\x00"), 9, defs.T_list)
	run_skipping_emu(t, []byte("\x0b\x00\x00\x00\x01\x00\x00\x00\x0chello, world"), 21, defs.T_list)
}

func TestSkippingEmu_SkipSetOrListButGotErrors(t *testing.T) {
	run_skipping_emu(t, []byte{}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{2}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{2, 0}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{2, 0, 0}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{2, 0, 0, 0}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{2, 0, 0, 0, 1}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{2, 0, 0, 0, 2, 1}, EEOF, defs.T_list)
	run_skipping_emu(t, []byte{9, 0, 0, 0, 1, 2}, ETAG, defs.T_list)
	run_skipping_emu(t, []byte("\x0b\x00\x00\x00\x01"), EEOF, defs.T_list)
}

func TestSkipListMap(t *testing.T) {
	listMap, err := hex.DecodeString("0f00010d000000020b0b00000001000000016100000001620b0b000000010000000161000000016200")
	if err != nil {
		t.Fatal(err)
	}
	var fsm _skipbuf_t
	var in = (*rt.GoSlice)(unsafe.Pointer(&listMap))
	rv := do_skip(&fsm, in.Ptr, in.Len, defs.T_struct)
	if rv != len(listMap) {
		t.Fatalf("skip failed: %d", rv)
	}
}
