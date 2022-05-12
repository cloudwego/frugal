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

package decoder

import (
    `fmt`
    `html`
    `io/ioutil`
    `strings`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/cloudwego/frugal/internal/atm/pgen`
    `github.com/cloudwego/frugal/internal/atm/ssa`
    `github.com/cloudwego/frugal/internal/loader`
    `github.com/oleiade/lane`
)

func dumpbb(bb *ssa.BasicBlock, cfg *ssa.CFG) string {
    var w int
    var phi []string
    var ins []string
    var term []string
    for _, v := range bb.Phi {
        for _, ss := range strings.Split(v.String(), "\n") {
            vv := strings.ReplaceAll(html.EscapeString(ss), " ", "&nbsp;")
            phi = append(phi, fmt.Sprintf("<tr><td align=\"left\">%s</td></tr>\n", vv))
            if len(ss) > w {
                w = len(ss)
            }
        }
    }
    for _, v := range bb.Ins {
        for _, ss := range strings.Split(v.String(), "\n") {
            vv := strings.ReplaceAll(html.EscapeString(ss), " ", "&nbsp;")
            ins = append(ins, fmt.Sprintf("<tr><td align=\"left\">%s</td></tr>\n", vv))
            if len(ss) > w {
                w = len(ss)
            }
        }
    }
    for _, ss := range strings.Split(bb.Term.String(), "\n") {
        vv := strings.ReplaceAll(html.EscapeString(ss), " ", "&nbsp;")
        term = append(term, fmt.Sprintf("<tr><td align=\"left\">%s</td></tr>\n", vv))
        if len(ss) > w {
            w = len(ss)
        }
    }
    var pred []string
    for _, d := range bb.Pred {
        pred = append(pred, fmt.Sprintf("bb_%d", d.Id))
    }
    idomby := "∅"
    if d := cfg.DominatedBy[bb.Id]; d != nil {
        idomby = fmt.Sprintf("bb_%d", d.Id)
    }
    var idomof []string
    for _, d := range cfg.DominatorOf[bb.Id] {
        idomof = append(idomof, fmt.Sprintf("bb_%d", d.Id))
    }
    var df []string
    for _, d := range cfg.DominanceFrontier[bb.Id] {
        df = append(df, fmt.Sprintf("bb_%d", d.Id))
    }
    meta := []string {
        fmt.Sprintf("# pred = {%s}", strings.Join(pred, ", ")),
        fmt.Sprintf("# idom_by = %s", idomby),
        fmt.Sprintf("# idom_of = {%s}", strings.Join(idomof, ", ")),
        fmt.Sprintf("# df = {%s}", strings.Join(df, ", ")),
    }
    for i, ss := range meta {
        meta[i] = fmt.Sprintf("<tr><td align=\"left\">%s</td></tr>\n", ss)
        if len(ss) > w {
            w = len(ss)
        }
    }
    buf := []string {
        "<table border=\"1\" cellborder=\"0\" cellspacing=\"0\">\n",
        fmt.Sprintf("<tr><td width=\"%d\">bb_%d</td></tr>\n", w * 10 + 5, bb.Id),
    }
    if len(meta) != 0 {
        buf = append(buf, "<hr/>\n")
        buf = append(buf, meta...)
    }
    if len(bb.Phi) != 0 {
        buf = append(buf, "<hr/>\n")
        buf = append(buf, phi...)
    }
    if len(bb.Ins) != 0 {
        buf = append(buf, "<hr/>\n")
        buf = append(buf, ins...)
    }
    buf = append(buf, "<hr/>\n")
    buf = append(buf, term...)
    buf = append(buf, "</table>")
    return strings.Join(buf, "")
}

func cfgdot(cfg *ssa.CFG, fn string) {
    q := lane.NewQueue()
    n := make(map[int]bool)
    e := make(map[struct{A, B int}]bool)
    buf := []string {
        "digraph CFG {",
        `    xdotversion = "15"`,
        `    graph [ fontname = "Fira Code" ]`,
        `    node [ fontname = "Fira Code" fontsize="16" shape = "plaintext" ]`,
        `    edge [ fontname = "Fira Code" ]`,
        `    START [ shape = "circle" ]`,
        fmt.Sprintf(`    START -> bb_%d`, cfg.Root.Id),
    }
    for q.Enqueue(cfg.Root); !q.Empty(); {
        f := true
        p := q.Dequeue().(*ssa.BasicBlock)
        it := p.Term.Successors()
        buf = append(buf, fmt.Sprintf(`    bb_%d [ label = < %s > ]`, p.Id, dumpbb(p, cfg)))
        n[p.Id] = true
        for it.Next() {
            ln := it.Block()
            if !n[ln.Id] {
                q.Enqueue(ln)
            }
            edge := struct{A, B int}{p.Id, ln.Id}
            if !e[edge] {
                e[edge] = true
                if v, ok := it.Value(); ok {
                    f = false
                    buf = append(buf, fmt.Sprintf(`    bb_%d -> bb_%d [ label = "%d" ]`, p.Id, ln.Id, v))
                } else if f {
                    buf = append(buf, fmt.Sprintf(`    bb_%d -> bb_%d [ label = "goto" ]`, p.Id, ln.Id))
                } else {
                    buf = append(buf, fmt.Sprintf(`    bb_%d -> bb_%d [ label = "otherwise" ]`, p.Id, ln.Id))
                }
            }
        }
    }
    buf = append(buf, "}")
    err := ioutil.WriteFile(fn, []byte(strings.Join(buf, "\n")), 0644)
    if err != nil {
        panic(err)
    }
}

type (
    LinkerAMD64 struct{}
)

const (
    _NativeStackSize = _stack__do_skip
)

func init() {
    SetLinker(new(LinkerAMD64))
}

func (LinkerAMD64) Link(p hir.Program) Decoder {
    cfgdot(ssa.Compile(p), "/tmp/cfg.gv")
    fn := pgen.CreateCodeGen((Decoder)(nil)).Generate(p, _NativeStackSize)
    fp := loader.Loader(fn.Code).Load("decoder", fn.Frame)
    return *(*Decoder)(unsafe.Pointer(&fp))
}
