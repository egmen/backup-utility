# GitHub Actions CI/CD для Docker

## Обзор

Этот проект использует GitHub Actions для автоматической сборки и публикации Docker образов в GitHub Container Registry (ghcr.io).

## Workflow: docker-publish.yml

### Триггеры

Workflow запускается при:

- **Push в ветку**: `main`
- **Push тегов**: любые теги начинающиеся с `v` (например, `v1.0.0`, `v2.1.3`)
- **Pull Request**: в ветку `main` (только сборка, без публикации)

### Что делает workflow

1. **Checkout кода** - клонирует репозиторий
2. **Настройка Docker Buildx** - для поддержки мультиплатформенной сборки
3. **Авторизация в GHCR** - использует `GITHUB_TOKEN` для доступа к GitHub Container Registry
4. **Генерация метаданных** - автоматически создает теги и метки для образа
5. **Сборка и публикация** - собирает образ для `linux/amd64` и `linux/arm64`, публикует в registry
6. **Attestation** - создает подтверждение происхождения образа для безопасности

### Теги образов

Workflow автоматически создает следующие теги:

| Событие          | Примеры тегов                           |
| ---------------- | --------------------------------------- |
| Push в `main`    | `main`, `latest`, `sha-d7e9e06`         |
| Tag `v1.2.3`     | `v1.2.3`, `1.2.3`, `1.2`, `1`, `latest` |
| Pull Request #42 | `pr-42` (только сборка)                 |

### Использование образов

После публикации, образы доступны по адресу:

```bash
# Последняя версия из main
docker pull ghcr.io/egmen/backup-utility:latest

# Конкретная версия
docker pull ghcr.io/egmen/backup-utility:v1.0.0

# Версия по SHA коммита
docker pull ghcr.io/egmen/backup-utility:sha-d7e9e06
```

## Настройка

### 1. Включение GitHub Container Registry

Убедитесь, что GitHub Container Registry включен для вашего репозитория:

1. Перейдите в **Settings** → **Packages**
2. Убедитесь, что пакеты могут быть созданы

### 2. Permissions для workflow

Workflow использует встроенный `GITHUB_TOKEN` с правами:

- `contents: read` - чтение кода
- `packages: write` - публикация в GHCR
- `id-token: write` - для attestation

Эти права уже настроены в workflow и не требуют дополнительной настройки.

### 3. Видимость образов

По умолчанию образы могут быть приватными. Чтобы сделать их публичными:

1. Перейдите на страницу пакета: `https://github.com/users/egmen/packages/container/backup-utility`
2. Нажмите **Package settings**
3. В разделе **Danger Zone** измените видимость на **Public**

## Релизы

### Создание нового релиза

Для создания нового релиза с версионированным образом:

```bash
# Создайте и запушьте тег
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

Это автоматически:

1. Запустит workflow
2. Соберет образ для обеих платформ
3. Опубликует с тегами: `v1.0.0`, `1.0.0`, `1.0`, `1`, `latest`

### Semantic Versioning

Workflow поддерживает семантическое версионирование:

- `v1.2.3` → создаст теги `1.2.3`, `1.2`, `1`, `latest`
- `v2.0.0-beta.1` → создаст тег `2.0.0-beta.1`
- `v1.0.0-rc.1` → создаст тег `1.0.0-rc.1`

## Мультиплатформенная поддержка

Образы собираются для следующих платформ:

- `linux/amd64` - x86_64 архитектура
- `linux/arm64` - ARM64 архитектура (Apple Silicon, серверы ARM)

Docker автоматически выберет правильный образ для вашей платформы.

## Кеширование

Workflow использует GitHub Actions Cache для ускорения сборки:

- Кеш Docker слоев между запусками
- Значительно ускоряет повторные сборки
- Автоматически управляется GitHub

## Безопасность

### Attestation

Workflow создает подтверждение происхождения (provenance attestation) для каждого образа, что позволяет:

- Верифицировать, что образ был собран из конкретного исходного кода
- Отслеживать цепочку поставок (supply chain)
- Соответствовать требованиям безопасности

### Проверка attestation

```bash
# Установите GitHub CLI
gh attestation verify oci://ghcr.io/egmen/backup-utility:latest \
  --owner egmen
```

## Мониторинг

### Просмотр запусков workflow

1. Перейдите на вкладку **Actions** в репозитории
2. Выберите **Build and Publish Docker Image**
3. Просмотрите историю запусков и логи

### Просмотр опубликованных образов

1. Перейдите на вкладку **Packages** в профиле GitHub
2. Найдите `backup-utility`
3. Просмотрите все доступные теги и метаданные

## Troubleshooting

### Ошибка при публикации

Если workflow падает на этапе публикации:

1. Проверьте права доступа в **Settings** → **Actions** → **General** → **Workflow permissions**
2. Убедитесь, что выбрано "Read and write permissions"

### Образ не появляется в GHCR

1. Проверьте логи workflow в разделе Actions
2. Убедитесь, что это не Pull Request (PR только собирают, но не публикуют)
3. Проверьте, что push был в ветку `main` или создан тег

### Проблемы с мультиплатформенной сборкой

Если сборка для ARM64 падает:

1. Проверьте, что все зависимости поддерживают ARM64
2. Временно можно убрать `linux/arm64` из `platforms` в workflow

## Дополнительная настройка

### Изменение имени образа

Измените переменную в workflow:

```yaml
env:
  IMAGE_NAME: custom-name # вместо ${{ github.repository }}
```

### Добавление дополнительных платформ

Добавьте в список `platforms`:

```yaml
platforms: linux/amd64,linux/arm64,linux/arm/v7
```

### Публикация в несколько registry

Добавьте дополнительный шаг логина и измените метаданные:

```yaml
- name: Log in to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}

- name: Extract metadata
  uses: docker/metadata-action@v5
  with:
    images: |
      ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
      dockerhub-username/backup-utility
```
