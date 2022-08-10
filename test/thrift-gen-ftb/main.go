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
	"flag"
	"io/ioutil"
	"os"

	"github.com/cloudwego/thriftgo/plugin"
)

func main() {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	err := f.Parse(os.Args[1:])
	if err != nil {
		println(err)
		os.Exit(2)
	}
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		println("Failed to get input:", err.Error())
		os.Exit(1)
	}

	req, err := plugin.UnmarshalRequest(data)
	if err != nil {
		println("Failed to unmarshal request:", err.Error())
		os.Exit(1)
	}

	os.Exit(exit(run(req)))
}

func run(req *plugin.Request) *plugin.Response {
	var warnings []string

	g := newGenerator(req)
	contents, err := g.generate()
	if err != nil {
		return &plugin.Response{Warnings: []string{err.Error()}}
	}

	warnings = append(warnings, g.warnings...)
	for i := range warnings {
		warnings[i] = "[thrift-gen-ftb] " + warnings[i]
	}
	res := &plugin.Response{
		Warnings: warnings,
		Contents: contents,
	}

	return res
}

func exit(res *plugin.Response) int {
	data, err := plugin.MarshalResponse(res)
	if err != nil {
		println("[thrift-gen-ftb] Failed to marshal response:", err.Error())
		return 1
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		println("[thrift-gen-ftb] Error at writing response out:", err.Error())
		return 1
	}
	return 0
}
