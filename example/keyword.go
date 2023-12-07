package example

import "context"

//go:generate gorsx -service Keyword -impl impl

// Keyword
// @GORS @Path(/api)  @Path(/v1)
// @CQRS @QueryPath(./app) @CommandPath(./app) @AssemblerPath(./assembler) @QueryBusPath(./bus/query.go) @CommandBusPath(./bus/command.go)
type Keyword interface {
	// BindBookCallback
	// @GORS @POST @Path(/keyword/bind_book_callback) @JSONBinding @JSONRender
	BindBookCallback(context.Context, *BindBookCallbackReq) (*BindBookCallbackResp, error)

	// ManualAudit
	// @GORS @POST @Path(/keyword/manual_audit) @JSONBinding @JSONRender
	ManualAudit(context.Context, *ManualAuditReq) (*ManualAuditResp, error)

	// BatchDelete
	// @GORS @POST @Path(/keyword/batch_delete) @JSONBinding @JSONRender
	BatchDelete(context.Context, *BatchDeleteReq) (*BatchDeleteResp, error)

	// CreateKocChannelCache
	// @GORS @POST @Path(/keyword/create_koc_channel_cache) @JSONBinding @JSONRender
	// @CQRS @Command
	CreateKocChannelCache(context.Context, *CreateKocChCacheReq) (*CreateKocChCacheResp, error)
	// CreateKocChannelCache2
	// @GORS @POST @Path(/keyword/create_koc_channel_cache) @JSONBinding @JSONRender
	// @CQRS @Query
	CreateKocChannelCache2(context.Context, *CreateKocChCacheReq) (*CreateKocChCacheResp, error)
	// CreateKocChannelCache3
	// @GORS @POST @Path(/keyword/create_koc_channel_cache) @JSONBinding @JSONRender
	// @CQRS @Query
	CreateKocChannelCache3(context.Context, *CreateKocChCacheReq) (*CreateKocChCacheResp, error)
	// CreateKocChannelCache4
	// @GORS @POST @Path(/keyword/create_koc_channel_cache) @JSONBinding @JSONRender
	CreateKocChannelCache4(context.Context, *CreateKocChCacheReq) (*CreateKocChCacheResp, error)
}

type BindBookCallbackReq struct {
	BookID int    `json:"book_id"`
	Word   string `json:"word"`
	Insert int    `json:"insert"`
	Msg    string `json:"msg"`
}

type BindBookCallbackResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type ManualAuditReq struct {
	Data interface{} `json:"data"`
}

type ManualAuditResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type BatchDeleteReq struct {
	ProductId     int    `json:"product_id"`
	SearchKeyword string `json:"search_keyword"`
	Reason        string `json:"reason"`
	AdminId       int    `json:"admin_id"`
	AdminName     string `json:"admin_name"`
}

type BatchDeleteResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type CreateKocChCacheReq struct {
	AppID       int    `json:"app_id"`
	Keyword     string `json:"keyword"`
	ChannelName string `json:"channel_name"`
}

type CreateKocChCacheResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
