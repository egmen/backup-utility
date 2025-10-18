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

### Настройки бэкапов

| Переменная            | По умолчанию | Описание                                                   |
| --------------------- | ------------ | ---------------------------------------------------------- |
| `BACKUP_RETENTION`    | `7d`         | Политика хранения бэкапов (1d, 2w, 1m, 1w2d)               |
| `BACKUP_MIN_RETAINED` | `3`          | Минимальное количество бэкапов, которые всегда сохраняются |
| `BACKUP_PREFIX`       | `backup`     | Префикс имен архивов                                       |

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

### Сборка образа

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

### Проверка логов

Для просмотра логов работающего контейнера:

```bash
docker logs backup-service -f
```

## Для разработки

set SOURCE_DIR $(pwd)/example/source
set TARGET_STORAGE $(pwd)/example/target
go run main.go

SOURCE_DIR=(pwd)/example/source TARGET_STORAGE=(pwd)/example/target go run main.go
