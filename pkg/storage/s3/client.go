package s3

import "context"

// HeaderKV 表示一个 HTTP header 列表项（一对多值），等价于 S3 的 x-amz-meta-* 头。
// 设计成切片而不是 map 因为 header 允许多个同名 key。
type HeaderKV struct {
	Key    string
	Values []string
}

// PresignedURLResult 描述一次预签名 URL 生成的完整响应：
// 客户端直接 PUT/POST/GET url 即可完成操作，query 和 headers 已内嵌。
type PresignedURLResult struct {
	// URL 已经带有签名 query string 的完整 URL，可直接使用。
	URL string
	// Headers 是客户端必须带上的额外 HTTP header（例如 Content-Type、X-Amz-Security-Token）。
	Headers []HeaderKV
	// ExpireAt 是该预签名 URL 的过期时间（Unix 时间戳，毫秒）。
	ExpireAt int64
}

// PostFormPresignedResult 是 S3 POST form 上传场景的返回值。
// 客户端构造一个 multipart/form-data 表单：action=URL，加上 formData 里所有键值对，最后附上文件字段 file。
type PostFormPresignedResult struct {
	URL      string
	FormData map[string]string
	Headers  []HeaderKV
	ExpireAt int64
}

// MultipartCompletedPart 描述一次 CompleteMultipartUpload 请求里的每个分片。
// 语义与 S3 CompleteMultipartUpload 完全对齐。
type MultipartCompletedPart struct {
	PartNumber int32
	ETag       string
}

// ObjectInfo 描述一个对象的基础元数据。
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified int64 // Unix ms
	ContentType  string
}

// S3Client 是对象存储适配层的统一接口。
// 所有方法都对 "RustFS / MinIO / AWS S3" 三种后端保持同一语义。
// 任何需要替换底层存储的地方都只需替换这个接口的实现。
type S3Client interface {
	// ============ Multipart Upload ============

	// InitiateMultipartUpload 发起一次分片上传，返回 S3 uploadID。
	// key 是对象在 bucket 内的完整路径，例如 "goim/abc123/file.png"。
	InitiateMultipartUpload(ctx context.Context, key, contentType string) (uploadID string, err error)

	// GenerateUploadPartPresignedURL 为指定分片号生成一个预签名 PUT URL。
	// 客户端拿到后可以直接 PUT part 对应的字节。
	GenerateUploadPartPresignedURL(ctx context.Context, key, uploadID string, partNumber int32) (PresignedURLResult, error)

	// CompleteMultipartUpload 合并所有已上传的分片。
	// completedParts 里的 PartNumber 和 ETag 由客户端在每次 PUT 成功后保存，最终提交。
	CompleteMultipartUpload(ctx context.Context, key, uploadID string, completedParts []MultipartCompletedPart) (finalURL string, err error)

	// AbortMultipartUpload 放弃一次未完成的分片上传，释放已上传分片占用的存储。
	AbortMultipartUpload(ctx context.Context, key, uploadID string) error

	// ============ FormData Upload（小文件直传）============

	// GeneratePostFormPresignedData 生成一次 S3 POST form 上传所需的全部参数。
	// 客户端不需要后端代理，直接把 multipart/form-data POST 到返回的 URL。
	GeneratePostFormPresignedData(ctx context.Context, key, contentType string) (PostFormPresignedResult, error)

	// ============ 下载 ============

	// GenerateGetObjectPresignedURL 生成一个带过期时间的 GET 预签名 URL。
	// expireSeconds 为 0 时使用 Config.PresignExpire。
	GenerateGetObjectPresignedURL(ctx context.Context, key string, expireSeconds int64) (PresignedURLResult, error)

	// HeadObject 拉取对象元数据。
	HeadObject(ctx context.Context, key string) (*ObjectInfo, error)

	// ============ 管理 ============

	// DeleteObject 删除指定对象。
	DeleteObject(ctx context.Context, key string) error

	// ListObjects 列出指定 prefix 下的所有对象。
	ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error)
}
