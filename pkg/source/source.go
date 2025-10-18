package source

import (
	"fmt"
	"strings"
)

// SourcePlugin определяет интерфейс для плагинов источников данных
type SourcePlugin interface {
	GetName() string                      // Имя плагина для суффикса файлов
	GetExtension() string                 // Расширение файла (.tar.gz, .sql.gz и т.д.)
	Validate() error                      // Проверка доступности источника
	CreateBackup(backupPath string) error // Создание бэкапа
	GetInfo() (PluginInfo, error)         // Информация об источнике
}

// PluginInfo содержит информацию о плагине источника
type PluginInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	TotalSize int64                  `json:"total_size"`
	FileCount int                    `json:"file_count"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Manager управляет различными плагинами источников данных для бэкапа
type Manager struct {
	plugins []SourcePlugin
}

// NewManager создает новый менеджер источников
func NewManager(sourceDir, sourceDB string) (*Manager, error) {
	manager := &Manager{
		plugins: make([]SourcePlugin, 0),
	}

	// Добавляем плагин директории, если указан путь
	if sourceDir != "" {
		dirPlugin := NewDirectoryPlugin(sourceDir)
		manager.plugins = append(manager.plugins, dirPlugin)
	}

	// Добавляем MySQL плагин, если указан URI БД
	if sourceDB != "" {
		if strings.HasPrefix(sourceDB, "mysql://") {
			mysqlPlugin, err := NewMySQLPlugin(sourceDB)
			if err != nil {
				return nil, fmt.Errorf("failed to create MySQL plugin: %v", err)
			}
			manager.plugins = append(manager.plugins, mysqlPlugin)
		} else {
			// TODO: добавить поддержку других типов БД (postgres://, mongodb://)
			return nil, fmt.Errorf("unsupported database type in URI: %s", sourceDB)
		}
	}

	if len(manager.plugins) == 0 {
		return nil, fmt.Errorf("no source plugins configured: either SOURCE_DIR or SOURCE_DB must be set")
	}

	return manager, nil
}

// ValidateSource проверяет доступность всех настроенных источников
func (sm *Manager) ValidateSource() error {
	if len(sm.plugins) == 0 {
		return fmt.Errorf("no source plugins available")
	}

	for _, plugin := range sm.plugins {
		if err := plugin.Validate(); err != nil {
			return fmt.Errorf("plugin %s validation failed: %v", plugin.GetName(), err)
		}
	}

	return nil
}

// CreateBackups создает бэкапы из всех доступных источников
// Возвращает список созданных файлов бэкапов
func (sm *Manager) CreateBackups(baseBackupPath string) ([]string, error) {
	if len(sm.plugins) == 0 {
		return nil, fmt.Errorf("no source plugins available")
	}

	var createdFiles []string

	for _, plugin := range sm.plugins {
		// Генерируем путь с суффиксом плагина
		backupPath := sm.generateBackupPath(baseBackupPath, plugin.GetName(), plugin.GetExtension())

		if err := plugin.CreateBackup(backupPath); err != nil {
			return createdFiles, fmt.Errorf("plugin %s backup failed: %v", plugin.GetName(), err)
		}

		createdFiles = append(createdFiles, backupPath)
	}

	return createdFiles, nil
}

// generateBackupPath генерирует путь к файлу бэкапа с суффиксом плагина
// Формат: prefix_YYYY-MM-DD_HH-MM_pluginname.extension
func (sm *Manager) generateBackupPath(basePath, pluginName, extension string) string {
	// Убираем расширение из базового пути
	baseWithoutExt := strings.TrimSuffix(basePath, ".tar.gz")
	// Добавляем суффикс плагина и расширение
	return baseWithoutExt + "_" + pluginName + extension
}

// CreateBackup создает бэкап из первого доступного источника (для обратной совместимости)
func (sm *Manager) CreateBackup(backupPath string) error {
	if len(sm.plugins) == 0 {
		return fmt.Errorf("no source plugins available")
	}

	// Используем первый плагин для обратной совместимости
	firstPlugin := sm.plugins[0]
	pluginBackupPath := sm.generateBackupPath(backupPath, firstPlugin.GetName(), firstPlugin.GetExtension())

	return firstPlugin.CreateBackup(pluginBackupPath)
}

// GetSourceInfo возвращает информацию обо всех плагинах источников
func (sm *Manager) GetSourceInfo() ([]PluginInfo, error) {
	var infos []PluginInfo

	for _, plugin := range sm.plugins {
		info, err := plugin.GetInfo()
		if err != nil {
			return nil, fmt.Errorf("failed to get info from plugin %s: %v", plugin.GetName(), err)
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// GetPlugins возвращает список всех загруженных плагинов
func (sm *Manager) GetPlugins() []SourcePlugin {
	return sm.plugins
}
