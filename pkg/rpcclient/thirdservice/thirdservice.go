package thirdservice

import (
	"context"

	pbthird "github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// ThirdService 是 third RPC 服务的 Go 接口，给 im-api 等上层调用方使用。
// 所有方法签名与 proto 生成的 ThirdClient 一致。
type ThirdService interface {
	// PartLimit 返回 S3 Multipart Upload 的限制（min/max part size、max num parts）。
	PartLimit(ctx context.Context, in *pbthird.PartLimitReq, opts ...grpc.CallOption) (*pbthird.PartLimitResp, error)
	// PartSize 根据总大小返回建议的分片大小。
	PartSize(ctx context.Context, in *pbthird.PartSizeReq, opts ...grpc.CallOption) (*pbthird.PartSizeResp, error)
	// InitiateMultipartUpload 发起一次 Multipart Upload，返回 uploadID + 所有分片预签名 URL。
	InitiateMultipartUpload(ctx context.Context, in *pbthird.InitiateMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.InitiateMultipartUploadResp, error)
	// AuthSign 为指定分片号列表重新生成预签名 URL。
	AuthSign(ctx context.Context, in *pbthird.AuthSignReq, opts ...grpc.CallOption) (*pbthird.AuthSignResp, error)
	// CompleteMultipartUpload 合并所有已上传的分片。
	CompleteMultipartUpload(ctx context.Context, in *pbthird.CompleteMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.CompleteMultipartUploadResp, error)
	// AbortMultipartUpload 放弃一次 Multipart Upload，释放已上传分片。
	AbortMultipartUpload(ctx context.Context, in *pbthird.AbortMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.AbortMultipartUploadResp, error)
	// InitiateFormData 发起一次小文件直传（POST form），返回 S3 POST form 预签名参数。
	InitiateFormData(ctx context.Context, in *pbthird.InitiateFormDataReq, opts ...grpc.CallOption) (*pbthird.InitiateFormDataResp, error)
	// CompleteFormData 小文件直传完成后回调（拼最终访问 URL）。
	CompleteFormData(ctx context.Context, in *pbthird.CompleteFormDataReq, opts ...grpc.CallOption) (*pbthird.CompleteFormDataResp, error)
	// AccessURL 生成已上传文件的 GET 预签名下载 URL。
	AccessURL(ctx context.Context, in *pbthird.AccessURLReq, opts ...grpc.CallOption) (*pbthird.AccessURLResp, error)
	// DeleteOutdatedData 过期对象清理（预留接口）。
	DeleteOutdatedData(ctx context.Context, in *pbthird.DeleteOutdatedDataReq, opts ...grpc.CallOption) (*pbthird.DeleteOutdatedDataResp, error)
}

type defaultThirdService struct {
	cli zrpc.Client
}

// NewThirdService 创建一个连接到真实 third RPC 服务的客户端。
func NewThirdService(cli zrpc.Client) ThirdService {
	return &defaultThirdService{cli: cli}
}

func (s *defaultThirdService) client() pbthird.ThirdClient {
	return pbthird.NewThirdClient(s.cli.Conn())
}

func (s *defaultThirdService) PartLimit(ctx context.Context, in *pbthird.PartLimitReq, opts ...grpc.CallOption) (*pbthird.PartLimitResp, error) {
	return s.client().PartLimit(ctx, in, opts...)
}

func (s *defaultThirdService) PartSize(ctx context.Context, in *pbthird.PartSizeReq, opts ...grpc.CallOption) (*pbthird.PartSizeResp, error) {
	return s.client().PartSize(ctx, in, opts...)
}

func (s *defaultThirdService) InitiateMultipartUpload(ctx context.Context, in *pbthird.InitiateMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.InitiateMultipartUploadResp, error) {
	return s.client().InitiateMultipartUpload(ctx, in, opts...)
}

func (s *defaultThirdService) AuthSign(ctx context.Context, in *pbthird.AuthSignReq, opts ...grpc.CallOption) (*pbthird.AuthSignResp, error) {
	return s.client().AuthSign(ctx, in, opts...)
}

func (s *defaultThirdService) CompleteMultipartUpload(ctx context.Context, in *pbthird.CompleteMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.CompleteMultipartUploadResp, error) {
	return s.client().CompleteMultipartUpload(ctx, in, opts...)
}

func (s *defaultThirdService) AbortMultipartUpload(ctx context.Context, in *pbthird.AbortMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.AbortMultipartUploadResp, error) {
	return s.client().AbortMultipartUpload(ctx, in, opts...)
}

func (s *defaultThirdService) InitiateFormData(ctx context.Context, in *pbthird.InitiateFormDataReq, opts ...grpc.CallOption) (*pbthird.InitiateFormDataResp, error) {
	return s.client().InitiateFormData(ctx, in, opts...)
}

func (s *defaultThirdService) CompleteFormData(ctx context.Context, in *pbthird.CompleteFormDataReq, opts ...grpc.CallOption) (*pbthird.CompleteFormDataResp, error) {
	return s.client().CompleteFormData(ctx, in, opts...)
}

func (s *defaultThirdService) AccessURL(ctx context.Context, in *pbthird.AccessURLReq, opts ...grpc.CallOption) (*pbthird.AccessURLResp, error) {
	return s.client().AccessURL(ctx, in, opts...)
}

func (s *defaultThirdService) DeleteOutdatedData(ctx context.Context, in *pbthird.DeleteOutdatedDataReq, opts ...grpc.CallOption) (*pbthird.DeleteOutdatedDataResp, error) {
	return s.client().DeleteOutdatedData(ctx, in, opts...)
}

// ===== stub =====

// stubThirdService 用于 dev / test 模式下没有第三方存储时的零依赖跑通。
// 所有方法返回空的 proto Resp。
type stubThirdService struct{}

// NewStubThirdService 创建一个 stub 实现，所有 RPC 调用都返回空响应。
func NewStubThirdService() ThirdService {
	return &stubThirdService{}
}

func (s *stubThirdService) PartLimit(ctx context.Context, in *pbthird.PartLimitReq, opts ...grpc.CallOption) (*pbthird.PartLimitResp, error) {
	return &pbthird.PartLimitResp{}, nil
}

func (s *stubThirdService) PartSize(ctx context.Context, in *pbthird.PartSizeReq, opts ...grpc.CallOption) (*pbthird.PartSizeResp, error) {
	return &pbthird.PartSizeResp{}, nil
}

func (s *stubThirdService) InitiateMultipartUpload(ctx context.Context, in *pbthird.InitiateMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.InitiateMultipartUploadResp, error) {
	return &pbthird.InitiateMultipartUploadResp{}, nil
}

func (s *stubThirdService) AuthSign(ctx context.Context, in *pbthird.AuthSignReq, opts ...grpc.CallOption) (*pbthird.AuthSignResp, error) {
	return &pbthird.AuthSignResp{}, nil
}

func (s *stubThirdService) CompleteMultipartUpload(ctx context.Context, in *pbthird.CompleteMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.CompleteMultipartUploadResp, error) {
	return &pbthird.CompleteMultipartUploadResp{}, nil
}

func (s *stubThirdService) AbortMultipartUpload(ctx context.Context, in *pbthird.AbortMultipartUploadReq, opts ...grpc.CallOption) (*pbthird.AbortMultipartUploadResp, error) {
	return &pbthird.AbortMultipartUploadResp{}, nil
}

func (s *stubThirdService) InitiateFormData(ctx context.Context, in *pbthird.InitiateFormDataReq, opts ...grpc.CallOption) (*pbthird.InitiateFormDataResp, error) {
	return &pbthird.InitiateFormDataResp{}, nil
}

func (s *stubThirdService) CompleteFormData(ctx context.Context, in *pbthird.CompleteFormDataReq, opts ...grpc.CallOption) (*pbthird.CompleteFormDataResp, error) {
	return &pbthird.CompleteFormDataResp{}, nil
}

func (s *stubThirdService) AccessURL(ctx context.Context, in *pbthird.AccessURLReq, opts ...grpc.CallOption) (*pbthird.AccessURLResp, error) {
	return &pbthird.AccessURLResp{}, nil
}

func (s *stubThirdService) DeleteOutdatedData(ctx context.Context, in *pbthird.DeleteOutdatedDataReq, opts ...grpc.CallOption) (*pbthird.DeleteOutdatedDataResp, error) {
	return &pbthird.DeleteOutdatedDataResp{}, nil
}
