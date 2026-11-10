package s3

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	// ErrNilConfig 表示 NewS3Client 收到了 nil 配置。
	ErrNilConfig = errors.New("s3: config is nil")
	// ErrEmptyBucket 表示 bucket 未配置。
	ErrEmptyBucket = errors.New("s3: bucket is empty")
)

// s3Impl 是基于 AWS SDK v2 的 S3Client 实现。
// RustFS / MinIO / AWS S3 三者都使用这份实现，通过 Config 的 Endpoint / UsePathStyle 区分。
type s3Impl struct {
	cfg    *Config
	client *s3v2.Client
}

// NewS3Client 构造一个连接到 RustFS / MinIO / AWS S3 的 S3Client。
// cfg 为空或 Bucket 为空时返回错误。
func NewS3Client(ctx context.Context, cfg *Config) (S3Client, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}
	if cfg.Bucket == "" {
		return nil, ErrEmptyBucket
	}
	if cfg.PresignExpire == 0 {
		cfg.PresignExpire = time.Hour
	}

	endpointResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
				Source:            aws.EndpointSourceCustom,
			}, nil
		})

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(endpointResolver),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: load aws config: %w", err)
	}

	s3Client := s3v2.NewFromConfig(awsCfg, func(o *s3v2.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &s3Impl{cfg: cfg, client: s3Client}, nil
}

// ==================== Multipart Upload ====================

func (s *s3Impl) InitiateMultipartUpload(ctx context.Context, key, contentType string) (string, error) {
	out, err := s.client.CreateMultipartUpload(ctx, &s3v2.CreateMultipartUploadInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("s3: CreateMultipartUpload: %w", err)
	}
	return aws.ToString(out.UploadId), nil
}

func (s *s3Impl) GenerateUploadPartPresignedURL(ctx context.Context, key, uploadID string, partNumber int32) (PresignedURLResult, error) {
	presignClient := s3v2.NewPresignClient(s.client)
	req, err := presignClient.PresignUploadPart(ctx, &s3v2.UploadPartInput{
		Bucket:     aws.String(s.cfg.Bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
	}, s3v2.WithPresignExpires(s.cfg.PresignExpire))
	if err != nil {
		return PresignedURLResult{}, fmt.Errorf("s3: PresignUploadPart: %w", err)
	}
	return fromPresignedHTTPRequest(req, s.cfg.PresignExpire), nil
}

func (s *s3Impl) CompleteMultipartUpload(ctx context.Context, key, uploadID string, completedParts []MultipartCompletedPart) (string, error) {
	parts := make([]types.CompletedPart, 0, len(completedParts))
	for _, p := range completedParts {
		parts = append(parts, types.CompletedPart{
			PartNumber: aws.Int32(p.PartNumber),
			ETag:       aws.String(p.ETag),
		})
	}
	out, err := s.client.CompleteMultipartUpload(ctx, &s3v2.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.cfg.Bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("s3: CompleteMultipartUpload: %w", err)
	}

	finalKey := aws.ToString(out.Key)
	return s.buildPublicURL(finalKey), nil
}

func (s *s3Impl) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx, &s3v2.AbortMultipartUploadInput{
		Bucket:   aws.String(s.cfg.Bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return fmt.Errorf("s3: AbortMultipartUpload: %w", err)
	}
	return nil
}

// ==================== FormData Upload ====================

func (s *s3Impl) GeneratePostFormPresignedData(ctx context.Context, key, contentType string) (PostFormPresignedResult, error) {
	presignClient := s3v2.NewPresignClient(s.client)
	presigned, err := presignClient.PresignPostObject(ctx, &s3v2.PutObjectInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(o *s3v2.PresignPostOptions) { o.Expires = s.cfg.PresignExpire })
	if err != nil {
		return PostFormPresignedResult{}, fmt.Errorf("s3: PresignPostObject: %w", err)
	}

	// 注意：AWS SDK v2 的 PresignPostObject 在 V1 EndpointResolver 模式下返回的 URL 是空字符串。
	// 这里手动拼 path-style 的 POST action URL（endpoint/bucket），与 Multipart 的 presign 保持一致。
	postURL := presigned.URL
	if postURL == "" {
		base := s.cfg.Endpoint
		for len(base) > 0 && base[len(base)-1] == '/' {
			base = base[:len(base)-1]
		}
		postURL = fmt.Sprintf("%s/%s", base, s.cfg.Bucket)
	}

	return PostFormPresignedResult{
		URL:      postURL,
		FormData: presigned.Values,
		ExpireAt: timex.Now().Add(s.cfg.PresignExpire).UnixMilli(),
	}, nil
}

// ==================== 下载 ====================

func (s *s3Impl) GenerateGetObjectPresignedURL(ctx context.Context, key string, expireSeconds int64) (PresignedURLResult, error) {
	expire := s.cfg.PresignExpire
	if expireSeconds > 0 {
		expire = time.Duration(expireSeconds) * time.Second
	}
	presignClient := s3v2.NewPresignClient(s.client)
	req, err := presignClient.PresignGetObject(ctx, &s3v2.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	}, s3v2.WithPresignExpires(expire))
	if err != nil {
		return PresignedURLResult{}, fmt.Errorf("s3: PresignGetObject: %w", err)
	}
	return fromPresignedHTTPRequest(req, expire), nil
}

func (s *s3Impl) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3v2.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: HeadObject: %w", err)
	}
	var lastModified int64
	if out.LastModified != nil {
		lastModified = out.LastModified.UnixMilli()
	}
	return &ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(out.ContentLength),
		LastModified: lastModified,
		ContentType:  aws.ToString(out.ContentType),
	}, nil
}

// ==================== 管理 ====================

func (s *s3Impl) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3v2.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3: DeleteObject: %w", err)
	}
	return nil
}

func (s *s3Impl) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var results []ObjectInfo
	paginator := s3v2.NewListObjectsV2Paginator(s.client, &s3v2.ListObjectsV2Input{
		Bucket: aws.String(s.cfg.Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("s3: ListObjectsV2: %w", err)
		}
		for _, obj := range page.Contents {
			var lm int64
			if obj.LastModified != nil {
				lm = obj.LastModified.UnixMilli()
			}
			results = append(results, ObjectInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: lm,
			})
		}
	}
	return results, nil
}

// ==================== 内部辅助 ====================

func fromPresignedHTTPRequest(req *v4.PresignedHTTPRequest, expire time.Duration) PresignedURLResult {
	var kvs []HeaderKV
	for k, vs := range req.SignedHeader {
		kvs = append(kvs, HeaderKV{Key: k, Values: vs})
	}
	return PresignedURLResult{
		URL:      req.URL,
		Headers:  kvs,
		ExpireAt: timex.Now().Add(expire).UnixMilli(),
	}
}

// buildPublicURL 用 PublicEndpoint 拼出 bucket/key 的完整访问 URL。
func (s *s3Impl) buildPublicURL(key string) string {
	base := s.cfg.PublicEndpoint
	if base == "" {
		base = s.cfg.Endpoint
	}
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	return fmt.Sprintf("%s/%s/%s", base, s.cfg.Bucket, key)
}
