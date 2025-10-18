# Backup Utility

Минималистичная утилита для бэкапа в Docker с поддержкой ротации.

## Переменные окружения

Все переменные окружения являются необязательными. Если переменная не указана, используется значение по умолчанию.

### Настройки источников

При минимальной конфигурации должен быть доступен хотя бы один.

| Переменная   | По умолчанию   | Описание                                                                                      |
| ------------ | -------------- | --------------------------------------------------------------------------------------------- |
| `SOURCE_DIR` | `/data/source` | Папка для бэкапа (можно передать как аргумент). Если не смонтирована, то бэкап не выполняется |
| `SOURCE_DB`  | -              | URI подключения к БД (postgres://, mysql://, mongodb://)                                      |

### Настройки хранилища

Обязательно должен быть доступен для записи.

| Переменная       | По умолчанию   | Описание                                                   |
| ---------------- | -------------- | ---------------------------------------------------------- |
| `TARGET_STORAGE` | `/data/target` | Локальная папка или URI хранилища (s3://, gs://, azure://) |

**Поддерживаемые типы хранилищ:**

- **Локальная файловая система**: `/path/to/backups`
- **Amazon S3**: `s3://access_key:secret_key@bucket-name/prefix`
- **Yandex Cloud Object Storage**: `s3://access_key:secret_key@storage.yandexcloud.net/bucket/prefix`
- **Другие S3-совместимые хранилища**: `s3://access_key:secret_key@endpoint.com/bucket/prefix`

**Примеры:**

- AWS S3: `s3://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY@my-bucket/backups`
- Yandex Cloud: `s3://YCAJEWXOyY8Bmyk2eJL-nls:YCPs52ajb2jNXxOUsL4-pFDL8gZNC1tK5OC8ICOg@storage.yandexcloud.net/my-bucket/backups`

### Настройки бэкапов

| Переменная            | По умолчанию | Описание                                                        |
| --------------------- | ------------ | --------------------------------------------------------------- |
| `BACKUP_RETENTION`    | `7d`         | Политика хранения бэкапов (1d, 2w, 1m, 1w2d)                    |
| `BACKUP_MIN_RETAINED` | `3`          | Минимальное количество бэкапов, которые всегда сохраняются      |
| `BACKUP_PREFIX`       | `backup`     | Префикс имен архивов (формат: `prefix_YYYY-MM-DD_HH-MM.tar.gz`) |

### Настройки выполнения

| Переменная           | По умолчанию | Описание                                      |
| -------------------- | ------------ | --------------------------------------------- |
| `RUN_SCHEDULE`       | `0 4 * * *`  | Расписание в cron-формате (ежедневно в 04:00) |
| `RUN_ON_STARTUP`     | `true`       | Сделать бэкап при старте (`true`/`false`)     |
| `RUN_LOG_LEVEL`      | `info`       | Уровень логирования (debug/info/warn/error)   |
| `RUN_NOTIFY_WEBHOOK` | -            | URL для GET-уведомлений (шаблон: {message})   |

## Примеры

### Минимальная конфигурация

```yaml
version: "3"
services:
  backup:
    image: your-backup-image
    # Тома для монтирования локальных директорий
    volumes:
      - ./source:/data/source # Исходная директория для бэкапа
      - ./backups:/data/target # Целевая директория для хранения бэкапов
```

### Полная конфигурация

```yaml
version: "3"
services:
  backup:
    image: your-backup-image
    environment:
      SOURCE_DIR: /data/source
      TARGET_STORAGE: /data/backups
      BACKUP_RETENTION: 2w
      BACKUP_PREFIX: myapp
      RUN_LOG_LEVEL: debug
      RUN_SCHEDULE: "0 3 * * *"
      RUN_ON_STARTUP: "true"
      RUN_NOTIFY_WEBHOOK: "https://example.com/alert?msg={message}"
    # Тома для монтирования локальных директорий
    volumes:
      - ./source:/data/source # Исходная директория для бэкапа
      - ./backups:/data/backups # Директория для хранения бэкапов
```

### Запуск с cron-расписанием

```yaml
version: "3"
services:
  backup:
    image: your-backup-image
    environment:
      # Формат: "минута час день_месяца месяц день_недели"
      # Примеры:
      # - Каждые 6 часов: "0 */6 * * *"
      # - Ежедневно в 4:00 (по умолчанию): "0 4 * * *"
      # - Каждые 30 минут: "*/30 * * * *"
      RUN_SCHEDULE: "0 */6 * * *"
    volumes:
      - ./source:/data/source
      - ./backups:/data/backups
```

## Формат файлов бэкапов

Файлы бэкапов создаются с именами в формате `prefix_YYYY-MM-DD_HH-MM_plugin.extension`:

- `backup_2025-10-18_15-30_directory.tar.gz`
- `backup_2025-10-18_15-30_mysql.sql.gz`
- `myapp_2025-10-18_09-45_directory.tar.gz`

Где:

- `prefix` - значение переменной `BACKUP_PREFIX` (по умолчанию `backup`)
- `YYYY-MM-DD` - дата в формате год-месяц-день
- `HH-MM` - время в формате час-минуты (24-часовой формат, UTC)
- `plugin` - имя плагина источника (`directory`, `mysql`, и т.д.)
- `extension` - расширение файла, зависящее от типа плагина (`.tar.gz`, `.sql.gz`)

**Важно:** При одновременном использовании нескольких источников (например, `SOURCE_DIR` и `SOURCE_DB`) создаются отдельные файлы для каждого плагина с одинаковой датой и временем, но разными суффиксами плагинов.

## Детали хранения бэкапов

При дефолтных настройках (BACKUP_RETENTION=7d, BACKUP_MIN_RETAINED=3):

- Хранятся все бэкапы за последние 7 дней
- Но не менее 3 бэкапов (даже если они старше 7 дней)
- Примерный объем: 7 \* размер_бэкапа + резерв (3 бэкапа)

Примеры конфигураций:

1. Бэкап каждый час, хранить неделю:

```yaml
RUN_SCHEDULE: "0 * * * *" # Каждый час (24 бэкапа в день)
BACKUP_RETENTION: "7d" # Хранить 7 дней (примерно 168 бэкапов)
BACKUP_MIN_RETAINED: 3 # Гарантированно хранить минимум 3 бэкапа
```

2. Бэкап раз в день, хранить 1 день:

```yaml
RUN_SCHEDULE: "0 0 * * *" # Раз в день в полночь
BACKUP_RETENTION: "1d" # Хранить 1 день
# Будет храниться минимум 3 бэкапа (BACKUP_MIN_RETAINED)
# даже если они старше 1 дня
```

## Обработка ошибок и уведомления

При неудачном бэкапе утилита:

1. Запишет ошибку в лог (уровень ERROR)
2. Завершится с ненулевым кодом возврата (1)
3. Не будет удалять старые бэкапы при ошибке
4. При RUN_ON_STARTUP=true - попробует сделать бэкап при следующем запуске

Zero-config уведомления:

- По умолчанию ошибки только в логах
- Можно настроить вебхук через переменную RUN_NOTIFY_WEBHOOK (опционально)

### Поведение при повторных ошибках:

- После 3 неудачных попыток подряд утилита перейдет в режим "ошибка"
- В этом режиме:
  - Не будет пытаться делать новые бэкапы
  - Будет писать ошибки в лог
  - Требует ручного вмешательства (устранение причины ошибки)

## Логирование

Утилита пишет логи в stdout в формате:

```
[LEVEL] timestamp message
```

## Docker

### Использование готового образа

Образы автоматически публикуются в GitHub Container Registry через CI/CD:

```bash
# Последняя стабильная версия
docker pull ghcr.io/egmen/backup-utility:latest

# Конкретная версия
docker pull ghcr.io/egmen/backup-utility:v1.0.0

# По хешу коммита
docker pull ghcr.io/egmen/backup-utility:d7e9e06
```

Образы доступны для платформ:

- `linux/amd64` (x86_64)
- `linux/arm64` (ARM64, включая Apple Silicon)

### Сборка образа локально

Для сборки Docker образа выполните команду из корневой директории проекта:

```bash
docker build -t backup-utility .
```

### Запуск с .env файлом

Создайте `.env` файл с необходимыми переменными окружения:

```bash
# .env
SOURCE_DIR=/data/source
TARGET_STORAGE=/data/backups
BACKUP_RETENTION=7d
BACKUP_MIN_RETAINED=3
BACKUP_PREFIX=backup
RUN_SCHEDULE=0 4 * * *
RUN_ON_STARTUP=true
RUN_LOG_LEVEL=info
# RUN_NOTIFY_WEBHOOK=https://example.com/alert?msg={message}
```

Запустите контейнер с .env файлом:

```bash
docker run -d \
  --name backup-service \
  --env-file .env \
  -v ./example/source:/data/source \
  -v ./example/backups:/data/backups \
  backup-utility
```

### Запуск разового бэкапа

Для выполнения разового бэкапа без расписания:

```bash
docker run --rm \
  --env-file .env \
  -e RUN_SCHEDULE="" \
  -v ./example/source:/data/source \
  -v ./example/backups:/data/backups \
  backup-utility
```

### Остановка контейнера

Для остановки работающего контейнера с cron:

```bash
docker stop backup-service
```

Для удаления остановленного контейнера:

```bash
docker rm backup-service
```

### Запуск с автоматическим удалением

Если нужно, чтобы контейнер автоматически удалялся после остановки:

```bash
docker run -d \
  --name backup-service \
  --rm \
  --env-file .env \
  -v ./source:/data/source \
  -v ./backups:/data/backups \
  backup-utility
```

### Запуск с ограничением по времени

Для запуска контейнера на определенное время (например, на 1 час):

```bash
timeout 3600 docker run --rm \
  --env-file .env \
  -v ./source:/data/source \
  -v ./backups:/data/backups \
  backup-utility
```

### Запуск с S3 хранилищем

Создайте `.env` файл для S3-совместимых хранилищ:

```bash
# .env для AWS S3
SOURCE_DIR=/data/source
TARGET_STORAGE=s3://your_access_key:your_secret_key@my-backup-bucket/app-backups
BACKUP_RETENTION=14d
BACKUP_MIN_RETAINED=5
```

```bash
# .env для Yandex Cloud Object Storage
SOURCE_DIR=/data/source
TARGET_STORAGE=s3://your_access_key:your_secret_key@storage.yandexcloud.net/my-bucket/app-backups
BACKUP_RETENTION=14d
BACKUP_MIN_RETAINED=5
```

### Запуск с Yandex Cloud Object Storage

Для Yandex Cloud используйте следующий формат:

```bash
docker run -d \
  --name backup-service \
  -e SOURCE_DIR=/data/source \
  -e TARGET_STORAGE=s3://your_access_key:your_secret_key@storage.yandexcloud.net/your-bucket/backups \
  -v ./example/source:/data/source \
  backup-utility
```

### Запуск с AWS S3

Для AWS S3 используйте упрощенный формат:

```bash
docker run -d \
  --name backup-service \
  -e SOURCE_DIR=/data/source \
  -e TARGET_STORAGE=s3://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY@my-bucket/backups \
  -v ./example/source:/data/source \
  backup-utility
```

### Запуск с другими S3-совместимыми хранилищами

Для других провайдеров (MinIO, DigitalOcean Spaces и т.д.):

```bash
docker run -d \
  --name backup-service \
  -e SOURCE_DIR=/data/source \
  -e TARGET_STORAGE=s3://access_key:secret_key@your-endpoint.com/bucket/backups \
  -v ./example/source:/data/source \
  backup-utility
```

**Примечание:** При использовании S3-совместимых хранилищ не нужно монтировать volume для `/data/backups`, так как бэкапы будут храниться в облаке.

### Запуск с бэкапом MySQL/MariaDB

Для бэкапа MySQL базы данных используйте переменную `SOURCE_DB`:

```bash
docker run -d \
  --name backup-service \
  -e SOURCE_DB=mysql://user:password@mysql-host:3306/database_name \
  -e TARGET_STORAGE=/data/backups \
  -v ./backups:/data/backups \
  backup-utility
```

**Примечания:**

- Контейнер должен иметь доступ к утилите `mariadb-dump` или `mysqldump` (включена в образ)
- Автоматически определяется доступная утилита (предпочитается `mariadb-dump`)
- Можно комбинировать бэкап директории и базы данных одновременно:

```bash
docker run -d \
  --name backup-service \
  -e SOURCE_DIR=/data/source \
  -e SOURCE_DB=mysql://user:password@mysql-host:3306/database_name \
  -e TARGET_STORAGE=/data/backups \
  -v ./source:/data/source \
  -v ./backups:/data/backups \
  --network host \
  backup-utility
```

При такой конфигурации будут созданы два файла:

- `backup_2025-10-18_15-30_directory.tar.gz` - архив директории
- `backup_2025-10-18_15-30_mysql.sql.gz` - дамп базы данных

### Проверка логов

Для просмотра логов работающего контейнера:

```bash
docker logs backup-service -f
```

## Архитектура

Утилита использует модульную плагинную архитектуру:

```
pkg/
├── source/
│   ├── source.go            # Интерфейс и менеджер источников
│   ├── plugin_directory.go  # Плагин бэкапа локальной директории
│   └── plugin_mysql.go      # Плагин бэкапа MySQL/MariaDB
└── storage/
    ├── storage.go           # Интерфейс и менеджер хранилищ
    ├── plugin_local.go      # Плагин локального файлового хранилища
    └── plugin_s3.go         # Плагин S3-совместимых хранилищ
```

### Плагины источников (source):

- **Directory Plugin** - создает tar.gz архивы из локальной директории
- **MySQL Plugin** - создает SQL дампы MySQL/MariaDB баз данных (через `mariadb-dump` или `mysqldump`)

Плагины источников можно комбинировать. При указании и `SOURCE_DIR` и `SOURCE_DB` будут созданы оба бэкапа.

### Плагины хранилища (storage):

- **Local Plugin** - локальная файловая система
- **S3 Plugin** - AWS S3, Yandex Cloud, MinIO и другие S3-совместимые сервисы
