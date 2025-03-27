package filex

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DecompressWithPermissions 解压 Tar.gz 文件并保留文件权限
func DecompressWithPermissions(tarFile, dst string) error {
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

		header.Name = filepath.ToSlash(header.Name)
		filePath := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			// 显式创建父目录
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tr)
			file.Close()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type: %v", header.Typeflag)
		}
	}
	return nil
}

// CompressDirectoryWithPermissions 压缩目录为 Tar.gz 文件并保留文件权限
func CompressDirectoryWithPermissions(sourceDir, outputFilePath string) error {
	err := IsNotExistMkDir(filepath.Dir(outputFilePath))
	if err != nil {
		return fmt.Errorf("创建文件目录 %s 错误：%v", outputFilePath, err)
	}
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	gw := gzip.NewWriter(outputFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 获取 baseName 并准备 tarTemp 路径
	baseName := filepath.Base(sourceDir)
	wd, _ := os.Getwd()
	tarSrc := filepath.Join(wd, "tarTemp", baseName)

	// ✅ 优化点：只复制一次 baseName 目录，不嵌套两层
	if err := CopyDir(sourceDir, tarSrc); err != nil {
		return fmt.Errorf("复制目录失败: %v", err)
	}

	err = filepath.Walk(tarSrc, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(filepath.Dir(tarSrc), filePath)
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

		if !info.IsDir() {
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

	// 清理 tarTemp 临时目录
	_ = os.RemoveAll(filepath.Join(wd, "tarTemp"))

	return err
}
