package logic

import (
	"context"
	"fmt"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

// InitiateFormDataLogic 生成一次 S3 POST form 上传所需的全部参数。
// 客户端拿到后直接把 multipart/form-data POST 到返回的 URL。
type InitiateFormDataLogic struct {
	logicBase
}

func NewInitiateFormDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InitiateFormDataLogic {
	return &InitiateFormDataLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *InitiateFormDataLogic) InitiateFormData(req *third.InitiateFormDataReq) (*third.InitiateFormDataResp, error) {
	if req.Name == "" {
		l.Errorf("initiate formdata: name empty")
		return nil, errxArgs("name is required")
	}

	ctx := l.ctx
	// InitiateFormDataReq 没有 urlPrefix 字段，用 group 做业务前缀
	key := req.Name
	if req.Group != "" {
		key = fmt.Sprintf("%s/%s", req.Group, req.Name)
	}
	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	presigned, err := l.svcCtx.S3Client.GeneratePostFormPresignedData(ctx, key, contentType)
	if err != nil {
		l.Errorf("initiate formdata failed, key=%s, err=%v", key, err)
		return nil, errxInternal(err)
	}

	presignExpireSec := int64(l.svcCtx.Config.RustFS.PresignExpire.Seconds())
	expireAt := timex.Now().UnixMilli() + presignExpireSec*1000

	id := key
	if len(id) > 36 {
		id = id[:36]
	}

	return &third.InitiateFormDataResp{
		Id:           id,
		Url:          presigned.URL,
		File:         key,
		Header:       toProtoHeaders(presigned.Headers),
		FormData:     presigned.FormData,
		Expires:      expireAt,
		SuccessCodes: []int32{200, 201, 204},
	}, nil
}

// CompleteFormDataLogic 小文件直传的"完成"回调。
// 实际上文件已经在客户端 POST 的那一刻直接进入 RustFS。
// CompleteFormDataReq 只有 id 和 urlPrefix——后端把两者拼回 object key，返回最终访问 URL。
type CompleteFormDataLogic struct {
	logicBase
}

func NewCompleteFormDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompleteFormDataLogic {
	return &CompleteFormDataLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *CompleteFormDataLogic) CompleteFormData(req *third.CompleteFormDataReq) (*third.CompleteFormDataResp, error) {
	if req.Id == "" {
		l.Errorf("complete formdata: id empty")
		return nil, errxArgs("id is required")
	}

	key := req.Id
	if req.UrlPrefix != "" {
		key = req.UrlPrefix + "/" + key
	}

	base := l.svcCtx.Config.RustFS.PublicEndpoint
	if base == "" {
		base = l.svcCtx.Config.RustFS.Endpoint
	}
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	finalURL := fmt.Sprintf("%s/%s/%s", base, l.svcCtx.Config.RustFS.Bucket, key)

	return &third.CompleteFormDataResp{
		Url: finalURL,
	}, nil
}
