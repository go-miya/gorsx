package internal

import (
	"fmt"
)

type AssemblerCore struct {
	IsQuery         bool
	FuncName        string
	ToParamsIdent   *Param
	ToResultIdent   *Result
	FromParamsIdent *Param
	FromResultIdent *Result
}

func NewAssemblerCore(isQuery bool, funcName string, ToParamsIdent *Param, ToResultIdent *Result, FromParamsIdent *Param, FromResultIdent *Result) *AssemblerCore {
	return &AssemblerCore{
		IsQuery:         isQuery,
		FuncName:        funcName,
		ToParamsIdent:   ToParamsIdent,
		ToResultIdent:   ToResultIdent,
		FromParamsIdent: FromParamsIdent,
		FromResultIdent: FromResultIdent,
	}
}

const templateAssemblerTo = `
func %sTo(in *%s) *%s {
	panic("to implemented")
}
`
const templateAssemblerFrom = `
func %sFrom(in *%s) *%s {
	panic("to implemented")
}
`

func (c *AssemblerCore) Gen() string {
	to := c.GenTextTo()
	if c.IsQuery {
		to += "\n"
		to += c.GenTextFrom()
	}
	return to
}

func (c *AssemblerCore) GenTextTo() string {
	reqObj := c.ToParamsIdent.ObjectArgs
	respObj := c.ToResultIdent.ObjectArgs
	return fmt.Sprintf(templateAssemblerTo,
		c.FuncName,
		reqObj.GoImportPath.Ident(reqObj.Name).Qualify(),
		respObj.GoImportPath.Ident(respObj.Name).Qualify(),
	)
}

func (c *AssemblerCore) GenTextFrom() string {
	reqObj := c.FromParamsIdent.ObjectArgs
	respObj := c.FromResultIdent.ObjectArgs
	return fmt.Sprintf(templateAssemblerFrom,
		c.FuncName,
		reqObj.GoImportPath.Ident(reqObj.Name).Qualify(),
		respObj.GoImportPath.Ident(respObj.Name).Qualify(),
	)
}

func (c *AssemblerCore) GetFuncNameTo() string {
	return c.FuncName + "To"
}

func (c *AssemblerCore) GetFuncNameFrom() string {
	return c.FuncName + "From"
}
