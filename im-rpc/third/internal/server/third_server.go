package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
)

// ThirdServer 实现 proto 里定义的 Third gRPC 服务。
// 每个 RPC 委托给对应 logic 处理。
type ThirdServer struct {
	svcCtx *svc.ServiceContext
	third.UnimplementedThirdServer
}

func NewThirdServer(svcCtx *svc.ServiceContext) *ThirdServer {
	return &ThirdServer{svcCtx: svcCtx}
}

// ----- Multipart Upload -----

func (s *ThirdServer) PartLimit(ctx context.Context, req *third.PartLimitReq) (*third.PartLimitResp, error) {
	l := logic.NewPartLimitLogic(ctx, s.svcCtx)
	return l.PartLimit(req)
}

func (s *ThirdServer) PartSize(ctx context.Context, req *third.PartSizeReq) (*third.PartSizeResp, error) {
	l := logic.NewPartSizeLogic(ctx, s.svcCtx)
	return l.PartSize(req)
}

func (s *ThirdServer) InitiateMultipartUpload(ctx context.Context, req *third.InitiateMultipartUploadReq) (*third.InitiateMultipartUploadResp, error) {
	l := logic.NewInitiateMultipartLogic(ctx, s.svcCtx)
	return l.InitiateMultipartUpload(req)
}

func (s *ThirdServer) AuthSign(ctx context.Context, req *third.AuthSignReq) (*third.AuthSignResp, error) {
	l := logic.NewAuthSignLogic(ctx, s.svcCtx)
	return l.AuthSign(req)
}

func (s *ThirdServer) CompleteMultipartUpload(ctx context.Context, req *third.CompleteMultipartUploadReq) (*third.CompleteMultipartUploadResp, error) {
	l := logic.NewCompleteMultipartLogic(ctx, s.svcCtx)
	return l.CompleteMultipartUpload(req)
}

func (s *ThirdServer) AbortMultipartUpload(ctx context.Context, req *third.AbortMultipartUploadReq) (*third.AbortMultipartUploadResp, error) {
	l := logic.NewAbortMultipartLogic(ctx, s.svcCtx)
	return l.AbortMultipartUpload(req)
}

// ----- FormData Upload（小文件直传）-----

func (s *ThirdServer) InitiateFormData(ctx context.Context, req *third.InitiateFormDataReq) (*third.InitiateFormDataResp, error) {
	l := logic.NewInitiateFormDataLogic(ctx, s.svcCtx)
	return l.InitiateFormData(req)
}

func (s *ThirdServer) CompleteFormData(ctx context.Context, req *third.CompleteFormDataReq) (*third.CompleteFormDataResp, error) {
	l := logic.NewCompleteFormDataLogic(ctx, s.svcCtx)
	return l.CompleteFormData(req)
}

// ----- 下载 -----

func (s *ThirdServer) AccessURL(ctx context.Context, req *third.AccessURLReq) (*third.AccessURLResp, error) {
	l := logic.NewAccessURLLogic(ctx, s.svcCtx)
	return l.AccessURL(req)
}

// ----- 管理 -----

func (s *ThirdServer) DeleteOutdatedData(ctx context.Context, req *third.DeleteOutdatedDataReq) (*third.DeleteOutdatedDataResp, error) {
	l := logic.NewDeleteOutdatedLogic(ctx, s.svcCtx)
	return l.DeleteOutdatedData(req)
}

// ----- 推送 & 日志 -----
// FcmUpdateToken / SetAppBadge / UploadLogs / DeleteLogs / SearchLogs 暂不实现。
// 客户端调用时 gRPC 框架会返回 Unimplemented 状态码。
