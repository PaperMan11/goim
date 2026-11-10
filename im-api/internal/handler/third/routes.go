package third

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

// InitRoutes 注册第三方存储相关的 HTTP 路由组。
// 所有路由挂在 /api/v1/third 前缀下，调用方不需要感知底层是 RustFS / MinIO / AWS S3。
func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	thirdRoute := r.Group("/third")

	// Multipart Upload 分片上传协议三步闭环
	thirdRoute.POST("/part_limit", PartLimit(svc))
	thirdRoute.POST("/part_size", PartSize(svc))
	thirdRoute.POST("/multipart/init", InitiateMultipartUpload(svc))
	thirdRoute.POST("/multipart/sign", AuthSign(svc))
	thirdRoute.POST("/multipart/complete", CompleteMultipartUpload(svc))
	thirdRoute.POST("/multipart/abort", AbortMultipartUpload(svc))

	// FormData 小文件直传
	thirdRoute.POST("/formdata/init", InitiateFormData(svc))
	thirdRoute.POST("/formdata/complete", CompleteFormData(svc))

	// 下载
	thirdRoute.GET("/access_url", AccessURL(svc))

	// 管理（预留）
	thirdRoute.POST("/delete_outdated", DeleteOutdatedData(svc))
}
