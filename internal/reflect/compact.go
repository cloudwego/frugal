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

// Compact Protocol low-level encode/decode helpers.

// appendVarint encodes v as an unsigned LEB128 varint into b.
func appendVarint(b []byte, v uint64) []byte {
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			c |= 0x80
		}
		b = append(b, c)
		if v == 0 {
			break
		}
	}
	return b
}

// decodeVarint decodes an unsigned LEB128 varint from b.
// Returns the value and number of bytes consumed.
func decodeVarint(b []byte) (uint64, int) {
	var v uint64
	var s uint
	for i, c := range b {
		v |= uint64(c&0x7f) << s
		s += 7
		if c&0x80 == 0 {
			return v, i + 1
		}
	}
	// Should not reach here for valid data; return what we have.
	return 0, 0
}

// varintLen returns the number of bytes needed to encode v as a varint.
func varintLen(v uint64) int {
	n := 1
	for v >>= 7; v != 0; v >>= 7 {
		n++
	}
	return n
}

func zigzag32(n int32) uint32 {
	return (uint32(n) << 1) ^ uint32(n>>31)
}

func unzigzag32(n uint32) int32 {
	return int32((n >> 1) ^ -(n & 1))
}

func zigzag64(n int64) uint64 {
	return (uint64(n) << 1) ^ uint64(n>>63)
}

func unzigzag64(n uint64) int64 {
	return int64((n >> 1) ^ -(n & 1))
}

// appendZigzag32 encodes n as zigzag + varint.
func appendZigzag32(b []byte, n int32) []byte {
	return appendVarint(b, uint64(zigzag32(n)))
}

// appendZigzag64 encodes n as zigzag + varint.
func appendZigzag64(b []byte, n int64) []byte {
	return appendVarint(b, zigzag64(n))
}

// decodeZigzag32 decodes a zigzag+varint encoded int32 from b.
// Returns the value and number of bytes consumed.
func decodeZigzag32(b []byte) (int32, int) {
	v, n := decodeVarint(b)
	return unzigzag32(uint32(v)), n
}

// decodeZigzag64 decodes a zigzag+varint encoded int64 from b.
// Returns the value and number of bytes consumed.
func decodeZigzag64(b []byte) (int64, int) {
	v, n := decodeVarint(b)
	return unzigzag64(v), n
}

// zigzag32Size returns the varint length of a zigzag-encoded int32.
func zigzag32Size(n int32) int {
	return varintLen(uint64(zigzag32(n)))
}

// zigzag64Size returns the varint length of a zigzag-encoded int64.
func zigzag64Size(n int64) int {
	return varintLen(zigzag64(n))
}

// appendUint64LE appends v as 8 bytes in little-endian order (Apache Thrift Compact format).
func appendUint64LE(b []byte, v uint64) []byte {
	return append(b,
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
		byte(v>>32),
		byte(v>>40),
		byte(v>>48),
		byte(v>>56),
	)
}

// readUint64LE reads 8 bytes in little-endian order.
func readUint64LE(b []byte) uint64 {
	return uint64(b[0]) |
		uint64(b[1])<<8 |
		uint64(b[2])<<16 |
		uint64(b[3])<<24 |
		uint64(b[4])<<32 |
		uint64(b[5])<<40 |
		uint64(b[6])<<48 |
		uint64(b[7])<<56
}

// Compact field header encoding, matching Apache Thrift TCompactProtocol.
//
// Byte layout: [dddd|tttt]
//   dddd = delta nibble (high 4 bits) - delta from last field ID
//   tttt = type nibble (low 4 bits)  - Compact wire type
//
// If low nibble == 0 (STOP): struct end marker.
// If high nibble == 0: not a delta - full 2-byte big-endian field ID follows.
// If high nibble != 0: field ID = lastFieldId + high_nibble.

// writeCompactFieldHeader writes a Compact field header to b.
func writeCompactFieldHeader(b []byte, lastId, id uint16, wt ttype) []byte {
	if id > lastId {
		delta := id - lastId
		if delta <= 15 {
			return append(b, byte(delta<<4)|byte(wt))
		}
	}
	b = append(b, byte(wt))
	return appendUint16(b, id)
}

// readCompactFieldHeader reads a Compact field header starting at b[i].
// Returns the wire type, field ID, and number of bytes consumed.
func readCompactFieldHeader(b []byte, i int, lastId uint16) (wt ttype, id uint16, n int) {
	if i >= len(b) {
		return ctSTOP, 0, 0
	}
	h := b[i]
	n = 1
	wt = ttype(h & 0x0F)
	if wt == ctSTOP {
		return ctSTOP, 0, n
	}
	delta := uint16(h >> 4)
	if delta == 0 {
		if i+2 >= len(b) {
			return ctSTOP, 0, 0
		}
		id = uint16(b[i+1])<<8 | uint16(b[i+2])
		n += 2
	} else {
		id = lastId + delta
	}
	return wt, id, n
}

// compactFieldHeaderSize returns the number of bytes needed for a Compact field header.
func compactFieldHeaderSize(lastId, id uint16) int {
	if id > lastId && id-lastId <= 15 {
		return 1
	}
	return 3 // type byte + 2-byte ID
}
