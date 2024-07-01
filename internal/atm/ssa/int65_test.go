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
	"testing"
)

func TestInt65_All(t *testing.T) {
	println("       0 - 1 =", Int65i(0).OneLess().String())
	println("       0     =", Int65i(0).String())
	println("       0 + 1 =", Int65i(0).OneMore().String())
	println("-------------------------------------------")
	println("       1 - 1 =", Int65i(1).OneLess().String())
	println("       1     =", Int65i(1).String())
	println("       1 + 1 =", Int65i(1).OneMore().String())
	println("-------------------------------------------")
	println("      -1 - 1 =", Int65i(-1).OneLess().String())
	println("      -1     =", Int65i(-1).String())
	println("      -1 + 1 =", Int65i(-1).OneMore().String())
	println("-------------------------------------------")
	println("MaxInt65 - 1 =", MaxInt65.OneLess().String())
	println("MaxInt65     =", MaxInt65.String())
	println("MaxInt65 + 1 =", MaxInt65.OneMore().String())
	println("-------------------------------------------")
	println("MinInt65 - 1 =", MinInt65.OneLess().String())
	println("MinInt65     =", MinInt65.String())
	println("MinInt65 + 1 =", MinInt65.OneMore().String())
}
