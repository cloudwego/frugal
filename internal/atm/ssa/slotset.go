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
	"fmt"
	"sort"
	"strings"
)

type (
	SlotSet map[IrSpillSlot]struct{}
)

func (self SlotSet) add(r IrSpillSlot) bool {
	if _, ok := self[r]; ok {
		return false
	} else {
		self[r] = struct{}{}
		return true
	}
}

func (self SlotSet) clone() (rs SlotSet) {
	rs = make(SlotSet, len(self))
	for r := range self {
		rs.add(r)
	}
	return
}

func (self SlotSet) remove(r IrSpillSlot) bool {
	if _, ok := self[r]; !ok {
		return false
	} else {
		delete(self, r)
		return true
	}
}

func (self SlotSet) String() string {
	nb := len(self)
	rs := make([]string, 0, nb)
	rr := make([]IrSpillSlot, 0, nb)

	/* extract all slot */
	for r := range self {
		rr = append(rr, r)
	}

	/* sort by slot ID */
	sort.Slice(rr, func(i int, j int) bool {
		return rr[i] < rr[j]
	})

	/* convert every slot */
	for _, r := range rr {
		rs = append(rs, r.String())
	}

	/* join them together */
	return fmt.Sprintf(
		"{%s}",
		strings.Join(rs, ", "),
	)
}
