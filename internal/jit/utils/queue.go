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

package utils

import (
	"container/list"
	"sync"
)

// Queue ...
type Queue struct {
	sync.Mutex

	l *list.List
}

// NewQueue ...
func NewQueue() *Queue {
	return &Queue{l: list.New()}
}

// Enqueue adds an item at the back of the queue
func (q *Queue) Enqueue(item interface{}) {
	q.Lock()
	defer q.Unlock()
	_ = q.l.PushBack(item)
}

// Dequeue removes and returns the front queue item
func (q *Queue) Dequeue() interface{} {
	q.Lock()
	defer q.Unlock()
	return q.l.Remove(q.l.Front())
}

// Empty checks if the queue is empty
func (q *Queue) Empty() bool {
	q.Lock()
	defer q.Unlock()
	return q.l.Len() <= 0
}
