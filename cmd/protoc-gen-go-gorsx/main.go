package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	main2 "github.com/go-miya/gorsx/cmd"
	"github.com/go-miya/gorsx/internal"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-go-gors %v\n", "v0.0.1")
		return
	}

	var flags flag.FlagSet
	protogen.Options{ParamFunc: flags.Set}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f)
		}
		return nil
	})
}

func generateFile(_ *protogen.Plugin, file *protogen.File) {
	if len(file.Services) == 0 {
		return
	}
	for _, service := range file.Services {
		getFileInfo(file, service)
	}
}

// cwd: /Users/zhaoxing/Documents/work/miya/gorsx,
// outDir: /Users/zhaoxing/Documents/work/miya/gorsx/example_rpc,
// servicePath: ./impl, relative_proto_path: example_rpc/koc_data.proto,
// abs_proto_path: /Users/zhaoxing/Documents/work/miya/gorsx/example_rpc/koc_data.proto,
// abs_query_path: /Users/zhaoxing/Documents/work/miya/gorsx/example_rpc/app,
// abs_command_path: /Users/zhaoxing/Documents/work/miya/gorsx/example_rpc/app
func getFileInfo(file *protogen.File, service *protogen.Service) {
	path := internal.NewPath(splitComment(service.Comments.Leading.String()))
	cwd, _ := os.Getwd()
	outDir := filepath.Dir(filepath.Join(cwd, file.Desc.Path()))
	queryAbs := filepath.Join(outDir, path.Query)
	commandAbs := filepath.Join(filepath.Dir(filepath.Join(cwd, file.Desc.Path())), path.Command)
	var cqrsFiles []*internal.CQRSFile
	g := &main2.Generate{
		Buf:              &bytes.Buffer{},
		HeaderBuf:        &bytes.Buffer{},
		ImportsBuf:       &bytes.Buffer{},
		FunctionBuf:      &bytes.Buffer{},
		SrvName:          service.GoName,
		Funcs:            nil,
		Imports:          make(map[string]*internal.GoImport),
		UsedPackageNames: make(map[string]bool),
	}
	for _, method := range service.Methods {
		if !method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
			methodName := method.GoName
			funcInfo := internal.NewRPCMethodInfo(methodName)
			funcInfo.Param2 = checkAndGetParam2(method.Input)
			funcInfo.Result1 = checkAndGetResult1(method.Output)
			g.Funcs = append(g.Funcs, funcInfo)

			// Unary RPC method
			cqrsFile := internal.NewFileFromComment(
				methodName, queryAbs, commandAbs, path.Query, path.Command, splitComment(method.Comments.Leading.String()), path.NamePrefix)
			if cqrsFile == nil {
				continue
			}
			cqrsFiles = append(cqrsFiles, cqrsFile)

			funcInfo.CQRS = cqrsFile
			funcInfo.Assembler = internal.NewAssemblerCore(
				cqrsFile.IsQuery(),
				methodName,
				funcInfo.Param2,
				&internal.Result{ObjectArgs: &internal.ObjectArgs{Name: cqrsFile.GetReqName(), GoImportPath: internal.GoImportPath(cqrsFile.Package)}},
				&internal.Param{ObjectArgs: &internal.ObjectArgs{Name: cqrsFile.GetRespName(), GoImportPath: internal.GoImportPath(cqrsFile.Package)}},
				funcInfo.Result1,
			)
		}
	}
	g.Generate(outDir, buildGoImportPath(path.GoBasePath, strings.Trim(string(file.GoImportPath), "\"")), path.ServiceImplPath, path)
	for _, f := range cqrsFiles {
		if err := f.Gen(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s.%s error: %s \n", service.Desc.FullName(), f.Endpoint, err)
			continue
		}
	}
}

func buildGoImportPath(basePath string, fileGoImportPath string) string {
	goImportPathList := strings.Split(fileGoImportPath, "/")
	goImportPathList = append([]string{basePath}, goImportPathList[1:]...)
	return strings.Join(goImportPathList, "/")
}

func splitComment(leadingComment string) []string {
	var comments []string
	scanner := bufio.NewScanner(strings.NewReader(leadingComment))
	for scanner.Scan() {
		line := scanner.Text()
		comments = append(comments, line)
	}
	return comments
}

func checkAndGetParam2(in *protogen.Message) *internal.Param {
	return &internal.Param{
		ObjectArgs: &internal.ObjectArgs{
			Name: in.GoIdent.GoName,
		},
	}
}

func checkAndGetResult1(in *protogen.Message) *internal.Result {
	return &internal.Result{
		ObjectArgs: &internal.ObjectArgs{
			Name: in.GoIdent.GoName,
		},
	}
}
