// Copyright 2015 The GoGraph Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mini

import (
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
)

var closeNode = &ast.BadExpr{}

func goInspect(node ast.Node) chan ast.Node {
	ch := make(chan ast.Node)
	deep := 0
	go ast.Inspect(node, func(n ast.Node) bool {
		if deep < 0 {
			return false
		}
		// 保证只发送一次 nil, 表示遍历结束.
		if n != nil {
			deep++
		} else {
			deep--
			if deep != 0 {
				return true
			}
		}

		if _, ok := n.(*ast.CommentGroup); ok {
			return true
		}
		if _, ok := n.(*ast.Comment); ok {
			return true
		}

		ch <- n
		if closeNode == <-ch || deep == 0 {
			deep = -1
			close(ch)
		}
		return true
	})

	return ch
}

// Diff 遍历两个 AST 节点 a, b, 比较权威导入路径 (Custom Import Path)
// 和其它非注释节点, 相同返回 nil, nil, 否则返回一对儿不同的子节点.
//
// NOTE(yhc): 算法缺陷:
// 	如果两个参数都不是 minified Go 节点,
// 	比较权威导入路径的算法可能会误判. 例如下列极端情况:
// 		package name
// 		// import "name"
// 	如果该情况出现, 那么上段代码因为有换行, 所以未定义权威导入路径,
// 	分析换行需要引入 token.File 或者 token.FileSet. Diff 未引入它们,
// 	所以会误判为定义了权威导入路径. 经格式化的代码不会出现此情况,
// 	Diff 精简了参数, 忽略了此情况.
//
func Diff(a, b ast.Node) (na, nb ast.Node) {
	if a == nil || b == nil {
		return a, b
	}

	ca := goInspect(a)
	cb := goInspect(b)
	for {
		na = <-ca
		nb = <-cb

		if na == nb { // 相同
			ca <- closeNode
			cb <- closeNode
			return nil, nil
		}

		if !compare(na, nb) {
			ca <- closeNode
			cb <- closeNode
			break
		}
		ca <- nil
		cb <- nil
	}
	return
}

// Compare 遍历比较 a, b 是否相同(除注释之外).
// 事实上只是简单调用了 Diff:
//
// 	a, b = Diff(a, b)
// 	return a == b
//
func Compare(a, b ast.Node) bool {
	a, b = Diff(a, b)
	return a == b
}

// compare 要求 a, b 不能为 nil
func compare(a, b ast.Node) bool {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	switch x := a.(type) {
	case *ast.AssignStmt:
		y := b.(*ast.AssignStmt)
		return x.Tok == y.Tok
	case *ast.BasicLit:
		y := b.(*ast.BasicLit)
		if x.Kind != y.Kind {
			return false
		}
		if x.Value == y.Value {
			return true
		}
		if x.Kind != token.STRING || x.Value[0] == y.Value[0] {
			return false
		}
		// `string`
		s1, s2 := x.Value, y.Value
		if s1[0] == '`' {
			s1 = strconv.Quote(s1[1 : len(s1)-1])
		} else {
			s2 = strconv.Quote(s2[1 : len(s2)-1])
		}
		return s1 == s2

	case *ast.BinaryExpr:
		y := b.(*ast.BinaryExpr)
		return x.Op == y.Op

	case *ast.BranchStmt:
		y := b.(*ast.BranchStmt)
		return x.Tok == y.Tok

	case *ast.CallExpr:
		y := b.(*ast.CallExpr)
		return x.Ellipsis == y.Ellipsis

	case *ast.ChanType:
		y := b.(*ast.ChanType)
		return x.Dir == y.Dir

	case *ast.File:
		y := b.(*ast.File)
		xpos := x.Name.End()
		ypos := y.Name.End()

		xc := findComment(x.Comments, xpos)
		yc := findComment(y.Comments, ypos)

		if xc == nil && yc == nil {
			return true
		}

		var xpath, ypath string
		var xoffset, yoffset token.Pos = 10, 10

		if xc != nil {
			xoffset = xc.Pos() - xpos
			xpath = fetchImportPaths(xc.Text)
		}
		if yc != nil {
			yoffset = yc.Pos() - ypos
			ypath = fetchImportPaths(yc.Text)
		}

		if xpath != ypath {
			return false
		}
		// none "Custom Import Path"
		if xpath == "" {
			return true
		}

		return xoffset < 2 && yoffset < 2 && xpath == ypath

	case *ast.GenDecl:
		y := b.(*ast.GenDecl)
		return x.Tok == y.Tok

	case *ast.Ident:
		y := b.(*ast.Ident)
		return x.Name == y.Name

	case *ast.IncDecStmt:
		y := b.(*ast.IncDecStmt)
		return x.Tok == y.Tok

	case *ast.InterfaceType:
		y := b.(*ast.InterfaceType)
		return !(x.Incomplete || y.Incomplete)

	case *ast.Package:
		y := b.(*ast.Package)
		return x.Name == y.Name

	case *ast.RangeStmt:
		y := b.(*ast.RangeStmt)
		return x.Tok == y.Tok

	case *ast.StructType:
		y := b.(*ast.StructType)
		return !(x.Incomplete || y.Incomplete)

	case *ast.UnaryExpr:
		y := b.(*ast.UnaryExpr)
		return x.Op == y.Op
	}
	return true
}

// findComment 在 comments 中查找第一个 comment.Pos>=pos 注释
func findComment(comments []*ast.CommentGroup, pos token.Pos) (c *ast.Comment) {
	for _, group := range comments {
		if group.End() < pos {
			continue
		}
		for _, comment := range group.List {
			if comment.Pos() >= pos {
				return comment
			}
		}
	}
	return nil
}
