# Быстрый старт: Публикация Docker образов

## 📦 Что уже настроено

✅ GitHub Actions workflow для автоматической сборки и публикации
✅ Поддержка мультиплатформенности (amd64 и arm64)
✅ Автоматическое тегирование образов
✅ Публикация в GitHub Container Registry (ghcr.io)
✅ Кеширование для быстрых сборок
✅ Security attestation для образов

## 🚀 Как использовать

### Вариант 1: Автоматическая публикация при push

Просто запушьте код в ветку `main` или `release`:

```bash
git add .
git commit -m "Ваши изменения"
git push origin release
```

Образ автоматически соберется и опубликуется с тегом `release`.

### Вариант 2: Релиз с версией

Создайте тег версии и запушьте его:

```bash
# Создать тег
git tag -a v1.0.0 -m "Release version 1.0.0"

# Запушить тег
git push origin v1.0.0
```

Образ опубликуется с тегами:

- `ghcr.io/egmen/backup-utility:v1.0.0`
- `ghcr.io/egmen/backup-utility:1.0.0`
- `ghcr.io/egmen/backup-utility:1.0`
- `ghcr.io/egmen/backup-utility:1`
- `ghcr.io/egmen/backup-utility:latest`

### Вариант 3: Проверка перед публикацией (Pull Request)

Создайте Pull Request в `main` или `release`:

```bash
git checkout -b feature/my-feature
git add .
git commit -m "Моя фича"
git push origin feature/my-feature
```

Создайте PR на GitHub → образ соберется, но НЕ опубликуется.
Это позволяет проверить, что образ собирается без ошибок.

## 📥 Использование опубликованных образов

### Локальное использование

```bash
# Скачать последнюю версию
docker pull ghcr.io/egmen/backup-utility:latest

# Скачать конкретную версию
docker pull ghcr.io/egmen/backup-utility:v1.0.0

# Запустить
docker run ghcr.io/egmen/backup-utility:latest
```

### В Docker Compose

```yaml
version: "3.8"

services:
  backup:
    image: ghcr.io/egmen/backup-utility:latest
    volumes:
      - ./data:/data/source
      - ./backups:/data/backups
    environment:
      - BACKUP_RETENTION=7d
      - RUN_SCHEDULE=0 4 * * *
```

### В Kubernetes

```yaml
apiVersion: apps/v1
kind: CronJob
metadata:
  name: backup-job
spec:
  schedule: "0 4 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: ghcr.io/egmen/backup-utility:v1.0.0
              volumeMounts:
                - name: data
                  mountPath: /data/source
```

## 🔍 Мониторинг

### Посмотреть статус сборки

1. Откройте GitHub → перейдите в репозиторий
2. Вкладка **Actions**
3. Выберите последний запуск **Build and Publish Docker Image**

### Посмотреть опубликованные образы

1. Откройте профиль GitHub: https://github.com/egmen
2. Вкладка **Packages**
3. Найдите `backup-utility`

Или напрямую: https://github.com/egmen/backup-utility/pkgs/container/backup-utility

## 🔐 Настройка видимости образа

По умолчанию образы могут быть приватными. Чтобы сделать их публичными:

1. Перейдите на страницу пакета
2. **Package settings** (справа)
3. **Change visibility** → **Public**

## 🛠️ Что делать при ошибках

### "Error: Unable to publish package"

**Решение:**

1. Settings → Actions → General
2. Workflow permissions → **Read and write permissions**
3. Сохранить

### "Error: buildx failed"

**Решение:**
Проблема со сборкой кода. Проверьте логи workflow и исправьте ошибки в Dockerfile или коде.

### "Rate limit exceeded"

**Решение:**
GitHub имеет лимиты на сборки. Подождите или оптимизируйте workflow (используйте кеш).

## 📋 Checklist первого запуска

- [ ] Запушить код в `release` или `main`
- [ ] Открыть Actions и проверить, что workflow запустился
- [ ] Дождаться завершения сборки (5-10 минут)
- [ ] Проверить, что образ появился в Packages
- [ ] (Опционально) Сделать образ публичным
- [ ] Протестировать: `docker pull ghcr.io/egmen/backup-utility:latest`

## 💡 Советы

1. **Используйте теги для релизов** - это позволяет откатиться к предыдущей версии
2. **Проверяйте через PR** - создавайте Pull Request для проверки сборки
3. **Semantic Versioning** - используйте формат `v1.2.3` для версий
4. **Changelog** - ведите список изменений в каждом релизе

## 📚 Дополнительные материалы

- Полная документация: [.github/workflows/README.md](../workflows/README.md)
- GitHub Container Registry: https://docs.github.com/packages
- Docker multi-platform: https://docs.docker.com/build/building/multi-platform/

---

**Готово!** 🎉 Теперь ваши Docker образы публикуются автоматически.
