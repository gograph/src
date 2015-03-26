ScanLooker
==========

ScanLooker 是个回调函数, 可在解析 Go 源码的时传递扫描结果.
该函数声明为:

```go
// pos, tok, lit 来自每次 scanner.Scan 的返回值.
// 返回 false 表示撤销回调, true 表示继续回调.
func (pos token.Pos, tok token.Token, lit string) bool
```

实现
----

给 `parser` 类型增加属性, 并修改 `init`, `next0` 方法, 支持多个 ScanLooker.

给 `interface.go` 中相关导出函数添加可选参数 `opts ...interface{}`:

```go
func ParseFile(fset *token.FileSet, filename string, src interface{}, mode Mode, opts ...interface{}) (f *ast.File, err error)

func ParseExpr(x string, opts ...interface{}) (ast.Expr, error)

func ParseDir(fset *token.FileSet, path string, filter func(os.FileInfo) bool, mode Mode, opts ...interface{}) (pkgs map[string]*ast.Package, first error)
```


上述三个函数, 可选 ScanLooker 参数有两种形式:

```go
func (pos token.Pos, tok token.Token, lit string) bool
func (filename string, pos token.Pos, tok token.Token, lit string) bool
```

支持 filename 参数只是做了简单的包装. 类似这样:

```go
// support scanning looker
for i, opt := range opts {
    if x, ok := opt.(func(string, token.Pos, token.Token, string) bool); ok {
        opts[i] = func(pos token.Pos, tok token.Token, lit string) bool{
            return x(filename, pos, tok, lit)
        }
    }
}

// parse source
p.init(fset, filename, text, mode, opts)
```

