package filex

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const defaultMaxSize = int64(10 * 1024 * 1024 * 1024) // 默认最大分卷 10G

// CompressAuto 压缩文件或目录，自动识别平台并生成 zip（Windows）或 tar.gz（Linux/macOS）压缩包。
// src：待压缩路径，可为目录或单个文件
// dst：压缩文件前缀（不带扩展名）
// maxSizeInGB：每个分卷最大大小（单位：GB），小于 1 则默认 1GB
func CompressAuto(src, dst string) (string, error) {
	if runtime.GOOS == "windows" {
		return dst + ".zip", ZipDirBase(src, dst+".zip")
	}
	return dst + ".tar.gz", CompressFilesOrFoldsBase(src, dst+".tar.gz")
}

// DecompressAuto 自动根据文件后缀解压 zip 或 tar.gz
func DecompressAuto(filePath, dst string) error {
	if strings.HasSuffix(filePath, ".zip") {
		return UnzipDir(filePath, dst)
	} else if strings.HasSuffix(filePath, ".tar.gz") {
		return DecompressTarGz(filePath, dst)
	}
	return fmt.Errorf("不支持的压缩格式: %s", filePath)
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

// CompressFilesOrFolds  单个压缩（无分卷）
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

// CompressFilesOrFoldsBase 单个压缩（无分卷）保留顶层目录
func CompressFilesOrFoldsBase(src, dst string) error {
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

	// 获取压缩包中顶层目录名称（例如 "db"）
	baseDir := filepath.Base(src)

	err = filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对于 src 的路径
		relPath, err := filepath.Rel(src, filePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// 拼接顶层目录（baseDir）和相对路径，确保解压后包含顶层 "db" 目录
		var tarName string
		if relPath == "." {
			tarName = baseDir
		} else {
			tarName = baseDir + "/" + relPath
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		// 确保保留文件所有的权限位（包括执行权限）
		header.Mode = int64(info.Mode())
		header.Name = tarName

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// 如果是普通文件则写入文件内容
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

// DecompressTarGz 解压 .tar.gz
func DecompressTarGz(tarFile, dst string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	gr, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(targetPath, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			out, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
