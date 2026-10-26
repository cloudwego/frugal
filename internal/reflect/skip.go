/*
 * Copyright 2026 CloudWeGo Authors
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
	"encoding/binary"
	"io"
)

const maxSkipDepth = 64

// skipScalar skips a fixed-size or string value at b[i:] and returns the new offset.
// It returns -1 for containers or invalid data so the caller can fall back to skipValue.
// Keep this function small so it can be inlined in loops.
// Use b[i:j:j] instead of slicing b[i:] at each step to avoid per-element slice bounds overhead.
func skipScalar(b []byte, i int, t ttype) int {
	if n := int(typeToSize[t]); n > 0 {
		if n <= len(b)-i {
			return i + n
		}
		return -1
	}
	if t == tSTRING && strHeaderLen <= len(b)-i {
		j := i + strHeaderLen
		if l := int(int32(binary.BigEndian.Uint32(b[i:j:j]))); l >= 0 && l <= len(b)-j {
			return j + l
		}
	}
	return -1
}

// skipValue returns the byte length of a value of wire type t.
func skipValue(b []byte, t ttype, maxdepth int) (int, error) {
	if minWireSize[t] == 0 {
		return 0, newUnknownTypeException(t)
	}
	if n := int(typeToSize[t]); n > 0 {
		if len(b) < n {
			return 0, io.ErrShortBuffer
		}
		return n, nil
	}
	if t == tSTRING {
		if len(b) < strHeaderLen {
			return 0, io.ErrShortBuffer
		}
		l := int(int32(binary.BigEndian.Uint32(b)))
		if l < 0 {
			return 0, errNegativeSize
		}
		if l > len(b)-strHeaderLen {
			return 0, newSizeExceedsBufferException(l, len(b)-strHeaderLen)
		}
		return strHeaderLen + l, nil
	}
	if maxdepth == 0 {
		return 0, errDepthLimitExceeded
	}
	switch t {
	case tSTRUCT:
		i := 0
		for {
			if uint(i) >= uint(len(b)) {
				return i, io.ErrShortBuffer
			}
			tp := ttype(b[i])
			i++
			if tp == tSTOP {
				return i, nil
			}
			if uint(len(b)-i) < fieldHeaderLen-1 {
				return i, io.ErrShortBuffer
			}
			i += fieldHeaderLen - 1
			end := skipScalar(b, i, tp)
			if end < 0 {
				n, err := skipValue(b[i:], tp, maxdepth-1)
				if err != nil {
					return i, err
				}
				end = i + n
			}
			i = end
		}
	case tMAP:
		if len(b) < mapHeaderLen {
			return 0, io.ErrShortBuffer
		}
		l := int(int32(binary.BigEndian.Uint32(b[2:])))
		if l < 0 {
			return 0, errNegativeSize
		}
		if l == 0 {
			// match thrift.Binary.Skip: ignore key/value types when map is empty
			return mapHeaderLen, nil
		}
		kt, vt := ttype(b[0]), ttype(b[1])
		if minWireSize[kt] == 0 {
			return 0, newUnknownTypeException(kt)
		}
		if minWireSize[vt] == 0 {
			return 0, newUnknownTypeException(vt)
		}
		minSize := int(minWireSize[kt]) + int(minWireSize[vt])
		// multiply instead of dividing to avoid slow integer division
		if uint64(l)*uint64(minSize) > uint64(len(b)-mapHeaderLen) {
			return 0, newSizeExceedsBufferException(l, len(b)-mapHeaderLen)
		}
		if typeToSize[kt] > 0 && typeToSize[vt] > 0 {
			return mapHeaderLen + l*minSize, nil
		}
		i := mapHeaderLen
		for j := 0; j < l; j++ {
			end := skipScalar(b, i, kt)
			if end < 0 {
				n, err := skipValue(b[i:], kt, maxdepth-1)
				if err != nil {
					return i, err
				}
				end = i + n
			}
			i = end
			end = skipScalar(b, i, vt)
			if end < 0 {
				n, err := skipValue(b[i:], vt, maxdepth-1)
				if err != nil {
					return i, err
				}
				end = i + n
			}
			i = end
		}
		return i, nil
	case tLIST, tSET:
		if len(b) < listHeaderLen {
			return 0, io.ErrShortBuffer
		}
		l := int(int32(binary.BigEndian.Uint32(b[1:])))
		if l < 0 {
			return 0, errNegativeSize
		}
		if l == 0 {
			return listHeaderLen, nil // ignore element type when list is empty
		}
		et := ttype(b[0])
		if minWireSize[et] == 0 {
			return 0, newUnknownTypeException(et)
		}
		minSize := int(minWireSize[et])
		if uint64(l)*uint64(minSize) > uint64(len(b)-listHeaderLen) {
			return 0, newSizeExceedsBufferException(l, len(b)-listHeaderLen)
		}
		if typeToSize[et] > 0 {
			return listHeaderLen + l*minSize, nil
		}
		i := listHeaderLen
		for j := 0; j < l; j++ {
			end := skipScalar(b, i, et)
			if end < 0 {
				n, err := skipValue(b[i:], et, maxdepth-1)
				if err != nil {
					return i, err
				}
				end = i + n
			}
			i = end
		}
		return i, nil
	}
	return 0, newUnknownTypeException(t)
}
