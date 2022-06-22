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

import (
    `fmt`
    `sort`
    `strings`

    `github.com/oleiade/lane`
)

type _Term interface {
    fmt.Stringer
    term()
}

type (
    _TrRel     uint8
    _RegTerm   Reg
    _ValueTerm Int65
)

func (_Stmt)      term() {}
func (_RegTerm)   term() {}
func (_ValueTerm) term() {}

func (self _RegTerm)   String() string { return Reg(self).String() }
func (self _ValueTerm) String() string { return Int65(self).String() }

const (
    _R_eq _TrRel = iota
    _R_ne
    _R_lt
    _R_ltu
    _R_ge
    _R_geu
)

func (self _TrRel) String() string {
    switch self {
        case _R_eq  : return "=="
        case _R_ne  : return "!="
        case _R_lt  : return "<"
        case _R_ltu : return "<#"
        case _R_ge  : return ">="
        case _R_geu : return ">=#"
        default     : panic("unreachable")
    }
}

type _Edge struct {
    bb *BasicBlock
    to *BasicBlock
}

func (self _Edge) String() string {
    return fmt.Sprintf("bb_%d => bb_%d", self.bb.Id, self.to.Id)
}

type _Stmt struct {
    lhs Reg
    rhs _Term
    rel _TrRel
}

func (self _Stmt) String() string {
    return fmt.Sprintf("%s %s %s", self.lhs, self.rel, self.rhs)
}

func (self _Stmt) negated() _Stmt {
    switch self.rel {
        case _R_eq  : return _Stmt { self.lhs, self.rhs, _R_ne }
        case _R_ne  : return _Stmt { self.lhs, self.rhs, _R_eq }
        case _R_lt  : return _Stmt { self.lhs, self.rhs, _R_ge }
        case _R_ltu : return _Stmt { self.lhs, self.rhs, _R_geu }
        case _R_ge  : return _Stmt { self.lhs, self.rhs, _R_lt }
        case _R_geu : return _Stmt { self.lhs, self.rhs, _R_ltu }
        default     : panic("unreachable")
    }
}

func (self _Stmt) condition(cond bool) _Stmt {
    if cond {
        return self
    } else {
        return self.negated()
    }
}

type _Range struct {
    rr []Int65
}

func newRange(lower Int65, upper Int65) *_Range {
    return &_Range {
        rr: []Int65 { lower, upper },
    }
}

func (self *_Range) lower() Int65 {
    if len(self.rr) == 0 {
        panic("empty range")
    } else {
        return self.rr[0]
    }
}

func (self *_Range) upper() Int65 {
    if n := len(self.rr); n == 0 {
        panic("empty range")
    } else {
        return self.rr[n - 1]
    }
}

func (self *_Range) truth() (bool, bool) {
    var lower Int65
    var upper Int65

    /* empty range */
    if len(self.rr) == 0 {
        return false, false
    }

    /* fast path: there is only one range */
    if len(self.rr) == 2 {
        if self.rr[0].CompareZero() == 0 && self.rr[1].CompareZero() == 0 {
            return false, true
        } else if self.rr[0].CompareZero() > 0 || self.rr[1].CompareZero() < 0 {
            return true, true
        } else {
            return false, false
        }
    }

    /* check if any range contains the zero */
    for i := 0; i < len(self.rr); i += 2 {
        lower = self.rr[i]
        upper = self.rr[i + 1]

        /* the range contains zero, the truth cannot be determained */
        if lower.CompareZero() <= 0 && upper.CompareZero() >= 0 {
            return false, false
        }
    }

    /* no, the range can be interpreted as true */
    return true, true
}

