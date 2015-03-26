// Copyright 2015 The GoGraph Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mini

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"io/ioutil"
	"testing"
)

var config = printer.Config{Mode: 0, Tabwidth: 0, Indent: 0}

func TestDiff(t *testing.T) {
	err := ExampleDiff("mini.go")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrite(t *testing.T) {
	buf, err := ExampleWrite("compare.go")
	if err != nil {
		t.Fatal(err)
	}
	err = Examine(buf)
	if err != nil {
		t.Fatal(err)
	}
}

func ExampleWrite(filename string) ([]byte, error) {
	var buf bytes.Buffer
	src, _ := ioutil.ReadFile(filename)
	m := New(&buf, 0)
	_, err := m.Write(src)
	if err == nil {
		// testing
		_, err = parser.ParseFile(m.FileSet(), "", src, parser.ParseComments)
	}

	return buf.Bytes(), err
}

func ExampleDiff(filename string) error {
	var buf bytes.Buffer
	// mini.New
	m := New(&buf, 0) // Mode(0) for "Custom Import Path"
	src, err := ioutil.ReadFile(filename)
	if err == nil {
		_, err = m.Write(src)
	}
	if err != nil {
		return err
	}

	var srcFile, miniFile *ast.File
	fset := m.FileSet()

	// ParseFile with ParseComments for "Custom Import Path"
	srcFile, err = parser.ParseFile(fset, "", src, parser.ParseComments)
	if err == nil {
		miniFile, err = parser.ParseFile(fset, "", buf.Bytes(), parser.ParseComments)
	}

	if err != nil {
		return err
	}
	// mini.Diff
	na, nb := Diff(srcFile, miniFile)

	// nil == nil
	if na != nb {
		var mini string
		if nb != nil {
			src := string(buf.Bytes())
			//fmt.Println(src) // debug

			b, e := nb.Pos()-miniFile.Pos(), nb.End()-miniFile.Pos()
			if b >= 0 && int(e) <= len(src) {
				mini = src[b:e]
			}
		}
		return fmt.Errorf("Diff: %s: %s", fset.Position(na.Pos()), mini)
	}
	return nil
}
