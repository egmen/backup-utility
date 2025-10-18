package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"

	"github.com/egmen/backup-utility/pkg/source"
	"github.com/egmen/backup-utility/pkg/storage"
)

// Config содержит параметры бэкапа
type Config struct {
	SourceDir       string
	SourceDB        string
	TargetStorage   string
	BackupRetention string
	MinRetained     int
	BackupPrefix    string
	Schedule        string
	RunOnStartup    bool
	LogLevel        string
	NotifyWebhook   string
}

func main() {
	config := loadConfig()
	log.Printf("Starting backup utility with config: %+v", config)

	if config.RunOnStartup {
		if err := runBackup(config); err != nil {
			log.Fatalf("Initial backup failed: %v", err)
		}
	}

	if config.Schedule != "" {
		runScheduledBackups(config)
	} else {
		log.Println("No schedule set, running one-time backup")
		if err := runBackup(config); err != nil {
			log.Fatalf("Backup failed: %v", err)
		}
	}
}

func loadConfig() Config {
	return Config{
		SourceDir:       getEnv("SOURCE_DIR", "/data/source"),
		SourceDB:        getEnv("SOURCE_DB", ""),
		TargetStorage:   getEnv("TARGET_STORAGE", "/data/backups"),
		BackupRetention: getEnv("BACKUP_RETENTION", "7d"),
		MinRetained:     getEnvAsInt("BACKUP_MIN_RETAINED", 3),
		BackupPrefix:    getEnv("BACKUP_PREFIX", "backup"),
		Schedule:        getEnv("RUN_SCHEDULE", "0 4 * * *"),
		RunOnStartup:    getEnvAsBool("RUN_ON_STARTUP", true),
		LogLevel:        getEnv("RUN_LOG_LEVEL", "info"),
		NotifyWebhook:   getEnv("RUN_NOTIFY_WEBHOOK", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func runBackup(config Config) error {
	log.Printf("Starting backup process")

	// Создаем менеджеры для источников и хранилища
	sourceManager, err := source.NewManager(config.SourceDir, config.SourceDB)
	if err != nil {
		return fmt.Errorf("failed to create source manager: %v", err)
	}

	storageManager, err := storage.NewManager(
		config.TargetStorage,
		config.BackupPrefix,
		config.BackupRetention,
		config.MinRetained,
	)
	if err != nil {
		return fmt.Errorf("failed to create storage manager: %v", err)
	}

	// Проверяем доступность источников
	if err := sourceManager.ValidateSource(); err != nil {
		return fmt.Errorf("source validation failed: %v", err)
	}

	// Убеждаемся что директория хранилища существует
	if err := storageManager.EnsureStorageExists(); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Генерируем базовый ключ для нового бэкапа
	baseBackupKey := storageManager.GenerateBackupPath()
	log.Printf("Creating backup with base key: %s", baseBackupKey)

	// Для S3 создаем временные локальные файлы, для локального хранилища используем путь напрямую
	var isS3 = strings.HasPrefix(config.TargetStorage, "s3://")

	if isS3 {
		// Создаем временные файлы для S3
		baseTemp := "/tmp/" + filepath.Base(baseBackupKey)
		createdFiles, err := sourceManager.CreateBackups(baseTemp)
		if err != nil {
			// Очищаем созданные файлы при ошибке
			for _, f := range createdFiles {
				os.Remove(f)
			}
			return fmt.Errorf("failed to create backups: %v", err)
		}

		// Получаем префикс пути из baseBackupKey (если есть)
		baseDir := filepath.Dir(baseBackupKey)
		if baseDir == "." {
			baseDir = ""
		}

		// Загружаем все файлы в S3 и удаляем временные файлы
		for _, tempFile := range createdFiles {
			// Вычисляем целевой ключ в S3, сохраняя структуру пути
			filename := filepath.Base(tempFile)
			var s3Key string
			if baseDir != "" {
				s3Key = baseDir + "/" + filename
			} else {
				s3Key = filename
			}

			if err := storageManager.UploadBackup(tempFile, s3Key); err != nil {
				// Очищаем все временные файлы при ошибке
				for _, f := range createdFiles {
					os.Remove(f)
				}
				return fmt.Errorf("failed to upload backup to S3: %v", err)
			}
			os.Remove(tempFile) // очищаем временный файл после успешной загрузки
			log.Printf("Backup uploaded successfully: %s", s3Key)
		}
	} else {
		// Для локального хранилища создаем архивы напрямую в нужное место
		createdFiles, err := sourceManager.CreateBackups(baseBackupKey)
		if err != nil {
			return fmt.Errorf("failed to create backups: %v", err)
		}
		for _, file := range createdFiles {
			log.Printf("Backup created successfully: %s", file)
		}
	}

	// Применяем политику ротации
	if err := storageManager.ApplyRetentionPolicy(); err != nil {
		return fmt.Errorf("failed to apply retention policy: %v", err)
	}

	return nil
}

func runScheduledBackups(config Config) {
	scheduler := cron.New()
	defer scheduler.Stop()

	_, err := scheduler.AddFunc(config.Schedule, func() {
		log.Println("Running scheduled backup")
		if err := runBackup(config); err != nil {
			log.Printf("Scheduled backup failed: %v", err)

			if config.NotifyWebhook != "" {
				sendNotification(config.NotifyWebhook, fmt.Sprintf("Backup failed: %v", err))
			}
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule backup: %v", err)
	}

	log.Printf("Backup scheduler started with schedule: %s", config.Schedule)
	scheduler.Run()
}

func sendNotification(webhookURL string, message string) {
	if webhookURL == "" {
		return
	}

	// Формируем URL с сообщением
	fullURL := fmt.Sprintf("%s?message=%s", webhookURL, url.QueryEscape(message))

	// Отправляем GET-запрос
	resp, err := http.Get(fullURL)
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Notification failed with status: %s", resp.Status)
	} else {
		log.Printf("Notification sent successfully")
	}
}
