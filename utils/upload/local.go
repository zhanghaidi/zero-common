// upload/local.go
package upload

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type LocalUploader struct {
	directory string
}

// NewLocalUploader 创建一个本地存储上传器实例
func NewLocalUploader(directory string) *LocalUploader {
	return &LocalUploader{directory: directory}
}

// InitiateMultipartUpload 初始化分片上传并返回 uploadID
func (l *LocalUploader) InitiateMultipartUpload(objectKey string) (string, string, error) {

	uploadId := fmt.Sprintf("%d", time.Now().UnixNano()) // 使用时间戳生成唯一 uploadID

	objectKey = fmt.Sprintf("chunk-upload/%s%s", uploadId, strings.ToLower(path.Ext(objectKey)))

	return uploadId, objectKey, nil
}

// UploadPart 上传单个分片
func (l *LocalUploader) UploadPart(objectKey, uploadID string, partNumber int, reader io.Reader, partSize int64) (string, error) {
	// 生成分片文件路径
	partDir := filepath.Join(l.directory, "chunk-upload", "_header")
	partPath := filepath.Join(partDir, fmt.Sprintf("%s_part%d", uploadID, partNumber))

	// 确保分片文件的目录存在
	if err := os.MkdirAll(filepath.Dir(partPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for part: %w", err)
	}

	// 创建或覆盖分片文件
	file, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 将分片内容写入文件
	if _, err = io.CopyN(file, reader, partSize); err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to write part: %w", err)
	}

	// 返回分片 ETag
	return fmt.Sprintf("etag-%s-part%d", uploadID, partNumber), nil
}

// CompleteMultipartUpload 完成分片上传并合并所有分片
func (l *LocalUploader) CompleteMultipartUpload(objectKey, uploadID string, parts []Part) (string, error) {
	finalFilePath := filepath.Join(l.directory, objectKey)
	// 确保最终文件的目录存在
	if err := os.MkdirAll(filepath.Dir(finalFilePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for final file: %w", err)
	}
	// 创建最终合并文件
	finalFile, err := os.OpenFile(finalFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	// 使用缓冲区加速文件写入
	buffer := make([]byte, 5*1024*1024) // 5MB 缓冲区
	// 合并所有分片
	for _, part := range parts {
		partFilePath := filepath.Join(l.directory, "chunk-upload", "_header", fmt.Sprintf("%s_part%d", uploadID, part.PartNumber))
		partFile, err := os.Open(partFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to open part file: %w", err)
		}

		for {
			n, err := partFile.Read(buffer)
			if err != nil && err != io.EOF {
				partFile.Close()
				return "", fmt.Errorf("failed to read part file: %w", err)
			}
			if n == 0 {
				break
			}

			if _, err := finalFile.Write(buffer[:n]); err != nil {
				partFile.Close()
				return "", fmt.Errorf("failed to write to final file: %w", err)
			}
		}
		partFile.Close()

		// 删除临时分片文件
		if err = os.Remove(partFilePath); err != nil {
			return "", fmt.Errorf("failed to delete part file: %w", err)
		}
	}
	// 返回合并后的文件路径
	return objectKey, nil
}

// UploadFile 直接上传文件
func (l *LocalUploader) UploadFile(objectKey string, reader io.Reader) (string, error) {
	fullPath := filepath.Join(l.directory, objectKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return objectKey, nil
}

// CopyFolder 复制目录及其内容
func (l *LocalUploader) CopyFolder(srcFolder, newFolder string) error {
	err := filepath.Walk(srcFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path %q: %w", path, err)
		}

		// 生成目标路径
		relativePath, err := filepath.Rel(srcFolder, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}
		targetPath := filepath.Join(newFolder, relativePath)

		if info.IsDir() {
			// 创建目标目录
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", targetPath, err)
			}
		} else {
			// 复制文件
			srcFile, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open source file %q: %w", path, err)
			}
			defer srcFile.Close()

			targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
			if err != nil {
				return fmt.Errorf("failed to create target file %q: %w", targetPath, err)
			}
			defer targetFile.Close()

			if _, err := io.Copy(targetFile, srcFile); err != nil {
				return fmt.Errorf("failed to copy content from %q to %q: %w", path, targetPath, err)
			}
		}

		return nil
	})

	return err
}

// DeleteFolder 删除目录及其子文件，排除特定文件夹
func (l *LocalUploader) DeleteFolder(folderPath string, exclude ...string) error {
	excludeMap := make(map[string]struct{})
	for _, e := range exclude {
		absPath, err := filepath.Abs(filepath.Join(folderPath, e))
		if err != nil {
			return fmt.Errorf("failed to resolve exclude path %q: %w", e, err)
		}
		excludeMap[absPath] = struct{}{}
	}

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path %q: %w", path, err)
		}

		// 如果是排除的目录，跳过
		if info.IsDir() {
			if _, ok := excludeMap[path]; ok {
				return filepath.SkipDir
			}
		}

		// 删除文件或目录
		if _, ok := excludeMap[path]; !ok {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to delete %q: %w", path, err)
			}
		}
		return nil
	})

	return err
}

// DeleteFile 删除本地文件
func (l *LocalUploader) DeleteFile(objectKey string) error {
	fullPath := filepath.Join(l.directory, objectKey)
	err := os.Remove(fullPath) // 删除文件
	if err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

// ListFiles 获取本地目录下的所有文件，可以通过后缀进行过滤
func (l *LocalUploader) ListFiles(directory string, suffixFilters ...string) ([]string, error) {
	var files []string

	// 遍历目录中的所有文件
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("无法访问路径 %q: %w", path, err)
		}

		// 跳过目录，仅处理文件
		if info.IsDir() {
			return nil
		}

		// 如果没有指定后缀过滤器，则添加所有文件
		if len(suffixFilters) == 0 {
			files = append(files, path)
			return nil
		}

		// 按后缀筛选文件
		for _, suffix := range suffixFilters {
			if strings.HasSuffix(strings.ToLower(info.Name()), strings.ToLower(suffix)) {
				files = append(files, path)
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
