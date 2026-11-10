package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/third/internal/svc"
	"github.com/PaperMan11/goim/pkg/protocol/third"
)

// S3 Multipart Upload 通用限制：最小 5MB，最大 5GB，最多 10000 片。
// RustFS / MinIO / AWS S3 三者一致，硬编码即可。
const (
	minPartSize     int64 = 5 * 1024 * 1024            // 5MB
	maxPartSize     int64 = 5 * 1024 * 1024 * 1024     // 5GB
	maxNumParts     int32 = 10000
	defaultPartSize int64 = 16 * 1024 * 1024            // 16MB
)

// PartLimitLogic 返回 S3 Multipart Upload 的约束。
type PartLimitLogic struct {
	logicBase
}

func NewPartLimitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PartLimitLogic {
	return &PartLimitLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *PartLimitLogic) PartLimit(req *third.PartLimitReq) (*third.PartLimitResp, error) {
	return &third.PartLimitResp{
		MinPartSize: minPartSize,
		MaxPartSize: maxPartSize,
		MaxNumSize:  int32(maxNumParts), // proto 字段名叫 maxNumSize，语义是最大分片数（S3 = 10000）
	}, nil
}

// PartSizeLogic 根据文件总大小返回一个建议的分片大小。
// 策略：ceil(totalSize / maxNumParts)，下限 minPartSize，上限 maxPartSize。
type PartSizeLogic struct {
	logicBase
}

func NewPartSizeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PartSizeLogic {
	return &PartSizeLogic{logicBase: newLogicBase(ctx, svcCtx)}
}

func (l *PartSizeLogic) PartSize(req *third.PartSizeReq) (*third.PartSizeResp, error) {
	total := req.Size
	if total <= 0 {
		return &third.PartSizeResp{Size: defaultPartSize}, nil
	}
	// ceil(total / maxNumParts)
	partSize := (total + int64(maxNumParts) - 1) / int64(maxNumParts)
	if partSize < minPartSize {
		partSize = minPartSize
	}
	if partSize > maxPartSize {
		partSize = maxPartSize
	}
	return &third.PartSizeResp{Size: partSize}, nil
}
