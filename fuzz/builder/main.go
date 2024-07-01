// Copyright 2022 CloudWeGo Authors
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
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"github.com/cloudwego/frugal"
)

var (
	OutputDir  string
	ThriftDir  string
	MaxFileNum int64
)

func init() {
	flag.StringVar(&OutputDir, "out", "testdata", "output directory")
	flag.StringVar(&ThriftDir, "search", ".", "directory of thrift files")
	flag.Int64Var(&MaxFileNum, "max-file-num", 0, "max number of files to generate")
}

func checkArgs() {
	if ThriftDir == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func ThriftSearcher() <-chan (string) {
	out := make(chan (string))
	// read file path from stdin
	if ThriftDir == "" {
		br := bufio.NewReader(os.Stdin)
		go func() {
			str, err := br.ReadString('\n')
			for ; err == nil; str, err = br.ReadString('\n') {
				out <- str
			}
			if err == io.EOF {
				close(out)
			}
			log.Fatalln(fmt.Errorf("read from stdin failed: %w", err))
		}()
		return out
	}
	// search ThriftDir to find thrift files
	dirs := make(chan (string))
	var searchDir func(string)
	searchDir = func(dir string) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Println(fmt.Errorf("read directory %s failed: %w", dir, err))
		}
		dirs <- dir
		for _, file := range files {
			if file.IsDir() {
				searchDir(filepath.Join(ThriftDir, file.Name()))
			}
		}
	}
	go func() {
		searchDir(ThriftDir)
		close(dirs)
	}()
	go func() {
		for dir := range dirs {
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				log.Println(fmt.Errorf("read directory %s failed: %w", dir, err))
			}
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".thrift") {
					out <- filepath.Join(dir, file.Name())
				}
			}
		}
		close(out)
	}()
	return out
}

var fileCounter int64

func main() {
	flag.Parse()
	checkArgs()
	for thrift := range ThriftSearcher() {
		builder := NewStructBuilder()
		flow, err := builder.BuildThriftStruct(thrift)
		if err != nil {
			log.Println(fmt.Errorf("build struct for %s failed: %w", thrift, err))
			continue
		}
		for st := range flow {
			data := reflect.New(st.Elem()).Interface()
			// FIXME: prevent duplicate elements in sets
			err = gofakeit.Struct(data)
			if err != nil {
				log.Fatal(fmt.Errorf("fake struct %s for %s failed: %w", st.Name(), thrift, err))
			}
			buf := make([]byte, frugal.EncodedSize(data))
			length, err := frugal.EncodeObject(buf, nil, data)
			if err != nil {
				log.Fatal(fmt.Errorf("encode struct %s for %s failed: %w", st.Name(), thrift, err))
			}
			no := atomic.AddInt64(&fileCounter, 1)
			err = ioutil.WriteFile(filepath.Join(OutputDir, strconv.FormatInt(no, 10)), buf[:length], 0o644)
			if err != nil {
				log.Fatal(fmt.Errorf("write struct %s for %s failed: %w", st.Name(), thrift, err))
			}
			if no >= MaxFileNum {
				log.Println("reach max file num, stop")
				return
			}
		}
	}
}
