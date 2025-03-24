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
	versionID := ""
	return c.minioClient.RestoreObject(ctx, bucketName, objectName, versionID, minio.RestoreRequest{
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

// RestoreObjectsLife 恢复某个Bucket中的所有冷存储对象文件
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

	logx.Infof("本次共发现需要恢复的对象 %d 个", count)
	return
}
