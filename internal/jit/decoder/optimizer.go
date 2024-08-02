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

package decoder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/frugal/internal/jit/rt"
	"github.com/cloudwego/frugal/internal/jit/utils"
)

type BasicBlock struct {
	P    Program
	Src  int
	End  int
	Link []*BasicBlock
}

func (self *BasicBlock) Len() int {
	return self.End - self.Src
}

func (self *BasicBlock) Free() {
	q := utils.NewQueue()
	m := make(map[*BasicBlock]struct{})

	/* traverse the graph with BFS */
	for q.Enqueue(self); !q.Empty(); {
		v := q.Dequeue()
		p := v.(*BasicBlock)

		/* add branch to queue */
		for _, b := range p.Link {
			q.Enqueue(b)
		}

		/* clear branch, and add to free list */
		m[p] = struct{}{}
		p.Link = p.Link[:0]
	}

	/* reset and free all the nodes */
	for p := range m {
		freeBasicBlock(p)
	}
}

func (self *BasicBlock) String() string {
	n := self.End - self.Src
	v := make([]string, n+1)

	/* dump every instructions */
	for i := self.Src; i < self.End; i++ {
		v[i-self.Src+1] = "    " + self.P[i].Disassemble()
	}

	/* add the entry label */
	v[0] = fmt.Sprintf("L_%d:", self.Src)
	return strings.Join(v, "\n")
}

type GraphBuilder struct {
	Pin   map[int]bool
	Graph map[int]*BasicBlock
}

func (self *GraphBuilder) scan(p Program) {
	for _, v := range p {
		if _OpBranches[v.Op] {
			self.Pin[v.To] = true
		}
	}
}

func (self *GraphBuilder) block(p Program, i int, bb *BasicBlock) {
	bb.Src = i
	bb.End = i

	/* traverse down until it hits a branch instruction */
	for i < len(p) && !_OpBranches[p[i].Op] {
		i++
		bb.End++

		/* hit a merge point, merge with existing block */
		if self.Pin[i] {
			bb.Link = append(bb.Link, self.branch(p, i))
			return
		}
	}

	/* end of basic block */
	if i == len(p) {
		return
	}

	/* also include the branch instruction */
	if bb.End++; p[i].Op != OP_struct_switch {
		bb.Link = append(bb.Link, self.branch(p, p[i].To))
	} else {
		for _, v := range p[i].IntSeq() {
			if v >= 0 {
				bb.Link = append(bb.Link, self.branch(p, v))
			}
		}
	}

	/* GOTO instruction doesn't technically "branch", anything
	 * sits between it and the next branch target are unreachable. */
	if p[i].Op != OP_goto {
		bb.Link = append(bb.Link, self.branch(p, i+1))
	}
}

func (self *GraphBuilder) branch(p Program, i int) *BasicBlock {
	var ok bool
	var bb *BasicBlock

	/* check for existing basic blocks */
	if bb, ok = self.Graph[i]; ok {
		return bb
	}

	/* create a new block */
	bb = newBasicBlock()
	bb.P, bb.Link = p, bb.Link[:0]

	/* process the new block */
	self.Graph[i] = bb
	self.block(p, i, bb)
	return bb
}

func (self *GraphBuilder) Free() {
	rt.MapClear(self.Pin)
	rt.MapClear(self.Graph)
	freeGraphBuilder(self)
}

func (self *GraphBuilder) Build(p Program) *BasicBlock {
	self.scan(p)
	return self.branch(p, 0)
}

func (self *GraphBuilder) BuildAndFree(p Program) (bb *BasicBlock) {
	bb = self.Build(p)
	self.Free()
	return
}

type _OptimizerState struct {
	buf  []*BasicBlock
	refs map[int]int
	mask map[*BasicBlock]bool
}

func (self *_OptimizerState) visit(bb *BasicBlock) bool {
	var mm bool
	var ok bool

	/* check for duplication */
	if mm, ok = self.mask[bb]; mm && ok {
		return false
	}

	/* add to block buffer */
	self.buf = append(self.buf, bb)
	self.mask[bb] = true
	return true
}

func Optimize(p Program) Program {
	acc := 0
	ret := newProgram()
	buf := utils.NewQueue()
	ctx := newOptimizerState()
	cfg := newGraphBuilder().BuildAndFree(p)

	/* travel with BFS */
	for buf.Enqueue(cfg); !buf.Empty(); {
		v := buf.Dequeue()
		b := v.(*BasicBlock)

		/* check for duplication, and then mark as visited */
		if !ctx.visit(b) {
			continue
		}

		/* optimize each block */
		for _, pass := range _PassTab {
			pass(b)
		}

		/* add conditional branches if any */
		for _, q := range b.Link {
			buf.Enqueue(q)
		}
	}

	/* sort the blocks by entry point */
	sort.Slice(ctx.buf, func(i int, j int) bool {
		return ctx.buf[i].Src < ctx.buf[j].Src
	})

	/* remap all the branch locations */
	for _, bb := range ctx.buf {
		ctx.refs[bb.Src] = acc
		acc += bb.End - bb.Src
	}

	/* adjust all the branch targets */
	for _, bb := range ctx.buf {
		if end := bb.End; bb.Src != end {
			if ins := &bb.P[end-1]; _OpBranches[ins.Op] {
				if ins.Op != OP_struct_switch {
					ins.To = ctx.refs[ins.To]
				} else {
					for i, v := range ins.IntSeq() {
						if v >= 0 {
							ins.IntSeq()[i] = ctx.refs[v]
						}
					}
				}
			}
		}
	}

	/* merge all the basic blocks */
	for _, bb := range ctx.buf {
		ret = append(ret, bb.P[bb.Src:bb.End]...)
	}

	/* release the original program */
	p.Free()
	freeOptimizerState(ctx)
	return ret
}

var _PassTab = [...]func(p *BasicBlock){
	_PASS_SeekMerging,
	_PASS_NopElimination,
	_PASS_Compacting,
}

const (
	_NOP OpCode = 0xff
)

func init() {
	_OpNames[_NOP] = "(nop)"
}

// Seek Merging Pass: merges seeking instructions as much as possible.
func _PASS_SeekMerging(bb *BasicBlock) {
	for i := bb.Src; i < bb.End; i++ {
		if p := &bb.P[i]; p.Op == OP_seek {
			for r, j := true, i+1; r && j < bb.End; i, j = i+1, j+1 {
				switch bb.P[j].Op {
				case _NOP:
					break
				case OP_seek:
					p.Iv += bb.P[j].Iv
					bb.P[j].Op = _NOP
				default:
					r = false
				}
			}
		}
	}
}

// NOP Elimination Pass: remove instructions that are effectively NOPs (`seek 0`)
func _PASS_NopElimination(bb *BasicBlock) {
	for i := bb.Src; i < bb.End; i++ {
		if bb.P[i].Iv == 0 && bb.P[i].Op == OP_seek {
			bb.P[i].Op = _NOP
		}
	}
}

// Compacting Pass: remove all the placeholder NOP instructions inserted in the previous pass.
func _PASS_Compacting(bb *BasicBlock) {
	var i int
	var j int

	/* copy instructins excluding NOPs */
	for i, j = bb.Src, bb.Src; i < bb.End; i++ {
		if bb.P[i].Op != _NOP {
			bb.P[j] = bb.P[i]
			j++
		}
	}

	/* update basic block end if needed */
	if i != j {
		bb.End = j
	}
}
