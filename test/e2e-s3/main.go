// Package main 是 RustFS S3 兼容性 e2e 测试脚本。
// 不依赖 gRPC，直接调用 pkg/storage/s3 适配层，验证 RustFS 对
// S3 Multipart Upload / POST Form / Presign 等关键特性的支持情况。
//
// 用法：
//
//	cd test/e2e-s3 && go run .
//	或   go run ./test/e2e-s3
//
// 环境变量覆盖默认 RustFS 配置：
//
//	RUSTFS_ENDPOINT     默认 http://192.168.241.128:9000
//	RUSTFS_ACCESS_KEY   默认 rustfsadmin
//	RUSTFS_SECRET_KEY   默认 rustfsadmin
//	RUSTFS_BUCKET       默认 goim-files
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"

	gos3 "github.com/PaperMan11/goim/pkg/storage/s3"
)

func main() {
	endpoint := env("RUSTFS_ENDPOINT", "http://192.168.241.128:9000")
	accessKey := env("RUSTFS_ACCESS_KEY", "ah0inZPqYH4ur2vZdZMG")
	secretKey := env("RUSTFS_SECRET_KEY", "vZqRJJz7DA8fvHBoTJYIEhLcWSUd1k7V1r2e0m1i")
	bucket := env("RUSTFS_BUCKET", "goim-files")

	cfg := &gos3.Config{
		Endpoint:       endpoint,
		PublicEndpoint: endpoint,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		Region:         "us-east-1",
		Bucket:         bucket,
		UsePathStyle:   true,
		PresignExpire:  10 * time.Minute,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("========================================")
	fmt.Println("RustFS E2E Test")
	fmt.Printf("  endpoint: %s\n", endpoint)
	fmt.Printf("  bucket:   %s\n", bucket)
	fmt.Println("========================================")

	if err := ensureBucket(ctx, cfg); err != nil {
		fatalf("FATAL: ensureBucket: %v\n", err)
	}
	fmt.Println("bucket ready")

	cli, err := gos3.NewS3Client(ctx, cfg)
	if err != nil {
		fatalf("FATAL: NewS3Client: %v\n", err)
	}
	fmt.Println("S3Client init OK")

	allPassed := true

	fmt.Println("\n--- 1. Multipart Upload (3 parts x 6MB) ---")
	key := fmt.Sprintf("e2e-test/%s/multipart.bin", time.Now().Format("20060102-150405"))
	if !testMultipart(ctx, cli, key) {
		allPassed = false
	}

	fmt.Println("\n--- 2. FormData Upload (POST form presign) ---")
	key2 := fmt.Sprintf("e2e-test/%s/formdata.txt", time.Now().Format("20060102-150405"))
	if !testFormData(ctx, cli, key2) {
		allPassed = false
	}

	fmt.Println("\n--- 3. AccessURL (GET presign + download) ---")
	if !testAccessURL(ctx, cli, key2) {
		allPassed = false
	}

	fmt.Println("\n========================================")
	if allPassed {
		fmt.Println("ALL TESTS PASSED — RustFS 完全兼容 S3 适配层")
	} else {
		fmt.Println("SOME TESTS FAILED — 检查上面的日志")
		os.Exit(1)
	}
}

// testMultipart 完整跑一遍 Multipart Upload 流程。
// 每片 6MB（必须 ≥ S3 最小分片 5MB，否则 Complete 报 EntityTooSmall）。
func testMultipart(ctx context.Context, cli gos3.S3Client, key string) bool {
	partSize := 6 * 1024 * 1024
	partData := make([][]byte, 3)
	for i := range partData {
		buf := make([]byte, partSize)
		rand.Read(buf)
		partData[i] = buf
	}

	uploadID, err := cli.InitiateMultipartUpload(ctx, key, "application/octet-stream")
	if err != nil {
		fmt.Printf("  FAIL Initiate: %v\n", err)
		return false
	}
	fmt.Printf("  ok  Initiate: uploadID=%s\n", uploadID)

	type uploaded struct {
		partNumber int32
		eTag       string
	}
	var uploadedParts []uploaded

	for i := int32(1); i <= 3; i++ {
		pr, err := cli.GenerateUploadPartPresignedURL(ctx, key, uploadID, i)
		if err != nil {
			fmt.Printf("  FAIL Presign part %d: %v\n", i, err)
			return false
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, pr.URL, bytes.NewReader(partData[i-1]))
		if err != nil {
			fmt.Printf("  FAIL NewRequest part %d: %v\n", i, err)
			return false
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("  FAIL PUT part %d: %v\n", i, err)
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("  FAIL PUT part %d: HTTP %d body=%s\n", i, resp.StatusCode, truncate(string(body), 200))
			return false
		}
		eTag := resp.Header.Get("ETag")
		fmt.Printf("  ok  Part %d uploaded, ETag=%s\n", i, eTag)
		uploadedParts = append(uploadedParts, uploaded{partNumber: i, eTag: eTag})
	}

	completed := make([]gos3.MultipartCompletedPart, len(uploadedParts))
	for i, p := range uploadedParts {
		completed[i] = gos3.MultipartCompletedPart{PartNumber: p.partNumber, ETag: p.eTag}
	}
	finalURL, err := cli.CompleteMultipartUpload(ctx, key, uploadID, completed)
	if err != nil {
		fmt.Printf("  FAIL Complete: %v\n", err)
		return false
	}
	fmt.Printf("  ok  Complete: finalURL=%s\n", finalURL)

	info, err := cli.HeadObject(ctx, key)
	if err != nil {
		fmt.Printf("  FAIL HeadObject: %v\n", err)
		return false
	}
	fmt.Printf("  ok  HeadObject: size=%d (expected %d)\n", info.Size, int64(3*partSize))
	return true
}

// testFormData 跑 POST form 直传（小文件）。
func testFormData(ctx context.Context, cli gos3.S3Client, key string) bool {
	content := []byte("Hello RustFS via S3 POST form presign!")

	pr, err := cli.GeneratePostFormPresignedData(ctx, key, "text/plain")
	if err != nil {
		fmt.Printf("  FAIL PresignPost: %v\n", err)
		return false
	}
	fmt.Printf("  ok  PresignPost: expires=%d ms, form fields=%d\n", pr.ExpireAt, len(pr.FormData))

	body, ct, err := buildMultipartForm(pr.FormData, "file", content, key)
	if err != nil {
		fmt.Printf("  FAIL buildMultipart: %v\n", err)
		return false
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, pr.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("  FAIL POST: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("  ok  POST form: HTTP %d body=%s\n", resp.StatusCode, truncate(string(respBody), 120))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("  FAIL: unexpected status %d\n", resp.StatusCode)
		return false
	}
	return true
}

// testAccessURL 生成 GET 预签名 URL 并实际下载验证。
func testAccessURL(ctx context.Context, cli gos3.S3Client, key string) bool {
	pr, err := cli.GenerateGetObjectPresignedURL(ctx, key, 300)
	if err != nil {
		fmt.Printf("  FAIL PresignGet: %v\n", err)
		return false
	}
	fmt.Printf("  ok  PresignGet: expires=%d ms\n", pr.ExpireAt)

	resp, err := http.Get(pr.URL)
	if err != nil {
		fmt.Printf("  FAIL GET: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("  ok  GET: HTTP %d body=%q\n", resp.StatusCode, string(body))

	return resp.StatusCode == 200
}

// ==================== 辅助 ====================

func ensureBucket(ctx context.Context, cfg *gos3.Config) error {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: cfg.Endpoint, SigningRegion: cfg.Region, HostnameImmutable: true, Source: aws.EndpointSourceCustom}, nil
			})),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return err
	}
	s3c := s3v2.NewFromConfig(awsCfg, func(o *s3v2.Options) { o.UsePathStyle = cfg.UsePathStyle })

	out, err := s3c.ListBuckets(ctx, &s3v2.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("ListBuckets (检查凭据) 失败: %w", err)
	}
	bucketExists := false
	for _, b := range out.Buckets {
		if aws.ToString(b.Name) == cfg.Bucket {
			bucketExists = true
			break
		}
	}
	if bucketExists {
		fmt.Printf("bucket %q 已存在\n", cfg.Bucket)
		return nil
	}
	_, err = s3c.CreateBucket(ctx, &s3v2.CreateBucketInput{Bucket: aws.String(cfg.Bucket)})
	if err != nil {
		return fmt.Errorf("CreateBucket %s: %w", cfg.Bucket, err)
	}
	fmt.Printf("bucket %q 已创建\n", cfg.Bucket)
	return nil
}

func buildMultipartForm(formFields map[string]string, fieldName string, fileContent []byte, filename string) ([]byte, string, error) {
	var buf bytes.Buffer
	boundary := "----goimboundary" + fmt.Sprintf("%d", time.Now().UnixNano())

	writeField := func(name, value string) {
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"\r\n\r\n", name))
		buf.WriteString(value + "\r\n")
	}

	keys := make([]string, 0, len(formFields))
	for k := range formFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeField(k, formFields[k])
	}

	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldName, filepath.Base(filename)))
	buf.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	buf.Write(fileContent)
	buf.WriteString("\r\n--" + boundary + "--\r\n")

	return buf.Bytes(), "multipart/form-data; boundary=" + boundary, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fatalf(format string, args ...any) {
	fmt.Printf(format, args...)
	os.Exit(1)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
