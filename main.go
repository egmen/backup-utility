package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
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

func createArchive(archivePath string, sourceDir string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if _, err := io.Copy(tarWriter, file); err != nil {
			return err
		}

		return nil
	})
}

func applyRetentionPolicy(config Config) error {
	files, err := os.ReadDir(config.TargetStorage)
	if err != nil {
		return err
	}

	var backups []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), config.BackupPrefix) {
			backups = append(backups, file)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		infoI, _ := backups[i].Info()
		infoJ, _ := backups[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	if len(backups) > config.MinRetained {
		for i := config.MinRetained; i < len(backups); i++ {
			filePath := filepath.Join(config.TargetStorage, backups[i].Name())
			if err := os.Remove(filePath); err != nil {
				log.Printf("Failed to remove old backup: %v", err)
			} else {
				log.Printf("Removed old backup: %s", filePath)
			}
		}
	}

	return nil
}

func runBackup(config Config) error {
	log.Printf("Starting backup process")

	if config.SourceDir != "" {
		if _, err := os.Stat(config.SourceDir); os.IsNotExist(err) {
			return fmt.Errorf("source directory does not exist: %s", config.SourceDir)
		}
	}

	if err := os.MkdirAll(config.TargetStorage, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	backupPath := fmt.Sprintf("%s/%s-%s.tar.gz",
		config.TargetStorage,
		config.BackupPrefix,
		time.Now().UTC().Format(time.RFC3339))

	log.Printf("Creating backup at: %s", backupPath)

	if config.SourceDir != "" {
		if err := createArchive(backupPath, config.SourceDir); err != nil {
			return fmt.Errorf("failed to create archive: %v", err)
		}
		log.Printf("Backup created successfully: %s", backupPath)
	}

	if err := applyRetentionPolicy(config); err != nil {
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
