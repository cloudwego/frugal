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
    `math`
)

type ReachabilityMatrix struct {
    dist [][]uint64
}

func buildReachabilityMatrix(cfg *CFG) {
    nb := cfg.MaxBlock() + 1
    edge := make(map[[2]int]bool)
    cfg.dist = make([][]uint64, nb)

    /* initialize each row */
    for i := range cfg.dist {
        cfg.dist[i] = make([]uint64, nb)
        setu64(cfg.dist[i], math.MaxInt64)
    }

    /* add each block and edge */
    cfg.PostOrder(func(bb *BasicBlock) {
        it := bb.Term.Successors()
        cfg.dist[bb.Id][bb.Id] = 0

        /* add every edge */
        for it.Next() {
            i := it.Block().Id
            e := [2]int { bb.Id, i }

            /* this is a new edge */
            if !edge[e] {
                edge[e] = true
                cfg.dist[bb.Id][i] = 1
            }
        }
    })

    /* Floyd-Warshall algorithm */
    for k := 1; k < nb; k++ {
        for i := 1; i < nb; i++ {
            for j := 1; j < nb; j++ {
                cfg.dist[i][j] = minu64(
                    cfg.dist[i][j],
                    cfg.dist[i][k] + cfg.dist[k][j],
                )
            }
        }
    }
}
