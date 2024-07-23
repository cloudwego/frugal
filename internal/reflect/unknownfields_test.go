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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnknownFields(t *testing.T) {
	b := []byte("0123456789")

	p := unknownFieldsPool.Get().(*unknownFields)
	defer unknownFieldsPool.Put(p)
	p.Reset()

	assert.Equal(t, 0, p.Size())
	assert.Equal(t, []byte{}, p.Copy(b))

	p.Add(1, 1)
	assert.Equal(t, 1, p.Size())
	assert.Equal(t, []byte{'1'}, p.Copy(b))

	p.Add(2, 2)
	assert.Equal(t, 3, p.Size())
	assert.Equal(t, []byte{'1', '2', '3'}, p.Copy(b))

	p.Add(8, 2)
	assert.Equal(t, 5, p.Size())
	assert.Equal(t, []byte{'1', '2', '3', '8', '9'}, p.Copy(b))

}
