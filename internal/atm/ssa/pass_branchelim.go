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
    `math`
    `strings`
)

type _Term interface {
    fmt.Stringer
    term()
}

type (
    _TrRel     uint8
    _RegTerm   Reg
    _ValueTerm int64
)

func (_Stmt)      term() {}
func (_RegTerm)   term() {}
func (_ValueTerm) term() {}

func (self _RegTerm)   String() string { return Reg(self).String() }
func (self _ValueTerm) String() string { return fmt.Sprint(int64(self)) }

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
    rr []int64
}

func newRange(lower int64, upper int64) *_Range {
    return &_Range {
        rr: []int64 { lower, upper },
    }
}

func (self *_Range) lower() int64 {
    if len(self.rr) == 0 {
        panic("empty range")
    } else {
        return self.rr[0]
    }
}

func (self *_Range) upper() int64 {
    if n := len(self.rr); n == 0 {
        panic("empty range")
    } else {
        return self.rr[n - 1]
    }
}

func (self *_Range) truth() (bool, bool) {
    var lower int64
    var upper int64

    /* empty range */
    if len(self.rr) == 0 {
        return false, false
    }

    /* fast path: there is only one range */
    if len(self.rr) == 2 {
        if self.rr[0] == 0 && self.rr[1] == 0 {
            return false, true
        } else if self.rr[0] > 0 || self.rr[1] < 0 {
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
        if lower <= 0 && upper >= 0 {
            return false, false
        }
    }

    /* no, the range can be interpreted as true */
    return true, true
}

func (self *_Range) remove(lower int64, upper int64) {
    for i := 0; i < len(self.rr); i += 2 {
        l := self.rr[i]
        u := self.rr[i + 1]

        /* not intersecting */
        if lower > u { break }
        if upper < l { continue }

        /* splicing */
        if l < lower && upper < u {
            next := []int64 { l, lower - 1, upper + 1, u }
            self.rr = append(self.rr[:i], append(next, self.rr[i + 2:]...)...)
            i += 2
            break
        }

        /* remove the upper half */
        if l < lower {
            self.rr[i + 1] = lower - 1
            continue
        }

        /* remove the lower half */
        if upper < u {
            self.rr[i] = upper + 1
            break
        }

        /* remove the entire range */
        copy(self.rr[i:], self.rr[i + 2:])
        self.rr = self.rr[:len(self.rr) - 2]
        i -= 2
    }
}

func (self *_Range) intersect(lower int64, upper int64) {
    if upper < math.MaxInt64 { self.remove(upper + 1, math.MaxInt64) }
    if lower > math.MinInt64 { self.remove(math.MinInt64, lower - 1) }
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
        if s.WriteRune('['); l == math.MinInt64 {
            s.WriteString("-∞")
        } else {
            s.WriteString(fmt.Sprint(l))
        }

        /* upper bounds */
        if s.WriteString(", "); u == math.MaxInt64 {
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
    r.rr[Rz] = newRange(0, 0)
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
    rr = newRange(math.MinInt64, math.MaxInt64)
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
            var x int64
            var p _ValueTerm

            /* must be a value term */
            if p, f = v.rhs.(_ValueTerm); !f {
                continue
            }

            /* evaluate the range */
            switch x = int64(p); v.rel {
                default: {
                    panic("unreachable")
                }

                /* simple ranges */
                case _R_ne: rr.of(v.lhs).remove(x, x)
                case _R_eq: rr.of(v.lhs).intersect(x, x)
                case _R_ge: rr.of(v.lhs).intersect(x, math.MaxInt64)

                /* signed less-than */
                case _R_lt: {
                    if x == math.MinInt64 {
                        return false
                    } else {
                        rr.of(v.lhs).intersect(math.MinInt64, x - 1)
                    }
                }

                /* unsigned greater-than-or-equal-to */
                case _R_geu: {
                    if x < 0 {
                        panic(fmt.Sprintf("unsigned comparison to a negative value %d", x))
                    } else {
                        rr.of(v.lhs).intersect(x, math.MaxInt64)
                    }
                }

                /* unsigned less-than */
                case _R_ltu: {
                    if x <= 0 {
                        panic(fmt.Sprintf("unsigned comparison to a non-positive value %d", x))
                    } else {
                        rr.of(v.lhs).intersect(0, x - 1)
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
                    rt = true
                    st = append(st, _Stmt { v.lhs, _ValueTerm(r.upper()), _R_ltu })
                }

                /* unsigned greater-than-or-equal-to */
                case _R_geu: {
                    rt = true
                    st = append(st, _Stmt { v.lhs, _ValueTerm(r.lower()), _R_geu })
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
                ps.define(p.R, _ValueTerm(p.V), _R_eq)
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

    /* create a save-point */
    ps.checkpoint()
    rem := make(map[_Edge]bool)

    /* prove every branch */
    for v, p := range sw.Br {
        if ps.isContradiction(_Stmt { sw.V, _ValueTerm(v), _R_eq }) {
            // TODO: handle unreachable
            rem[_Edge { bb, p }] = true
            println(fmt.Sprintf("branch bb_%d => bb_%d (%d) is unreachable", bb.Id, p.Id, v))
        }
    }

    /* add all the negated conditions */
    for i := range sw.Br {
        ps.define(sw.V, _ValueTerm(i), _R_ne)
    }

    /* prove the default branch */
    reachable := ps.verifyCorrectness()
    ps.restore()

    /* check for reachability */
    if !reachable {
        // TODO: handle unreachable
        rem[_Edge { bb, sw.Ln }] = true
        println(fmt.Sprintf("default branch bb_%d => bb_%d is unreachable", bb.Id, sw.Ln.Id))
    }

    /* DFS the dominator tree */
    for _, p := range cfg.DominatorOf[bb.Id] {
        var f bool
        var v _ValueTerm

        /* no need to recurse into unreachable branches */
        if rem[_Edge { bb, p }] {
            continue
        }

        /* find the branch value */
        for i, b := range sw.Br {
            if b == p {
                f, v = true, _ValueTerm(i)
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
}
