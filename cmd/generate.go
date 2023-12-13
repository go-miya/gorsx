package cmd

import (
	"bytes"
	"fmt"
	"github.com/go-miya/gorsx/internal"
	"github.com/samber/lo"
	"go/ast"
	"go/format"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	contextPackage = internal.GoImportPath("context")
	ioPackage      = internal.GoImportPath("io")
)

type Generate struct {
	Buf              *bytes.Buffer
	HeaderBuf        *bytes.Buffer
	ImportsBuf       *bytes.Buffer
	FunctionBuf      *bytes.Buffer
	pkgImpl          string
	pkgAssembler     string
	pkgBus           string
	pkgImportPath    string // 源文件
	implDeclFuncs    []*ast.FuncDecl
	implDeclImports  []*ast.GenDecl
	implRemainDecls  []ast.Decl
	Imports          map[string]*internal.GoImport
	SrvName          string
	SrvTypeShort     string
	UsedPackageNames map[string]bool
	Funcs            []*internal.FuncInfo
	CQRSList         CQRSList
}

type CQRSList []*internal.CQRSFile

func (l CQRSList) GetQueries() []*internal.CQRSFile {
	return lo.Filter(l, func(x *internal.CQRSFile, index int) bool {
		return x.IsQuery()
	})
}

func (l CQRSList) GetCommands() []*internal.CQRSFile {
	return lo.Filter(l, func(x *internal.CQRSFile, index int) bool {
		return x.IsCommand()
	})
}

func (g *Generate) Generate(outDir, pkgPath, ImplPath string, carsPath *internal.Path) {
	g.generateServiceImpl(outDir, pkgPath, ImplPath)
	g.generateAssembler(outDir, pkgPath, carsPath)
	g.generateBus(outDir, pkgPath, carsPath, true)
	g.generateBus(outDir, pkgPath, carsPath, false)
}

// 暂时还未解决bus的增量补充问题，待解决
func (g *Generate) GenerateProto(outDir, pkgPath, ImplPath string, carsPath *internal.Path) {
	g.generateServiceImpl(outDir, pkgPath, ImplPath)
	g.generateAssembler(outDir, pkgPath, carsPath)
}

func (g *Generate) generateServiceImpl(outDir, pkgPath, ImplPath string) {
	// gen service impl
	implOutputPath := filepath.Join(outDir, ImplPath, fmt.Sprintf("%s.go", strings.ToLower(g.SrvName)))
	g.pkgImportPath = pkgPath
	_, g.pkgImpl = filepath.Split(ImplPath)
	g.pkgImpl = fmt.Sprintf("package %s", g.pkgImpl)

	var content []byte
	if _, err := os.Stat(implOutputPath); err != nil {
		content = g.contentImpl()
	} else {
		astFile, err := internal.ParserGoFile(implOutputPath)
		if err != nil {
			log.Fatalf("generateServiceImpl.ParserGoFile failed, %v", err)
		}
		g.implDeclImports, g.implRemainDecls, g.implDeclFuncs = internal.InspectAstFile(astFile)
		content = g.contentImplAppend()
	}

	// Format the output.
	src, err := format.Source(content)
	if err != nil {
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		src = content
	}
	if err := writeContent(implOutputPath, src); err != nil {
		log.Fatalf("writing output: %s", err)
	}
	log.Printf("%s.%s wrote impl %s", pkgPath, g.SrvName, implOutputPath)
}

