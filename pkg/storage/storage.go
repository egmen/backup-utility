package storage

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// Provider определяет интерфейс для различных типов хранилищ
type Provider interface {
	EnsureExists() error
	Upload(key string, data []byte) error
	Delete(key string) error
	List(prefix string) ([]ObjectInfo, error)
	GenerateKey(prefix string) string
}

// ObjectInfo содержит информацию об объекте в хранилище
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// Manager управляет целевым хранилищем для бэкапов
type Manager struct {
	provider    Provider
	prefix      string
	retention   string
	minRetained int
}

// NewManager создает новый менеджер хранилища
func NewManager(targetPath, prefix, retention string, minRetained int) (*Manager, error) {
	provider, err := createProvider(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider: %v", err)
	}

	return &Manager{
		provider:    provider,
		prefix:      prefix,
		retention:   retention,
		minRetained: minRetained,
	}, nil
}

// createProvider создает провайдер хранилища на основе URI
func createProvider(targetPath string) (Provider, error) {
	if strings.HasPrefix(targetPath, "s3://") {
		return NewS3Provider(targetPath)
	}
	return NewLocalProvider(targetPath), nil
}

// EnsureStorageExists создает хранилище, если оно не существует
func (sm *Manager) EnsureStorageExists() error {
	return sm.provider.EnsureExists()
}

// GenerateBackupPath генерирует ключ для нового архива бэкапа
func (sm *Manager) GenerateBackupPath() string {
	return sm.provider.GenerateKey(sm.prefix)
}

// SaveBackup сохраняет данные бэкапа в хранилище
func (sm *Manager) SaveBackup(backupKey string, data []byte) error {
	return sm.provider.Upload(backupKey, data)
}

// UploadBackup загружает готовый файл бэкапа в хранилище
func (sm *Manager) UploadBackup(sourcePath, backupKey string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}
	return sm.provider.Upload(backupKey, data)
}

// ApplyRetentionPolicy применяет политику ротации бэкапов
func (sm *Manager) ApplyRetentionPolicy() error {
	backups, err := sm.provider.List(sm.prefix)
	if err != nil {
		return fmt.Errorf("failed to list backups: %v", err)
	}

	// Сортируем по времени создания (новые первые)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].LastModified.After(backups[j].LastModified)
	})

	// Применяем политику ротации
	if len(backups) > sm.minRetained {
		for i := sm.minRetained; i < len(backups); i++ {
			if err := sm.provider.Delete(backups[i].Key); err != nil {
				log.Printf("Failed to remove old backup: %v", err)
			} else {
				log.Printf("Removed old backup: %s", backups[i].Key)
			}
		}
	}

	return nil
}

// ListBackups возвращает список всех бэкапов
func (sm *Manager) ListBackups() ([]string, error) {
	backups, err := sm.provider.List(sm.prefix)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, backup := range backups {
		keys = append(keys, backup.Key)
	}

	return keys, nil
}

// GetStorageInfo возвращает информацию о хранилище
func (sm *Manager) GetStorageInfo() (Info, error) {
	backups, err := sm.provider.List(sm.prefix)
	if err != nil {
		return Info{}, err
	}

	var totalSize int64
	for _, backup := range backups {
		totalSize += backup.Size
	}

	return Info{
		BackupCount: len(backups),
		TotalSize:   totalSize,
		Path:        "managed by provider",
	}, nil
}

// Info содержит информацию о хранилище
type Info struct {
	BackupCount int
	TotalSize   int64
	Path        string
}
