/*
 * Copyright 2021 ByteDance Inc.
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

package iovec

// IoVec contains a list of buffers. It enables zero-copy serialization.
type IoVec interface {
    // Put writes v into the underlying IO vector. v must be allocated
    // from the Add method below.
    Put(v []byte)

    // Cat writes v and w into the underlying IO vector one after the other.
    // v must be allocated from the Add method below.
    Cat(v []byte, w []byte)

    // Add writes v into the underlying IO vector and allocates a new buffer
    // with capacity of at least n bytes from the underlying implementation.
    //
    // v must either be nil or a buffer allocated by previous call to the Add
    // method.
    Add(n int, v []byte) []byte
}
