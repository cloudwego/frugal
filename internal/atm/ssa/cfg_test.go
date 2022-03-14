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

func dumpbb(bb *BasicBlock) string {
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
    buf := []string {
        "<table border=\"1\" cellborder=\"0\" cellspacing=\"0\">\n",
        fmt.Sprintf("<tr><td width=\"%d\">bb_%d</td></tr>\n", w * 10 + 5, bb.Id),
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

func cfgdot(bb *BasicBlock, fn string) {
    q := lane.NewQueue()
    m := make(map[*BasicBlock]struct{})
    buf := []string {
        "digraph CFG {",
        `    xdotversion = "15"`,
        `    graph [ fontname = "monospace" ]`,
        `    node [ fontname = "monospace" fontsize="16" shape = "plaintext" ]`,
        `    edge [ fontname = "monospace" ]`,
        `    START [ shape = "circle" ]`,
        fmt.Sprintf(`    bb_%d [ label = < %s > ]`, bb.Id, dumpbb(bb)),
        fmt.Sprintf(`    START -> bb_%d`, bb.Id),
    }
    for q.Enqueue(bb); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        if sw, ok := p.Term.(*IrSwitch); ok {
            var vvs []int64
            var lns []*BasicBlock
            for v, ln := range sw.Br {
                vvs = append(vvs, v)
                lns = append(lns, ln)
            }
            lns = append(lns, sw.Ln)
            for i, ln := range lns {
                if _, ok := m[ln]; !ok {
                    buf = append(buf, fmt.Sprintf(`    bb_%d [ label = < %s > ]`, ln.Id, dumpbb(ln)))
                    q.Enqueue(ln)
                }
                if i < len(vvs) {
                    buf = append(buf, fmt.Sprintf(`    bb_%d -> bb_%d [ label = "%d" ]`, p.Id, ln.Id, vvs[i]))
                } else if len(sw.Br) == 0 {
                    buf = append(buf, fmt.Sprintf(`    bb_%d -> bb_%d [ label = "goto" ]`, p.Id, ln.Id))
                } else {
                    buf = append(buf, fmt.Sprintf(`    bb_%d -> bb_%d [ label = "otherwise" ]`, p.Id, ln.Id))
                }
            }
        }
        m[p] = struct{}{}
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
    b := CreateGraphBuilder()
    g := b.Build(p.Build())
    t.Logf("Generating DOT file ...")
    cfgdot(g, "/tmp/cfg.gv")
}
