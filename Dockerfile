FROM golang:rc-alpine AS builder

# Устанавливаем необходимые пакеты для CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY server/ ./
RUN go mod download
# Сборка основного сервера с CGO
RUN GOOS=linux go build -o loyalityserver ./cmd/loyalityserver
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

COPY config/ ./config/

# Создаем файл .dockerenv для определения запуска в Docker
RUN touch /.dockerenv

# Устанавливаем переменные окружения для путей
ENV ADMIN_STATIC_DIR="/app/static/admin"
ENV ADMIN_ADMINS_PATH="/app/config/admins.json"
ENV CONFIG_PATH="/app/config/config.yaml"
ENV TOKEN_PATH="/app/config/token.txt"
ENV ADMIN_TOKEN_PATH="/app/config/admin_token.txt"
ENV API_TOKEN_PATH="/app/config/api_token.txt"
ENV MIGRATIONS_PATH="/app/migrations/sqlite"
ENV DB_PATH="/app/data/loyality_system.db"

# Настройка supervisord для управления процессами
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 80

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