func (g *Generate) generateBus(outDir, pkgPath string, cqrsPath *internal.Path, isQuery bool) {
	if isQuery && len(g.CQRSList.GetQueries()) == 0 {
		return
	}
	if !isQuery && len(g.CQRSList.GetCommands()) == 0 {
		return
	}
	var content []byte
	g.Reset()
	path := cqrsPath.BusQuery
	if !isQuery {
		path = cqrsPath.BusCommand
	}
	tarFilePath := filepath.Join(outDir, path)
	g.pkgBus = fmt.Sprintf("package %s", filepath.Base(filepath.Dir(tarFilePath)))
	if _, err := os.Stat(tarFilePath); err != nil {
		content = g.contentBus(nil, isQuery)
	} else {
		busQueryFile, err := internal.ParserGoFile(tarFilePath)
		if err != nil {
			log.Fatalf("generateBus.ParserGoFile failed, %v", err)
		}
		content = g.contentBus(busQueryFile, isQuery)
	}
	if content == nil {
		return
	}
	src, err := format.Source(content)
	if err != nil {
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		src = content
	}
	err = writeContent(tarFilePath, src)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
	log.Printf("%s.%s wrote cqrs %s", pkgPath, g.SrvName, tarFilePath)
}

func (g *Generate) generateAssembler(outDir, pkgPath string, cqrsPath *internal.Path) {
	if len(g.CQRSList) == 0 {
		return
	}
	var content []byte
	g.Reset()
	assemblerOPath := filepath.Join(outDir, cqrsPath.AssemblerPath, fmt.Sprintf("%s.go", strings.ToLower(g.SrvName)))
	_, g.pkgAssembler = filepath.Split(cqrsPath.AssemblerPath)
	g.pkgAssembler = fmt.Sprintf("package %s", g.pkgAssembler)
	isAppend := false
	if _, err := os.Stat(assemblerOPath); err != nil {
		content = g.contentAssembler(isAppend)
	} else {
		isAppend = true
		astFile, err := internal.ParserGoFile(assemblerOPath)
		if err != nil {
			log.Fatalf("generateAssembler.ParserGoFile failed, %v", err)
		}
		g.implDeclImports, g.implRemainDecls, g.implDeclFuncs = internal.InspectAstFile(astFile)
		content = g.contentAssembler(isAppend)
	}
	src, err := format.Source(content)
	if err != nil {
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		src = content
	}
	if !isAppend {
		err = writeContent(assemblerOPath, src)
	} else {
		err = appendContent(assemblerOPath, src)
	}
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
	log.Printf("%s.%s wrote assembler %s", pkgPath, g.SrvName, assemblerOPath)
}

func writeContent(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}

func appendContent(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(content)
	if err != nil {
		return err
	}
	return f.Close()
}

func (g *Generate) checkResult2MustBeError(rpcType *ast.FuncType, methodName *ast.Ident) {
	result2 := rpcType.Results.List[1]
	result2Iden, ok := result2.Type.(*ast.Ident)
	if !ok {
		log.Fatalf("error: func %s 2th result is not error", methodName)
	}
	if result2Iden.Name != "error" {
		log.Fatalf("error: func %s 2th result is not error", methodName)
	}
}

