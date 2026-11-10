package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/storage/s3"
)

// AbortMultipartLogic 放弃一次未完成的 Multipart Upload，释放已上传分片占用的存储。
type AbortMultipartLogic struct {
	logicBase
}

func NewAbortMultipartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AbortMultipartLogic {
	return &AbortMultipartLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *AbortMultipartLogic) AbortMultipartUpload(req *third.AbortMultipartUploadReq) (*third.AbortMultipartUploadResp, error) {
	if req.UploadID == "" {
		l.Errorf("abort multipart: uploadID empty")
		return nil, errxArgs("uploadID is required")
	}

	ctx := l.ctx
	key := req.Name
	if req.UrlPrefix != "" {
		key = req.UrlPrefix + "/" + key
	}

	if err := l.svcCtx.S3Client.AbortMultipartUpload(ctx, key, req.UploadID); err != nil {
		l.Errorf("abort multipart failed, key=%s, uploadID=%s, err=%v", key, req.UploadID, err)
		return nil, errxInternal(err)
	}

	return &third.AbortMultipartUploadResp{}, nil
}

var _ s3.S3Client = nil
