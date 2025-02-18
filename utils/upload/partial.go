package upload

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/zhanghaidi/zero-common/config"

	"fmt"
	"io"
	"os"
	"path/filepath"
)

// 分片上传使用云端存储过程中，无法本地处理切片，此切片分块为医维度改版前固定本地上传本地处理切片核心方法，可以参考医维度chunkUpload方法
type Partial struct {
	TempName   string
	Path       string
	RealPath   string
	HeaderPath string
	ChunkIndex int
}

func NewPartial(temp, extension string) *Partial {
	tempName := fmt.Sprintf("%s%s", temp, extension)
	path := filepath.Join(config.GlobalStorage.Local.Directory, "chunk-upload", tempName+".part")
	headerPath := filepath.Join(config.GlobalStorage.Local.Directory, "chunk-upload", "_header", temp)

	return &Partial{
		TempName:   tempName,
		Path:       path,
		HeaderPath: headerPath,
		ChunkIndex: 0,
	}
}

func (p *Partial) Create() error {
	folder := filepath.Dir(p.Path)
	if err := os.MkdirAll(folder, 0777); err != nil {
		return err
	}
	headerFolder := filepath.Dir(p.HeaderPath)
	if err := os.MkdirAll(headerFolder, 0777); err != nil {
		return err
	}

	file, err := os.Create(p.Path)
	if err != nil {
		return err
	}
	file.Close()

	headerFile, err := os.Create(p.HeaderPath)
	if err != nil {
		return err
	}
	headerFile.WriteString("0")
	headerFile.Close()

	return nil
}

func (p *Partial) Delete() error {
	return os.Remove(p.Path)
}

func (p *Partial) Rename() (string, error) {
	newPath := filepath.Join(filepath.Dir(p.Path), p.TempName)
	err := os.Rename(p.Path, newPath)
	if err != nil {
		return "", err
	}

	return newPath, nil
}

func (p *Partial) Exists() bool {
	_, err := os.Stat(p.Path)
	return !os.IsNotExist(err)
}

func (p *Partial) CalculateHash() (string, error) {
	file, err := os.Open(p.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (p *Partial) SetChunkIndex(index int) error {
	file, err := os.OpenFile(p.HeaderPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d", index))
	return err
}

func (p *Partial) GetChunkIndex() (int, error) {
	file, err := os.Open(p.HeaderPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	buffer := make([]byte, 5) // Assuming chunk index will be within 5 digits
	_, err = file.Read(buffer)
	if err != nil {
		return 0, err
	}

	var index int
	_, err = fmt.Sscanf(string(buffer), "%d", &index)
	if err != nil {
		return 0, err
	}

	return index, nil
}

func (p *Partial) UnsetChunkIndex() error {
	return os.Remove(p.HeaderPath)
}
