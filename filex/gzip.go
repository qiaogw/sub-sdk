package filex

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const defaultMaxSize = int64(10 * 1024 * 1024 * 1024) // 默认最大分卷 10G

// Compress 压缩文件或目录，自动识别平台并生成 zip（Windows）或 tar.gz（Linux/macOS）压缩包。
// src：待压缩路径，可为目录或单个文件
// dst：压缩文件前缀（不带扩展名）
// maxSizeInGB：每个分卷最大大小（单位：GB），小于 1 则默认 1GB
func Compress(src, dst string, mSize int64) error {
	maxSize := mSize * 1024 * 1024 * 1024
	if mSize < 1 {
		maxSize = int64(1 * 1024 * 1024 * 1024)
	}
	prefix := filepath.Base(dst)

	if runtime.GOOS == "windows" {
		return ZipDirAndSplit(src, dst+".zip", prefix, maxSize)
	}
	return CompressAndSplitFiles(src, dst+".tar.gz", prefix, maxSize)
}

// CompressAndSplitFiles 压缩并分卷
func CompressAndSplitFiles(src, dst, prefix string, maxSize int64) error {
	destDir := filepath.Dir(dst)
	err := IsNotExistMkDir(destDir)
	if err != nil {
		return fmt.Errorf("创建目录 %s 错误：%v", destDir, err)
	}
	if maxSize > defaultMaxSize {
		maxSize = defaultMaxSize
	}

	var outputFile *os.File
	var gw *gzip.Writer
	var tw *tar.Writer
	var currentSize int64
	partNumber := 0

	err = createNewSplitFile(destDir, prefix, partNumber, &outputFile, &gw, &tw)
	if err != nil {
		return err
	}
	defer func() {
		if tw != nil {
			tw.Close()
		}
		if gw != nil {
			gw.Close()
		}
		if outputFile != nil {
			outputFile.Close()
		}
	}()

	err = filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, filePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			header.Size = 0
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			n, err := io.Copy(tw, file)
			file.Close()
			if err != nil {
				return fmt.Errorf("压缩失败: [%s], 错误: %v", filePath, err)
			}
			currentSize += n

			if currentSize >= maxSize {
				partNumber++
				err := createNewSplitFile(destDir, prefix, partNumber, &outputFile, &gw, &tw)
				if err != nil {
					return err
				}
				currentSize = 0
			}
		}

		return nil
	})

	return err
}

// createNewSplitFile 创建新分卷
func createNewSplitFile(destDir, prefix string, partNumber int, outputFile **os.File, gw **gzip.Writer, tw **tar.Writer) error {
	if *tw != nil {
		(*tw).Close()
	}
	if *gw != nil {
		(*gw).Close()
	}
	if *outputFile != nil {
		(*outputFile).Close()
	}

	partFileName := fmt.Sprintf("%s.tar.gz", prefix)
	if partNumber > 0 {
		partFileName = fmt.Sprintf("%s_%d.tar.gz", prefix, partNumber)
	}
	partFilePath := filepath.Join(destDir, partFileName)

	file, err := os.Create(partFilePath)
	if err != nil {
		return err
	}

	*outputFile = file
	*gw = gzip.NewWriter(file)
	*tw = tar.NewWriter(*gw)

	return nil
}

// CompressFilesOrFolds 单个压缩（无分卷）
func CompressFilesOrFolds(src, dst string) error {
	err := IsNotExistMkDir(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("创建目录 %s 错误：%v", dst, err)
	}
	outputFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	gw := gzip.NewWriter(outputFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, filePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, file)
			file.Close()
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