func (self *_Range) remove(lower Int65, upper Int65) {
    for i := 0; i < len(self.rr); i += 2 {
        l := self.rr[i]
        u := self.rr[i + 1]

        /* not intersecting */
        if lower.Compare(u) > 0 { break }
        if upper.Compare(l) < 0 { continue }

        /* splicing */
        if lower.Compare(l) > 0 && upper.Compare(u) < 0 {
            next := []Int65 { l, lower.OneLess(), upper.OneMore(), u }
            self.rr = append(self.rr[:i], append(next, self.rr[i + 2:]...)...)
            i += 2
            break
        }

        /* remove the upper half */
        if lower.Compare(l) > 0 {
            self.rr[i + 1] = lower.OneLess()
            continue
        }

        /* remove the lower half */
        if upper.Compare(u) < 0 {
            self.rr[i] = upper.OneMore()
            break
        }

        /* remove the entire range */
        copy(self.rr[i:], self.rr[i + 2:])
        self.rr = self.rr[:len(self.rr) - 2]
        i -= 2
    }
}

func (self *_Range) intersect(lower Int65, upper Int65) {
    if lower != MinInt65 { self.remove(MinInt65, lower.OneLess()) }
    if upper != MaxInt65 { self.remove(upper.OneMore(), MaxInt65) }
}

func (self *_Range) removeRange(r *_Range) {
    for i := 0; i < len(r.rr); i += 2 {
        self.remove(r.rr[i], r.rr[i + 1])
    }
}

func (self *_Range) intersectRange(r *_Range) {
    for i := 0; i < len(r.rr); i += 2 {
        self.intersect(r.rr[i], r.rr[i + 1])
    }
}

func (self *_Range) String() string {
    nb := len(self.rr)
    rb := make([]string, nb / 2)

    /* empty ranges */
    if nb == 0 {
        return "{ (empty) }"
    }

    /* dump every range */
    for i := 0; i < nb; i += 2 {
        l := self.rr[i]
        u := self.rr[i + 1]
        s := new(strings.Builder)

        /* lower bounds */
        if s.WriteRune('['); l == MinInt65 {
            s.WriteString("-∞")
        } else {
            s.WriteString(fmt.Sprint(l))
        }

        /* upper bounds */
        if s.WriteString(", "); u == MaxInt65 {
            s.WriteString("+∞")
        } else {
            s.WriteString(fmt.Sprint(u))
        }

        /* build the range */
        s.WriteRune(']')
        rb[i / 2] = s.String()
    }

    /* join them together */
    return fmt.Sprintf(
        "{ %s }",
        strings.Join(rb, " ∪ "),
    )
}

type _Ranges struct {
    rr map[Reg]*_Range
}

func newRanges(nb int) (r _Ranges) {
    r.rr = make(map[Reg]*_Range, nb)
    r.rr[Rz] = newRange(Int65{}, Int65{})
    return
}

func (self _Ranges) of(reg Reg) (r *_Range) {
    var ok bool
    var rr *_Range

    /* check for existing range */
    if rr, ok = self.rr[reg]; ok {
        return rr
    }

    /* create a new one if needed */
    rr = newRange(MinInt65, MaxInt65)
    self.rr[reg] = rr
    return rr
}

func (self _Ranges) at(reg Reg) (r *_Range, ok bool) {
    r, ok = self.rr[reg]
    return
}

type _Proof struct {
    cp []int
    st []_Stmt
}

func (self *_Proof) define(lhs Reg, rhs _Term, rel _TrRel) {
    self.st = append(self.st, _Stmt { lhs, rhs, rel })
}

func (self *_Proof) assume(ref Reg, lhs Reg, rhs _Term, rel _TrRel) {
    self.st = append(self.st, _Stmt {
        lhs: ref,
        rel: _R_eq,
        rhs: _Stmt { lhs, rhs, rel },
    })
}

func (self *_Proof) restore() {
    p := len(self.cp) - 1
    self.st, self.cp = self.st[:self.cp[p]], self.cp[:p]
}

func (self *_Proof) checkpoint() {
    self.cp = append(self.cp, len(self.st))
}

func (self *_Proof) isContradiction(st _Stmt) (ret bool) {
    self.checkpoint()
    self.st = append(self.st, st)
    ret = !self.verifyCorrectness()
    self.restore()
    return
}

