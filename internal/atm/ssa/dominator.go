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

/** This is an implementation of the Lengauer-Tarjan algorithm described in
 *  https://doi.org/10.1145%2F357062.357071
 */

package ssa

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
    return &_LengauerTarjan {
        vertex: make(map[int]int),
    }
}

func (self *_LengauerTarjan) dfs(bb *BasicBlock) {
    i := len(self.nodes)
    self.vertex[bb.Id] = i

    /* create a new node */
    p := &_LtNode {
        semi   : i,
        node   : bb,
        bucket : make(map[*_LtNode]struct{}),
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

func (self *_LengauerTarjan) compress(p *_LtNode) {
    if p.ancestor.ancestor != nil {
        self.compress(p.ancestor)
        if p.label.semi > p.ancestor.label.semi { p.label = p.ancestor.label }
        p.ancestor = p.ancestor.ancestor
    }
}

type DominatorTree struct {
    Root        *BasicBlock
    DominatedBy map[int]*BasicBlock
    DominatorOf map[int][]*BasicBlock
}

func minInt(a int, b int) int {
    if a < b {
        return a
    } else {
        return b
    }
}

func BuildDominatorTree(bb *BasicBlock) DominatorTree {
    domby := make(map[int]*BasicBlock)
    domof := make(map[int][]*BasicBlock)

    /* Step 1: Carry out a depth-first search of the problem graph. Number the vertices
     * from 1 to n as they are reached during the search. Initialize the variables used
     * in succeeding steps. */
    lt := newLengauerTarjan()
    lt.dfs(bb)

    /* perform Step 2 and Step 3 simultaneously */
    for i := len(lt.nodes) - 1; i > 0; i-- {
        p := lt.nodes[i]
        q := (*_LtNode)(nil)

        /* Step 2: Compute the semidominators of all vertices by applying Theorem 4.
         * Carry out the computation vertex by vertex in decreasing order by number. */
        for _, v := range p.pred {
            q = lt.eval(v)
            p.semi = minInt(p.semi, q.semi)
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

    /* map the dominator relations */
    for _, p := range lt.nodes[1:] {
        domby[p.node.Id] = p.dom.node
        domof[p.dom.node.Id] = append(domof[p.dom.node.Id], p.node)
    }

    /* construct the dominator tree */
    return DominatorTree {
        Root        : bb,
        DominatorOf : domof,
        DominatedBy : domby,
    }
}
