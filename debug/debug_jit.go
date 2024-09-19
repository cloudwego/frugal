//go:build !go1.24 && amd64 && !windows

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

package debug

import (
	"github.com/cloudwego/frugal/internal/jit/decoder"
	"github.com/cloudwego/frugal/internal/jit/encoder"
	"github.com/cloudwego/frugal/internal/jit/loader"
)

// GetStats returns statistics of the JIT compiler.
func GetStats() Stats {
	return Stats{
		Memory: MemStats{
			Count: int(loader.FnCount),
			Alloc: int(loader.LoadSize),
		},
		Encoder: CacheStats{
			Miss: int(encoder.MissCount),
			Size: int(encoder.TypeCount),
		},
		Decoder: CacheStats{
			Miss: int(decoder.MissCount),
			Size: int(decoder.TypeCount),
		},
	}
}
