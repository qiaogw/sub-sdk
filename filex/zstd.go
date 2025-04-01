package filex

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CompressZstd 压缩函数，支持可选的压缩级别
// 参数：srcPath（文件或目录路径），dstPath（压缩文件保存路径）、level（可选，若未指定则默认为 5）
// 返回值：dstPath（压缩文件名），error（错误）
func CompressZstd(srcPath, dstPath string, level ...int) (string, error) {
	// 解析可选的压缩级别参数，默认级别为 5
	encLevel := 5
	if len(level) > 0 {
		encLevel = level[0]
	}
	// 打开目标文件准备写入压缩数据
	outFile, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("创建压缩文件失败: %v", err)
	}
	defer outFile.Close()

	// 创建 zstd 压缩器，设置压缩级别及并发数（利用所有 CPU 核心）
	encoder, err := zstd.NewWriter(outFile,
		zstd.WithEncoderLevel(zstd.EncoderLevel(encLevel)),
		zstd.WithEncoderConcurrency(runtime.NumCPU()),
	)
	if err != nil {
		return "", fmt.Errorf("创建 zstd writer 失败: %v", err)
	}
	defer encoder.Close()

	// 判断 srcPath 是文件还是目录
	info, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("获取源路径信息失败: %v", err)
	}

	// 如果是目录，则先打包成 tar 格式
	if info.IsDir() {
		// 创建一个 tar 写入器，将 tar 数据写入到 zstd writer 中
		tw := tar.NewWriter(encoder)
		defer tw.Close()

		// 遍历目录下所有文件
		err = filepath.Walk(srcPath, func(file string, fi fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// 构造 tar 头
			header, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			// 为了保持相对路径
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
				return tw.WriteHeader(header)
			}
			// 写入头信息
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			// 如果是目录，不写入内容
			if fi.IsDir() {
				return nil
			}

			// 打开文件并写入内容
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
			return "", fmt.Errorf("打包目录失败: %v", err)
		}
	} else {
		// 如果是单个文件，直接将文件内容写入 zstd writer
		inFile, err := os.Open(srcPath)
		if err != nil {
			return "", fmt.Errorf("打开源文件失败: %v", err)
		}
		defer inFile.Close()

		if _, err := io.Copy(encoder, inFile); err != nil {
			return "", fmt.Errorf("写入压缩数据失败: %v", err)
		}
	}

	return dstPath, nil
}

// DecompressZstd 解压函数
// 参数：srcPath（压缩文件路径），dstDir（解压后存放位置）
// 返回值：dstDir（解压位置），error（错误）
func DecompressZstd(srcPath, dstDir string) (string, error) {
	// 打开压缩文件
	inFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("打开压缩文件失败: %v", err)
	}
	defer inFile.Close()

	// 创建 zstd 解压器
	decoder, err := zstd.NewReader(inFile)
	if err != nil {
		return "", fmt.Errorf("创建 zstd reader 失败: %v", err)
	}
	defer decoder.Close()

	// 读取所有解压后的数据到内存中
	decompressedData, err := io.ReadAll(decoder)
	if err != nil {
		return "", fmt.Errorf("读取解压数据失败: %v", err)
	}

	// 尝试将解压数据作为 tar 包处理
	tarReader := tar.NewReader(bytes.NewReader(decompressedData))
	// 尝试读取第一个 tar 头
	_, err = tarReader.Next()
	if err == nil {
		// 如果能读出 tar 头，说明原始数据为 tar 包，进行解包
		// 保证解压目录存在
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return "", fmt.Errorf("创建解压目录失败: %v", err)
		}
		// 重置 tarReader，重新遍历所有条目
		tarReader = tar.NewReader(bytes.NewReader(decompressedData))
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("读取 tar 条目失败: %v", err)
			}

			// 构造目标路径
			targetPath := filepath.Join(dstDir, header.Name)
			// 防止路径穿越攻击
			if !strings.HasPrefix(targetPath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
				return "", fmt.Errorf("非法文件路径: %s", targetPath)
			}

			switch header.Typeflag {
			case tar.TypeDir:
				// 创建目录
				if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
					return "", err
				}
			case tar.TypeReg:
				// 创建目录（如果不存在）
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
			default:
				// 可根据需要处理其他类型
			}
		}
	} else {
		// 如果不是 tar 包，则认为是单个文件解压
		// 保证目标目录存在
		if err := os.MkdirAll(filepath.Dir(dstDir), 0755); err != nil {
			return "", fmt.Errorf("创建目标目录失败: %v", err)
		}
		outFile, err := os.Create(dstDir)
		if err != nil {
			return "", fmt.Errorf("创建输出文件失败: %v", err)
		}
		defer outFile.Close()

		// 将数据写入目标文件
		if _, err := io.Copy(outFile, bytes.NewReader(decompressedData)); err != nil {
			return "", fmt.Errorf("写入输出文件失败: %v", err)
		}
	}

	return dstDir, nil
}
