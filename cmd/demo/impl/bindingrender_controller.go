package impl

import (
	context "context"
	demo "github.com/go-miya/gorsx/cmd/demo"
)

type BindingRenderController struct{}

func (provider *BindingRenderController) UriBindingIndentedJSONRender(c context.Context, req *demo.UriBindingReq) (res *demo.IndentedJSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) QueryBindingSecureJSONRender(c context.Context, req *demo.QueryBindingReq) (res *demo.SecureJSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) HeaderBindingJsonpJSONRender(c context.Context, req *demo.HeaderBindingReq) (res *demo.JsonpJSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) JSONBindingJSONRender(c context.Context, req *demo.JSONBindingReq) (res *demo.JSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) XMLBindingXMLRender(c context.Context, req *demo.XMLBindingReq) (res *demo.XMLRenderResp, err error) {
	return
}

func (provider *BindingRenderController) FormBindingJSONRender(c context.Context, req *demo.FormBindingReq) (res *demo.JSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) FormPostBindingPureJSONRender(c context.Context, req *demo.FormPostBindingReq) (res *demo.PureJSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) FormMultipartBindingAsciiJSONRender(c context.Context, req *demo.FormMultipartBindingReq) (res *demo.AsciiJSONRenderResp, err error) {
	return
}

func (provider *BindingRenderController) MsgPackBindingMsgPackRender(c context.Context, req *demo.MsgPackBindingReq) (res *demo.MsgPackRenderResp, err error) {
	return
}

func (provider *BindingRenderController) YAMLBindingYAMLRender(c context.Context, req *demo.YAMLBindingReq) (res *demo.YAMLRenderResp, err error) {
	return
}

func (provider *BindingRenderController) TOMLBindingTOMLRender(c context.Context, req *demo.TOMLBindingReq) (res *demo.TOMLRenderResp, err error) {
	return
}

func (provider *BindingRenderController) TOMLBindingTOMLRender2(c context.Context, req demo.TOMLBindingReq) (res demo.TOMLBindingReq, err error) {
	return
}
