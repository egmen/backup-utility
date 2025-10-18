FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o backup .

FROM alpine:latest

# Устанавливаем утилиты для бэкапа баз данных
RUN apk add --no-cache mysql-client

WORKDIR /app
COPY --from=builder /app/backup .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

VOLUME ["/data/source", "/data/backups"]

ENV SOURCE_DIR=/data/source \
    TARGET_STORAGE=/data/backups \
    BACKUP_RETENTION=7d \
    BACKUP_MIN_RETAINED=3 \
    BACKUP_PREFIX=backup \
    RUN_SCHEDULE="0 4 * * *" \
    RUN_ON_STARTUP=true \
    RUN_LOG_LEVEL=info

ENTRYPOINT ["./backup"]
