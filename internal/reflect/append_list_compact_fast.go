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

import "unsafe"

// Compact list fast paths.

func init() {
	// List fast-path registration for Compact.
	// The Binary list registrations are in append_list_fast.go.
	registerListAppendFuncCompact(tBYTE, appendListCompact_I08)
	registerListAppendFuncCompact(tI16, appendListCompact_I16)
	registerListAppendFuncCompact(tI32, appendListCompact_I32)
	registerListAppendFuncCompact(tI64, appendListCompact_I64)
	registerListAppendFuncCompact(tDOUBLE, appendListCompact_DOUBLE)
	registerListAppendFuncCompact(tENUM, appendListCompact_ENUM)
	registerListAppendFuncCompact(tSTRING, appendListCompact_STRING)
	registerListAppendFuncCompact(tSTRUCT, appendListCompact_Other)
	registerListAppendFuncCompact(tMAP, appendListCompact_Other)
	registerListAppendFuncCompact(tSET, appendListCompact_Other)
	registerListAppendFuncCompact(tLIST, appendListCompact_Other)
}

func appendListCompact_I08(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b = append(b, *((*byte)(vp)))
	}
	return b, nil
}

func appendListCompact_I16(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b = appendZigzag32(b, int32(*(*int16)(vp)))
	}
	return b, nil
}

func appendListCompact_I32(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b = appendZigzag32(b, *(*int32)(vp))
	}
	return b, nil
}

func appendListCompact_I64(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b = appendZigzag64(b, *(*int64)(vp))
	}
	return b, nil
}

func appendListCompact_DOUBLE(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b = appendUint64LE(b, *(*uint64)(vp))
	}
	return b, nil
}

func appendListCompact_ENUM(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		b = appendZigzag32(b, int32(*(*int64)(vp)))
	}
	return b, nil
}

func appendListCompact_STRING(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	var s string
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		s = *((*string)(vp))
		b = appendVarint(b, uint64(len(s)))
		b = append(b, s...)
	}
	return b, nil
}

func appendListCompact_Other(t *tType, b []byte, p unsafe.Pointer) ([]byte, error) {
	t = t.V
	b, n, vp := appendCompactListHeader(t, b, p)
	if n == 0 {
		return b, nil
	}
	var err error
	for i := uint32(0); i < n; i++ {
		if i != 0 {
			vp = unsafe.Add(vp, t.Size)
		}
		if t.IsPointer {
			b, err = t.AppendFuncCompact(t, b, *(*unsafe.Pointer)(vp))
		} else {
			b, err = t.AppendFuncCompact(t, b, vp)
		}
		if err != nil {
			return b, err
		}
	}
	return b, nil
}
