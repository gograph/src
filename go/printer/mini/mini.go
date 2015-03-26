// Copyright 2015 The GoGraph Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// mini 包提供 minified Go 支持, 以紧凑格式输出 Go 源码.
// 该格式为:
//
// 	单行 Go 源码, 并且无法再缩减掉任何一个间隔符.
//
// 此单行(尾部无换行符) Go 源码不包含除权威导入路径之外的注释.
package mini

import (
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"strconv"
	"strings"
)

type Mode int

const (
	OmitImportPath Mode = 1 << iota // 忽略权威导入路径; 默认包含.
)

type errorString string

func (s errorString) Error() string {
	return string(s)
}

const ErrILLEGAL = errorString("ILLEGAL") // 扫描错误 ILLEGAL.

type stringWriter interface {
	WriteString(string) (int, error)
}

// Mini 提供 minified Go 支持, 必须使用 New 创建.
type Mini struct {
	w   io.Writer
	sw  stringWriter
	n   int
	err error

	file *token.File
	fset *token.FileSet
	// 上次写入的信息
	importPath int // 处理 ImportPath 的步骤
	tok        token.Token
	ch         byte
	lit        string // 推延字符串
}

// New 新建并返回一个 Mini. 参数 w 为 minified Go 输出目标.
// 如果 w 为 nil, 使用 os.Stdout 替代.
// 参数 mode 参见 Mode 常量.
func New(w io.Writer, mode Mode) *Mini {
	if w == nil {
		w = os.Stdout
	}
	m := new(Mini)
	m.w = w
	m.sw = w.(stringWriter)
	if mode&OmitImportPath == 0 {
		m.importPath = 1
	}
	return m
}

// Write 等同于 New(w, 0).Write(src); 不负责语法检查.
func Write(w io.Writer, src []byte) (int, error) {
	return New(w, 0).Write(src)
}

// WriteString 等同于 New(w, 0).Write([]byte(src)); 不负责语法检查.
func WriteString(w io.Writer, src string) (int, error) {
	return New(w, 0).Write([]byte(src))
}

func (m *Mini) errorHandle(pos token.Position, msg string) {
	m.err = fmt.Errorf("scanner: %d: %s", pos.Line, msg)
}

// delay 处理定界符 ',' ';' 推延写入
func (m *Mini) delay(ch byte) bool {
	// "," ";" " "
	switch m.ch {
	case 0:
		m.ch = ch // 推延
		return true
	case ';':
		if m.tok == token.FOR {
			// 特别优化
			// for;;i++{}
			// for;;{} -> for{}
			if ch == ';' {
				m.ch = 0
				m.lit = ";;"
			}
			return true
		}
		// 忽略
		if ch == ' ' || ch == ';' {
			return true
		}
		// ,
		m.ch = 0
	case ',':
		return true
	default:
		panic("mini: internal error")
	}
	return m.write(ch)
}

// write 直接写入一个字节
func (m *Mini) write(ch byte) bool {
	if m.err == nil {
		_, m.err = m.w.Write([]byte{ch})
		m.n++
	}
	return m.err == nil
}

func (m *Mini) writeString(s string) bool {
	if m.err == nil {
		m.ch = 0
		n := 0
		if m.sw == nil {
			n, m.err = m.w.Write([]byte(s))
		} else {
			n, m.err = m.sw.WriteString(s)
		}
		m.n += n
	}
	return m.err == nil
}

// FileSet 返回最后一次 Write 新建的 token.FileSet.
// 它可能为 nil, 或只包含一个 token.File 的 token.FileSet.
func (m *Mini) FileSet() *token.FileSet {
	return m.fset
}

// Write 方法扫描 src, 并向输出目标写入 minified Go; 不负责语法检查.
// src 为已经格式化的 Go 源码(片段).
// Write 总是新建 token.FileSet 和命名为 "minifiedGo" 的 token.File,
// 调用者可以用 FileSet 方法得到 token.FileSet.
func (m *Mini) Write(src []byte) (int, error) {
	var s scanner.Scanner

	m.fset = token.NewFileSet()
	m.file = m.fset.AddFile("minifiedGo", -1, len(src))

	s.Init(m.file, src, m.errorHandle, 0)

	for s.ErrorCount == 0 && m.WriteToken(s.Scan()) {
	}
	if m.err == nil && s.ErrorCount != 0 {
		m.err = fmt.Errorf("Scanner.ErrorCount: %d", s.ErrorCount)
	}

	m.file = nil
	return m.n, m.err
}

// WriteString 等同于 m.Write([]byte(src)); 不负责语法检查.
func (m *Mini) WriteString(src string) (int, error) {
	return m.Write([]byte(src))
}

