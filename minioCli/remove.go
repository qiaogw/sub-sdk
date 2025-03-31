package minioCli

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
	"time"
)

// RemoveObject 删除指定存储桶中的对象
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string) error {
	err := c.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	return err
}

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
				logx.Errorf("列举出错: %v\n", object.Err)
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
			errMsgs = append(errMsgs, msg)
		} else {
			logx.Infof("✅ 删除对象: %s\n", removeResp.ObjectName)
		}
	}
	// 循环完毕，看是否有错误
	if len(errMsgs) > 0 {
		return fmt.Errorf("some objects failed to remove:\n%s", strings.Join(errMsgs, "\n"))
	}
	return nil
}

// RemoveExpiredObjectsByPrefix 删除指定存储桶中指定路径下已超过 lifeDay 天的对象
func (c *Client) RemoveExpiredObjectsByPrefix(ctx context.Context, bucketName, prefix string, lifeDay float64) error {
	// 去除前导斜杠（如果需要）
	prefix = strings.TrimPrefix(prefix, "/")

	// 1) 列举对象，过滤出超过 lifeDay 天的对象，打包到一个 channel
	objectInfoCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectInfoCh)
		objectsCh := c.minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		})
		for object := range objectsCh {
			if object.Err != nil {
				logx.Errorf("列举对象出错: %v\n", object.Err)
				// 出错时这里记录错误后继续处理后续对象
				continue
			}
			// 判断对象是否超过 lifeDay 天
			if time.Since(object.LastModified).Hours() > lifeDay*24 {
				objectInfoCh <- object
			}
		}
	}()

	// 2) 使用 RemoveObjectsWithResult 批量删除筛选后的对象
	removeCh := c.minioClient.RemoveObjectsWithResult(ctx, bucketName, objectInfoCh, minio.RemoveObjectsOptions{})

	// 3) 遍历删除结果，记录错误信息
	var errMsgs []string
	for removeResp := range removeCh {
		if removeResp.Err != nil {
			msg := fmt.Sprintf("Failed to remove %s: %v", removeResp.ObjectName, removeResp.Err)
			errMsgs = append(errMsgs, msg)
		} else {
			logx.Infof("✅ 删除对象: %s", removeResp.ObjectName)
		}
	}

	// 如有删除失败的对象，返回错误信息
	if len(errMsgs) > 0 {
		return fmt.Errorf("some objects failed to remove:\n%s", strings.Join(errMsgs, "\n"))
	}
	return nil
}
