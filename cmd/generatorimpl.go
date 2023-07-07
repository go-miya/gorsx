package main

import (
	"bytes"
	"github.com/go-miya/gorsx/internal/pkg"
	"go/ast"
	"go/token"
	"io"
	"log"
	"strconv"
	"strings"
)

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
	for _, info := range g.routerInfos {
		g.printRouterInfoImpl(typeName, info)
	}
}

func (g *generate) printRouterInfoImpl(typeName string, info *pkg.RouterInfo) {
	if info.Param2 == nil {
		return
	}
	if info.Result1 == nil {
		return
	}

	builds := []any{"func(provider *", typeName, ") ", info.RpcMethodName, "(c ", contextPackage.Ident("Context"), ","}

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
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.RpcMethodName)
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
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.RpcMethodName)
	}

	builds = append(builds, ", err error)", " {")

	g.P(g.functionBuf, builds...)
	g.P(g.functionBuf, "// code your logic here")
	g.P(g.functionBuf, "return")
	g.P(g.functionBuf, "}")
	g.P(g.functionBuf)
}

func (g *generate) contentImplAppend() []byte {
	g.appendFuncs()
	g.appendImports()
	buffer := bytes.NewBuffer([]byte(""))
	buffer.Write([]byte(g.pkgImpl))
	for _, decl := range g.implDeclImports {
		err := pkg.AstToGo(buffer, decl)
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, decl := range g.implRemainDecls {
		err := pkg.AstToGo(buffer, decl)
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, decl := range g.implDeclFuncs {
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
	for _, info := range g.routerInfos {
		if g.isExistFunc(info.RpcMethodName) {
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

func (g *generate) buildFuncDecl(importPath, typeName string, info *pkg.RouterInfo) *ast.FuncDecl {
	decl := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("provider")},
					Type:  &ast.StarExpr{X: &ast.Ident{Name: typeName}},
				},
			},
		},
		Name: ast.NewIdent(info.RpcMethodName),
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ReturnStmt{},
		}},
	}

	// params
	var params []*ast.Field
	params = append(params, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("c")},
		Type:  &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")},
	})
	param2 := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("req")},
	}
	if objectArgs := info.Param2.ObjectArgs; objectArgs != nil {
		paramObj := *info.Param2.ObjectArgs
		if paramObj.GoImportPath == "" {
			paramObj.GoImportPath = pkg.GoImportPath(importPath)
		}
		gp := paramObj.GoImportPath.Ident(paramObj.Name)
		gp.GoImport.Enable = true
		g.imports[gp.GoImport.ImportPath] = gp.GoImport
		param2.Type = &ast.SelectorExpr{X: ast.NewIdent(gp.GoImport.PackageName), Sel: ast.NewIdent(gp.GoName)}
	} else {
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.RpcMethodName)
	}
	params = append(params, param2)

	// results
	var results []*ast.Field
	result1 := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("res")},
	}
	if objectArgs := info.Result1.ObjectArgs; objectArgs != nil {
		resultObj := *info.Param2.ObjectArgs
		if resultObj.GoImportPath == "" {
			resultObj.GoImportPath = pkg.GoImportPath(importPath)
		}
		gp := resultObj.GoImportPath.Ident(resultObj.Name)
		gp.GoImport.Enable = true
		g.imports[gp.GoImport.ImportPath] = gp.GoImport
		result1.Type = &ast.SelectorExpr{X: ast.NewIdent(gp.GoImport.PackageName), Sel: ast.NewIdent(gp.GoName)}
	} else {
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.RpcMethodName)
	}
	results = append(results, result1)
	results = append(results, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("err")},
		Type:  ast.NewIdent("error"),
	})

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
