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

package opts

type Options struct {
	MaxInlineDepth   int
	MaxInlineILSize  int
	MaxPretouchDepth int
}

func (self *Options) CanInline(sp int, pc int) bool {
	return (self.MaxInlineDepth > sp || self.MaxInlineDepth == 0) && (self.MaxInlineILSize > pc || self.MaxInlineILSize == 0)
}

func (self *Options) CanPretouch(d int) bool {
	return self.MaxPretouchDepth > d || self.MaxPretouchDepth == 0
}

func GetDefaultOptions() Options {
	return Options{
		MaxInlineDepth:   MaxInlineDepth,
		MaxInlineILSize:  MaxInlineILSize,
		MaxPretouchDepth: 0,
	}
}
