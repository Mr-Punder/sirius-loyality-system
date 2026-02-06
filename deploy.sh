#!/bin/bash

REMOTE_PATH="/opt/sirius"

print_usage() {
    echo "Использование: ./deploy.sh USER@HOST [REMOTE_PATH]"
    echo ""
    echo "Примеры:"
    echo "  ./deploy.sh root@192.168.1.100"
    echo "  ./deploy.sh admin@example.com /opt/sirius"
}

if [ -z "$1" ]; then
    echo "Ошибка: не указан адрес удаленной машины"
    echo ""
    print_usage
    return 1 2>/dev/null || true
fi

REMOTE_TARGET="$1"

if [[ ! "$REMOTE_TARGET" =~ @ ]]; then
    echo "Ошибка: неверный формат. Используйте USER@HOST"
    echo ""
    print_usage
    return 1 2>/dev/null || true
fi

if [ -n "$2" ]; then
    REMOTE_PATH="$2"
fi

echo "🔨 Сборка бинарников для Linux..."
cd server || { echo "Ошибка: директория server не найдена"; return 1 2>/dev/null || true; }

export GOOS=linux
export GOARCH=amd64

go mod download

echo "  - loyalityserver..."
CGO_ENABLED=0 go build -o loyalityserver ./cmd/loyalityserver

echo "  - userbot..."
CGO_ENABLED=0 go build -o userbot ./cmd/telegrambot/user

echo "  - adminbot..."
CGO_ENABLED=0 go build -o adminbot ./cmd/telegrambot/admin

cd ..
echo "✅ Сборка завершена"
echo ""

echo "⏸️  Остановка сервисов..."
ssh "$REMOTE_TARGET" "sudo systemctl stop sirius-server.service sirius-userbot.service sirius-adminbot.service || true"
echo ""

echo "📦 Загрузка файлов..."
scp server/loyalityserver server/userbot server/adminbot "$REMOTE_TARGET:$REMOTE_PATH/bin/"
scp -r server/static/* "$REMOTE_TARGET:$REMOTE_PATH/static/"
scp systemd/*.service "$REMOTE_TARGET:/tmp/"
ssh "$REMOTE_TARGET" "sudo mv /tmp/*.service /etc/systemd/system/ && sudo systemctl daemon-reload"
echo ""

echo "▶️  Запуск сервисов..."
ssh "$REMOTE_TARGET" "sudo chmod 755 $REMOTE_PATH/bin/*"
ssh "$REMOTE_TARGET" "sudo systemctl start sirius-server.service"
sleep 2
ssh "$REMOTE_TARGET" "sudo systemctl start sirius-userbot.service"
ssh "$REMOTE_TARGET" "sudo systemctl start sirius-adminbot.service"
echo ""

echo "✅ Деплой завершен"
echo ""
ssh "$REMOTE_TARGET" "sudo systemctl status sirius-server.service --no-pager -l || true"
