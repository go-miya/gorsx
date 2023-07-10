package pkg

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
)

func ParserFile(file string) (*ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	fmt.Println(111, fset.Position(token.Pos(129)), fset.PositionFor(token.Pos(129), false))
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
