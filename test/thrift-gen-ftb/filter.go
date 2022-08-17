// Copyright 2022 ByteDance Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"

	"github.com/cloudwego/thriftgo/generator/golang"
	"github.com/cloudwego/thriftgo/parser"
)

func FilterOut(st *golang.StructLike) bool {
	for _, fi := range st.Fields() {
		if strings.HasPrefix(fi.GoName().String(), "_") {
			return true
		}
	}
	return false
}

func HasCircularDependency(scope *golang.Scope, structLike *golang.StructLike, typ *parser.Type) bool {
	m := map[string]bool{scope.AST().Filename + "." + structLike.Name: true}
	return hasCircularDependency(scope, m, typ)
}

func hasCircularDependency(scope *golang.Scope, structLikeMap map[string]bool, typ *parser.Type) bool {
	scope, typ, name := getUnderlay(scope, typ)
	if typ.Category.IsList() || typ.Category.IsSet() {
		return hasCircularDependency(scope, structLikeMap, typ.ValueType)
	}
	if typ.Category.IsMap() {
		return hasCircularDependency(scope, structLikeMap, typ.KeyType) || hasCircularDependency(scope, structLikeMap, typ.ValueType)
	}
	if typ.Category.IsStructLike() {
		if _, ok := structLikeMap[scope.AST().Filename+"."+name]; ok {
			return true
		}
		structLikeMap[scope.AST().Filename+"."+name] = true
		for _, fi := range scope.StructLike(name).Fields() {
			if hasCircularDependency(scope, structLikeMap, fi.Type) {
				return true
			}
		}
	}
	return false
}

func HasDeepContainer(scope *golang.Scope, typ *parser.Type) bool {
	visited := make(map[string]bool)
	return hasDeepContainer(scope, typ, visited, 0)
}

func hasDeepContainer(scope *golang.Scope, typ *parser.Type, visited map[string]bool, depth int) bool {
	if depth > 5 {
		return true
	}
	scope, typ, name := getUnderlay(scope, typ)
	if typ.Category.IsList() || typ.Category.IsSet() {
		return hasDeepContainer(scope, typ.ValueType, visited, depth+1)
	}
	if typ.Category.IsMap() {
		return hasDeepContainer(scope, typ.KeyType, visited, depth+1) || hasDeepContainer(scope, typ.ValueType, visited, depth+1)
	}
	if typ.Category.IsStructLike() {
		if _, ok := visited[scope.AST().Filename+"."+name]; ok {
			return false
		}
		visited[scope.AST().Filename+"."+name] = true
		for _, fi := range scope.StructLike(name).Fields() {
			if hasDeepContainer(scope, fi.Type, visited, depth) {
				return true
			}
		}
	}
	return false
}
