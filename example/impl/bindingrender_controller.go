package impl

import (
	context "context"
	example "github.com/go-miya/gorsx/example"
)

type BindingRenderController struct{}

func (provider *BindingRenderController) UriBindingIndentedJSONRender(c context.Context, req *example.UriBindingReq) (res *example.IndentedJSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) QueryBindingSecureJSONRender(c context.Context, req *example.QueryBindingReq) (res *example.SecureJSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) HeaderBindingJsonpJSONRender(c context.Context, req *example.HeaderBindingReq) (res *example.JsonpJSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) JSONBindingJSONRender(c context.Context, req *example.JSONBindingReq) (res *example.JSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) XMLBindingXMLRender(c context.Context, req *example.XMLBindingReq) (res *example.XMLRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) FormBindingJSONRender(c context.Context, req *example.FormBindingReq) (res *example.JSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) FormPostBindingPureJSONRender(c context.Context, req *example.FormPostBindingReq) (res *example.PureJSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) FormMultipartBindingAsciiJSONRender(c context.Context, req *example.FormMultipartBindingReq) (res *example.AsciiJSONRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) MsgPackBindingMsgPackRender(c context.Context, req *example.MsgPackBindingReq) (res *example.MsgPackRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) YAMLBindingYAMLRender(c context.Context, req *example.YAMLBindingReq) (res *example.YAMLRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) TOMLBindingTOMLRender(c context.Context, req *example.TOMLBindingReq) (res *example.TOMLRenderResp, err error) {
	// code your logic here
	return
}

func (provider *BindingRenderController) TOMLBindingTOMLRender2(c context.Context, req *example.TOMLBindingReq) (res *example.TOMLRenderResp, err error) {
	// code your logic here
	return
}
