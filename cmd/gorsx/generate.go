package main

import (
	"bytes"
	"fmt"
	"github.com/go-miya/gorsx/internal/pkg"
	"go/ast"
	"go/token"
	"io"
	"log"
	"strconv"
	"strings"
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
	funcs            []*pkg.FuncInfo
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

func (g *generate) getParamsAndResults(rpcType *ast.FuncType) (*ast.FieldList, *ast.FieldList) {
	params := rpcType.Params
	param1, param2 := params.List[0], params.List[1]
	if len(param1.Names) == 0 {
		param1.Names = append(param1.Names, ast.NewIdent("c"))
	}
	if len(param2.Names) == 0 {
		param2.Names = append(param2.Names, ast.NewIdent("req"))
	}

	results := rpcType.Results
	result1, result2 := results.List[0], results.List[1]
	if len(result1.Names) == 0 {
		result1.Names = append(result1.Names, ast.NewIdent("res"))
	}
	if len(result2.Names) == 0 {
		result2.Names = append(result2.Names, ast.NewIdent("err"))
	}
	return rpcType.Params, rpcType.Results
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
	g.checkParam1MustBeContext(rpcType, methodName)
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

func (g *generate) contentImpl() []byte {
	g.printHeaderImpl()
	g.printFunctionImpl()
	g.printImports()
	g.combine()
	return g.buf.Bytes()

}

func (g *generate) printImports() {
	g.P(g.importsBuf, "import (")
	for _, imp := range g.imports {
		if !imp.Enable {
			continue
		}
		if imp.ImportPath == "" {
			continue
		}
		g.P(g.importsBuf, imp.PackageName, " ", strconv.Quote(imp.ImportPath))
	}
	g.P(g.importsBuf, ")")

}

func (g *generate) combine() {
	_, _ = io.Copy(g.buf, g.headerBuf)
	_, _ = io.Copy(g.buf, g.importsBuf)
	_, _ = io.Copy(g.buf, g.functionBuf)
}

func (g *generate) printHeaderImpl() {
	g.P(g.headerBuf, g.pkgImpl)
}

func (g *generate) printFunctionImpl() {
	typeName := g.srvName + "Controller"
	g.P(g.functionBuf, "type ", typeName, " struct {}")
	g.P(g.functionBuf)
	g.P(g.functionBuf)
	for _, info := range g.funcs {
		g.printRouterInfoImpl(typeName, info)
	}
}

func (g *generate) printRouterInfoImpl(typeName string, info *pkg.FuncInfo) {
	if info.Param2 == nil {
		return
	}
	if info.Result1 == nil {
		return
	}

	builds := []any{"func(provider *", typeName, ") ", info.FuncName, "(c ", contextPackage.Ident("Context"), ","}

	if info.Param2.Bytes {
		builds = append(builds, "req []byte")
	} else if info.Param2.String {
		builds = append(builds, "req string")
	} else if info.Param2.Reader {
		builds = append(builds, "req ", ioPackage.Ident("Reader"))
	} else if objectArgs := info.Param2.ObjectArgs; objectArgs != nil {
		paramObj := *info.Param2.ObjectArgs
		if paramObj.GoImportPath == "" {
			paramObj.GoImportPath = pkg.GoImportPath(g.pkgImportPath)
		}
		builds = append(builds, "req *", paramObj.GoImportPath.Ident(objectArgs.Name))
	} else {
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.FuncName)
	}

	builds = append(builds, ") (")

	if info.Result1.Bytes {
		builds = append(builds, "res []byte")
	} else if info.Result1.String {
		builds = append(builds, "res string")
	} else if info.Result1.Reader {
		builds = append(builds, "res ", ioPackage.Ident("Reader"))
	} else if objectArgs := info.Result1.ObjectArgs; objectArgs != nil {
		resultObj := *info.Result1.ObjectArgs
		if resultObj.GoImportPath == "" {
			resultObj.GoImportPath = pkg.GoImportPath(g.pkgImportPath)
		}
		builds = append(builds, "res *", resultObj.GoImportPath.Ident(objectArgs.Name))
	} else {
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.FuncName)
	}

	builds = append(builds, ", err error)", " {")

	g.P(g.functionBuf, builds...)
	g.P(g.functionBuf, "return")
	g.P(g.functionBuf, "}")
	g.P(g.functionBuf)
}

