package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SourceManager управляет различными источниками данных для бэкапа
type SourceManager struct {
	sourceDir string
	sourceDB  string
}

// NewSourceManager создает новый менеджер источников
func NewSourceManager(sourceDir, sourceDB string) *SourceManager {
	return &SourceManager{
		sourceDir: sourceDir,
		sourceDB:  sourceDB,
	}
}

// ValidateSource проверяет доступность источников данных
func (sm *SourceManager) ValidateSource() error {
	if sm.sourceDir == "" && sm.sourceDB == "" {
		return fmt.Errorf("no source specified: either SOURCE_DIR or SOURCE_DB must be set")
	}

	if sm.sourceDir != "" {
		if _, err := os.Stat(sm.sourceDir); os.IsNotExist(err) {
			return fmt.Errorf("source directory does not exist: %s", sm.sourceDir)
		}
	}

	// TODO: добавить проверку подключения к БД, когда будет реализована поддержка БД

	return nil
}

// CreateBackup создает бэкап из доступных источников
func (sm *SourceManager) CreateBackup(backupPath string) error {
	if sm.sourceDir != "" {
		return sm.createDirectoryArchive(backupPath, sm.sourceDir)
	}

	if sm.sourceDB != "" {
		return sm.createDatabaseBackup(backupPath, sm.sourceDB)
	}

	return fmt.Errorf("no valid source found")
}

// createDirectoryArchive создает tar.gz архив из директории
func (sm *SourceManager) createDirectoryArchive(archivePath, sourceDir string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %v", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking directory %s: %v", path, err)
		}

		// Пропускаем директории
		if info.IsDir() {
			return nil
		}

		// Открываем файл для чтения
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", path, err)
		}
		defer file.Close()

		// Создаем заголовок для tar
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %v", path, err)
		}

		// Получаем относительный путь
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}
		header.Name = relPath

		// Записываем заголовок
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %v", path, err)
		}

		// Копируем содержимое файла
		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("failed to copy file content for %s: %v", path, err)
		}

		return nil
	})
}

// createDatabaseBackup создает бэкап базы данных (заглушка для будущей реализации)
func (sm *SourceManager) createDatabaseBackup(backupPath, dbURI string) error {
	// TODO: реализовать поддержку различных БД
	if strings.HasPrefix(dbURI, "postgres://") {
		return sm.createPostgresBackup(backupPath, dbURI)
	}

	if strings.HasPrefix(dbURI, "mysql://") {
		return sm.createMySQLBackup(backupPath, dbURI)
	}

	if strings.HasPrefix(dbURI, "mongodb://") {
		return sm.createMongoBackup(backupPath, dbURI)
	}

	return fmt.Errorf("unsupported database type in URI: %s", dbURI)
}

// createPostgresBackup создает бэкап PostgreSQL (заглушка)
func (sm *SourceManager) createPostgresBackup(backupPath, dbURI string) error {
	// TODO: реализовать с помощью pg_dump
	return fmt.Errorf("PostgreSQL backup not yet implemented")
}

// createMySQLBackup создает бэкап MySQL (заглушка)
func (sm *SourceManager) createMySQLBackup(backupPath, dbURI string) error {
	// TODO: реализовать с помощью mysqldump
	return fmt.Errorf("MySQL backup not yet implemented")
}

// createMongoBackup создает бэкап MongoDB (заглушка)
func (sm *SourceManager) createMongoBackup(backupPath, dbURI string) error {
	// TODO: реализовать с помощью mongodump
	return fmt.Errorf("MongoDB backup not yet implemented")
}

// GetSourceInfo возвращает информацию об источнике данных
func (sm *SourceManager) GetSourceInfo() (SourceInfo, error) {
	info := SourceInfo{}

	if sm.sourceDir != "" {
		dirInfo, err := sm.getDirectoryInfo(sm.sourceDir)
		if err != nil {
			return info, err
		}
		info.Directory = &dirInfo
	}

	if sm.sourceDB != "" {
		info.Database = &DatabaseInfo{
			URI:  sm.sourceDB,
			Type: sm.getDatabaseType(sm.sourceDB),
		}
	}

	return info, nil
}

// getDirectoryInfo получает информацию о директории
func (sm *SourceManager) getDirectoryInfo(dirPath string) (DirectoryInfo, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return DirectoryInfo{
		Path:      dirPath,
		TotalSize: totalSize,
		FileCount: fileCount,
	}, err
}

// getDatabaseType определяет тип базы данных по URI
func (sm *SourceManager) getDatabaseType(dbURI string) string {
	if strings.HasPrefix(dbURI, "postgres://") {
		return "PostgreSQL"
	}
	if strings.HasPrefix(dbURI, "mysql://") {
		return "MySQL"
	}
	if strings.HasPrefix(dbURI, "mongodb://") {
		return "MongoDB"
	}
	return "Unknown"
}

// SourceInfo содержит информацию об источнике данных
type SourceInfo struct {
	Directory *DirectoryInfo `json:"directory,omitempty"`
	Database  *DatabaseInfo  `json:"database,omitempty"`
}

// DirectoryInfo содержит информацию о директории
type DirectoryInfo struct {
	Path      string `json:"path"`
	TotalSize int64  `json:"total_size"`
	FileCount int    `json:"file_count"`
}

// DatabaseInfo содержит информацию о базе данных
type DatabaseInfo struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}
