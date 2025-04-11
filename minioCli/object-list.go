package minioCli

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"strings"
)

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
	// 1. 先通过 StatObject 检查对象存储类型和恢复状态
	stat, err := c.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("StatObject failed: %w", err)
	}
	// 2. 如果对象不是 GLACIER 存储类型，直接下载
	if stat.StorageClass != StorageClassCold {
		return c.minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	}

	// 3. 对象在 GLACIER，查看 RestoreInfo
	if stat.Restore == nil {
		// => 尚未发起过恢复请求
		//    发起一次恢复请求
		restoreErr := c.RestoreObject(ctx, bucketName, objectName, 7)
		if restoreErr != nil {
			return nil, fmt.Errorf("对象 %q 属于 GLACIER，提交恢复请求失败: %w", objectName, restoreErr)
		}
		// 提示用户稍后再来
		return nil, fmt.Errorf("对象 %q 在GLACIER状态，已提交恢复请求，请等待几分钟后再下载", objectName)

	} else {
		// => 已有 RestoreInfo
		//    判断 OngoingRestore 是 true 还是 false
		if stat.Restore.OngoingRestore {
			// 正在恢复中
			return nil, fmt.Errorf("对象 %q 仍在恢复中(OngoingRestore=true)，请等待恢复完成后再下载", objectName)
		} else {
			// OngoingRestore = false => 已恢复并在可读窗口内
			// 直接下载即可
			return c.minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
		}
	}
}

// QueryObjects 根据关键字查询指定存储桶中的对象
// bucketName 为存储桶名称，prefix 为路径前缀，query 为查询关键字
func (c *Client) QueryObjects(ctx context.Context, bucketName, prefix, query string) ([]minio.ObjectInfo, error) {
	objectsCh := c.minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var matchedObjects []minio.ObjectInfo
	for object := range objectsCh {
		if object.Err != nil {
			return nil, object.Err
		}
		// 过滤包含查询关键字的对象名
		if strings.Contains(object.Key, query) {
			matchedObjects = append(matchedObjects, object)
		}
	}
	return matchedObjects, nil
}
