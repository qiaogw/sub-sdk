package minioCli

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// PutObject 将文件上传到指定存储桶中
func (c *Client) PutObject(ctx context.Context,
	bucketName, objectName string, reader io.Reader, size int64, storeClass string) error {
	if len(storeClass) < 1 {
		storeClass = StorageClassLow
	}
	_, err := c.minioClient.PutObject(ctx,
		bucketName, objectName, reader, size, minio.PutObjectOptions{
			ContentType:  "application/octet-stream",
			StorageClass: storeClass,
		})
	return err
}

// PresignedGetObjectURL 返回一个对象的预签名 URL
func (c *Client) PresignedGetObjectURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	// 1. 先通过 StatObject 检查对象存储类型和恢复状态
	stat, err := c.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("StatObject failed: %w", err)
	}
	// 2. 如果对象不是 GLACIER 存储类型，直接下载
	if stat.StorageClass != StorageClassCold {
		return c.doPresignedGetObjectURL(ctx, bucketName, objectName, expiry)
	}

	// 3. 如果是 GLACIER，则检查恢复状态
	if stat.Restore == nil {
		// => 尚未发起过恢复
		if err := c.RestoreObject(ctx, bucketName, objectName, 7); err != nil {
			return "", fmt.Errorf("对象 %q 为 GLACIER，提交恢复请求失败: %w", objectName, err)
		}
		return "", fmt.Errorf("对象 %q 在GLACIER状态，已提交恢复请求，请等待解冻完成后再下载", objectName)

	} else {
		// => 已有 RestoreInfo
		if stat.Restore.OngoingRestore {
			// 正在恢复中
			return "", fmt.Errorf("对象 %q 正在恢复中，请等待完成后再下载", objectName)
		} else {
			// 已恢复
			return c.doPresignedGetObjectURL(ctx, bucketName, objectName, expiry)
		}
	}
}

// doPresignedGetObjectURL 为内部方法，真正生成预签名 URL
func (c *Client) doPresignedGetObjectURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := c.minioClient.PresignedGetObject(ctx, bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("PresignedGetObject failed: %w", err)
	}
	return presignedURL.String(), nil
}

// doGetObjectURL 为内部方法，返回原先的公开访问URL
func (c *Client) doGetObjectURL(bucketName, objectName string) string {
	endpointURL, _ := url.Parse(c.minioClient.EndpointURL().String())
	endpointURL.Path = filepath.Join(endpointURL.Path, bucketName, objectName)
	return endpointURL.String()
}

// GetObjectURL 获取对象的公开访问 URL
func (c *Client) GetObjectURL(ctx context.Context, bucketName, objectName string) (string, error) {
	// 1. 先获取对象信息
	stat, err := c.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("StatObject failed: %w", err)
	}

	// 2. 若不是 GLACIER，直接返回公开URL
	if stat.StorageClass != "GLACIER" {
		return c.doGetObjectURL(bucketName, objectName), nil
	}

	// 3. 若是GLACIER，检查 Restore
	if stat.Restore == nil {
		// 未发起过恢复
		if err := c.RestoreObject(ctx, bucketName, objectName, 1); err != nil {
			return "", fmt.Errorf("对象 %q 为 GLACIER，提交恢复请求失败: %w", objectName, err)
		}
		return "", fmt.Errorf("对象 %q 为GLACIER，已提交恢复请求，请等待解冻完成后再访问公开URL", objectName)
	} else {
		if stat.Restore.OngoingRestore {
			return "", fmt.Errorf("对象 %q 正在恢复中，请等待完成后再访问公开URL", objectName)
		} else {
			// 已恢复
			return c.doGetObjectURL(bucketName, objectName), nil
		}
	}
}

// StateObject 获取对象的状态信息
func (c *Client) StateObject(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	return c.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})

}

// RestoreObject 恢复某个bucket中的冷存储对象文件
func (c *Client) RestoreObject(ctx context.Context, bucketName, objectName string, days int) (err error) {
	ctx1, cancel := context.WithCancel(context.Background())
	defer cancel()
	versionID := ""
	return c.minioClient.RestoreObject(ctx1, bucketName, objectName, versionID, minio.RestoreRequest{
		Days: &days,
		GlacierJobParameters: &minio.GlacierJobParameters{
			Tier: minio.TierExpedited,
			// 取回选项，支持三种取
			//值：[TierExpedited|TierStandard|
			//TierBulk]。
			//Expedited表示快速取回对
			//象，取回耗时1~5 min，
			//Standard表示标准取回对
			//象，取回耗时3~5 h，
			//Bulk表示批量取回对象，
			//取回耗时5~12 h。
			//默认取值为Standard。
		},
	})
}

// RestoreObjectsLife 恢复某个Bucket中的某个目录下的所有冷存储对象文件
func (c *Client) RestoreObjectsLife(bucketName, prefix string, lifeDay int) (count int, err error) {
	logx.Debug("开始恢复文件。。。", prefix)
	if len(prefix) == 0 {
		prefix = "/"
	}
	objectPrefix := strings.TrimPrefix(prefix, `/`)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 配置 ListObjects 参数
	listOpts := minio.ListObjectsOptions{
		Prefix:    objectPrefix,
		Recursive: true, // 你是否需要递归列出子目录
	}
	// 遍历对象列表
	objectCh := c.minioClient.ListObjects(ctx, bucketName, listOpts)
	for object := range objectCh {
		if object.Err != nil {
			// 记录遍历时出现的错误
			logx.Errorf("ListObjects error: %v", object.Err)
			// 可以选择中断，也可以选择跳过
			err = object.Err
			continue
		}

		// 判断存储类型是否为 GLACIER（或其他归档类型）
		if object.StorageClass == "GLACIER" {
			count++
			restoreErr := c.RestoreObject(ctx, bucketName, object.Key, lifeDay)
			if restoreErr != nil {
				// 如果只想统计多少成功或不关心错误，可以不处理
				logx.Errorf("RestoreObject error: key=%s, err=%v", object.Key, restoreErr)
			}
		}
	}

	logx.Infof("✅ 本次共恢复的对象 %d 个", count)
	return
}
