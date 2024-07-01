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
	"github.com/cloudwego/frugal/internal/rt"
)

type _RegSrc struct {
	r Reg
	p IrNode
}

// PhiElim transforms Phi nodes into copies if possible.
type PhiElim struct{}

func (self PhiElim) addSource(src map[_RegSrc]struct{}, phi *IrPhi, defs map[Reg]IrNode, visited map[*IrPhi]struct{}) {
	var ok bool
	var pp *IrPhi
	var def IrNode

	/* circles back to itself */
	if _, ok = visited[phi]; ok {
		return
	} else {
		visited[phi] = struct{}{}
	}

	/* add definitions for this node */
	for _, r := range phi.V {
		if r.Kind() == K_zero {
			src[_RegSrc{r: *r, p: nil}] = struct{}{}
		} else if def, ok = defs[*r]; !ok {
			panic("phixform: undefined register: " + r.String())
		} else if pp, ok = def.(*IrPhi); ok {
			self.addSource(src, pp, defs, visited)
		} else {
			src[_RegSrc{r: *r, p: def}] = struct{}{}
		}
	}
}

func (self PhiElim) Apply(cfg *CFG) {
	buf := make(map[Reg]IrNode)
	vis := make(map[*IrPhi]struct{})
	src := make(map[_RegSrc]struct{})

	/* scan for all instruction usages and register definitions */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var ok bool
		var def IrDefinitions

		/* mark all Phi definitions */
		for _, v := range bb.Phi {
			buf[v.R] = v
		}

		/* scan instructions */
		for _, v := range bb.Ins {
			if def, ok = v.(IrDefinitions); ok {
				for _, r := range def.Definitions() {
					buf[*r] = v
				}
			}
		}
	})

	/* scan for unused Phi nodes */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var p *IrPhi
		var ins []IrNode

		/* filter the Phi nodes */
		phi := bb.Phi
		bb.Phi = bb.Phi[:0]

		/* check all Phi nodes */
		for _, p = range phi {
			rt.MapClear(src)
			rt.MapClear(vis)

			/* resolve all the value sources */
			if self.addSource(src, p, buf, vis); len(src) != 1 {
				bb.Phi = append(bb.Phi, p)
				continue
			}

			/* all values come from a single source */
			for s := range src {
				ins = append(ins, IrCopy(p.R, s.r))
				break
			}
		}

		/* patch instructions if needed */
		if len(ins) != 0 {
			bb.Ins = append(ins, bb.Ins...)
		}
	})
}
