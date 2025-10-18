package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalProvider реализует локальное файловое хранилище
type LocalProvider struct {
	basePath string
}

// NewLocalProvider создает провайдер для локального хранилища
func NewLocalProvider(basePath string) *LocalProvider {
	return &LocalProvider{basePath: basePath}
}

// EnsureExists создает директорию хранилища
func (lp *LocalProvider) EnsureExists() error {
	return os.MkdirAll(lp.basePath, 0755)
}

// GenerateKey генерирует ключ для нового файла
func (lp *LocalProvider) GenerateKey(prefix string) string {
	timestamp := time.Now().UTC().Format("2006-01-02_15-04")
	filename := fmt.Sprintf("%s_%s.tar.gz", prefix, timestamp)
	return filepath.Join(lp.basePath, filename)
}

// Upload сохраняет данные в локальный файл
func (lp *LocalProvider) Upload(key string, data []byte) error {
	return os.WriteFile(key, data, 0644)
}

// Delete удаляет локальный файл
func (lp *LocalProvider) Delete(key string) error {
	return os.Remove(key)
}

// List возвращает список файлов в директории
func (lp *LocalProvider) List(prefix string) ([]ObjectInfo, error) {
	files, err := os.ReadDir(lp.basePath)
	if err != nil {
		return nil, err
	}

	var objects []ObjectInfo
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), prefix) {
			info, err := file.Info()
			if err != nil {
				continue
			}
			objects = append(objects, ObjectInfo{
				Key:          filepath.Join(lp.basePath, file.Name()),
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}
	}
	return objects, nil
}
