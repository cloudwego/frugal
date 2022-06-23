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
    `os`
    `sort`
    `strings`

    `github.com/ajstarks/svgo`
)

func draw_liverange(fn string, bb []*BasicBlock, lr map[Reg]*_LiveRange) {
    leni := 0
    maxw := 0
    maxi := 0
    regs := make([]Reg, 0, len(lr))
    insi := make(map[_LivePoint]int)
    for _, b := range bb {
        for _, v := range b.Ins {
            s := v.String()
            s = strings.TrimSpace(strings.Split(s, "# ")[0])
            if len(s) > maxi {
                maxi = len(s)
            }
        }
        s := b.Term.String()
        s = strings.TrimSpace(strings.Split(s, "# ")[0])
        if len(s) > maxi {
            maxi = len(s)
        }
    }
    for r := range lr {
        s := r.String()
        regs = append(regs, r)
        if len(s) > maxw {
            maxw = len(s)
        }
    }
    insw := maxi * 9 + 120
    regw := (maxw + 1) * 8 + 16
    sort.Slice(regs, func(i, j int) bool {
        k1, k2 := regs[i].Kind(), regs[j].Kind()
        n1, n2 := regs[i].Name(), regs[j].Name()
        return k1 < k2 || (k1 == k2 && n1 < n2) || (k1 == k2 && n1 == n2 && regs[i] < regs[j])
    })
    for _, b := range bb {
        leni += len(b.Ins) + 1
    }
    fp, err := os.OpenFile(fn, os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0644)
    if err != nil {
        panic(err)
    }
    p := svg.New(fp)
    p.Start(len(regs) * regw + insw + 100, leni * 24 + 100)
    if _, err = fp.WriteString(`<rect width="100%" height="100%" fill="white" />` + "\n"); err != nil {
        panic(err)
    }
    bbi := 0
    for i, b := range bb {
        p.Text(16, 100 + bbi * 24, fmt.Sprintf("bb_%d", b.Id), "fill:gray;font-size:16px;font-family:monospace")
        p.Line(10, 84 + bbi * 24, insw + 5, 84 + bbi * 24, "stroke:lightgray")
        for j, v := range b.Ins {
            s := v.String()
            h := 95 + bbi * 24
            s = strings.TrimSpace(strings.Split(s, "# ")[0])
            insi[_LivePoint { b: i, i: j }] = h
            p.Text(insw, 100 + bbi * 24, s, "fill:black;font-size:16px;font-family:monospace;text-anchor:end")
            p.Line(insw + 10, h, len(regs) * regw + insw + 50, h, "stroke:gray")
            bbi++
        }
        s := b.Term.String()
        h := 95 + bbi * 24
        s = strings.TrimSpace(strings.Split(s, "# ")[0])
        insi[_LivePoint { b: i, i: len(b.Ins) }] = h
        p.Text(insw, 100 + bbi * 24, s, "fill:black;font-size:16px;font-family:monospace;text-anchor:end")
        p.Line(insw + 10, h, len(regs) * regw + insw + 50, h, "stroke:gray")
        bbi++
    }
    for i, r := range regs {
        x := insw + i * regw + 50
        p.Text(x, 70, r.String(), "fill:black;font-size:16px;font-family:monospace;text-anchor:middle")
        p.Line(x, insi[lr[r].p[0]], x, insi[lr[r].p[len(lr[r].p) - 1]], "stroke:black;stroke-width:3")
        for _, pt := range lr[r].p {
            ins := bb[pt.b].Ins
            isdef := false
            if pt.i < len(ins) {
                if def, ok := bb[pt.b].Ins[pt.i].(IrDefinitions); ok {
                    for _, v := range def.Definitions() {
                        if r == *v {
                            isdef = true
                            break
                        }
                    }
                }
            }
            if !isdef {
                p.Circle(x, insi[pt], 4, "fill:black;stroke:black;stroke-width:2")
            } else {
                p.Circle(x, insi[pt], 4, "fill:white;stroke:black;stroke-width:2")
            }
        }
    }
    p.End()
    if err = fp.Close(); err != nil {
        panic(err)
    }
}
