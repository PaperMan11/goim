package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/PaperMan11/goim/pkg/utils/convert"
)

// AccessURLLogic 为已上传的文件生成一个带过期时间的 GET 预签名 URL。
// 客户端拿到后可以直接下载，后端不需要代理流量。
type AccessURLLogic struct {
	logicBase
}

func NewAccessURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AccessURLLogic {
	return &AccessURLLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *AccessURLLogic) AccessURL(req *third.AccessURLReq) (*third.AccessURLResp, error) {
	if req.Name == "" {
		l.Errorf("access url: name empty")
		return nil, errxArgs("name is required")
	}

	ctx := l.ctx
	key := buildObjectKey(req.Name, req.UrlPrefix)

	var expireSeconds int64
	if req.Query != nil {
		if v, ok := req.Query["expires"]; ok && v != "" {
			if n, err := convert.ToInt64E(v); err == nil {
				expireSeconds = n
			} else {
				l.Infof("access url: invalid expires value %q, ignoring", v)
			}
		}
	}

	presigned, err := l.svcCtx.S3Client.GenerateGetObjectPresignedURL(ctx, key, expireSeconds)
	if err != nil {
		l.Errorf("access url failed, key=%s, err=%v", key, err)
		return nil, errxInternal(err)
	}

	// presigned.ExpireAt 是 Unix ms
	return &third.AccessURLResp{
		Url:        presigned.URL,
		ExpireTime: presigned.ExpireAt,
	}, nil
}