func (self *_Proof) verifyCorrectness() bool {
    rt := true
    rr := newRanges(len(self.st))
    sp := make([]_Stmt, 0, len(self.st))
    st := append([]_Stmt(nil), self.st...)

    /* calculate ranges for every variable */
    for rt {
        rt = false
        sp, st = st, sp[:0]

        /* update all the ranges */
        for _, v := range sp {
            var f bool
            var p _ValueTerm

            /* must be a value term */
            if p, f = v.rhs.(_ValueTerm); !f {
                continue
            }

            /* evaluate the range */
            switch x := Int65(p); v.rel {
                default: {
                    panic("unreachable")
                }

                /* simple ranges */
                case _R_ne: rr.of(v.lhs).remove(x, x)
                case _R_eq: rr.of(v.lhs).intersect(x, x)
                case _R_ge: rr.of(v.lhs).intersect(x, MaxInt65)

                /* signed less-than */
                case _R_lt: {
                    if x == MinInt65 {
                        return false
                    } else {
                        rr.of(v.lhs).intersect(MinInt65, x.OneLess())
                    }
                }

                /* unsigned greater-than-or-equal-to */
                case _R_geu: {
                    if x.CompareZero() < 0 {
                        panic(fmt.Sprintf("unsigned comparison to a negative value %s", x))
                    } else {
                        rr.of(v.lhs).intersect(x, MaxInt65)
                    }
                }

                /* unsigned less-than */
                case _R_ltu: {
                    if x.CompareZero() <= 0 {
                        panic(fmt.Sprintf("unsigned comparison to a non-positive value %s", x))
                    } else {
                        rr.of(v.lhs).intersect(Int65{}, x.OneLess())
                    }
                }
            }
        }

        /* expand all the definations */
        for _, v := range sp {
            if p, ok := v.rhs.(_Stmt); ok {
                if r, rk := rr.at(v.lhs); rk {
                    if t, tk := r.truth(); tk {
                        rt = true
                        st = append(st, p.condition(t))
                    }
                }
            }
        }

        /* evaluate all the registers */
        for _, v := range sp {
            var f bool
            var x Int65
            var r *_Range
            var t _RegTerm

            /* must be a register term with a valid range */
            if t, f = v.rhs.(_RegTerm) ; !f { continue }
            if r, f = rr.at(Reg(t))    ; !f { continue }

            /* empty range, already found contradictions */
            if len(r.rr) == 0 {
                return false
            }

            /* update the ranges */
            switch v.rel {
                default: {
                    panic("unreachable")
                }

                /* equality and inequality */
                case _R_ne: rr.of(v.lhs).removeRange(r)
                case _R_eq: rr.of(v.lhs).intersectRange(r)

                /* signed less-than */
                case _R_lt: {
                    rt = true
                    st = append(st, _Stmt { v.lhs, _ValueTerm(r.upper()), _R_lt })
                }

                /* signed greater-than */
                case _R_ge: {
                    rt = true
                    st = append(st, _Stmt { v.lhs, _ValueTerm(r.lower()), _R_ge })
                }

                /* unsigned less-than */
                case _R_ltu: {
                    if x, rt = r.upper(), true; x.CompareZero() > 0 {
                        st = append(st, _Stmt { v.lhs, _ValueTerm(x), _R_ltu })
                    } else {
                        return false
                    }
                }

                /* unsigned greater-than-or-equal-to */
                case _R_geu: {
                    if x, rt = r.lower(), true; x.CompareZero() >= 0 {
                        st = append(st, _Stmt { v.lhs, _ValueTerm(x), _R_geu })
                    } else {
                        st = append(st, _Stmt { v.lhs, _ValueTerm(Int65{}), _R_geu })
                    }
                }
            }
        }
    }

    /* the statements are valid iff there are no empty ranges */
    for _, r := range rr.rr {
        if len(r.rr) == 0 {
            return false
        }
    }

    /* all checked fine */
    return true
}

// BranchElim removes branches that can be proved unreachable.
type BranchElim struct{}

