package minioCli

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client 封装了与 S3 兼容存储服务交互的方法
type Client struct {
	minioClient *minio.Client
}

// NewClient 创建一个新的 ms3 客户端
func NewClient(endpoint, accessKeyID, secretAccessKey string, useSSL bool) (*Client, error) {
	// 初始化 MinIO 客户端对象
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		minioClient: minioClient,
	}, nil
}

// Close 关闭 ms3 客户端
func (c *Client) Close() error {
	// 执行任何必要的清理操作
	return nil
}
