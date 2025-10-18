package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// StorageManager управляет целевым хранилищем для бэкапов
type StorageManager struct {
	targetPath  string
	prefix      string
	retention   string
	minRetained int
}

// NewStorageManager создает новый менеджер хранилища
func NewStorageManager(targetPath, prefix, retention string, minRetained int) *StorageManager {
	return &StorageManager{
		targetPath:  targetPath,
		prefix:      prefix,
		retention:   retention,
		minRetained: minRetained,
	}
}

// EnsureStorageExists создает директорию хранилища, если она не существует
func (sm *StorageManager) EnsureStorageExists() error {
	return os.MkdirAll(sm.targetPath, 0755)
}

// GenerateBackupPath генерирует путь для нового архива бэкапа
func (sm *StorageManager) GenerateBackupPath() string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	filename := fmt.Sprintf("%s-%s.tar.gz", sm.prefix, timestamp)
	return filepath.Join(sm.targetPath, filename)
}

// SaveBackup сохраняет данные бэкапа в хранилище
func (sm *StorageManager) SaveBackup(backupPath string, data []byte) error {
	return os.WriteFile(backupPath, data, 0644)
}

// ApplyRetentionPolicy применяет политику ротации бэкапов
func (sm *StorageManager) ApplyRetentionPolicy() error {
	files, err := os.ReadDir(sm.targetPath)
	if err != nil {
		return fmt.Errorf("failed to read target directory: %v", err)
	}

	var backups []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), sm.prefix) {
			backups = append(backups, file)
		}
	}

	// Сортируем по времени создания (новые первые)
	sort.Slice(backups, func(i, j int) bool {
		infoI, _ := backups[i].Info()
		infoJ, _ := backups[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Применяем политику ротации
	if len(backups) > sm.minRetained {
		for i := sm.minRetained; i < len(backups); i++ {
			filePath := filepath.Join(sm.targetPath, backups[i].Name())
			if err := sm.removeBackup(filePath); err != nil {
				log.Printf("Failed to remove old backup: %v", err)
			} else {
				log.Printf("Removed old backup: %s", filePath)
			}
		}
	}

	return nil
}

// removeBackup удаляет файл бэкапа
func (sm *StorageManager) removeBackup(filePath string) error {
	return os.Remove(filePath)
}

// ListBackups возвращает список всех бэкапов
func (sm *StorageManager) ListBackups() ([]string, error) {
	files, err := os.ReadDir(sm.targetPath)
	if err != nil {
		return nil, err
	}

	var backups []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), sm.prefix) {
			backups = append(backups, file.Name())
		}
	}

	return backups, nil
}

// GetStorageInfo возвращает информацию о хранилище
func (sm *StorageManager) GetStorageInfo() (StorageInfo, error) {
	backups, err := sm.ListBackups()
	if err != nil {
		return StorageInfo{}, err
	}

	var totalSize int64
	for _, backup := range backups {
		filePath := filepath.Join(sm.targetPath, backup)
		if info, err := os.Stat(filePath); err == nil {
			totalSize += info.Size()
		}
	}

	return StorageInfo{
		BackupCount: len(backups),
		TotalSize:   totalSize,
		Path:        sm.targetPath,
	}, nil
}

// StorageInfo содержит информацию о хранилище
type StorageInfo struct {
	BackupCount int
	TotalSize   int64
	Path        string
}
