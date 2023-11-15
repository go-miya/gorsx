package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

type GoImportPath string

func (p GoImportPath) Ident(s string) *GoIdent {
	importPath := string(p)
	return &GoIdent{
		GoName: s,
		GoImport: &GoImport{
			PackageName: CleanPackageName(path.Base(importPath)),
			ImportPath:  importPath,
		},
	}
}

type GoIdent struct {
	GoImport *GoImport
	GoName   string
}

func (x *GoIdent) Qualify() string {
	if x.GoImport.ImportPath == "" {
		return x.GoName
	}
	return x.GoImport.PackageName + "." + x.GoName
}

type GoImport struct {
	PackageName string
	ImportPath  string
	Enable      bool
}

type ObjectArgs struct {
	Name         string
	GoImportPath GoImportPath
}

type Param struct {
	Bytes      bool
	String     bool
	ObjectArgs *ObjectArgs
	Reader     bool
}

type Result struct {
	Bytes      bool
	String     bool
	ObjectArgs *ObjectArgs
	Reader     bool
}

func NewMethodInfo(methodName string, t *ast.FuncType) *FuncInfo {
	return &FuncInfo{
		FuncName: methodName,
		FuncType: t,
	}
}

func NewRPCMethodInfo(methodName string) *FuncInfo {
	return &FuncInfo{}
}

type FuncInfo struct {
	FuncName  string
	FuncType  *ast.FuncType
	Param2    *Param
	Result1   *Result
	CQRS      *CQRSFile
	Assembler *AssemblerCore
}

const bodyQuery = `resp, err := provider.%s.Handle(ctx, assembler.%s(req))
	if err != nil {
		return 
	}
	return assembler.%s(resp), nil`

const bodyCommand = `err = provider.%s.Handle(ctx, assembler.%s(req))
	if err != nil {
		return 
	}
	return `

func (f *FuncInfo) GenBody() string {
	if f.CQRS == nil {
		return "return"
	}
	cqrsCall := f.CQRS.Endpoint
	tmpl := bodyQuery
	if f.CQRS.IsQuery() {
		cqrsCall = "queries." + cqrsCall
		return fmt.Sprintf(tmpl, cqrsCall, f.Assembler.GetFuncNameTo(), f.Assembler.GetFuncNameFrom())
	}
	cqrsCall = "commands." + cqrsCall
	tmpl = bodyCommand
	return fmt.Sprintf(tmpl, cqrsCall, f.Assembler.GetFuncNameTo())
}

func (f *FuncInfo) Check() error {
	err := f.checkParams()
	if err != nil {
		return err
	}
	return f.checkResults()
}

func (f *FuncInfo) checkParams() error {
	if f.FuncType.Params == nil {
		return fmt.Errorf("error: func %s params is empty", f.FuncName)
	}
	if len(f.FuncType.Params.List) != 2 {
		return fmt.Errorf("error: func %s params count is not equal 2", f.FuncName)
	}
	param1 := f.FuncType.Params.List[0]
	param0SelectorExpr, ok := param1.Type.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("error: func %s 1th param is not context.Context", f.FuncName)
	}
	if param0SelectorExpr.Sel.Name != "Context" {
		return fmt.Errorf("error: func %s 1th param is not context.Context", f.FuncName)
	}
	param0SelectorExprX, ok := param0SelectorExpr.X.(*ast.Ident)
	if !ok {
		return fmt.Errorf("error: func %s 1th param is not context.Context", f.FuncName)
	}
	if param0SelectorExprX.Name != "context" {
		return fmt.Errorf("error: func %s 1th param is not context.Context", f.FuncName)
	}
	return nil
}

func (f *FuncInfo) checkResults() error {
	if f.FuncType.Results == nil {
		return fmt.Errorf("error: func %s results is empty", f.FuncName)
	}
	if len(f.FuncType.Results.List) != 2 {
		return fmt.Errorf("error: func %s results count is not equal 2", f.FuncName)
	}
	result2 := f.FuncType.Results.List[1]
	result2Iden, ok := result2.Type.(*ast.Ident)
	if !ok {
		return fmt.Errorf("error: func %s 2th result is not error", f.FuncName)
	}
	if result2Iden.Name != "error" {
		return fmt.Errorf("error: func %s 2th result is not error", f.FuncName)
	}
	return nil
}

func CleanPackageName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, name)

	// Prepend '_' in the event of a Go keyword conflict or if
	// the identifier is invalid (does not start in the Unicode L category).
	r, _ := utf8.DecodeRuneInString(name)
	if token.Lookup(name).IsKeyword() || !unicode.IsLetter(r) {
		return "_" + name
	}
	return name
}
