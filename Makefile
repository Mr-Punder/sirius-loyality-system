.PHONY: help setup deploy install start stop restart status logs logs-server logs-userbot logs-adminbot enable disable build clean run-local run-server run-userbot run-adminbot

help:
	@echo "═══════════════════════════════════════════════════════"
	@echo "         Sirius Loyalty System - Команды"
	@echo "═══════════════════════════════════════════════════════"
	@echo ""
	@echo "🚀 БЫСТРЫЙ СТАРТ (первый раз):"
	@echo "  make setup          Создать /opt/sirius (один раз)"
	@echo "  make deploy         Собрать + установить + запустить"
	@echo "  make enable         Включить автозапуск"
	@echo ""
	@echo "🔄 ОБНОВЛЕНИЕ КОДА:"
	@echo "  make deploy         Собрать + установить + перезапустить"
	@echo ""
	@echo "🔧 ПО ШАГАМ:"
	@echo "  make build          1. Собрать бинарники"
	@echo "  make install        2. Скопировать в /opt/sirius"
	@echo "  make restart        3. Перезапустить сервисы"
	@echo ""
	@echo "📦 УПРАВЛЕНИЕ:"
	@echo "  make start          Запустить всё"
	@echo "  make stop           Остановить всё"
	@echo "  make restart        Перезапустить всё"
	@echo "  make status         Показать статус"
	@echo ""
	@echo "📋 ЛОГИ:"
	@echo "  make logs           Все логи (Ctrl+C для выхода)"
	@echo "  make logs-server    Только сервер"
	@echo "  make logs-userbot   Только пользовательский бот"
	@echo "  make logs-adminbot  Только административный бот"
	@echo ""
	@echo "💻 ЛОКАЛЬНАЯ РАЗРАБОТКА:"
	@echo "  make run-server     Запустить сервер локально"
	@echo "  make run-userbot    Запустить userbot локально"
	@echo "  make run-adminbot   Запустить adminbot локально"
	@echo ""
	@echo "🛠  УТИЛИТЫ:"
	@echo "  make enable         Автозапуск при загрузке"
	@echo "  make disable        Выключить автозапуск"
	@echo "  make clean          Удалить локальные бинарники"
	@echo ""
	@echo "═══════════════════════════════════════════════════════"

setup:
	@chmod +x setup-production.sh
	@./setup-production.sh

build:
	@echo "🔨 Сборка бинарников..."
	@cd server && go mod download
	@cd server && go build -o loyalityserver ./cmd/loyalityserver
	@cd server && go build -o userbot ./cmd/telegrambot/user
	@cd server && go build -o adminbot ./cmd/telegrambot/admin
	@echo "✅ Бинарники собраны в server/"