// WriteToken 接收扫描结果并以 minified Go 输出到目标; 不负责语法检查.
// 返回 false 表示调用者应停止调用该方法.
// 该方法应作为 ScanLooker 使用.
func (m *Mini) WriteToken(pos token.Pos, tok token.Token, lit string) bool {
	if m.err != nil || m.tok == token.EOF {
		return false
	}
	switch tok {
	case token.STRING:
		if lit[0] == '`' {
			lit = strconv.Quote(lit[0 : len(lit)-1])
		}
	case token.COMMENT:
		if m.importPath == 3 {
			m.importPath = 0
			lit = fetchImportPaths(lit)
			if lit != "" {
				m.writeString(lit)
			}
		}
		return true
	case token.COMMA:
		// 推延写入
		return m.delay(',')
	case token.SEMICOLON:
		// 权威导入路径必须是 package name 的行尾注释.
		if m.importPath == 3 {
			m.importPath = 0
		}

		return m.delay(';')

	case token.CASE:
		if m.tok.IsKeyword() {
			// 	break/return/fallthrough
			// case
			// 直接写入
			m.write(';')
			m.ch = 0
		}
	case token.MUL:
		if m.tok == token.QUO {
			// src/cmd/pprof/internal/driver/driver.go : Ratio: 1 / *f.flagDivideBy
			// 直接写入
			m.write(' ')
			m.ch = 0
		}
	case token.EOF:
		m.tok = tok
		return false
	case token.IMPORT:
		// 权威导入路径
		if m.importPath == 1 {
			m.importPath = 2
		}
		break
	case token.ILLEGAL:
		m.tok = tok // reset
		m.err = ErrILLEGAL
		return false

	default:
		// ])} 之前无需分隔符
		if isR(tok) {
			m.ch = 0
			break
		}
		// 权威导入路径
		if m.importPath == 2 && tok == token.IDENT {
			m.importPath = 3
		}

		if m.tok.IsKeyword() {
			if !tok.IsOperator() {
				if m.lit == "" {
					m.write(' ')
					m.ch = 0
				}
				break
			}
		}

		if m.tok.IsLiteral() {
			// var a int; var a map[key]{}; var a[]int
			if isL(tok) {
				break
			}
			if m.ch == ';' {
				break
			}
			if !tok.IsOperator() {
				// 直接写入
				if m.ch == 0 {
					m.write(' ')
				}
				break
			}
		}
	}

	if m.lit != "" {
		if m.tok == token.FOR {
			m.writeString(";;")
			m.lit = ""
		}
	} else if m.ch != 0 {
		m.write(m.ch)
	}

	m.tok = tok

	// https://github.com/golang/go/issues/10213
	if tok.IsOperator() {
		lit = tok.String()
	}
	return m.writeString(lit)
}

func isL(tok token.Token) bool {
	return tok >= token.LPAREN && tok <= token.LBRACE
}

func isR(tok token.Token) bool {
	return tok >= token.RPAREN && tok <= token.RBRACE
}

func fetchImportPaths(lit string) string {
	const imp = "import "
	if len(lit) < 2 {
		return ""
	}

	if lit[1] == '*' {
		lit = lit[2 : len(lit)-2]
	} else {
		lit = lit[2:]
	}
	lit = strings.Trim(lit, " ")
	if !strings.HasPrefix(lit, imp) {
		return ""
	}
	lit = strings.TrimLeft(lit[7:], " ")
	if len(lit) < 3 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		return ""
	}
	return imp + lit
}

func Examine(src []byte) error {
	var (
		info      string
		s         scanner.Scanner
		pos, end  token.Pos
		tok, pre  token.Token
		lit, txt  string
		offset    token.Pos
		isForSemi int
	)
	size := len(src)
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, size)

	s.Init(file, src, func(pos token.Position, msg string) {
		info = "error handing " + msg
	}, scanner.ScanComments)

	i := 0
	for {
		pos, tok, lit = s.Scan()
		i++
		if tok == token.EOF {
			return nil
		}
		// Scan 总是在 EOF 之前送出一个 SEMICOLON "\n"
		if int(pos) > size {
			continue
		}

		offset = pos - end

		if tok == token.ILLEGAL || lit == "\n" || offset > 1 {
			break
		}

		if i == 4 && tok == token.COMMENT {
			if fetchImportPaths(lit) == "" {
				break
			}
		}

		if tok == token.COMMENT {
			break
		}

		if offset == 1 {
			// 空格两边至少有一个关键字或者两边都是 IDENT
			// 或者 /* --> / *
			if !(pre.IsKeyword() || tok.IsKeyword() ||
				(pre == token.IDENT && tok == token.IDENT) ||
				pre == token.QUO && tok == token.MUL) {
				break
			}
			// for;;{} -> for{}
			// 总结: for 后跟空格的话, 必定不能是 ";" 或者 "{"
			if pre == token.FOR {
				if tok == token.SEMICOLON || tok == token.LBRACE {
					break
				}
			}
			// "{" "(" "[" 之前无需空格
			// select{},return(*p)(a),return[]int{1}
			if isL(tok) {
				break
			}
		}
		if offset == 0 {
			// /* --> / *
			if pre == token.QUO && tok == token.MUL {
				break
			}
			if isForSemi == 2 {
				// 不应该出现: for;;{}
				// 期望: for;;i++{}
				if tok == token.LBRACE {
					break
				}
				isForSemi = 0
			}

			// 连续的 "," ";" 和未精简的 ",]" ";}"
			if pre == token.SEMICOLON || pre == token.COMMA {
				if tok == pre || isR(tok) {
					// 期望: for;;i++{}
					if tok == token.SEMICOLON && isForSemi == 1 {
						isForSemi = 2
					} else {
						break
					}
				}
			}

			// for;;i++{}
			if pre == token.FOR && tok == token.SEMICOLON {
				isForSemi = 1
			}
		}

		if tok.IsOperator() {
			lit = tok.String()
		}
		end = pos + token.Pos(len(lit))
		pre = tok
		txt = lit
	}

	if info == "" {
		if offset == 1 {
			info = "%s -> %#v %#v"
		} else {
			info = "%s -> %#v%#v"
		}
		info = fmt.Sprintf(info, file.Position(pos), txt, lit)
	}
	return errorString(info)
}
