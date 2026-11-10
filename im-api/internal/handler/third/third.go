package third

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	thirdpb "github.com/PaperMan11/goim/pkg/protocol/third"
	"github.com/gin-gonic/gin"
)

// ===== Multipart Upload =====

// PartLimit  GET /api/v1/third/part_limit
func PartLimit(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.PartLimit)
	}
}

// PartSize  POST /api/v1/third/part_size
func PartSize(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.PartSize)
	}
}

// InitiateMultipartUpload  POST /api/v1/third/multipart/init
func InitiateMultipartUpload(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.InitiateMultipartUpload)
	}
}

// AuthSign  POST /api/v1/third/multipart/sign
func AuthSign(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.AuthSign)
	}
}

// CompleteMultipartUpload  POST /api/v1/third/multipart/complete
func CompleteMultipartUpload(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.CompleteMultipartUpload)
	}
}

// AbortMultipartUpload  POST /api/v1/third/multipart/abort
func AbortMultipartUpload(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.AbortMultipartUpload)
	}
}

// ===== FormData Upload（小文件直传）=====

// InitiateFormData  POST /api/v1/third/formdata/init
func InitiateFormData(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.InitiateFormData)
	}
}

// CompleteFormData  POST /api/v1/third/formdata/complete
func CompleteFormData(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.CompleteFormData)
	}
}

// ===== 下载 =====

// accessURLQuery GET 请求使用 query 参数，需要 form tag 才能按小写字段名绑定。
type accessURLQuery struct {
	Name      string `form:"name"`
	URLPrefix string `form:"urlPrefix"`
}

// AccessURL  GET /api/v1/third/access_url?name=xxx&urlPrefix=yyy
func AccessURL(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.CallQuery(c, func(q accessURLQuery) *thirdpb.AccessURLReq {
			return &thirdpb.AccessURLReq{Name: q.Name, UrlPrefix: q.URLPrefix}
		}, svc.ThirdService.AccessURL)
	}
}

// ===== 管理 =====

// DeleteOutdatedData  POST /api/v1/third/delete_outdated
func DeleteOutdatedData(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.ThirdService.DeleteOutdatedData)
	}
}