install:
	@if [ ! -d /opt/sirius ]; then \
		echo "❌ /opt/sirius не существует. Выполните: make setup"; \
		exit 1; \
	fi
	@echo "📦 Установка в /opt/sirius..."
	@sudo cp server/loyalityserver /opt/sirius/bin/
	@sudo cp server/userbot /opt/sirius/bin/
	@sudo cp server/adminbot /opt/sirius/bin/
	@sudo chmod 755 /opt/sirius/bin/*
	@sudo cp -r server/static/* /opt/sirius/static/
	@sudo cp -r server/migrations/* /opt/sirius/migrations/
	@sudo cp systemd/*.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@echo "✅ Установка завершена"

deploy:
	@echo "🚀 Деплой..."
	@$(MAKE) build
	@echo ""
	@sudo systemctl stop sirius-server.service sirius-userbot.service sirius-adminbot.service 2>/dev/null || true
	@$(MAKE) install
	@echo ""
	@sudo systemctl start sirius-server.service
	@sleep 2
	@sudo systemctl start sirius-userbot.service
	@sudo systemctl start sirius-adminbot.service
	@echo ""
	@echo "═══════════════════════════════════════════════════════"
	@echo "✅ Деплой завершен!"
	@echo "═══════════════════════════════════════════════════════"
	@sudo systemctl status sirius-server.service --no-pager -l || true
	@echo ""
	@echo "📋 Логи: make logs"

start:
	@echo "▶️  Запуск сервисов..."
	@sudo systemctl start sirius-server.service sirius-userbot.service sirius-adminbot.service
	@echo "✅ Сервисы запущены"

stop:
	@echo "⏸️  Остановка сервисов..."
	@sudo systemctl stop sirius-server.service sirius-userbot.service sirius-adminbot.service
	@echo "✅ Сервисы остановлены"

restart:
	@echo "🔄 Перезапуск сервисов..."
	@sudo systemctl restart sirius-server.service sirius-userbot.service sirius-adminbot.service
	@echo "✅ Сервисы перезапущены"

status:
	@sudo systemctl status sirius-server.service sirius-userbot.service sirius-adminbot.service --no-pager

logs:
	@echo "📋 Логи всех сервисов (Ctrl+C для выхода):"
	@sudo journalctl -u sirius-server.service -u sirius-userbot.service -u sirius-adminbot.service -f

logs-server:
	@sudo journalctl -u sirius-server.service -f

logs-userbot:
	@sudo journalctl -u sirius-userbot.service -f

logs-adminbot:
	@sudo journalctl -u sirius-adminbot.service -f

enable:
	@echo "⚙️  Включение автозапуска..."
	@sudo systemctl enable sirius-server.service sirius-userbot.service sirius-adminbot.service
	@echo "✅ Автозапуск включен"

disable:
	@echo "⚙️  Выключение автозапуска..."
	@sudo systemctl disable sirius-server.service sirius-userbot.service sirius-adminbot.service
	@echo "✅ Автозапуск выключен"

clean:
	@echo "🗑️  Удаление локальных бинарников..."
	@rm -f server/loyalityserver server/userbot server/adminbot
	@echo "✅ Очистка завершена"

run-server:
	@if [ ! -f local.yaml ]; then \
		echo "⚠ Файл local.yaml не найден"; \
		echo "Создайте local.yaml с настройками для локальной разработки"; \
		exit 1; \
	fi
	@echo "Запуск сервера локально (порт 8080)..."
	@mkdir -p logs
	@cd server && CONFIG_PATH=../local.yaml go run ./cmd/loyalityserver/main.go

run-userbot:
	@if [ ! -f local.yaml ]; then \
		echo "⚠ Файл local.yaml не найден"; \
		exit 1; \
	fi
	@if [ ! -f local-config/token.txt ]; then \
		echo "⚠ Файл local-config/token.txt не найден"; \
		echo "Создайте файл с токеном пользовательского бота"; \
		exit 1; \
	fi
	@echo "Запуск пользовательского бота локально..."
	@cd server && CONFIG_PATH=../local.yaml \
		TOKEN_PATH=../local-config/token.txt \
		API_TOKEN_PATH=../local-config/api_token.txt \
		SERVER_URL=http://localhost:8080 \
		go run ./cmd/telegrambot/user/main.go

run-adminbot:
	@if [ ! -f local.yaml ]; then \
		echo "⚠ Файл local.yaml не найден"; \
		exit 1; \
	fi
	@if [ ! -f local-config/admin_token.txt ]; then \
		echo "⚠ Файл local-config/admin_token.txt не найден"; \
		echo "Создайте файл с токеном административного бота"; \
		exit 1; \
	fi
	@if [ ! -f local-config/admins.json ]; then \
		echo "⚠ Файл local-config/admins.json не найден"; \
		echo "Создайте файл со списком администраторов"; \
		exit 1; \
	fi
	@echo "Запуск административного бота локально..."
	@cd server && CONFIG_PATH=../local.yaml \
		TOKEN_PATH=../local-config/admin_token.txt \
		API_TOKEN_PATH=../local-config/api_token.txt \
		ADMINS_PATH=../local-config/admins.json \
		SERVER_URL=http://localhost:8080 \
		go run ./cmd/telegrambot/admin/main.go
