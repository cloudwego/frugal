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
    `io/ioutil`
    `strings`
    `testing`

    `github.com/cloudwego/frugal/internal/atm/hir`
)

func dumpbb(bb *BasicBlock) string {
    var w int
    var phi []string
    var ins []string
    var term []string
    for _, v := range bb.Phi {
        for _, ss := range strings.Split(v.String(), "\n") {
            vv := html.EscapeString(ss)
            vv = strings.ReplaceAll(vv, "$", "$$")
            phi = append(phi, fmt.Sprintf("<tr><td align=\"left\">%s</td></tr>\n", vv))
            if len(ss) > w {
                w = len(ss)
            }
        }
    }
    for _, v := range bb.Ins {
        for _, ss := range strings.Split(v.String(), "\n") {
            vv := html.EscapeString(ss)
            vv = strings.ReplaceAll(vv, "$", "$$")
            ins = append(ins, fmt.Sprintf("<tr><td align=\"left\">%s</td></tr>\n", vv))
            if len(ss) > w {
                w = len(ss)
            }
        }
    }
    for _, ss := range strings.Split(bb.Term.String(), "\n") {
        vv := html.EscapeString(ss)
        vv = strings.ReplaceAll(vv, "$", "$$")
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


func cfgdot(cfg *CFG, fn string) {
    e := make(map[[2]int]bool)
    buf := []string {
        "digraph CFG {",
        `    xdotversion = "15"`,
        `    graph [ fontname = "Fira Code" ]`,
        `    node [ fontname = "Fira Code" fontsize = "16" shape = "plaintext" ]`,
        `    edge [ fontname = "Fira Code" ]`,
        `    START [ shape = "circle" ]`,
        fmt.Sprintf(`    START -> bb_%d`, cfg.Root.Id),
    }
    cfg.ReversePostOrder(func(p *BasicBlock) {
        f := true
        it := p.Term.Successors()
        buf = append(buf, fmt.Sprintf(`    bb_%d [ label = < %s > ]`, p.Id, dumpbb(p)))
        for it.Next() {
            ln := it.Block()
            edge := [2]int{p.Id, ln.Id}
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
    })
    buf = append(buf, "}")
    err := ioutil.WriteFile(fn, []byte(strings.Join(buf, "\n")), 0644)
    if err != nil {
        panic(err)
    }
}

func TestCFG_Build(t *testing.T) {
    p := hir.CreateBuilder()
    p.LDAP  (0, hir.P0)
    p.LDAP  (1, hir.P1)
    p.SP    (hir.P0, hir.P1, 16)
    p.LQ    (hir.P0, 8, hir.R0)
    p.SP    (hir.P0, hir.P1, 16)
    p.LQ    (hir.P0, 8, hir.R1)
    p.RET   ().R0(hir.R0).R1(hir.R1)
    t.Logf("Generating CFG ...")
    c := p.Build()
    g := Compile(c, (func())(nil))
    t.Logf("Generating DOT file ...")
    cfgdot(g, "/tmp/cfg.gv")
}