func (self BranchElim) dfs(cfg *CFG, bb *BasicBlock, ps *_Proof) {
    var ok bool
    var sw *IrSwitch

    /* add facts for this basic block */
    for _, v := range bb.Ins {
        switch p := v.(type) {
            default: {
                break
            }

            /* integer constant */
            case *IrConstInt: {
                ps.define(p.R, _ValueTerm(Int65i(p.V)), _R_eq)
            }

            /* binary operators */
            case *IrBinaryExpr: {
                switch p.Op {
                    case IrCmpEq  : ps.assume(p.R, p.X, _RegTerm(p.Y), _R_eq)
                    case IrCmpNe  : ps.assume(p.R, p.X, _RegTerm(p.Y), _R_ne)
                    case IrCmpLt  : ps.assume(p.R, p.X, _RegTerm(p.Y), _R_lt)
                    case IrCmpLtu : ps.assume(p.R, p.X, _RegTerm(p.Y), _R_ltu)
                    case IrCmpGeu : ps.assume(p.R, p.X, _RegTerm(p.Y), _R_geu)
                }
            }
        }
    }

    /* only care about switches */
    if sw, ok = bb.Term.(*IrSwitch); !ok {
        return
    }

    /* edges to be removed */
    rem := lane.NewQueue()
    del := make(map[_Edge]bool)
    val := make([]int32, 0, len(sw.Br))

    /* prove every branch */
    for v, p := range sw.Br {
        if val = append(val, v); ps.isContradiction(_Stmt { sw.V, _ValueTerm(Int65i(int64(v))), _R_eq }) {
            delete(sw.Br, v)
            rem.Enqueue(_Edge { bb, p.To })
        }
    }

    /* create a save-point */
    ps.checkpoint()
    sort.Slice(val, func(i int, j int) bool { return val[i] < val[j] })

    /* add all the negated conditions */
    for _, i := range val {
        ps.define(sw.V, _ValueTerm(Int65i(int64(i))), _R_ne)
    }

    /* prove the default branch */
    reachable := ps.verifyCorrectness()
    ps.restore()

    /* check for reachability */
    if !reachable {
        if rem.Enqueue(_Edge { bb, sw.Ln.To }); len(sw.Br) != 1 {
            sw.Ln = IrUnlikely(cfg.CreateUnreachable(bb))
        } else {
            sw.Ln, sw.Br = sw.Br[val[0]], make(map[int32]IrBranch)
        }
    }

    /* clear register reference if needed */
    if len(sw.Br) == 0 {
        sw.V = Rz
    }

    /* adjust all the edges */
    for !rem.Empty() {
        e := rem.Pop().(_Edge)
        del[e] = true

        /* adjust Phi nodes in the target block */
        for _, v := range e.to.Phi {
            delete(v.V, e.bb)
        }

        /* remove predecessors from the target block */
        for i, p := range e.to.Pred {
            if p == e.bb {
                e.to.Pred = append(e.to.Pred[:i], e.to.Pred[i + 1:]...)
                break
            }
        }

        /* remove the entire block if no more entry edges left */
        if len(e.to.Pred) == 0 {
            for it := e.to.Term.Successors(); it.Next(); {
                rem.Enqueue(_Edge {
                    bb: e.to,
                    to: it.Block(),
                })
            }
        }
    }

    /* DFS the dominator tree */
    for _, p := range cfg.DominatorOf[bb.Id] {
        var f bool
        var v _ValueTerm

        /* no need to recurse into unreachable branches */
        if del[_Edge { bb, p }] {
            continue
        }

        /* find the branch value */
        for i, b := range sw.Br {
            if b.To == p {
                f, v = true, _ValueTerm(Int65i(int64(i)))
                break
            }
        }

        /* it is not a direct successor, just pass all the facts down */
        if !f {
            self.dfs(cfg, p, ps)
            continue
        }

        /* add the fact and recurse into the node */
        ps.checkpoint()
        ps.define(sw.V, v, _R_eq)
        self.dfs(cfg, p, ps)
        ps.restore()
    }
}

func (self BranchElim) Apply(cfg *CFG) {
    self.dfs(cfg, cfg.Root, new(_Proof))
    cfg.Rebuild()
}
