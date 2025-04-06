FROM golang:1.22 AS builder

# Устанавливаем необходимые пакеты для CGO
RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app
COPY server/ ./
RUN go mod download
# Сборка основного сервера с CGO, статически связанного для Alpine
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-linkmode external -extldflags -static" -o loyalityserver ./cmd/loyalityserver
# Сборка пользовательского бота
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o userbot ./cmd/telegrambot/user
# Сборка административного бота
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o adminbot ./cmd/telegrambot/admin

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

RUN echo "$2a$12$aKiyCQdQMD//SK8RZrh/qunPiAoo.HF9PNjC73NODQmx2Kcfor65y" > ./data/admin_password.hash


# Создаем файл .dockerenv для определения запуска в Docker
RUN touch /.dockerenv

# Устанавливаем пеcat ременные окружения для путей
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
