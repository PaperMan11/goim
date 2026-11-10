package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/storage/s3"
)

// CompleteMultipartLogic 合并所有已上传的分片。
type CompleteMultipartLogic struct {
	logicBase
}

func NewCompleteMultipartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompleteMultipartLogic {
	return &CompleteMultipartLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *CompleteMultipartLogic) CompleteMultipartUpload(req *third.CompleteMultipartUploadReq) (*third.CompleteMultipartUploadResp, error) {
	if req.UploadID == "" {
		l.Errorf("complete multipart: uploadID empty")
		return nil, errxArgs("uploadID is required")
	}
	if len(req.Parts) == 0 {
		l.Errorf("complete multipart: parts empty")
		return nil, errxArgs("at least one part is required")
	}

	ctx := l.ctx
	key := req.Name
	if req.UrlPrefix != "" {
		key = req.UrlPrefix + "/" + key
	}

	completedParts := make([]s3.MultipartCompletedPart, 0, len(req.Parts))
	for _, p := range req.Parts {
		completedParts = append(completedParts, s3.MultipartCompletedPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	finalURL, err := l.svcCtx.S3Client.CompleteMultipartUpload(ctx, key, req.UploadID, completedParts)
	if err != nil {
		l.Errorf("complete multipart failed, key=%s, uploadID=%s, err=%v", key, req.UploadID, err)
		return nil, errxInternal(err)
	}

	return &third.CompleteMultipartUploadResp{
		Url: finalURL,
	}, nil
}

var _ s3.S3Client = nil
