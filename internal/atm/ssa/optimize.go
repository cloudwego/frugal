/*
 * Copyright 2022 ByteDance Inc.
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

type Pass interface {
    Apply(*CFG)
}

type _PassDescriptor struct {
    pass Pass
    desc string
}

var _passes = [...]_PassDescriptor {
    { desc: "Constant Propagation"              , pass: new(ConstProp) },
    { desc: "Common Sub-expression Elimination" , pass: new(CSE) },
    { desc: "Copy Elimination"                  , pass: new(CopyElim) },
    { desc: "Trivial Dead Code Elimination"     , pass: new(TDCE) },
}

func optimizeSSAGraph(cfg *CFG) {
    for _, p := range _passes {
        p.pass.Apply(cfg)
    }
}