func (g *generate) contentImplAppend() []byte {
	g.appendFuncs()
	g.appendImports()
	buffer := bytes.NewBuffer([]byte(""))
	buffer.Write([]byte(g.pkgImpl))
	var decls []ast.Decl
	for _, decl := range g.implDeclImports {
		decls = append(decls, decl)
	}
	for _, decl := range g.implRemainDecls {
		decls = append(decls, decl)
	}
	for _, decl := range g.implDeclFuncs {
		decls = append(decls, decl)

	}
	for _, decl := range decls {
		err := pkg.AstToGo(buffer, decl)
		if err != nil {
			log.Fatal(err)
		}
	}
	return buffer.Bytes()
}

func (g *generate) appendImports() {
	for _, imp := range g.imports {
		if g.isExistImport(imp.ImportPath) {
			continue
		}
		if !imp.Enable {
			continue
		}
		if imp.ImportPath == "" {
			continue
		}
		spec := g.buildImportSpec(imp)
		if len(g.implDeclImports) == 0 {
			g.implDeclImports = append(g.implDeclImports, &ast.GenDecl{
				Tok: token.IMPORT,
			})
		}
		g.implDeclImports[0].Specs = append(g.implDeclImports[0].Specs, spec)
	}
}

func (g *generate) appendFuncs() {
	typeName := g.srvName + "Controller"
	for _, info := range g.funcs {
		if g.isExistFunc(info.FuncName) {
			continue
		}
		g.implDeclFuncs = append(g.implDeclFuncs, g.buildFuncDecl(g.pkgImportPath, typeName, info))
	}
}

func (g *generate) isExistFunc(name string) bool {
	for _, info := range g.implDeclFuncs {
		if info.Name.Name == name {
			return true
		}
	}
	return false
}

func (g *generate) isExistImport(name string) bool {
	if len(g.implDeclImports) == 0 {
		return false
	}
	for _, spec := range g.implDeclImports[0].Specs {
		importSpec, ok := spec.(*ast.ImportSpec)
		if !ok {
			continue
		}
		if strings.Trim(importSpec.Path.Value, `"`) == name {
			return true
		}
	}
	return false
}

func (g *generate) buildFuncDecl(importPath, typeName string, info *pkg.FuncInfo) *ast.FuncDecl {
	decl := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("provider")},
					Type:  &ast.StarExpr{X: &ast.Ident{Name: typeName}},
				},
			},
		},
		Name: ast.NewIdent(info.FuncName),
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ReturnStmt{},
		}},
	}

	// params
	params := info.Params.List
	param2 := params[1]
	if objectArgs := info.Param2.ObjectArgs; objectArgs != nil {
		paramObj := *info.Param2.ObjectArgs
		if paramObj.GoImportPath == "" {
			paramObj.GoImportPath = pkg.GoImportPath(importPath)
		}
		gp := paramObj.GoImportPath.Ident(paramObj.Name)
		gp.GoImport.Enable = true
		g.imports[gp.GoImport.ImportPath] = gp.GoImport
		param2.Type = &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent(gp.GoImport.PackageName), Sel: ast.NewIdent(gp.GoName)}}
	}

	// results
	results := info.Results.List
	result1 := results[0]
	if objectArgs := info.Result1.ObjectArgs; objectArgs != nil {
		resultObj := *info.Result1.ObjectArgs
		if resultObj.GoImportPath == "" {
			resultObj.GoImportPath = pkg.GoImportPath(importPath)
		}
		gp := resultObj.GoImportPath.Ident(resultObj.Name)
		gp.GoImport.Enable = true
		g.imports[gp.GoImport.ImportPath] = gp.GoImport
		result1.Type = &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent(gp.GoImport.PackageName), Sel: ast.NewIdent(gp.GoName)}}
	}
	decl.Type = &ast.FuncType{
		Params:  &ast.FieldList{List: params},
		Results: &ast.FieldList{List: results},
	}
	return decl
}

func (g *generate) buildImportSpec(info *pkg.GoImport) *ast.ImportSpec {
	return &ast.ImportSpec{
		Name: ast.NewIdent(info.PackageName),
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + info.ImportPath + `"`,
		},
	}
}
