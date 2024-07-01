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

	"github.com/cloudwego/frugal/internal/rt"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

const (
	_W_likely   = 7.0 / 8.0 // must be exactly representable to avoid float precision loss
	_W_unlikely = 1.0 - _W_likely
)

var _WeightTab = [...]float64{
	Likely:   _W_likely,
	Unlikely: _W_unlikely,
}

// PhiProp propagates Phi nodes into it's source blocks,
// essentially get rid of them.
// The CFG is no longer in SSA form after this pass.
type PhiProp struct{}

func (self PhiProp) dfs(dag *simple.DirectedGraph, bb *BasicBlock, vis map[int]*BasicBlock, path map[int]struct{}) {
	vis[bb.Id] = bb
	path[bb.Id] = struct{}{}

	/* traverse all the successors */
	for it := bb.Term.Successors(); it.Next(); {
		v := it.Block()
		s, d := bb.Id, v.Id

		/* back edge */
		if _, ok := path[d]; ok {
			continue
		}

		/* forward or cross edge */
		p, _ := dag.NodeWithID(int64(s))
		q, _ := dag.NodeWithID(int64(d))
		dag.SetEdge(dag.NewEdge(p, q))

		/* visit the successor if not already */
		if _, ok := vis[d]; !ok {
			self.dfs(dag, v, vis, path)
		}
	}

	/* remove the node from path */
	if _, ok := path[bb.Id]; !ok {
		panic("phiprop: corrupted DFS stack")
	} else {
		delete(path, bb.Id)
	}
}

func (self PhiProp) Apply(cfg *CFG) {
	var err error
	var ord []graph.Node

	/* convert to DAG by removing back edges (assuming they never takes) */
	// FIXME: this might cause inaccuracy, loops don't affect path probabilities
	//  if they are looked as a whole.
	dag := simple.NewDirectedGraph()
	bbs := make(map[int]*BasicBlock, cfg.MaxBlock())
	self.dfs(dag, cfg.Root, bbs, make(map[int]struct{}, cfg.MaxBlock()))

	/* topologically sort the DAG */
	if ord, err = topo.Sort(dag); err != nil {
		panic("phiprop: topology sort: " + err.Error())
	}

	/* weight from block to another block */
	subs := make(map[Reg]Reg)
	bias := make(map[int]float64)
	weight := make(map[int]map[int]float64, cfg.MaxBlock())

	/* calculate block weights in topological order */
	for _, p := range ord {
		var in float64
		var sum float64

		/* find the basic block */
		id := p.ID()
		bb := bbs[int(id)]
		tr := bb.Term.Successors()

		/* special case for root node */
		if len(bb.Pred) == 0 {
			in = 1.0
		}

		/* add all the incoming weights */
		for _, v := range bb.Pred {
			if dag.HasEdgeFromTo(int64(v.Id), id) {
				in += weight[v.Id][bb.Id]
			}
		}

		/* allocate the output probability map */
		weight[bb.Id] = make(map[int]float64)
		rt.MapClear(bias)

		/* calculate the output bias factor */
		for tr.Next() {
			if vv := tr.Block(); dag.HasEdgeFromTo(id, int64(vv.Id)) {
				w := _WeightTab[tr.Likeliness()]
				sum += w
				bias[vv.Id] = w
			}
		}

		/* bias the weight with the bias factor */
		for i, v := range bias {
			weight[bb.Id][i] = in * (v / sum)
		}
	}

	/* choose the register with highest probability as the "primary" register */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		for _, p := range bb.Phi {
			var rs Reg
			var ps float64
			var pp float64

			/* find the branch with the highest probability */
			for b, r := range p.V {
				if pp = weight[b.Id][bb.Id]; ps < pp {
					rs = *r
					ps = pp
				}
			}

			/* mark the substitution */
			if _, ok := subs[p.R]; ok {
				panic(fmt.Sprintf("phiprop: duplicated substitution: %s -> %s", p.R, rs))
			} else {
				subs[p.R] = rs
			}
		}
	})

	/* register substitution routine */
	substitute := func(rr []*Reg) {
		for _, r := range rr {
			if d, ok := subs[*r]; ok {
				for subs[d] != 0 {
					d = subs[d]
				}
				*r = d
			}
		}
	}

	/* substitute every register */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		var ok bool
		var use IrUsages
		var def IrDefinitions

		/* process Phi nodes */
		for _, v := range bb.Phi {
			substitute(v.Usages())
			substitute(v.Definitions())
		}

		/* process instructions */
		for _, v := range bb.Ins {
			if use, ok = v.(IrUsages); ok {
				substitute(use.Usages())
			}
			if def, ok = v.(IrDefinitions); ok {
				substitute(def.Definitions())
			}
		}

		/* process the terminator */
		if use, ok = bb.Term.(IrUsages); ok {
			substitute(use.Usages())
		}
	})

	/* propagate Phi nodes upward */
	cfg.PostOrder().ForEach(func(bb *BasicBlock) {
		pp := bb.Phi
		bb.Phi = nil

		/* process every Phi node */
		for _, p := range pp {
			for b, r := range p.V {
				if *r != p.R {
					b.Ins = append(b.Ins, IrArchCopy(p.R, *r))
				}
			}
		}
	})
}
