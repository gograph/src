// Copyright 2015 The GoGraph Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"go/token"
	"testing"
)

func Test_ScanLooker(t *testing.T) {
	var f1, f2, f3 int

	for _, filename := range validFiles {
		var s1 int
		_, err := ParseFile(fset, filename, nil, 0,
			func(token.Pos, token.Token, string) bool {
				f1++
				s1++
				return s1 != 100
			})
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", filename, err)
		}
	}

	if f1 != len(validFiles)*100 {
		t.Fatalf("ScanLooker f1: %v", f1)
	}

	for _, filename := range validFiles {
		var s2 int
		_, err := ParseFile(fset, filename, nil, 0,
			func(token.Pos, token.Token, string) bool {
				s2++
				f2++
				return s2 != 200
			}, func(name string, _ token.Pos, _ token.Token, _ string) bool {
				if f3 < 0 {
					return false
				}
				if name != filename {
					f3 = -1
					return false
				}
				f3++
				return true
			})
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", filename, err)
		}
	}

	if f2 != len(validFiles)*200 {
		t.Fatalf("ScanLooker f2: %v", f2)
	}

	if f3 < 0 {
		t.Fatalf("ScanLooker f3: %v", f3)
	}
}
