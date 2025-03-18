package minioCli

import (
	"context"
	"github.com/minio/minio-go/v7"
)

// ListIncompleteUploads 检查或列举未完成的分段上传
// prefix 参数用于指定列出对象的路径，默认为根目录
func (c *Client) ListIncompleteUploads(ctx context.Context, bucketName, prefix string) ([]minio.ObjectMultipartInfo, error) {
	var resp []minio.ObjectMultipartInfo
	for obj := range c.minioClient.ListIncompleteUploads(ctx, bucketName, prefix, true) {
		if obj.Err != nil {
			// 处理报错
			return nil, obj.Err
		}
		resp = append(resp, obj)
	}
	return resp, nil
}

// RemoveIncompleteUpload 删除未完成的分段
// prefix 参数用于指定列出对象的路径，默认为根目录
func (c *Client) RemoveIncompleteUpload(ctx context.Context, bucketName, prefix string) error {
	for obj := range c.minioClient.ListIncompleteUploads(ctx, bucketName, prefix, true) {
		if obj.Err != nil {
			// 处理报错
			return obj.Err
		}
		//// 若确实存在分段，调用 RemoveIncompleteUpload 清理
		err := c.minioClient.RemoveIncompleteUpload(ctx, bucketName, obj.Key)
		if err != nil {
			return err
		}
	}
	return nil
}

// ClearPrefix 清理目录，删除该目录所有对象，包括清理未完成的分段
// prefix 参数用于指定列出对象的路径，默认为根目录
func (c *Client) ClearPrefix(ctx context.Context, bucketName, prefix string) error {
	for obj := range c.minioClient.ListIncompleteUploads(ctx, bucketName, prefix, true) {
		if obj.Err != nil {
			// 处理报错
			return obj.Err
		}
		//// 若确实存在分段，调用 RemoveIncompleteUpload 清理
		err := c.minioClient.RemoveIncompleteUpload(ctx, bucketName, obj.Key)
		if err != nil {
			return err
		}
	}
	return c.RemoveObjectsByPrefix(ctx, bucketName, prefix)
}
