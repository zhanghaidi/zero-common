// upload/oss.go
package upload

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/zhanghaidi/zero-common/config"

	"io"
	"path"
	"strings"
	"time"
)

type OssUploader struct {
	bucket *oss.Bucket
}

func NewOssUploader(cfg config.StorageConf) (*OssUploader, error) {
	client, err := oss.New(cfg.Oss.Endpoint, cfg.Oss.AccessKeyID, cfg.Oss.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	bucket, err := client.Bucket(cfg.Oss.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &OssUploader{bucket: bucket}, nil
}

func (o *OssUploader) InitiateMultipartUpload(objectKey string) (string, string, error) {

	objectKey = fmt.Sprintf("chunk-upload/%d%s", time.Now().UnixNano(), strings.ToLower(path.Ext(objectKey)))

	imur, err := o.bucket.InitiateMultipartUpload(objectKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return imur.UploadID, imur.Key, nil
}

func (o *OssUploader) UploadPart(objectKey, uploadID string, partNumber int, reader io.Reader, partSize int64) (string, error) {
	// 分片上传时，保持与初始化相同的路径
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   o.bucket.BucketName,
		Key:      objectKey,
		UploadID: uploadID,
	}
	result, err := o.bucket.UploadPart(imur, reader, partSize, partNumber)
	if err != nil {
		return "", fmt.Errorf("failed to upload part: %w", err)
	}
	return result.ETag, nil
}

func (o *OssUploader) CompleteMultipartUpload(objectKey, uploadID string, parts []Part) (string, error) {
	var ossParts []oss.UploadPart
	for _, part := range parts {
		ossParts = append(ossParts, oss.UploadPart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		})
	}

	imur := oss.InitiateMultipartUploadResult{
		Bucket:   o.bucket.BucketName,
		Key:      objectKey,
		UploadID: uploadID,
	}
	_, err := o.bucket.CompleteMultipartUpload(imur, ossParts)
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return objectKey, nil
}

func (o *OssUploader) UploadFile(objectKey string, reader io.Reader) (string, error) {
	if err := o.bucket.PutObject(objectKey, reader); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return objectKey, nil
}

// CopyFolder 复制OSS文件夹及其子文件到新路径
func (o *OssUploader) CopyFolder(srcFolder, destFolder string) error {
	marker := ""
	for {
		// 列出源文件夹中的所有文件
		lsRes, err := o.bucket.ListObjects(oss.Prefix(srcFolder), oss.Marker(marker))
		if err != nil {
			return fmt.Errorf("列出OSS对象失败: %v", err)
		}

		// 遍历文件并复制到目标文件夹
		for _, object := range lsRes.Objects {
			srcKey := object.Key
			// 构造目标文件路径
			destKey := strings.Replace(srcKey, srcFolder, destFolder, 1)
			// 复制文件
			_, err := o.bucket.CopyObject(srcKey, destKey)
			if err != nil {
				return fmt.Errorf("复制OSS对象失败: %v", err)
			}
		}

		// 如果文件列表未截断，退出循环
		if !lsRes.IsTruncated {
			break
		}
		marker = lsRes.NextMarker
	}
	return nil
}

// DeleteFolder 删除OSS文件夹及其子文件，排除指定的子文件夹
func (o *OssUploader) DeleteFolder(folderPath string, exclude ...string) error {
	excludeSet := make(map[string]struct{})
	for _, ex := range exclude {
		excludeSet[ex] = struct{}{}
	}

	marker := ""
	for {
		// 列出文件夹中的所有文件
		lsRes, err := o.bucket.ListObjects(oss.Prefix(folderPath), oss.Marker(marker))
		if err != nil {
			return fmt.Errorf("列出OSS对象失败: %v", err)
		}

		var keys []string
		for _, object := range lsRes.Objects {
			skip := false
			// 检查是否在排除列表中
			for excluded := range excludeSet {
				if strings.HasPrefix(object.Key, path.Join(folderPath, excluded)) {
					skip = true
					break
				}
			}
			if !skip {
				keys = append(keys, object.Key)
			}
		}

		// 批量删除未排除的文件
		if len(keys) > 0 {
			_, err = o.bucket.DeleteObjects(keys)
			if err != nil {
				return fmt.Errorf("删除OSS对象失败: %v", err)
			}
		}

		// 如果文件列表未截断，退出循环
		if !lsRes.IsTruncated {
			break
		}
		marker = lsRes.NextMarker
	}
	return nil
}

// DeleteFile 删除OSS上的文件
func (o *OssUploader) DeleteFile(objectKey string) error {
	err := o.bucket.DeleteObject(objectKey) // 调用 OSS API 删除文件
	if err != nil {
		return fmt.Errorf("删除OSS文件失败: %v", err)
	}
	return nil
}

// ListFiles 获取OSS存储桶目录下的所有文件列表，可以通过后缀进行过滤
func (o *OssUploader) ListFiles(directory string, suffixFilters ...string) ([]string, error) {
	var files []string
	marker := ""
	for {
		// 列出目录中的所有文件
		lsRes, err := o.bucket.ListObjects(oss.Prefix(directory), oss.Marker(marker))
		if err != nil {
			return nil, fmt.Errorf("列出OSS对象失败: %v", err)
		}

		// 遍历文件并根据后缀进行过滤
		for _, object := range lsRes.Objects {
			if len(suffixFilters) == 0 {
				files = append(files, object.Key)
				continue
			}
			for _, suffix := range suffixFilters {
				if strings.HasSuffix(object.Key, suffix) {
					files = append(files, object.Key)
					break
				}
			}
		}

		// 如果文件列表未截断，退出循环
		if !lsRes.IsTruncated {
			break
		}
		marker = lsRes.NextMarker
	}
	return files, nil
}