func (g *Generate) CheckAndGetResult1(rpcType *ast.FuncType, methodName *ast.Ident) *internal.Result {
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
		return &internal.Result{Bytes: true}
	case *ast.Ident:
		if r1.Name != "string" {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &internal.Result{String: true}
	case *ast.StarExpr:
		switch x := r1.X.(type) {
		case *ast.Ident:
			name := x.Name
			return &internal.Result{ObjectArgs: &internal.ObjectArgs{Name: name}}
		case *ast.SelectorExpr:
			ident, ok := x.X.(*ast.Ident)
			if !ok {
				log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			}
			for importPath, goImport := range g.Imports {
				if goImport.PackageName == ident.Name {
					return &internal.Result{ObjectArgs: &internal.ObjectArgs{Name: x.Sel.Name, GoImportPath: internal.GoImportPath(importPath)}}
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
		ioImport, ok := g.Imports["io"]
		if !ok {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if ioImport.PackageName != ident.Name {
			log.Fatalf("error: func %s 1th result is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &internal.Result{Reader: true}
	default:

	}
	return nil
}

func (g *Generate) getParamsAndResults(rpcType *ast.FuncType) (*ast.FieldList, *ast.FieldList) {
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

func (g *Generate) CheckAndGetParam2(rpcType *ast.FuncType, methodName *ast.Ident) *internal.Param {
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
		return &internal.Param{Bytes: true}
	case *ast.Ident:
		if p2.Name != "string" {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &internal.Param{String: true}
	case *ast.StarExpr:
		switch x := p2.X.(type) {
		case *ast.Ident:
			name := x.Name
			return &internal.Param{ObjectArgs: &internal.ObjectArgs{Name: name}}
		case *ast.SelectorExpr:
			ident, ok := x.X.(*ast.Ident)
			if !ok {
				log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
			}
			for importPath, goImport := range g.Imports {
				if goImport.PackageName == ident.Name {
					return &internal.Param{ObjectArgs: &internal.ObjectArgs{Name: x.Sel.Name, GoImportPath: internal.GoImportPath(importPath)}}
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
		ioImport, ok := g.Imports["io"]
		if !ok {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		if ioImport.PackageName != ident.Name {
			log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		}
		return &internal.Param{Reader: true}
	default:
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or io.Reader or *struct{}", methodName)
		return nil
	}
}

func (g *Generate) checkParam1MustBeContext(rpcType *ast.FuncType, methodName *ast.Ident) {
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

func (g *Generate) checkParams(rpcType *ast.FuncType, methodName *ast.Ident) {
	if rpcType.Params == nil {
		log.Fatalf("error: func %s params is empty", methodName)
	}
	if len(rpcType.Params.List) != 2 {
		log.Fatalf("error: func %s params count is not equal 2", methodName)
	}
	g.checkParam1MustBeContext(rpcType, methodName)
}

func (g *Generate) P(w io.Writer, v ...any) {
	for _, x := range v {
		switch x := x.(type) {
		case *internal.GoIdent:
			x.GoImport.Enable = true
			g.Imports[x.GoImport.ImportPath] = x.GoImport
			_, _ = fmt.Fprint(w, x.Qualify())
		default:
			_, _ = fmt.Fprint(w, x)
		}
	}
	_, _ = fmt.Fprintln(w)
}

func (g *Generate) Reset() {
	g.Buf.Reset()
	g.HeaderBuf.Reset()
	g.ImportsBuf.Reset()
	g.FunctionBuf.Reset()
}

func (g *Generate) contentImpl() []byte {
	g.printHeaderImpl()
	g.printFunctionImpl()
	g.printImports()
	g.combine()
	return g.Buf.Bytes()

}

func (g *Generate) contentImplAppend() []byte {
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
		err := internal.AstToGo(buffer, decl)
		if err != nil {
			log.Fatal(err)
		}
	}
	_, _ = io.Copy(buffer, g.FunctionBuf)
	return buffer.Bytes()
}

func (g *Generate) contentAssembler(isAppend bool) []byte {
	if !isAppend {
		g.P(g.HeaderBuf, g.pkgAssembler)
	}
	g.printAssemblerFunc()
	g.combine()
	return g.Buf.Bytes()
}

func (g *Generate) contentBus(existFile *ast.File, isQuery bool) []byte {
	var cqrsList []*internal.CQRSFile
	var tp string
	if isQuery {
		cqrsList = g.CQRSList.GetQueries()
		tp = "Queries"
	} else {
		cqrsList = g.CQRSList.GetCommands()
		tp = "Commands"
	}
	if len(cqrsList) == 0 {
		return nil
	}

	g.P(g.HeaderBuf, g.pkgBus)

	importSpecs, busFields := internal.InspectBus(existFile, tp)
	if len(importSpecs) != 0 {
		g.P(g.ImportsBuf, "import (")
		for _, spec := range importSpecs {
			var name string
			if spec.Name != nil {
				name = spec.Name.Name + " "
			}
			g.P(g.ImportsBuf, name, spec.Path.Value)
		}
		g.P(g.ImportsBuf, ")")
	}

	g.P(g.FunctionBuf, fmt.Sprintf(`type %s struct {`, tp))
	existName := make(map[string]struct{})
	if len(busFields) != 0 {
		for _, field := range busFields {
			ft := field.Type.(*ast.SelectorExpr)
			g.P(g.FunctionBuf, field.Names[0].Name, " ", ft.X.(*ast.Ident).Name, ".", ft.Sel.Name)
			existName[field.Names[0].Name] = struct{}{}
		}
	}

	for _, file := range cqrsList {
		if _, ok := existName[file.Endpoint]; ok {
			continue
		}
		g.P(g.FunctionBuf, fmt.Sprintf("%s %s.%s", file.Endpoint, g.CQRSList[0].Package, file.Endpoint))
	}
	g.P(g.FunctionBuf, `}`)
	g.combine()
	return g.Buf.Bytes()
}

func (g *Generate) printAssemblerFunc() {
	for _, info := range g.Funcs {
		if info.Assembler == nil {
			continue
		}
		if g.isExistFunc(info.Assembler.GetFuncNameTo()) || g.isExistFunc(info.Assembler.GetFuncNameFrom()) {
			continue
		}
		if info.Assembler.ToParamsIdent.ObjectArgs.GoImportPath == "" {
			info.Assembler.ToParamsIdent.ObjectArgs.GoImportPath = internal.GoImportPath(g.pkgImportPath)
		}
		if info.Assembler.FromResultIdent.ObjectArgs.GoImportPath == "" {
			info.Assembler.FromResultIdent.ObjectArgs.GoImportPath = internal.GoImportPath(g.pkgImportPath)
		}
		g.P(g.FunctionBuf, info.Assembler.Gen())
	}
}

func (g *Generate) printImports() {
	g.P(g.ImportsBuf, "import (")
	for _, imp := range g.Imports {
		if !imp.Enable {
			continue
		}
		if imp.ImportPath == "" {
			continue
		}
		g.P(g.ImportsBuf, imp.PackageName, " ", strconv.Quote(imp.ImportPath))
	}
	g.P(g.ImportsBuf, ")")

}

func (g *Generate) combine() {
	_, _ = io.Copy(g.Buf, g.HeaderBuf)
	_, _ = io.Copy(g.Buf, g.ImportsBuf)
	_, _ = io.Copy(g.Buf, g.FunctionBuf)
}

func (g *Generate) printHeaderImpl() {
	g.P(g.HeaderBuf, g.pkgImpl)
}

func (g *Generate) printFunctionImpl() {
	typeName := buildTypeName(g.SrvName)
	g.P(g.FunctionBuf, "type ", typeName, " struct", `{
	queries    *bus.Queries
	commands   *bus.Commands
}`)
	g.P(g.FunctionBuf)
	g.P(g.FunctionBuf)
	for _, info := range g.Funcs {
		if info.CQRS != nil {
			g.CQRSList = append(g.CQRSList, info.CQRS)
		}
		g.printRouterInfoImpl(typeName, info)
	}
}

func buildTypeName(name string) string {
	return name // + "Controller"
}

func (g *Generate) printRouterInfoImpl(typeName string, info *internal.FuncInfo) {
	if info.Param2 == nil {
		return
	}
	if info.Result1 == nil {
		return
	}

	typeShort := "provider"
	if g.SrvTypeShort != "" {
		typeShort = g.SrvTypeShort
	}
	builds := []any{fmt.Sprintf("func(%s *", typeShort), typeName, ") ", info.FuncName, "(ctx ", contextPackage.Ident("Context"), ","}

	if info.Param2.Bytes {
		builds = append(builds, "req []byte")
	} else if info.Param2.String {
		builds = append(builds, "req string")
	} else if info.Param2.Reader {
		builds = append(builds, "req ", ioPackage.Ident("Reader"))
	} else if objectArgs := info.Param2.ObjectArgs; objectArgs != nil {
		paramObj := *info.Param2.ObjectArgs
		if paramObj.GoImportPath == "" {
			paramObj.GoImportPath = internal.GoImportPath(g.pkgImportPath)
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
			resultObj.GoImportPath = internal.GoImportPath(g.pkgImportPath)
		}
		builds = append(builds, "res *", resultObj.GoImportPath.Ident(objectArgs.Name))
	} else {
		log.Fatalf("error: func %s 2th param is invalid, must be []byte or string or *struct{}", info.FuncName)
	}

	builds = append(builds, ", err error)", " {")

	g.P(g.FunctionBuf)
	g.P(g.FunctionBuf, builds...)
	g.P(g.FunctionBuf, info.GenBody())
	g.P(g.FunctionBuf, "}")
	g.P(g.FunctionBuf)
}

func (g *Generate) appendImports() {
	for _, imp := range g.Imports {
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

func (g *Generate) appendFuncs() {
	typeName := buildTypeName(g.SrvName)
	for _, info := range g.Funcs {
		if g.isExistFunc(info.FuncName) {
			continue
		}
		if info.CQRS != nil {
			g.CQRSList = append(g.CQRSList, info.CQRS)
		}
		//g.implDeclFuncs = append(g.implDeclFuncs, g.buildFuncDecl(g.pkgImportPath, typeName, info))
		g.buildFuncDecl(g.pkgImportPath, typeName, info)
	}
}

func (g *Generate) isExistFunc(name string) bool {
	for _, info := range g.implDeclFuncs {
		if info.Name.Name == name {
			return true
		}
	}
	return false
}

func (g *Generate) isExistImport(name string) bool {
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

func (g *Generate) buildFuncDecl(importPath, typeName string, info *internal.FuncInfo) *ast.FuncDecl {
	g.printRouterInfoImpl(typeName, info)
	//decl := &ast.FuncDecl{
	//	Recv: &ast.FieldList{
	//		List: []*ast.Field{
	//			{
	//				Names: []*ast.Ident{ast.NewIdent("provider")},
	//				Type:  &ast.StarExpr{X: &ast.Ident{Name: typeName}},
	//			},
	//		},
	//	},
	//	Name: ast.NewIdent(info.FuncName),
	//	Body: &ast.BlockStmt{List: []ast.Stmt{
	//		&ast.ReturnStmt{},
	//	}},
	//}
	//
	//// params
	//params := info.Params.List
	//param2 := params[1]
	//if objectArgs := info.Param2.ObjectArgs; objectArgs != nil {
	//	paramObj := *info.Param2.ObjectArgs
	//	if paramObj.GoImportPath == "" {
	//		paramObj.GoImportPath = internal.GoImportPath(importPath)
	//	}
	//	gp := paramObj.GoImportPath.Ident(paramObj.Name)
	//	gp.GoImport.Enable = true
	//	g.imports[gp.GoImport.ImportPath] = gp.GoImport
	//	param2.Type = &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent(gp.GoImport.PackageName), Sel: ast.NewIdent(gp.GoName)}}
	//}
	//
	//// results
	//results := info.Results.List
	//result1 := results[0]
	//if objectArgs := info.Result1.ObjectArgs; objectArgs != nil {
	//	resultObj := *info.Result1.ObjectArgs
	//	if resultObj.GoImportPath == "" {
	//		resultObj.GoImportPath = internal.GoImportPath(importPath)
	//	}
	//	gp := resultObj.GoImportPath.Ident(resultObj.Name)
	//	gp.GoImport.Enable = true
	//	g.imports[gp.GoImport.ImportPath] = gp.GoImport
	//	result1.Type = &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent(gp.GoImport.PackageName), Sel: ast.NewIdent(gp.GoName)}}
	//}
	//decl.Type = &ast.FuncType{
	//	Params:  &ast.FieldList{List: params},
	//	Results: &ast.FieldList{List: results},
	//}
	//return decl
	return nil
}

func (g *Generate) buildImportSpec(info *internal.GoImport) *ast.ImportSpec {
	return &ast.ImportSpec{
		Name: ast.NewIdent(info.PackageName),
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + info.ImportPath + `"`,
		},
	}
}

func (g *Generate) Clear() {
	g.Buf.Reset()
	g.HeaderBuf.Reset()
	g.ImportsBuf.Reset()
	g.FunctionBuf.Reset()
}
