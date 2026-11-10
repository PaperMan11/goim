package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

// AuthSignLogic 为指定分片号列表生成新的预签名 URL。
// 场景：initiate 时发的一批 URL 过期了，客户端用 AuthSign 续签其中某几片。
type AuthSignLogic struct {
	logicBase
}

func NewAuthSignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthSignLogic {
	return &AuthSignLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *AuthSignLogic) AuthSign(req *third.AuthSignReq) (*third.AuthSignResp, error) {
	if req.UploadID == "" || len(req.PartNumbers) == 0 {
		l.Errorf("auth sign: uploadID or partNumbers empty")
		return nil, errxArgs("uploadID and partNumbers are required")
	}
	if req.Name == "" {
		l.Errorf("auth sign: name (object key) empty, S3 需要 bucket+key+uploadID+partNumber 才能生成预签名 URL")
		return nil, errxArgs("name (object key) is required for presign")
	}

	ctx := l.ctx
	key := buildObjectKey(req.Name, req.UrlPrefix)

	signParts := make([]*third.SignPart, 0, len(req.PartNumbers))
	for _, partNum := range req.PartNumbers {
		presigned, err := l.svcCtx.S3Client.GenerateUploadPartPresignedURL(ctx, key, req.UploadID, partNum)
		if err != nil {
			l.Errorf("auth sign: generate part sign failed, key=%s, uploadID=%s, part=%d, err=%v",
				key, req.UploadID, partNum, err)
			return nil, errxInternal(err)
		}
		signParts = append(signParts, &third.SignPart{
			PartNumber: partNum,
			Url:        presigned.URL,
			Query:      toProtoHeaders(presigned.Headers),
		})
	}

	presignExpireSec := int64(l.svcCtx.Config.RustFS.PresignExpire.Seconds())
	expireAt := timex.Now().UnixMilli() + presignExpireSec*1000

	return &third.AuthSignResp{
		Parts:      signParts,
		ExpireTime: expireAt,
	}, nil
}
