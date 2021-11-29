/*
 * Copyright 2021 ByteDance Inc.
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

package atm

import (
    `fmt`
    `os`
    `os/exec`
    `strings`
    `testing`

    `github.com/oleiade/lane`
    `github.com/stretchr/testify/require`
)

func dumpbb(bb *BasicBlock, refs map[*Instr]string) string {
    p := bb.Src
    buf := []string{fmt.Sprintf(`%p BB_%d:\l`, bb.Src, bb.Id)}
    for i := 0; i < bb.Len; i++ {
        for _, line := range strings.Split(p.disassemble(refs), "\n") {
            buf = append(buf, fmt.Sprintf(`%p %4c%s\l`, p, ' ', line))
        }
        p = p.Ln
    }
    ret := strings.Join(buf, "")
    ret = strings.ReplaceAll(ret, `"`, `\"`)
    return ret
}

func cfgdot(b *GraphBuilder, bb *BasicBlock, fn string) {
    type Route struct {
        A int
        B int
    }
    q := lane.NewQueue()
    r := make(map[Route]bool)
    t := make(map[*Instr]string)
    m := make(map[*BasicBlock]struct{})
    for k, v := range b.Graph {
        t[k] = fmt.Sprintf("BB_%d", v.Id)
    }
    buf := []string {
        "digraph CFG {",
        `    graph [ fontname = "monospace" ]`,
        `    node [ fontname = "monospace", shape = "box" ]`,
        `    edge [ fontname = "monospace" ]`,
        `    START [ shape = "circle" ]`,
        fmt.Sprintf(`    BB_%d [ label = "%s" ]`, bb.Id, dumpbb(bb, t)),
        fmt.Sprintf(`    START -> BB_%d`, bb.Id),
    }
    for q.Enqueue(bb); !q.Empty(); {
        p := q.Dequeue().(*BasicBlock)
        for i, ln := range p.Link {
            if _, ok := m[ln]; !ok {
                buf = append(buf, fmt.Sprintf(`    BB_%d [ label = "%s" ]`, ln.Id, dumpbb(ln, t)))
                q.Enqueue(ln)
            }
            rt := Route{A: p.Id, B: ln.Id}
            if !r[rt] {
                tag := ""
                if p.Cond != nil {
                    if p.Cond.Op == OP_jal {
                        tag = ` [ color = "blue" ]`
                    } else if p.Cond.Op != OP_bsw {
                        if i == 0 {
                            tag = ` [ color = "red", label = "false" ]`
                        } else {
                            tag = ` [ color = "green", label = "true" ]`
                        }
                    } else {
                        if i == 0 {
                            tag = ` [ color = "red", label = "default" ]`
                        } else {
                            for sw, v := range p.Cond.Sw() {
                                if v == ln.Src {
                                    tag = fmt.Sprintf(` [ color = "green", label = "case %d" ]`, sw)
                                    break
                                }
                            }
                            if tag == "" {
                                panic("cfgdot: invalid switch tab")
                            }
                        }
                    }
                }
                buf = append(buf, fmt.Sprintf(`    BB_%d -> BB_%d%s`, p.Id, ln.Id, tag))
            }
            r[rt] = true
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
        A = R0
        B = R1
        C = R2
        D = R3
        E = R4
    )
    p := CreateBuilder()
    p.IQ(0, A)
    p.IQ(1, B)
    p.IQ(2, C)
    p.IQ(3, D)
    p.IQ(4, E)
    p.ADD(B, C, A)
    p.BSW(A, []string{"r", "a", "b"})
    p.SUB(Rz, A, D)
    p.Label("r")
    p.SUB(D, E, E)
    p.BEQ(D, Rz, "a")
    p.MULI(B, 2, E)
    p.JAL("b", Pn)
    p.SBITI(A, 1, B)
    p.Label("a")
    p.ADD(D, E, B)
    p.SUBI(E, 1, E)
    p.Label("b")
    p.ADD(A, C, B)
    p.BLT(B, Rz, "r")
    p.HALT()
    t.Logf("Generating CFG ...")
    b := CreateGraphBuilder()
    g := b.Build(p.Build())
    t.Logf("Generating DOT file ...")
    cfgdot(b, g, "/tmp/cfg.gv")
    t.Logf("Done. Launching 'xdot' to view the file ...")
    require.NoError(t, exec.Command("xdot", "/tmp/cfg.gv").Run())
}
