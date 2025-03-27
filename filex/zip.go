package filex

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipDir 压缩文件夹（不保留顶层目录）
func ZipDir(dir, zipFile string) error {
	err := IsNotExistMkDir(filepath.Dir(zipFile))
	if err != nil {
		return fmt.Errorf("创建文件目录 %s 错误：%v", zipFile, err)
	}
	fz, err := os.Create(zipFile)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %s", err)
	}
	defer fz.Close()

	w := zip.NewWriter(fz)
	defer w.Close()

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("遍历文件夹失败: %s", err)
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			fDest, err := w.Create(relPath)
			if err != nil {
				return fmt.Errorf("创建ZIP文件失败: %s", err)
			}
			fSrc, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("打开文件失败: %s", err)
			}
			_, err = io.Copy(fDest, fSrc)
			fSrc.Close()
			if err != nil {
				return fmt.Errorf("复制文件内容失败: %s", err)
			}
		}
		return nil
	})
	return err
}

// UnzipDir 解压 ZIP 文件到目录
func UnzipDir(zipFile, dir string) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dir, f.Name)

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return fmt.Errorf("创建目录失败: %s", err)
			}
			continue
		}

		err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			return fmt.Errorf("创建目录失败: %s", err)
		}

		fSrc, err := f.Open()
		if err != nil {
			return fmt.Errorf("打开ZIP文件中的文件失败: %s", err)
		}
		defer fSrc.Close() // defer 仍安全（作用范围单一）

		fDest, err := os.Create(path)
		if err != nil {
			fSrc.Close()
			return fmt.Errorf("创建文件失败: %s", err)
		}

		_, err = io.Copy(fDest, fSrc)
		fDest.Close()
		fSrc.Close()
		if err != nil {
			return fmt.Errorf("复制文件内容失败: %s", err)
		}
	}
	return nil
}

// ZipDirAndSplit 压缩文件夹并分卷
func ZipDirAndSplit(src, dst, prefix string, maxSize int64) error {
	outputDir := filepath.Dir(dst)

	err := IsNotExistMkDir(outputDir)
	if err != nil {
		return fmt.Errorf("创建目录路径 %s 错误：%v", outputDir, err)
	}

	var zw *zip.Writer
	var outputFile *os.File
	var currentSize int64
	var partNumber int

	defer func() {
		if zw != nil {
			zw.Close()
		}
		if outputFile != nil {
			outputFile.Close()
		}
	}()

	createNewZipWriter := func() error {
		if zw != nil {
			zw.Close()
		}
		if outputFile != nil {
			outputFile.Close()
		}

		currentSize = 0
		partFileName := fmt.Sprintf("%s.zip", prefix)
		if partNumber > 0 {
			partFileName = fmt.Sprintf("%s_%d.zip", prefix, partNumber)
		}
		partFileName = filepath.Join(outputDir, partFileName)

		outputFile, err = os.Create(partFileName)
		if err != nil {
			return fmt.Errorf("创建分割ZIP文件失败: %s", err)
		}

		zw = zip.NewWriter(outputFile)
		return nil
	}

	err = createNewZipWriter()
	if err != nil {
		return err
	}

	err = filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("遍历文件夹失败: %s", err)
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(src, filePath)
			if err != nil {
				return fmt.Errorf("计算相对路径失败: %s", err)
			}
			relPath = filepath.ToSlash(relPath)

			header := &zip.FileHeader{
				Name:   relPath,
				Method: zip.Deflate,
			}

			fDest, err := zw.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("创建ZIP文件失败: %s", err)
			}

			fSrc, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("打开文件失败: %s", err)
			}
			n, err := io.Copy(fDest, fSrc)
			fSrc.Close()
			if err != nil {
				return fmt.Errorf("复制文件内容失败: %s", err)
			}

			currentSize += n
			if currentSize >= maxSize {
				partNumber++
				err := createNewZipWriter()
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	return err
}

// ZipDirBase 压缩文件夹 保留顶层目录
func ZipDirBase(dir, zipFile string) (err error) {
	err = IsNotExistMkDir(filepath.Dir(zipFile))
	if err != nil {
		return fmt.Errorf("创建文件目录 %s 错误：%v", zipFile, err)
	}
	fz, err := os.Create(zipFile)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %s", err)
	}
	defer fz.Close()

	w := zip.NewWriter(fz)
	defer w.Close()
	baseDir := filepath.Base(dir)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("遍历文件夹失败: %s", err)
		}
		if !info.IsDir() {
			// 把 path 相对于 dir 的相对路径拼到 baseDir 下
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			zipPath := filepath.Join(baseDir, relPath)

			fDest, err := w.Create(zipPath)
			if err != nil {
				return fmt.Errorf("创建ZIP文件失败: %s", err)
			}

			fSrc, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("打开文件失败: %s", err)
			}
			defer fSrc.Close()

			_, err = io.Copy(fDest, fSrc)
			if err != nil {
				return fmt.Errorf("复制文件内容失败: %s", err)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
