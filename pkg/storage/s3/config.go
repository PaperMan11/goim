package s3

import (
	"os"
	"strconv"
	"time"
)

// 覆盖 RustFS 配置的环境变量名。
// 优先级：环境变量 > 配置文件 > DefaultConfig。
// 主要用于把 AccessKey/SecretKey 从配置文件里剥离，避免密钥入库。
const (
	EnvEndpoint       = "RUSTFS_ENDPOINT"
	EnvPublicEndpoint = "RUSTFS_PUBLIC_ENDPOINT"
	EnvAccessKey      = "RUSTFS_ACCESS_KEY"
	EnvSecretKey      = "RUSTFS_SECRET_KEY"
	EnvRegion         = "RUSTFS_REGION"
	EnvBucket         = "RUSTFS_BUCKET"
	EnvUsePathStyle   = "RUSTFS_USE_PATH_STYLE"
	EnvPresignExpire  = "RUSTFS_PRESIGN_EXPIRE"
)

// Config 封装 S3 兼容对象存储（RustFS、MinIO、AWS S3）的连接参数。
// RustFS/S3 适配层的所有实现都从这里读配置。
type Config struct {
	// Endpoint 是对象存储服务的内部访问地址，例如 http://127.0.0.1:9000
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	// PublicEndpoint 是客户端直传时使用的对外域名，例如 https://files.example.com。
	// 后端用它生成预签名 URL，客户端用它 PUT/GET。
	PublicEndpoint string `json:"publicEndpoint" yaml:"publicEndpoint"`
	// AccessKey 访问密钥 ID
	AccessKey string `json:"accessKey" yaml:"accessKey"`
	// SecretKey 访问密钥 Secret
	SecretKey string `json:"secretKey" yaml:"secretKey"`
	// Region AWS 风格区域号，RustFS/MinIO 可以填任意非空值
	Region string `json:"region" yaml:"region"`
	// Bucket 所有对象存放的 bucket 名
	Bucket string `json:"bucket" yaml:"bucket"`
	// UsePathStyle RustFS / MinIO 需要 path-style（endpoint/bucket/key），
	// AWS S3 默认 virtual-hosted style（bucket.endpoint/key）。
	UsePathStyle bool `json:"usePathStyle" yaml:"usePathStyle"`
	// PresignExpire 预签名 URL 默认过期时间（秒）。
	// 所有通过 GenerateXxxPresignedURL 生成的 URL 都用这个时间。
	PresignExpire time.Duration `json:"presignExpire" yaml:"presignExpire"`
}

// ApplyEnvOverrides 用环境变量覆盖配置字段。
// 只有环境变量非空时才覆盖；UsePathStyle 和 PresignExpire 解析失败时保留原值。
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv(EnvEndpoint); v != "" {
		c.Endpoint = v
	}
	if v := os.Getenv(EnvPublicEndpoint); v != "" {
		c.PublicEndpoint = v
	}
	if v := os.Getenv(EnvAccessKey); v != "" {
		c.AccessKey = v
	}
	if v := os.Getenv(EnvSecretKey); v != "" {
		c.SecretKey = v
	}
	if v := os.Getenv(EnvRegion); v != "" {
		c.Region = v
	}
	if v := os.Getenv(EnvBucket); v != "" {
		c.Bucket = v
	}
	if v := os.Getenv(EnvUsePathStyle); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.UsePathStyle = b
		}
	}
	if v := os.Getenv(EnvPresignExpire); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.PresignExpire = d
		}
	}
}

// DefaultConfig 返回一份开发用的默认配置（127.0.0.1:9000、admin/admin、1 小时过期）。
func DefaultConfig() *Config {
	return &Config{
		Endpoint:       "http://127.0.0.1:9000",
		PublicEndpoint: "http://127.0.0.1:9000",
		AccessKey:      "rustfsadmin",
		SecretKey:      "rustfsadmin",
		Region:         "us-east-1",
		Bucket:         "goim-files",
		UsePathStyle:   true,
		PresignExpire:  time.Hour,
	}
}
