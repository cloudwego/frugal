/*
 * Copyright 2023 ByteDance Inc.
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

package utils

import (
    `sync/atomic`
    `unsafe`
)

/* ListNode can be used to save references */
type ListNode struct{
    value unsafe.Pointer
    next  *ListNode
}

/* Prepend creates a new node with value=p and adds it at the beginning of this list */
func (n *ListNode) Prepend(p unsafe.Pointer) {
    for {
        oldNext := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&n.next)))
        newNode := &ListNode{
            value: p,
            next: (*ListNode)(oldNext),
        }
        success := atomic.CompareAndSwapPointer(
            (*unsafe.Pointer)(unsafe.Pointer(&n.next)),
            oldNext,
            unsafe.Pointer(newNode),
        )
        if success {
            break
        }
    }
}
