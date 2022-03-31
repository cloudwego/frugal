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
    `html`
    `os`
    `strings`
    `testing`

    `github.com/cloudwego/frugal/internal/atm/hir`
    `github.com/oleiade/lane`
)

func dumpbb(bb *BasicBlock, cfg CFG) string {
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

func cfgdot(cfg CFG, fn string) {
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
        p := q.Dequeue().(*BasicBlock)
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
    err := os.WriteFile(fn, []byte(strings.Join(buf, "\n")), 0644)
    if err != nil {
        panic(err)
    }
}

func TestCFG_Build(t *testing.T) {
    const (
        A = hir.R0
        B = hir.R1
        C = hir.R2
        D = hir.R3
        E = hir.R4
    )
    p := hir.CreateBuilder()
    // p.IQ(1, A)
    // p.IQ(1, B)
    // p.BSW(A, []string{"_5", "_9"})
    // p.IQ(2, A)
    // p.Label("_3")
    // p.IQ(3, A)
    // p.BNE(A, B, "_3")
    // p.Label("_4")
    // p.IQ(4, A)
    // p.JMP("_13")
    // p.Label("_5")
    // p.IQ(5, A)
    // p.BEQ(A, B, "_7")
    // p.IQ(6, A)
    // p.BGEU(A, B, "_4")
    // p.JMP("_8")
    // p.Label("_7")
    // p.IQ(7, A)
    // p.BLT(A, B, "_12")
    // p.Label("_8")
    // p.IQ(8, A)
    // p.BNE(A, B, "_5")
    // p.JMP("_13")
    // p.Label("_9")
    // p.IQ(9, A)
    // p.BEQ(A, B, "_11")
    // p.IQ(10, A)
    // p.JMP("_12")
    // p.Label("_11")
    // p.IQ(11, A)
    // p.Label("_12")
    // p.IQ(12, A)
    // p.Label("_13")
    // p.IQ(13, A)
    p.IQ(0, A)
    p.IQ(1, B)
    p.IQ(2, C)
    p.IQ(3, D)
    p.IQ(4, E)
    p.ADD(B, C, A)
    p.SUB(hir.Rz, A, D)
    p.Label("r")
    p.SUB(D, E, E)
    p.BEQ(D, hir.Rz, "a")
    p.MULI(B, 2, E)
    p.JMP("b")
    p.Label("a")
    p.ADD(D, E, B)
    p.SUBI(E, 1, E)
    p.Label("b")
    p.ADD(A, C, B)
    p.BLT(B, hir.Rz, "r")
    p.RET()
    t.Logf("Generating CFG ...")
    c := p.Build()
    g := BuildCFG(c)
    t.Logf("Generating DOT file ...")
    cfgdot(g, "/tmp/cfg.gv")
}
