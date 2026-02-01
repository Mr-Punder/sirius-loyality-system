#!/bin/bash

set -e

echo "=== Настройка production окружения в /opt/sirius ==="

if [ -d /opt/sirius ]; then
    echo "⚠ Директория /opt/sirius уже существует"
    read -p "Продолжить? Существующие файлы будут сохранены. (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Создание структуры директорий..."
sudo mkdir -p /opt/sirius/{bin,config,logs,static,migrations}

echo "Копирование статических файлов и миграций..."
sudo cp -r server/static/* /opt/sirius/static/
sudo cp -r server/migrations/* /opt/sirius/migrations/

if [ -f production.yaml ]; then
    echo "Копирование конфигурации..."
    sudo cp production.yaml /opt/sirius/production.yaml
    echo "✓ Скопирован как /opt/sirius/production.yaml"
fi

echo "Настройка прав доступа..."
sudo chown -R $USER:$USER /opt/sirius
sudo chmod -R 755 /opt/sirius

echo ""
echo "=== Установка systemd сервисов ==="
if [ -d systemd ]; then
    sudo cp systemd/*.service /etc/systemd/system/
    sudo systemctl daemon-reload
    echo "✓ Сервисы установлены"
fi

if [ ! -f /opt/sirius/config/token.txt ]; then
    echo ""
    echo "⚠ Создайте файлы конфигурации:"
    echo "  nano /opt/sirius/config/token.txt           # Токен пользовательского бота"
    echo "  nano /opt/sirius/config/admin_token.txt     # Токен административного бота"
    echo "  nano /opt/sirius/config/api_token.txt       # API токен"
    echo "  nano /opt/sirius/config/admins.json         # Список админов"
    echo ""
fi

echo ""
echo "═══════════════════════════════════════════════════════"
echo "✅ Production окружение готово!"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "📁 Структура /opt/sirius:"
echo "   bin/           - бинарные файлы (пусто, заполнится при deploy)"
echo "   config/        - конфигурация и токены (настройте сейчас)"
echo "   logs/          - логи приложения"
echo "   static/        - веб-интерфейс"
echo "   migrations/    - SQL миграции"
echo "   production.yaml - основной конфиг"
echo ""
echo "🔧 Следующие шаги:"
echo "   1. nano /opt/sirius/production.yaml"
echo "      Настройте PostgreSQL connection_string"
echo ""
echo "   2. Создайте токены:"
echo "      nano /opt/sirius/config/token.txt        # Токен userbot"
echo "      nano /opt/sirius/config/admin_token.txt  # Токен adminbot"  
echo "      nano /opt/sirius/config/api_token.txt    # API токен"
echo "      nano /opt/sirius/config/admins.json      # Telegram ID админов"
echo ""
echo "   3. make deploy"
echo "      Соберёт, установит и запустит всё"
echo ""
echo "   4. make enable"
echo "      Включит автозапуск при загрузке"
echo ""
echo "═══════════════════════════════════════════════════════"
