package filex

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"github.com/minio/minio-go/v7"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CompressAndUpload 实现边压缩边上传
// 参数：minioClient( minio 客户端)、bucketName(桶名称)
// srcPath（文件或目录路径）、objectName（S3中存储对象名称，即压缩文件名）、level（可选，若未指定则默认为 5）
// 返回值：objectName，错误
func CompressAndUpload(ctx context.Context, minioClient *minio.Client, bucketName, srcPath, objectName string, level ...int) (string, error) {
	// 解析可选的压缩级别参数，默认级别为 5
	encLevel := 5
	if len(level) > 0 {
		encLevel = level[0]
	}
	// 如果传入的 objectName 没有后缀或者后缀不是 ".zst"，则追加 ".zst"
	if ext := filepath.Ext(objectName); ext != ".zst" {
		objectName += ".zst"
	}
	// 利用 io.Pipe 实现流式传输：compressor 写入 pipe，minio 读取 pipe
	pr, pw := io.Pipe()

	// 启动 goroutine 执行压缩操作，将数据写入 pw
	go func() {
		defer pw.Close()
		// 创建 zstd 压缩器，设置压缩级别及并发数（利用所有 CPU 核心）
		encoder, err := zstd.NewWriter(pw,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(encLevel)),
			zstd.WithEncoderConcurrency(runtime.NumCPU()),
		)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("创建 zstd writer 失败: %v", err))
			return
		}
		defer encoder.Close()

		// 获取源文件信息
		info, err := os.Stat(srcPath)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("获取源路径信息失败: %v", err))
			return
		}

		if info.IsDir() {
			// 如果是目录，先打包成 tar 格式，再写入 zstd 压缩流
			tw := tar.NewWriter(encoder)
			defer tw.Close()
			err = filepath.Walk(srcPath, func(file string, fi fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// 对于符号链接，需要调用 Lstat 而非 Stat（Walk 已经用 Lstat 了）
				// 生成 tar 头信息
				header, err := tar.FileInfoHeader(fi, "")
				if err != nil {
					return err
				}
				// 保证使用相对路径（不带上父目录）
				relPath, err := filepath.Rel(filepath.Dir(srcPath), file)
				if err != nil {
					return err
				}
				header.Name = relPath

				// 处理符号链接
				if fi.Mode()&os.ModeSymlink != 0 {
					linkTarget, err := os.Readlink(file)
					if err != nil {
						return err
					}
					header.Typeflag = tar.TypeSymlink
					header.Linkname = linkTarget
					// 写入头信息后直接返回
					return tw.WriteHeader(header)
				}

				if err := tw.WriteHeader(header); err != nil {
					return err
				}

				// 如果是目录，不需要写入数据
				if fi.IsDir() {
					return nil
				}

				// 打开文件并写入数据
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				pw.CloseWithError(fmt.Errorf("打包目录失败: %v", err))
				return
			}
		} else {
			// 如果是单个文件，直接复制文件内容
			inFile, err := os.Open(srcPath)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("打开源文件失败: %v", err))
				return
			}
			defer inFile.Close()
			if _, err := io.Copy(encoder, inFile); err != nil {
				pw.CloseWithError(fmt.Errorf("写入压缩数据失败: %v", err))
				return
			}
		}
	}()

	// 使用 minioClient 上传，PutObject 接受 io.Reader 流式上传
	// 由于数据流未知大小，可以将 size 设置为 -1，并设置 PutObjectOptions 中的 PartSize
	uploadInfo, err := minioClient.PutObject(ctx, bucketName, objectName, pr, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", fmt.Errorf("上传失败: %v", err)
	}
	fmt.Printf("上传成功: %s, 大小: %d\n", uploadInfo.Key, uploadInfo.Size)
	return objectName, nil
}

// DownloadAndDecompress 实现边下载边解压缩
// 参数：minioClient( minio 客户端)、bucketName(桶名称)
// objectName（S3中的压缩对象名称，即压缩文件名）、dstDir（解压目标目录，单文件时为目标文件路径）
// 返回值：dstDir，错误
func DownloadAndDecompress(minioClient *minio.Client, bucketName, objectName, dstDir string) (string, error) {
	// 通过 minioClient 获取对象（返回 io.Reader）
	object, err := minioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("下载对象失败: %v", err)
	}
	defer object.Close()

	// 创建 zstd 解压器，从 object 流中解压
	decoder, err := zstd.NewReader(object)
	if err != nil {
		return "", fmt.Errorf("创建 zstd reader 失败: %v", err)
	}
	defer decoder.Close()

	// 尝试将解压数据作为 tar 包处理
	tarReader := tar.NewReader(decoder)
	// 读取第一个 tar header 判断是否为 tar 包
	_, err = tarReader.Next()
	if err == nil {
		// 如果能读取到 tar header，认为是 tar 包
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return "", fmt.Errorf("创建解压目录失败: %v", err)
		}
		// 由于 tarReader 已经提前读取了第一个 header，需要重建 tarReader
		// 这里采用将解压后的数据缓存到内存的方式（数据量较小时可行，若数据较大，可考虑其他方式）
		decodedData, err := io.ReadAll(decoder)
		if err != nil {
			return "", fmt.Errorf("读取解压数据失败: %v", err)
		}
		tarReader = tar.NewReader(bytes.NewReader(decodedData))
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("读取 tar 条目失败: %v", err)
			}
			// 构造目标路径，并防止路径穿越攻击
			targetPath := filepath.Join(dstDir, header.Name)
			if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(dstDir)+string(os.PathSeparator)) {
				return "", fmt.Errorf("非法文件路径: %s", targetPath)
			}

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
					return "", err
				}
			case tar.TypeReg:
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return "", err
				}
				outFile, err := os.Create(targetPath)
				if err != nil {
					return "", err
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					outFile.Close()
					return "", err
				}
				outFile.Close()
			case tar.TypeSymlink:
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return "", err
				}
				if err := os.Symlink(header.Linkname, targetPath); err != nil {
					return "", err
				}
			default:
				// 可根据需要处理其他类型
			}
		}
	} else {
		// 如果不是 tar 包，则认为是单个文件压缩结果
		if err := os.MkdirAll(filepath.Dir(dstDir), 0755); err != nil {
			return "", fmt.Errorf("创建目标目录失败: %v", err)
		}
		outFile, err := os.Create(dstDir)
		if err != nil {
			return "", fmt.Errorf("创建输出文件失败: %v", err)
		}
		defer outFile.Close()
		if _, err := io.Copy(outFile, decoder); err != nil {
			return "", fmt.Errorf("写入输出文件失败: %v", err)
		}
	}

	return dstDir, nil
}
