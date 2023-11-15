package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/go-leo/gox/slicex"
	"github.com/go-miya/gorsx/cmd"
	"github.com/go-miya/gorsx/internal"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

var (
	serviceName   = flag.String("service", "", "service interface Name; must be set")
	ImplPath      = flag.String("impl", "", "service implementation Path")
	AssemblerPath = flag.String("assemble", "", "assemble path")
)

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of gorsx:\n")
	fmt.Fprintf(os.Stderr, "\tgorsx -service S\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	fmt.Fprintf(os.Stderr, "\tgorsx -impl S\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("gorsx: ")
}

func main() {

	flag.Usage = Usage
	flag.Parse()

	// must set service names
	if len(*serviceName) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	// We accept either one directory or a list of files. Which do we have?
	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	// load package information
	pack := loadPkg(args)

	// inspect package
	serviceFile, serviceDecl, serviceSpec, serviceType, serviceMethods := inspect(pack)
	if serviceFile == nil || serviceDecl == nil || serviceSpec == nil || serviceType == nil {
		log.Fatal("error: not found service")
	}

	imports := getGoImports(serviceFile)
	g := &cmd.Generate{
		Buf:              &bytes.Buffer{},
		HeaderBuf:        &bytes.Buffer{},
		ImportsBuf:       &bytes.Buffer{},
		FunctionBuf:      &bytes.Buffer{},
		Imports:          imports,
		SrvName:          *serviceName,
		Funcs:            nil,
		UsedPackageNames: make(map[string]bool),
	}

	var files []*internal.CQRSFile
	if serviceDecl != nil && serviceSpec != nil && serviceType != nil && len(serviceMethods) > 0 {
		// cqrsx
		serviceName := serviceSpec.Name.String()
		if serviceDecl.Doc == nil {
			log.Println("not found", serviceName, "annotation:", `"@CQRS @QueryPath() @CommandPath()"`)
			os.Exit(2)
		}
		var comments []string
		for _, comment := range serviceDecl.Doc.List {
			comments = append(comments, comment.Text)
		}
		cqrsPath := internal.NewPath(comments)
		queryAbs, err := filepath.Abs(cqrsPath.Query)
		if err != nil {
			fmt.Printf("query path error: %s\n", err)
			os.Exit(2)
		}
		commandAbs, err := filepath.Abs(cqrsPath.Command)
		if err != nil {
			fmt.Printf("command path error: %s\n", err)
			os.Exit(2)
		}
		fmt.Println(queryAbs, queryAbs)

		// assembler
		for _, method := range serviceMethods {
			if slicex.IsEmpty(method.Names) {
				continue
			}
			methodName := method.Names[0]

			// controller
			funcType, ok := method.Type.(*ast.FuncType)
			if !ok {
				log.Fatalf("error: func %s not convert to *ast.FuncType", methodName)
			}

			funcInfo := internal.NewMethodInfo(methodName.Name, funcType)
			err := funcInfo.Check()
			if err != nil {
				log.Fatal(err)
			}

			funcInfo.Param2 = g.CheckAndGetParam2(funcType, methodName)
			funcInfo.Result1 = g.CheckAndGetResult1(funcType, methodName)
			g.Funcs = append(g.Funcs, funcInfo)

			// cqrs
			if method.Doc == nil {
				continue
			}
			comments := slicex.Map[[]*ast.Comment, []string](
				method.Doc.List,
				func(i int, e1 *ast.Comment) string { return e1.Text },
			)
			cqrsFile := internal.NewFileFromComment(
				methodName.Name, queryAbs, commandAbs, cqrsPath.Query, cqrsPath.Command, comments, cqrsPath.NamePrefix)
			if cqrsFile == nil {
				continue
			}
			files = append(files, cqrsFile)
			funcInfo.CQRS = cqrsFile
			funcInfo.Assembler = internal.NewAssemblerCore(
				cqrsFile.IsQuery(),
				methodName.Name,
				funcInfo.Param2,
				&internal.Result{ObjectArgs: &internal.ObjectArgs{Name: cqrsFile.GetReqName(), GoImportPath: internal.GoImportPath(cqrsFile.Package)}},
				&internal.Param{ObjectArgs: &internal.ObjectArgs{Name: cqrsFile.GetRespName(), GoImportPath: internal.GoImportPath(cqrsFile.Package)}},
				funcInfo.Result1,
			)
		}
	}
	// gen service implementation
	g.Generate(pack.PkgPath, *ImplPath, *AssemblerPath, pack.GoFiles)
	// gen cqrs
	for _, f := range files {
		if err := f.Gen(); err != nil {
			log.Printf("%s.%s.%s error: %s\n", pack.PkgPath, *serviceName, f.Endpoint, err)
			continue
		}
		log.Printf("%s.%s.%s wrote %s\n", pack.PkgPath, *serviceName, f.Endpoint, f.AbsFilename)
	}
}

func loadPkg(args []string) *packages.Package {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes,
	}
	pkgs, err := packages.Load(cfg, args...)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found", len(pkgs))
	}
	return pkgs[0]
}

func inspect(pkg *packages.Package) (*ast.File, *ast.GenDecl, *ast.TypeSpec, *ast.InterfaceType, []*ast.Field) {
	var serviceFile *ast.File
	var serviceDecl *ast.GenDecl
	var serviceSpec *ast.TypeSpec
	var serviceType *ast.InterfaceType
	var serviceMethods []*ast.Field
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			denDecl, ok := node.(*ast.GenDecl)
			if !ok {
				return true
			}
			if denDecl.Tok != token.TYPE {
				// We only care about type declarations.
				return true
			}
			for _, spec := range denDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				if typeSpec.Name.Name != *serviceName {
					// This is not the interface type we're looking for.
					continue
				}
				serviceFile = file
				serviceDecl = denDecl
				serviceSpec = typeSpec
				serviceType = interfaceType
				serviceMethods = interfaceType.Methods.List
				return false
			}
			return true
		})
	}
	return serviceFile, serviceDecl, serviceSpec, serviceType, serviceMethods
}

func getGoImports(serviceFile *ast.File) map[string]*internal.GoImport {
	goImports := make(map[string]*internal.GoImport)
	for _, importSpec := range serviceFile.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			log.Panicf("warning: unquote error: %s", err)
		}
		item := &internal.GoImport{
			ImportPath: importPath,
		}
		if importSpec.Name != nil {
			item.PackageName = importSpec.Name.Name
		} else {
			item.PackageName = internal.CleanPackageName(path.Base(importPath))
		}
		goImports[item.ImportPath] = item
	}
	return goImports
}
