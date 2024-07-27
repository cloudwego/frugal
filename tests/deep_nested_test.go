/*
 * Copyright 2023 CloudWeGo Authors
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

package tests

import (
	"testing"

	"github.com/cloudwego/frugal"
)

type L0 struct {
	L1 *L1 `thrift:"l1,1,optional" frugal:"1,optional,L1"`
}
type L1 struct {
	L2 *L2 `thrift:"l2,1,optional" frugal:"1,optional,L2"`
}
type L2 struct {
	L3 *L3 `thrift:"l3,1,optional" frugal:"1,optional,L3"`
}
type L3 struct {
	L4 *L4 `thrift:"l4,1,optional" frugal:"1,optional,L4"`
}
type L4 struct {
	L5 *L5 `thrift:"l5,1,optional" frugal:"1,optional,L5"`
}
type L5 struct {
	L6 *L6 `thrift:"l6,1,optional" frugal:"1,optional,L6"`
}
type L6 struct {
	L7 *L7 `thrift:"l7,1,optional" frugal:"1,optional,L7"`
}
type L7 struct {
	L8 *L8 `thrift:"l8,1,optional" frugal:"1,optional,L8"`
}
type L8 struct {
	L9 *L9 `thrift:"l9,1,optional" frugal:"1,optional,L9"`
}
type L9 struct {
	L10 *L10 `thrift:"l10,1,optional" frugal:"1,optional,L10"`
}
type L10 struct {
	L11 *L11 `thrift:"l11,1,optional" frugal:"1,optional,L11"`
}
type L11 struct {
	L12 *L12 `thrift:"l12,1,optional" frugal:"1,optional,L12"`
}
type L12 struct {
	L13 *L13 `thrift:"l13,1,optional" frugal:"1,optional,L13"`
}
type L13 struct {
	L14 *L14 `thrift:"l14,1,optional" frugal:"1,optional,L14"`
}
type L14 struct {
	L15 *L15 `thrift:"l15,1,optional" frugal:"1,optional,L15"`
}
type L15 struct {
	L16 *L16 `thrift:"l16,1,optional" frugal:"1,optional,L16"`
}
type L16 struct {
	L17 *L17 `thrift:"l17,1,optional" frugal:"1,optional,L17"`
}
type L17 struct {
	L18 *L18 `thrift:"l18,1,optional" frugal:"1,optional,L18"`
}
type L18 struct {
	L19 *L19 `thrift:"l19,1,optional" frugal:"1,optional,L19"`
}
type L19 struct {
	L20 *L20 `thrift:"l20,1,optional" frugal:"1,optional,L20"`
}
type L20 struct {
	End *bool `thrift:"bool,1,optional" frugal:"1,optional,bool"`
}

var (
	deepNested = &L0{&L1{&L2{&L3{&L4{&L5{
		&L6{&L7{&L8{&L9{&L10{
			&L11{&L12{&L13{&L14{&L15{
				&L16{&L17{&L18{&L19{&L20{}}}}}}}}}}}}}}}}}}}}}
)

func TestDeepNested(t *testing.T) {
	for i := 0; i < 100; i++ {
		frugal.EncodedSize(deepNested)
	}
}
