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
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var (
	OutputDir string
	ThriftDir string
)

func init() {
	flag.StringVar(&OutputDir, "out", "testdata", "output directory")
	flag.StringVar(&ThriftDir, "thrift_dir", "", "directory of thrift files")
}

func ThriftSearcher() <-chan (string) {
	out := make(chan (string))
	if ThriftDir == "" {
		br := bufio.NewReader(os.Stdin)
		go func() {
			str, err := br.ReadString('\n')
			for ; err == nil; str, err = br.ReadString('\n') {
				out <- str
			}
			log.Fatalln(fmt.Errorf("read from stdin failed: %w", err))
		}()
		return out
	}
	dirs := make(chan (string))
	go func() {
		for dir := range dirs {
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				log.Println(fmt.Errorf("read directory %s failed: %w", dir, err))
			}
			for _, file := range files {
				if !file.IsDir() {
					out <- file.Name()
				}
			}
		}
	}()
	var searchDir func(string)
	searchDir = func(dir string) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Println(fmt.Errorf("read directory %s failed: %w", dir, err))
		}
		dirs <- dir
		for _, file := range files {
			if file.IsDir() {
				searchDir(file.Name())
			}
		}
	}
	go searchDir(ThriftDir)
	return out
}

func main() {
	flag.Parse()
}
