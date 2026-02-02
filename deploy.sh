#!/bin/bash

set -e

REMOTE_USER=""
REMOTE_HOST=""
REMOTE_PATH="/opt/sirius"

print_usage() {
    echo "Использование: ./deploy.sh USER@HOST [REMOTE_PATH]"
    echo ""
    echo "Примеры:"
    echo "  ./deploy.sh root@192.168.1.100"
    echo "  ./deploy.sh admin@example.com /opt/sirius"
    echo ""
    echo "Опции:"
    echo "  USER@HOST    - логин и адрес удаленной машины"
    echo "  REMOTE_PATH  - путь на удаленной машине (по умолчанию: /opt/sirius)"
}

if [ -z "$1" ]; then
    echo "Ошибка: не указан адрес удаленной машины"
    echo ""
    print_usage
    exit 1
fi

REMOTE_TARGET="$1"

if [[ ! "$REMOTE_TARGET" =~ @ ]]; then
    echo "Ошибка: неверный формат. Используйте USER@HOST"
    echo ""
    print_usage
    exit 1
fi

if [ -n "$2" ]; then
    REMOTE_PATH="$2"
fi

echo "🔨 Шаг 1/4: Сборка бинарников для Linux..."
cd server
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1

go mod download

echo "  - Сборка loyalityserver..."
go build -o loyalityserver ./cmd/loyalityserver

echo "  - Сборка userbot..."
go build -o userbot ./cmd/telegrambot/user

echo "  - Сборка adminbot..."
go build -o adminbot ./cmd/telegrambot/admin

cd ..
echo "✅ Бинарники собраны"
echo ""

echo "⏸️  Шаг 2/4: Остановка удаленных сервисов..."
ssh "$REMOTE_TARGET" "sudo systemctl stop sirius-server.service sirius-userbot.service sirius-adminbot.service 2>/dev/null || true"
echo "✅ Сервисы остановлены"
echo ""

echo "📦 Шаг 3/4: Загрузка файлов на удаленную машину..."
echo "  - Загрузка бинарников..."
scp server/loyalityserver server/userbot server/adminbot "$REMOTE_TARGET:$REMOTE_PATH/bin/"

echo "  - Загрузка static файлов..."
scp -r server/static/* "$REMOTE_TARGET:$REMOTE_PATH/static/"

echo "  - Загрузка миграций..."
scp -r server/migrations/* "$REMOTE_TARGET:$REMOTE_PATH/migrations/"

echo "  - Загрузка systemd сервисов..."
scp systemd/*.service "$REMOTE_TARGET:/tmp/"
ssh "$REMOTE_TARGET" "sudo mv /tmp/*.service /etc/systemd/system/ && sudo systemctl daemon-reload"

echo "✅ Файлы загружены"
echo ""

echo "▶️  Шаг 4/4: Запуск сервисов..."
ssh "$REMOTE_TARGET" "sudo chmod 755 $REMOTE_PATH/bin/*"
ssh "$REMOTE_TARGET" "sudo systemctl start sirius-server.service"
sleep 2
ssh "$REMOTE_TARGET" "sudo systemctl start sirius-userbot.service"
ssh "$REMOTE_TARGET" "sudo systemctl start sirius-adminbot.service"
echo "✅ Сервисы запущены"
echo ""

echo "═══════════════════════════════════════════════════════"
echo "✅ Деплой завершен!"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "📋 Проверка статуса:"
ssh "$REMOTE_TARGET" "sudo systemctl status sirius-server.service --no-pager -l || true"
echo ""
echo "📋 Команды для просмотра логов:"
echo "  ssh $REMOTE_TARGET 'sudo journalctl -u sirius-server.service -f'"
echo "  ssh $REMOTE_TARGET 'sudo journalctl -u sirius-userbot.service -f'"
echo "  ssh $REMOTE_TARGET 'sudo journalctl -u sirius-adminbot.service -f'"
echo ""
