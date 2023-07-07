package main

import (
	"bytes"
	"fmt"
	"github.com/go-miya/gorsx/internal/pkg"
	"go/ast"
	"io"
	"log"
)

const (
	contextPackage = pkg.GoImportPath("context")
	ioPackage      = pkg.GoImportPath("io")
)

type generate struct {
	buf              *bytes.Buffer
	headerBuf        *bytes.Buffer
	importsBuf       *bytes.Buffer
	functionBuf      *bytes.Buffer
	header           string
	pkg              string
	pkgImpl          string
	pkgImportPath    string // 源文件
	implDeclFuncs    []*ast.FuncDecl
	implDeclImports  []*ast.GenDecl
	implRemainDecls  []ast.Decl
	imports          map[string]*pkg.GoImport
	srvName          string
	usedPackageNames map[string]bool
	routerInfos      []*pkg.RouterInfo
}

func (g *generate) checkResult2MustBeError(rpcType *ast.FuncType, methodName *ast.Ident) {
	result2 := rpcType.Results.List[1]
	result2Iden, ok := result2.Type.(*ast.Ident)
	if !ok {
		log.Fatalf("error: func %s 2th result is not error", methodName)
	}
	if result2Iden.Name != "error" {
		log.Fatalf("error: func %s 2th result is not error", methodName)
	}
}

func (g *generate) checkAndGetResult1(rpcType *ast.FuncType, methodName *ast.Ident) *pkg.Result {
	result1 := rpcType.Results.List[0]
	switch r1 := result1.Type.(type) {
	case *ast.ArrayType:
		ident, ok := r1.Elt.(*ast.Ident)
		if !ok {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if ident.Name != "byte" {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &pkg.Result{Bytes: true}
	case *ast.Ident:
		if r1.Name != "string" {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &pkg.Result{String: true}
	case *ast.StarExpr:
		switch x := r1.X.(type) {
		case *ast.Ident:
			name := x.Name
			return &pkg.Result{ObjectArgs: &pkg.ObjectArgs{Name: name}}
		case *ast.SelectorExpr:
			ident, ok := x.X.(*ast.Ident)
			if !ok {
				log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			}
			for importPath, goImport := range g.imports {
				if goImport.PackageName == ident.Name {
					return &pkg.Result{ObjectArgs: &pkg.ObjectArgs{Name: x.Sel.Name, GoImportPath: pkg.GoImportPath(importPath)}}
				}
			}
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			return nil
		default:
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			return nil
		}
	case *ast.SelectorExpr:
		if r1.Sel == nil {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if r1.Sel.Name != "Reader" {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		ident, ok := r1.X.(*ast.Ident)
		if !ok {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		ioImport, ok := g.imports["io"]
		if !ok {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if ioImport.PackageName != ident.Name {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &pkg.Result{Reader: true}
	default:

	}
	return nil
}

func (g *generate) checkResults(rpcType *ast.FuncType, methodName *ast.Ident) {
	if rpcType.Results == nil {
		log.Fatalf("error: func %s results is empty", methodName)
	}
	if len(rpcType.Results.List) != 2 {
		log.Fatalf("error: func %s results count is not equal 2", methodName)
	}
}

func (g *generate) checkAndGetParam2(rpcType *ast.FuncType, methodName *ast.Ident) *pkg.Param {
	param2 := rpcType.Params.List[1]
	switch p2 := param2.Type.(type) {
	case *ast.ArrayType:
		ident, ok := p2.Elt.(*ast.Ident)
		if !ok {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if ident.Name != "byte" {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &pkg.Param{Bytes: true}
	case *ast.Ident:
		if p2.Name != "string" {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &pkg.Param{String: true}
	case *ast.StarExpr:
		switch x := p2.X.(type) {
		case *ast.Ident:
			name := x.Name
			return &pkg.Param{ObjectArgs: &pkg.ObjectArgs{Name: name}}
		case *ast.SelectorExpr:
			ident, ok := x.X.(*ast.Ident)
			if !ok {
				log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			}
			for importPath, goImport := range g.imports {
				if goImport.PackageName == ident.Name {
					return &pkg.Param{ObjectArgs: &pkg.ObjectArgs{Name: x.Sel.Name, GoImportPath: pkg.GoImportPath(importPath)}}
				}
			}
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			return nil
		default:
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			return nil
		}

	case *ast.SelectorExpr:
		if p2.Sel == nil {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if p2.Sel.Name != "Reader" {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		ident, ok := p2.X.(*ast.Ident)
		if !ok {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		ioImport, ok := g.imports["io"]
		if !ok {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if ioImport.PackageName != ident.Name {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &pkg.Param{Reader: true}
	default:
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		return nil
	}
}

func (g *generate) checkParam1MustBeContext(rpcType *ast.FuncType, methodName *ast.Ident) {
	param1 := rpcType.Params.List[0]
	param0SelectorExpr, ok := param1.Type.(*ast.SelectorExpr)
	if !ok {
		log.Fatalf("error: func %s 1th param is not context.Context", methodName)
	}
	if param0SelectorExpr.Sel.Name != "Context" {
		log.Fatalf("error: func %s 1th param is not context.Context", methodName)
	}
	param0SelectorExprX, ok := param0SelectorExpr.X.(*ast.Ident)
	if !ok {
		log.Fatalf("error: func %s 1th param is not context.Context", methodName)
	}
	if param0SelectorExprX.Name != "context" {
		log.Fatalf("error: func %s 1th param is not context.Context", methodName)
	}
}

func (g *generate) checkParams(rpcType *ast.FuncType, methodName *ast.Ident) {
	if rpcType.Params == nil {
		log.Fatalf("error: func %s params is empty", methodName)
	}
	if len(rpcType.Params.List) != 2 {
		log.Fatalf("error: func %s params count is not equal 2", methodName)
	}
}

func (g *generate) P(w io.Writer, v ...any) {
	for _, x := range v {
		switch x := x.(type) {
		case *pkg.GoIdent:
			x.GoImport.Enable = true
			g.imports[x.GoImport.ImportPath] = x.GoImport
			_, _ = fmt.Fprint(w, x.Qualify())
		default:
			_, _ = fmt.Fprint(w, x)
		}
	}
	_, _ = fmt.Fprintln(w)
}

func (g *generate) Reset() {
	g.buf.Reset()
	g.headerBuf.Reset()
	g.importsBuf.Reset()
	g.functionBuf.Reset()
}
