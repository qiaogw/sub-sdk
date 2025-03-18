package minioCli

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/minio/minio-go/v7"
	"log"
	"time"
)

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

// BucketDetail 封装一个桶的常见信息
type BucketDetail struct {
	// 基础信息
	Name         string
	CreationTime time.Time
	Region       string

	// 配置项
	Versioning   string // Versioning 状态，如 "Enabled", "Suspended"...
	Policy       string // 存放桶策略 JSON
	Encryption   string // 存放桶加密配置 (XML 或 JSON)
	Lifecycle    string // 存放桶的 Lifecycle 配置 (XML)
	Tagging      string // "k1=v1&k2=v2" 等
	Notification string // Notification 配置的结构或序列化后的字符串

	// 其他可扩展
	// StorageClass   string // 如果云厂商提供了“桶级默认存储类”，可放在这儿
	// ObjectLock     string // 若支持 Object Lock，可在此存放 Lock 信息
}

// GetBucketDetail 获取单个桶的各种配置信息
func (c *Client) GetBucketDetail(ctx context.Context, bucketName string) (*BucketDetail, error) {
	detail := &BucketDetail{Name: bucketName}

	// 1. 获取 CreationTime (S3 并无单独 API，可从 ListBuckets 中匹配)
	buckets, err := c.minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	var found bool
	for _, b := range buckets {
		if b.Name == bucketName {
			detail.CreationTime = b.CreationDate
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("bucket %q not found", bucketName)
	}

	// 2. Region
	//    如果桶不存在或无权限，GetBucketLocation 也会出错
	region, err := c.minioClient.GetBucketLocation(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get region for bucket %q: %w", bucketName, err)
	}
	detail.Region = region

	// 3. Versioning
	versioningCfg, err := c.minioClient.GetBucketVersioning(ctx, bucketName)
	if err != nil {
		// 若不支持版本控制或权限不足，可能出错，这里仅记录日志
		log.Printf("GetBucketVersioning error for %q: %v\n", bucketName, err)
	} else {
		detail.Versioning = versioningCfg.Status // "Enabled", "Suspended", ""
	}

	// 4. Policy (访问策略)
	//    如果桶无策略或你无权限，也可能出错
	policyStr, err := c.minioClient.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		log.Printf("GetBucketPolicy error for %q: %v\n", bucketName, err)
	} else {
		detail.Policy = policyStr
	}
	// 5. Encryption (加密配置)
	//    返回 XML；若桶未启用加密，常见报错：The server side encryption configuration was not found
	encryptionCfg, err := c.minioClient.GetBucketEncryption(ctx, bucketName)
	data, err := xml.Marshal(encryptionCfg)
	if err != nil {
		log.Printf("GetBucketEncryption error for %q: %v\n", bucketName, err)
	} else {
		detail.Encryption = string(data)
	}
	// 6. Lifecycle (生命周期配置)
	//    若未配置 Lifecycle，也会报错
	lifecycleCfg, err := c.minioClient.GetBucketLifecycle(ctx, bucketName)
	data, err = xml.Marshal(lifecycleCfg)
	if err != nil {
		log.Printf("GetBucketLifecycle error for %q: %v\n", bucketName, err)
	} else {
		detail.Lifecycle = string(data)
	}

	// 7. Tagging (标签)
	//    返回类似 "Key1=Value1&Key2=Value2"；无标签会报错
	tagging, err := c.minioClient.GetBucketTagging(ctx, bucketName)
	if err != nil {
		log.Printf("GetBucketTagging error for %q: %v\n", bucketName, err)
	} else {
		detail.Tagging = tagging.String()
	}

	// 8. Notification (事件通知配置)
	//    返回一个结构体，可序列化成 JSON 或字符串
	notificationCfg, err := c.minioClient.GetBucketNotification(ctx, bucketName)
	if err != nil {
		log.Printf("GetBucketNotification error for %q: %v\n", bucketName, err)
	} else {
		detail.Notification = fmt.Sprintf("%+v", notificationCfg)
		// 或者用 json.Marshal(notificationCfg) 转成 JSON
	}

	return detail, nil
}
