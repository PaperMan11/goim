package logic

import (
	"context"
	"fmt"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

// InitiateMultipartLogic 发起一次 Multipart Upload。
// 流程：S3 CreateMultipartUpload → 为指定分片列表生成预签名 URL → 返回 UploadInfo。
type InitiateMultipartLogic struct {
	logicBase
}

func NewInitiateMultipartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InitiateMultipartLogic {
	return &InitiateMultipartLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *InitiateMultipartLogic) InitiateMultipartUpload(req *third.InitiateMultipartUploadReq) (*third.InitiateMultipartUploadResp, error) {
	if req.Name == "" {
		l.Errorf("initiate multipart: empty name")
		return nil, errxArgs("name is required")
	}

	ctx := l.ctx
	key := buildObjectKey(req.Name, req.UrlPrefix)
	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 1. 发起 Multipart
	uploadID, err := l.svcCtx.S3Client.InitiateMultipartUpload(ctx, key, contentType)
	if err != nil {
		l.Errorf("initiate multipart failed, key=%s, err=%v", key, err)
		return nil, errxInternal(err)
	}

	// 2. 计算分片大小和分片总数
	partSize := req.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	numParts := req.MaxParts
	if numParts <= 0 {
		numParts = estimateNumParts(req.Size, partSize)
	}

	// 3. 为每个分片生成预签名 URL
	signParts := make([]*third.SignPart, 0, numParts)
	for i := int32(1); i <= numParts; i++ {
		presigned, err := l.svcCtx.S3Client.GenerateUploadPartPresignedURL(ctx, key, uploadID, i)
		if err != nil {
			l.Errorf("generate part sign failed, key=%s, uploadID=%s, part=%d, err=%v", key, uploadID, i, err)
			return nil, errxInternal(err)
		}
		signParts = append(signParts, &third.SignPart{
			PartNumber: i,
			Url:        presigned.URL,
			Query:      toProtoHeaders(presigned.Headers),
			Header:     nil, // S3 presign 把签名 query 放在 URL 上，没有额外 header
		})
	}

	presignExpireSec := int64(l.svcCtx.Config.RustFS.PresignExpire.Seconds())
	expireAt := timex.Now().UnixMilli() + presignExpireSec*1000

	uploadInfo := &third.UploadInfo{
		UploadID: uploadID,
		PartSize: partSize,
		Sign: &third.AuthSignParts{
			Parts: signParts,
		},
		ExpireTime: expireAt,
	}

	return &third.InitiateMultipartUploadResp{
		Url:    key, // object key，客户端可以用它拼最终访问路径
		Upload: uploadInfo,
	}, nil
}

// estimateNumParts 根据总大小估算分片数。
func estimateNumParts(totalSize, partSize int64) int32 {
	if totalSize <= 0 {
		return 1
	}
	n := (totalSize + partSize - 1) / partSize
	if n > int64(maxNumParts) {
		n = int64(maxNumParts)
	}
	if n < 1 {
		n = 1
	}
	return int32(n)
}

// buildObjectKey 根据原始文件名和可选前缀拼出 S3 object key。
func buildObjectKey(name, urlPrefix string) string {
	if urlPrefix != "" {
		return fmt.Sprintf("%s/%s", urlPrefix, name)
	}
	return name
}
