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

package ssa

import (
	"math"
	"math/bits"
	"strconv"
)

var (
	MinInt65 = Int65{0, 1}
	MaxInt65 = Int65{math.MaxUint64, 0}
)

const (
	_MinInt65Str = "-18446744073709551616"
)

type Int65 struct {
	u uint64
	s uint64
}

func Int65i(v int64) Int65 {
	return Int65{
		u: uint64(v),
		s: uint64(v) >> 63,
	}
}

func (self Int65) String() string {
	if self.s == 0 {
		return strconv.FormatUint(self.u, 10)
	} else if self.u != 0 {
		return "-" + strconv.FormatUint(-self.u, 10)
	} else {
		return _MinInt65Str
	}
}

func (self Int65) OneLess() (r Int65) {
	r.u, r.s = bits.Sub64(self.u, 1, 0)
	r.s = (self.s - r.s) & 1
	return
}

func (self Int65) OneMore() (r Int65) {
	r.u, r.s = bits.Add64(self.u, 1, 0)
	r.s = (self.s + r.s) & 1
	return
}

func (self Int65) Compare(other Int65) int {
	if self.s == 0 && other.s != 0 {
		return 1
	} else if self.s != 0 && other.s == 0 {
		return -1
	} else {
		return cmpu64(self.u, other.u)
	}
}

func (self Int65) CompareZero() int {
	if self.s != 0 {
		return -1
	} else if self.u != 0 {
		return 1
	} else {
		return 0
	}
}
