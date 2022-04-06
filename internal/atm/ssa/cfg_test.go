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
    `github.com/cloudwego/frugal/internal/rt`
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

func test_native_entry()                               {}
func test_error_eof(_ int)                             {}
func test_error_type(_ uint8, _ uint8)                 {}
func test_error_skip(_ int)                            {}
func test_error_missing(_ *rt.GoType, _ int, _ uint64) {}

var (
    f_test_native_entry  = hir.RegisterCCall(uintptr(rt.FuncAddr(test_native_entry)), nil)
    f_test_error_eof     = hir.RegisterGCall(test_error_eof, nil)
    f_test_error_type    = hir.RegisterGCall(test_error_type, nil)
    f_test_error_skip    = hir.RegisterGCall(test_error_skip, nil)
    f_test_error_missing = hir.RegisterGCall(test_error_missing, nil)
)

func TestCFG_Build(t *testing.T) {
    p := hir.CreateBuilder()
    p.LDAP  (0, hir.P2)
    p.LDAQ  (2, hir.R2)
    p.LDAP  (3, hir.P1)
    p.LDAP  (4, hir.P3)
    p.LDAQ  (5, hir.R3)
    p.ADDI  (hir.Rz, 2097120, hir.R0)
    p.BGEU  (hir.R3, hir.R0, "L_0")
    p.ADDP  (hir.P3, hir.R3, hir.P0)
    p.SP    (hir.P1, hir.P0, 16)
    p.ADDI  (hir.R3, 32, hir.R3)
    p.Label ("L_5")
    p.ADDI  (hir.Rz, 1, hir.R0)
    p.LDAQ  (1, hir.R1)
    p.BLTU  (hir.R1, hir.R0, "L_1")
    p.ADDP  (hir.P2, hir.R2, hir.P5)
    p.ADDI  (hir.R2, 1, hir.R2)
    p.LB    (hir.P5, 0, hir.R4)
    p.BEQ   (hir.R4, hir.Rz, "L_2")
    p.ADDI  (hir.Rz, 2, hir.R0)
    p.LDAQ  (1, hir.R1)
    p.BLTU  (hir.R1, hir.R0, "L_1")
    p.ADDP  (hir.P2, hir.R2, hir.P5)
    p.ADDI  (hir.R2, 2, hir.R2)
    p.LW    (hir.P5, 0, hir.R0)
    p.SWAPW (hir.R0, hir.R0)
    p.BSW   (hir.R0, []string { "L_3" })
    p.Label ("L_6")
    p.ADDPI (hir.P3, 2097152, hir.P0)
    p.LDAQ  (1, hir.R0)
    p.SUB   (hir.R0, hir.R2, hir.R0)
    p.ADDP  (hir.P2, hir.R2, hir.P5)
    p.CCALL (f_test_native_entry).
      A0    (hir.P0).
      A1    (hir.P5).
      A2    (hir.R0).
      A3    (hir.R4).
      R0    (hir.R0)
    p.BLT   (hir.R0, hir.Rz, "L_4")
    p.ADD   (hir.R2, hir.R0, hir.R2)
    p.JMP   ("L_5")
    p.Label ("L_3")
    p.ADDI  (hir.Rz, 2, hir.R0)
    p.BNE   (hir.R4, hir.R0, "L_6")
    p.ADDPI (hir.P1, 0, hir.P1)
    p.ADDI  (hir.Rz, 1, hir.R0)
    p.LDAQ  (1, hir.R1)
    p.BLTU  (hir.R1, hir.R0, "L_1")
    p.ADDP  (hir.P2, hir.R2, hir.P5)
    p.LB    (hir.P5, 0, hir.R0)
    p.SB    (hir.R0, hir.P1, 0)
    p.ADDI  (hir.R2, 1, hir.R2)
    p.ADDPI (hir.P1, 0, hir.P1)
    p.JMP   ("L_5")
    p.Label ("L_2")
    p.ADDI  (hir.R3, -32, hir.R3)
    p.ADDP  (hir.P3, hir.R3, hir.P0)
    p.LP    (hir.P0, 16, hir.P1)
    p.SP    (hir.Pn, hir.P0, 16)
    p.JMP   ("L_7")
    p.Label ("L_7")
    p.ADDPI (hir.Pn, 0, hir.P4)
    p.ADDPI (hir.Pn, 0, hir.P5)
    p.Label ("L_8")
    p.RET   ().
      R0    (hir.R2).
      R1    (hir.P4).
      R2    (hir.P5)
    p.Label ("L_1")
    p.LDAQ  (1, hir.R1)
    p.SUB   (hir.R0, hir.R1, hir.R0)
    p.GCALL (f_test_error_eof).
      A0    (hir.R0).
      R0    (hir.P4).
      R1    (hir.P5)
    p.JMP   ("L_8")
    p.GCALL (f_test_error_type).
      A0    (hir.R1).
      A1    (hir.R0).
      R0    (hir.P4).
      R1    (hir.P5)
    p.JMP   ("L_8")
    p.Label ("L_4")
    p.GCALL (f_test_error_skip).
      A0    (hir.R0).
      R0    (hir.P4).
      R1    (hir.P5)
    p.JMP   ("L_8")
    p.GCALL (f_test_error_missing).
      A0    (hir.P4).
      A1    (hir.R1).
      A2    (hir.R0).
      R0    (hir.P4).
      R1    (hir.P5)
    p.JMP   ("L_8")
    p.Label ("L_0")
    p.IP    (new(int), hir.P0)
    p.LP    (hir.P0, 0, hir.P4)
    p.LP    (hir.P0, 8, hir.P5)
    p.JMP   ("L_8")
    t.Logf("Generating CFG ...")
    c := p.Build()
    g := BuildCFG(c)
    t.Logf("Generating DOT file ...")
    cfgdot(g, "/tmp/cfg.gv")
}
