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

/** This is an implementation of the Lengauer-Tarjan algorithm described in
 *  https://doi.org/10.1145%2F357062.357071
 */

package ssa

import (
	"sort"

	"github.com/cloudwego/frugal/internal/rt"
	"github.com/oleiade/lane"
)

type _LtNode struct {
	semi     int
	node     *BasicBlock
	dom      *_LtNode
	label    *_LtNode
	parent   *_LtNode
	ancestor *_LtNode
	pred     []*_LtNode
	bucket   map[*_LtNode]struct{}
}

type _LengauerTarjan struct {
	nodes  []*_LtNode
	vertex map[int]int
}

func newLengauerTarjan() *_LengauerTarjan {
	return &_LengauerTarjan{
		vertex: make(map[int]int),
	}
}

func (self *_LengauerTarjan) dfs(bb *BasicBlock) {
	i := len(self.nodes)
	self.vertex[bb.Id] = i

	/* create a new node */
	p := &_LtNode{
		semi:   i,
		node:   bb,
		bucket: make(map[*_LtNode]struct{}),
	}

	/* add to node list */
	p.label = p
	self.nodes = append(self.nodes, p)

	/* get it's successors iterator */
	tr := bb.Term
	it := tr.Successors()

	/* traverse the successors */
	for it.Next() {
		w := it.Block()
		idx, ok := self.vertex[w.Id]

		/* not visited yet */
		if !ok {
			self.dfs(w)
			idx = self.vertex[w.Id]
			self.nodes[idx].parent = p
		}

		/* add predecessors */
		q := self.nodes[idx]
		q.pred = append(q.pred, p)
	}
}

func (self *_LengauerTarjan) eval(p *_LtNode) *_LtNode {
	if p.ancestor == nil {
		return p
	} else {
		self.compress(p)
		return p.label
	}
}

func (self *_LengauerTarjan) link(p *_LtNode, q *_LtNode) {
	q.ancestor = p
}

func (self *_LengauerTarjan) relable(p *_LtNode) {
	if p.label.semi > p.ancestor.label.semi {
		p.label = p.ancestor.label
	}
}

func (self *_LengauerTarjan) compress(p *_LtNode) {
	if p.ancestor.ancestor != nil {
		self.compress(p.ancestor)
		self.relable(p)
		p.ancestor = p.ancestor.ancestor
	}
}

type _NodeDepth struct {
	d  int
	bb int
}

func updateDominatorTree(cfg *CFG) {
	rt.MapClear(cfg.DominatedBy)
	rt.MapClear(cfg.DominatorOf)

	/* Step 1: Carry out a depth-first search of the problem graph. Number the vertices
	 * from 1 to n as they are reached during the search. Initialize the variables used
	 * in succeeding steps. */
	lt := newLengauerTarjan()
	lt.dfs(cfg.Root)

	/* perform Step 2 and Step 3 for every node */
	for i := len(lt.nodes) - 1; i > 0; i-- {
		p := lt.nodes[i]
		q := (*_LtNode)(nil)

		/* Step 2: Compute the semidominators of all vertices by applying Theorem 4.
		 * Carry out the computation vertex by vertex in decreasing order by number. */
		for _, v := range p.pred {
			q = lt.eval(v)
			p.semi = minint(p.semi, q.semi)
		}

		/* link the ancestor */
		lt.link(p.parent, p)
		lt.nodes[p.semi].bucket[p] = struct{}{}

		/* Step 3: Implicitly define the immediate dominator of each vertex by applying Corollary 1 */
		for v := range p.parent.bucket {
			if q = lt.eval(v); q.semi < v.semi {
				v.dom = q
			} else {
				v.dom = p.parent
			}
		}

		/* clear the bucket */
		for v := range p.parent.bucket {
			delete(p.parent.bucket, v)
		}
	}

	/* Step 4: Explicitly define the immediate dominator of each vertex, carrying out the
	 * computation vertex by vertex in increasing order by number. */
	for _, p := range lt.nodes[1:] {
		if p.dom.node.Id != lt.nodes[p.semi].node.Id {
			p.dom = p.dom.dom
		}
	}

	/* map the dominator relationship */
	for _, p := range lt.nodes[1:] {
		cfg.DominatedBy[p.node.Id] = p.dom.node
		cfg.DominatorOf[p.dom.node.Id] = append(cfg.DominatorOf[p.dom.node.Id], p.node)
	}

	/* sort the dominators */
	for _, p := range cfg.DominatorOf {
		sort.Slice(p, func(i int, j int) bool {
			return p[i].Id < p[j].Id
		})
	}
}

func updateDominatorDepth(cfg *CFG) {
	r := cfg.Root.Id
	q := lane.NewQueue()

	/* add the root node */
	q.Enqueue(_NodeDepth{bb: r})
	rt.MapClear(cfg.Depth)

	/* calculate depth for every block */
	for !q.Empty() {
		d := q.Dequeue().(_NodeDepth)
		cfg.Depth[d.bb] = d.d

		/* add all the dominated nodes */
		for _, p := range cfg.DominatorOf[d.bb] {
			q.Enqueue(_NodeDepth{
				d:  d.d + 1,
				bb: p.Id,
			})
		}
	}
}

func updateDominatorFrontier(cfg *CFG) {
	r := cfg.Root
	q := lane.NewQueue()

	/* add the root node */
	q.Enqueue(r)
	rt.MapClear(cfg.DominanceFrontier)

	/* calculate dominance frontier for every block */
	for !q.Empty() {
		k := q.Dequeue().(*BasicBlock)
		addImmediateDominated(cfg.DominatorOf, k, q)
		computeDominanceFrontier(cfg.DominatorOf, k, cfg.DominanceFrontier)
	}
}

func isStrictlyDominates(dom map[int][]*BasicBlock, p *BasicBlock, q *BasicBlock) bool {
	for _, v := range dom[p.Id] {
		if v != p && (v == q || isStrictlyDominates(dom, v, q)) {
			return true
		}
	}
	return false
}

func addImmediateDominated(dom map[int][]*BasicBlock, node *BasicBlock, q *lane.Queue) {
	for _, p := range dom[node.Id] {
		q.Enqueue(p)
	}
}

func computeDominanceFrontier(dom map[int][]*BasicBlock, node *BasicBlock, dfm map[int][]*BasicBlock) []*BasicBlock {
	var it IrSuccessors
	var df map[*BasicBlock]struct{}

	/* check for cached values */
	if v, ok := dfm[node.Id]; ok {
		return v
	}

	/* get the successor iterator */
	it = node.Term.Successors()
	df = make(map[*BasicBlock]struct{})

	/* local(X) = set of successors of X that X does not immediately dominate */
	for it.Next() {
		if y := it.Block(); !isStrictlyDominates(dom, node, y) {
			df[y] = struct{}{}
		}
	}

	/* df(X) = union of local(X) and ( union of up(K) for all K that are children of X ) */
	for _, k := range dom[node.Id] {
		for _, y := range computeDominanceFrontier(dom, k, dfm) {
			if !isStrictlyDominates(dom, node, y) {
				df[y] = struct{}{}
			}
		}
	}

	/* convert to slice */
	nb := len(df)
	ret := make([]*BasicBlock, 0, nb)

	/* extract all the keys */
	for bb := range df {
		ret = append(ret, bb)
	}

	/* sort by ID */
	sort.Slice(ret, func(i int, j int) bool {
		return ret[i].Id < ret[j].Id
	})

	/* add to cache */
	dfm[node.Id] = ret
	return ret
}
