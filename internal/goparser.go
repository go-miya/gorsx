package internal

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
)

func ParserGoFile(file string) (*ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func InspectAstFile(astFile *ast.File) (importDecl []*ast.GenDecl, remainedDecl []ast.Decl, funcDecl []*ast.FuncDecl) {
	for _, decl := range astFile.Decls {
		switch t := decl.(type) {
		case *ast.GenDecl:
			if t.Tok == token.IMPORT {
				importDecl = append(importDecl, t)
			} else {
				remainedDecl = append(remainedDecl, t)
			}
		case *ast.FuncDecl:
			funcDecl = append(funcDecl, t)
		default:
			remainedDecl = append(remainedDecl, t)
		}
	}
	return
}

func AstToGo(dst *bytes.Buffer, node interface{}) error {
	addNewline := func() {
		err := dst.WriteByte('\n') // add newline
		if err != nil {
			log.Panicln(err)
		}
	}
	addNewline()
	err := format.Node(dst, token.NewFileSet(), node)
	if err != nil {
		return err
	}
	addNewline()
	return nil
}

func InspectBus(astFile *ast.File, busType string) (importSpecs []*ast.ImportSpec, busFields []*ast.Field) {
	if astFile == nil {
		return
	}
	for _, decl := range astFile.Decls {
		switch t := decl.(type) {
		case *ast.GenDecl:
			if t.Tok == token.IMPORT {
				importSpecs = parseImportDecl(t)
			} else if t.Tok == token.TYPE {
				busFields = parseTypeDeclForFields(busType, t)
			}
		}
	}
	return
}

func parseImportDecl(decl *ast.GenDecl) (res []*ast.ImportSpec) {
	for _, spec := range decl.Specs {
		importSpec, ok := spec.(*ast.ImportSpec)
		if !ok {
			continue
		}
		res = append(res, importSpec)
	}
	return res
}

func parseTypeDeclForFields(name string, decl *ast.GenDecl) (res []*ast.Field) {
	var tarTypeSpec *ast.TypeSpec
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		if typeSpec.Name.Name == name {
			tarTypeSpec = typeSpec
			break
		}
	}
	if tarTypeSpec == nil {
		return
	}

	st, ok := tarTypeSpec.Type.(*ast.StructType)
	if !ok {
		return
	}
	for _, f := range st.Fields.List {
		res = append(res, f)
	}
	return
}
