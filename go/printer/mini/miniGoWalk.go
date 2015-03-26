// Copyright 2015 The GoGraph Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// 遍历并 minified 给定目录(缺省 go/src) 所有非 testdata 源码. 编译:
//
// 	go build miniGoWalk.go
//
// 使用: miniGoWalk [directory]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "."
)

func main() {
	var i int
	var root, name string

	flag.Parse()
	root = flag.Arg(0)
	if root == "" {
		root = runtime.GOROOT() + string(os.PathSeparator) + "src"
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		src, err := miniWrite(path)
		// 原来就有错
		if src == nil {
			return nil
		}
		i++
		if err == nil {
			// 检查是否最精简
			err = Examine(src)
		}
		if err != nil {
			name = path
		}

		return err
	})

	if err != nil {
		fmt.Println(name, err.Error())
		return
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	fmt.Println("Great, minified Go files", i, "from", root)
	fmt.Println("     Alloc", mem.Alloc)
	fmt.Println("TotalAlloc", mem.TotalAlloc)
}

func miniWrite(filename string) ([]byte, error) {
	var buf bytes.Buffer
	src, _ := ioutil.ReadFile(filename)
	m := New(&buf, 0)
	_, err := m.Write(src)
	_, err0 := parser.ParseFile(m.FileSet(), "", src, parser.ParseComments)

	// 原来就有错误
	if err0 != nil {
		return nil, nil
	}
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}
