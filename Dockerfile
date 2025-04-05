FROM golang:1.23-alpine AS builder

# Устанавливаем необходимые пакеты для CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY server/ ./
RUN go mod download
# Сборка основного сервера с CGO
RUN CGO_ENABLED=1 GOOS=linux go build -o loyalityserver ./cmd/loyalityserver
# Сборка пользовательского бота
RUN go build -o userbot ./cmd/telegrambot/user
# Сборка административного бота
RUN go build -o adminbot ./cmd/telegrambot/admin

FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates tzdata sqlite supervisor

WORKDIR /app

# Копируем бинарные файлы из builder
COPY --from=builder /app/loyalityserver .
COPY --from=builder /app/userbot .
COPY --from=builder /app/adminbot .

# Копируем статические файлы и миграции
COPY server/static/ ./static/
COPY server/migrations/ ./migrations/

# Создаем директории для данных, логов и конфигурации
RUN mkdir -p /app/data /app/logs /app/config

# Копируем конфигурационный файл
COPY server/cmd/loyalityserver/config.yaml ./config/

# Настройка supervisord для управления процессами
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 80

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
