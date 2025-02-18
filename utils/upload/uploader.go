// upload/uploader.go
package upload

import (
	"fmt"
	"github.com/zhanghaidi/zero-common/config"

	"io"
)

// Part 定义上传分片
type Part struct {
	ETag       string
	PartNumber int
}

// Uploader 接口支持多种存储类型
type Uploader interface {
	InitiateMultipartUpload(ext string) (uploadID string, objectKey string, err error)
	UploadPart(objectKey, uploadID string, partNumber int, reader io.Reader, partSize int64) (string, error)
	CompleteMultipartUpload(objectKey, uploadID string, parts []Part) (string, error)
	UploadFile(objectKey string, reader io.Reader) (string, error)
	CopyFolder(srcFolder, destFolder string) error
	DeleteFolder(folderPath string, exclude ...string) error
	ListFiles(directory string, suffixFilters ...string) ([]string, error)
	DeleteFile(objectKey string) error
}

// NewUploader 根据传入的配置信息返回适当的存储实例
func NewUploader() (Uploader, error) {
	cfg := config.GlobalStorage
	switch cfg.Driver {
	case "local":
		return NewLocalUploader(cfg.Local.Directory), nil
	case "oss":
		return NewOssUploader(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.Driver)
	}
}
