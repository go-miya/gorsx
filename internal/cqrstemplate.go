package internal

import (
	_ "embed"
	"errors"
	"os"
	"text/template"
)

//go:embed command.go.template
var commandContent string

//go:embed query.go.template
var queryContent string

type CQRSFile struct {
	Type          string
	RelaPath      string
	AbsFilename   string
	Package       string
	Endpoint      string
	LowerEndpoint string
}

func (v CQRSFile) GetReqName() string {
	if v.Type == "command" {
		return v.Endpoint + "Cmd"
	}
	return v.Endpoint + "Query"
}

func (v CQRSFile) IsQuery() bool {
	return v.Type == "query"
}

func (v CQRSFile) IsCommand() bool {
	return v.Type == "command"
}

func (v CQRSFile) GetRespName() string {
	if v.Type == "command" {
		return "nil"
	}
	return v.Endpoint + "Result"
}

func (v CQRSFile) Gen() error {
	if v.RelaPath == "" {
		return errors.New("@QueryPath or @CommandPath is empty")
	}
	if v.IsCommand() {
		return v.genCommand()
	} else if v.IsQuery() {
		return v.genQuery()
	}
	return errors.New("unknown endpoint type")
}

func (v CQRSFile) genQuery() error {
	tmpl, err := template.New("query").Parse(queryContent)
	if err != nil {
		return err
	}
	_, err = os.Stat(v.AbsFilename)
	if os.IsNotExist(err) {
		file, err := os.Create(v.AbsFilename)
		if err != nil {
			return err
		}
		return tmpl.Execute(file, &v)
	}
	if err != nil {
		return err
	}
	return nil
}

func (v CQRSFile) genCommand() error {
	tmpl, err := template.New("command").Parse(commandContent)
	if err != nil {
		return err
	}
	_, err = os.Stat(v.AbsFilename)
	if os.IsNotExist(err) {
		file, err := os.Create(v.AbsFilename)
		if err != nil {
			return err
		}
		return tmpl.Execute(file, &v)
	}
	if err != nil {
		return err
	}
	return nil
}
