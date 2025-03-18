package minioCli

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

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

// CreateBucket 创建一个新的存储桶
func (c *Client) CreateBucket(ctx context.Context, bucketName string) error {
	err := c.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	return err
}

// ListBuckets 列出所有存储桶
func (c *Client) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	buckets, err := c.minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	return buckets, nil
}

// BucketExists 检查指定存储桶是否存在
func (c *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	exists, err := c.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// RemoveBucket 删除指定存储桶及其内所有对象
func (c *Client) RemoveBucket(ctx context.Context, bucketName string) error {
	err := c.minioClient.RemoveBucket(ctx, bucketName)
	return err
}

// ListObjects 列出指定存储桶中的对象
// prefix 参数用于指定列出对象的路径，默认为根目录
func (c *Client) ListObjects(ctx context.Context, bucketName, prefix string) ([]minio.ObjectInfo, error) {
	objectsCh := c.minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix: prefix,
	})
	var objects []minio.ObjectInfo
	for object := range objectsCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

// GetObject 获取指定存储桶中的对象
func (c *Client) GetObject(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	obj, err := c.minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// PutObject 将文件上传到指定存储桶中
func (c *Client) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64) error {

	_, err := c.minioClient.PutObject(context.Background(),
		bucketName, objectName, reader, size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

// RemoveObject 删除指定存储桶中的对象
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string) error {
	err := c.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	return err
}

//// RemoveObjectsByPrefix 删除指定存储桶中指定路径下的所有对象
//func (c *Client) RemoveObjectsByPrefix(ctx context.Context, bucketName, prefix string) error {
//	doneCh := make(chan struct{})
//	defer close(doneCh)
//
//	objectsCh := c.minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
//		Prefix:    prefix,
//		Recursive: true,
//	})
//	for object := range objectsCh {
//		if object.Err != nil {
//			return object.Err
//		}
//		err := c.minioClient.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{})
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}

// RemoveObjectsByPrefix 删除指定存储桶中指定路径下的所有对象
func (c *Client) RemoveObjectsByPrefix(ctx context.Context, bucketName, prefix string) error {
	// 1) 列举出所有对象，打包到一个 channel
	objectInfoCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectInfoCh)
		objectsCh := c.minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		})
		for object := range objectsCh {
			// 如果列举出错，要么直接退出，要么记录后续再处理
			if object.Err != nil {
				fmt.Printf("List error: %v\n", object.Err)
				return
			}
			// 把 Key 放到管道中
			objectInfoCh <- minio.ObjectInfo{Key: object.Key}
		}
	}()

	// 2) 使用 RemoveObjectsWithResult 批量删除
	removeCh := c.minioClient.RemoveObjectsWithResult(ctx, bucketName, objectInfoCh, minio.RemoveObjectsOptions{})

	// 3) 遍历删除结果
	var errMsgs []string
	for removeResp := range removeCh {
		if removeResp.Err != nil {
			// 这里可以仅记录错误，不立刻 return，以保证后续对象继续删除
			// 如果想一旦失败就退出，也可以直接 return
			msg := fmt.Sprintf("Failed to remove %s: %v", removeResp.ObjectName, removeResp.Err)
			fmt.Println(msg) // 先打印日志
			errMsgs = append(errMsgs, msg)
		} else {
			fmt.Printf("Removed: %s\n", removeResp.ObjectName)
		}
	}
	// 循环完毕，看是否有错误
	if len(errMsgs) > 0 {
		return fmt.Errorf("some objects failed to remove:\n%s", strings.Join(errMsgs, "\n"))
	}
	return nil
}

// PresignedGetObjectURL 返回一个对象的预签名 URL
func (c *Client) PresignedGetObjectURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := c.minioClient.PresignedGetObject(ctx, bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

// GetObjectURL 获取对象的公开访问 URL
func (c *Client) GetObjectURL(bucketName, objectName string) string {
	endpointURL, _ := url.Parse(c.minioClient.EndpointURL().String())
	endpointURL.Path = filepath.Join(endpointURL.Path, bucketName, objectName)
	return endpointURL.String()
}

// StateObject 获取对象的状态信息
func (c *Client) StateObject(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	return c.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})

}

// Close 关闭 ms3 客户端
func (c *Client) Close() error {
	// 执行任何必要的清理操作
	return nil
}
