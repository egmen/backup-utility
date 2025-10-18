package source

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
)

// DirectoryPlugin реализует бэкап локальной директории
type DirectoryPlugin struct {
	sourceDir string
}

// NewDirectoryPlugin создает новый плагин для бэкапа директории
func NewDirectoryPlugin(sourceDir string) *DirectoryPlugin {
	return &DirectoryPlugin{
		sourceDir: sourceDir,
	}
}

// GetName возвращает имя плагина для суффикса файлов
func (dp *DirectoryPlugin) GetName() string {
	return "directory"
}

// GetExtension возвращает расширение файла бэкапа
func (dp *DirectoryPlugin) GetExtension() string {
	return ".tar.gz"
}

// Validate проверяет доступность директории
func (dp *DirectoryPlugin) Validate() error {
	if dp.sourceDir == "" {
		return fmt.Errorf("source directory is not specified")
	}

	if _, err := os.Stat(dp.sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", dp.sourceDir)
	}

	return nil
}

// CreateBackup создает tar.gz архив из директории
func (dp *DirectoryPlugin) CreateBackup(backupPath string) error {
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %v", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(dp.sourceDir, func(path string, info os.FileInfo, err error) error {
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
		relPath, err := filepath.Rel(dp.sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}
		header.Name = relPath

		// Записываем заголовок
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %v", path, err)
		}

		// Копируем содержимое файла
		if _, err := file.WriteTo(tarWriter); err != nil {
			return fmt.Errorf("failed to copy file content for %s: %v", path, err)
		}

		return nil
	})
}

// GetInfo возвращает информацию о директории
func (dp *DirectoryPlugin) GetInfo() (PluginInfo, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(dp.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		return PluginInfo{}, err
	}

	return PluginInfo{
		Name:      dp.GetName(),
		Type:      "directory",
		Source:    dp.sourceDir,
		TotalSize: totalSize,
		FileCount: fileCount,
	}, nil
}
