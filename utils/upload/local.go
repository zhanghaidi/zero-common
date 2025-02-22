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

// CopyFolder 复制文件夹
func (l *LocalUploader) CopyFolder(srcFolder, destFolder string) error {
	absSrc := filepath.Join(l.directory, srcFolder)
	absDest := filepath.Join(l.directory, destFolder)

	// 确保源目录存在
	if _, err := os.Stat(absSrc); os.IsNotExist(err) {
		return fmt.Errorf("源目录不存在: %q", absSrc)
	}

	return filepath.WalkDir(absSrc, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("访问路径失败 %q: %w", path, err)
		}

		relativePath, err := filepath.Rel(absSrc, path)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}
		targetPath := filepath.Join(absDest, relativePath)

		// 复制目录
		if d.IsDir() {
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("获取目录信息失败: %w", err)
			}
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("创建目录失败 %q: %w", targetPath, err)
			}
			return nil
		}

		// 复制文件
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("获取文件信息失败: %w", err)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("打开源文件失败 %q: %w", path, err)
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return fmt.Errorf("创建目标文件失败 %q: %w", targetPath, err)
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return fmt.Errorf("复制文件失败 %q -> %q: %w", path, targetPath, err)
		}

		return nil
	})
}

// DeleteFolder 删除目录及其子文件，排除特定文件夹
func (l *LocalUploader) DeleteFolder(folderPath string, exclude ...string) error {
	absFolder := filepath.Join(l.directory, folderPath)

	// 确保删除路径在 l.directory 内
	if !strings.HasPrefix(absFolder, filepath.Clean(l.directory)+string(os.PathSeparator)) {
		return fmt.Errorf("删除路径不安全: %q 超出 %q", absFolder, l.directory)
	}

	// 目录不存在时直接跳过
	if _, err := os.Stat(absFolder); os.IsNotExist(err) {
		fmt.Printf("Warning: 目录不存在, 跳过删除: %s\n", absFolder)
		return nil
	}

	// 处理排除列表
	excludeMap := make(map[string]struct{})
	for _, e := range exclude {
		excludePath := filepath.Join(absFolder, e)
		excludeMap[excludePath] = struct{}{}
	}

	// 先收集路径，确保删除顺序（先文件后目录）
	var paths []string
	err := filepath.WalkDir(absFolder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Warning: 访问路径失败, 文件已不存在: %s\n", path)
				return nil // 忽略文件不存在的错误
			}
			return fmt.Errorf("访问路径失败 %q: %w", path, err)
		}

		// 记录路径（文件先，目录后）
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return err
	}

	// **逆序删除，确保先删文件再删目录**
	for i := len(paths) - 1; i >= 0; i-- {
		p := paths[i]

		// 跳过排除项
		if _, ok := excludeMap[p]; ok {
			continue
		}

		if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除失败 %q: %w", p, err)
		}
	}

	return nil
}

// DeleteFile 删除本地文件
func (l *LocalUploader) DeleteFile(objectKey string) error {
	fullPath := filepath.Join(l.directory, objectKey)

	// 确保文件路径在 l.directory 内
	if !strings.HasPrefix(fullPath, filepath.Clean(l.directory)+string(os.PathSeparator)) {
		return fmt.Errorf("删除路径不安全: %q 超出 %q", fullPath, l.directory)
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
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
