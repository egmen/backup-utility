package source

import (
	"compress/gzip"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// MySQLPlugin реализует бэкап MySQL/MariaDB базы данных
type MySQLPlugin struct {
	dbURI    string
	host     string
	port     string
	username string
	password string
	database string
}

// NewMySQLPlugin создает новый плагин для бэкапа MySQL
func NewMySQLPlugin(dbURI string) (*MySQLPlugin, error) {
	parsed, err := url.Parse(dbURI)
	if err != nil {
		return nil, fmt.Errorf("invalid MySQL URI: %v", err)
	}

	plugin := &MySQLPlugin{
		dbURI: dbURI,
		host:  parsed.Hostname(),
		port:  parsed.Port(),
	}

	// Извлекаем credentials
	if parsed.User != nil {
		plugin.username = parsed.User.Username()
		plugin.password, _ = parsed.User.Password()
	}

	// Извлекаем имя базы данных из пути
	plugin.database = strings.TrimPrefix(parsed.Path, "/")

	// Устанавливаем порт по умолчанию
	if plugin.port == "" {
		plugin.port = "3306"
	}

	return plugin, nil
}

// GetName возвращает имя плагина для суффикса файлов
func (mp *MySQLPlugin) GetName() string {
	return "mysql"
}

// GetExtension возвращает расширение файла бэкапа
func (mp *MySQLPlugin) GetExtension() string {
	return ".sql.gz"
}

// Validate проверяет доступность MySQL сервера
func (mp *MySQLPlugin) Validate() error {
	// Проверяем наличие утилиты mysqldump или mariadb-dump
	dumpCmd := mp.getDumpCommand()
	if _, err := exec.LookPath(dumpCmd); err != nil {
		return fmt.Errorf("dump utility (%s) not found: %v", dumpCmd, err)
	}

	// TODO: Проверить подключение к серверу
	// Можно использовать mysql/mariadb -e "SELECT 1" для проверки

	return nil
}

// getDumpCommand возвращает команду для создания дампа
// Предпочитает mariadb-dump (новая утилита), затем mysqldump (устаревшая)
func (mp *MySQLPlugin) getDumpCommand() string {
	// Сначала проверяем наличие mariadb-dump (новая утилита)
	if _, err := exec.LookPath("mariadb-dump"); err == nil {
		return "mariadb-dump"
	}
	// Иначе используем mysqldump (устаревшая, но всё ещё работает)
	return "mysqldump"
}

// CreateBackup создает дамп базы данных с помощью mysqldump/mariadb-dump
func (mp *MySQLPlugin) CreateBackup(backupPath string) error {
	// Создаем файл для записи
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %v", err)
	}
	defer file.Close()

	// Создаем gzip writer для сжатия
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Определяем команду для дампа
	dumpCmd := mp.getDumpCommand()

	// Формируем аргументы для mysqldump/mariadb-dump
	args := []string{
		fmt.Sprintf("-h%s", mp.host),
		fmt.Sprintf("-P%s", mp.port),
	}

	if mp.username != "" {
		args = append(args, fmt.Sprintf("-u%s", mp.username))
	}

	if mp.password != "" {
		args = append(args, fmt.Sprintf("-p%s", mp.password))
	}

	// Дополнительные опции
	args = append(args,
		"--single-transaction", // Для InnoDB без блокировок
		"--quick",              // Построчное чтение
		"--routines",           // Включить процедуры и функции
		"--triggers",           // Включить триггеры
	)

	if mp.database != "" {
		args = append(args, mp.database)
	} else {
		// Дамп всех баз данных
		args = append(args, "--all-databases")
	}

	// Создаем команду mysqldump/mariadb-dump
	cmd := exec.Command(dumpCmd, args...)
	cmd.Stdout = gzipWriter
	cmd.Stderr = os.Stderr

	// Выполняем команду
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %v", dumpCmd, err)
	}

	return nil
}

// GetInfo возвращает информацию о MySQL источнике
func (mp *MySQLPlugin) GetInfo() (PluginInfo, error) {
	// TODO: Получить размер базы данных через SQL запрос
	// SELECT table_schema, SUM(data_length + index_length)
	// FROM information_schema.tables
	// WHERE table_schema = 'database_name'

	dbName := mp.database
	if dbName == "" {
		dbName = "all-databases"
	}

	return PluginInfo{
		Name:   mp.GetName(),
		Type:   "mysql",
		Source: fmt.Sprintf("%s@%s:%s/%s", mp.username, mp.host, mp.port, dbName),
		Extra: map[string]interface{}{
			"host":     mp.host,
			"port":     mp.port,
			"database": dbName,
		},
	}, nil
}
