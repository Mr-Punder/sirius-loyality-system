# Инструкция по развертыванию системы лояльности на удаленном сервере

Эта инструкция описывает процесс развертывания системы лояльности на удаленном сервере с использованием Docker.

## Предварительные требования

- Удаленный сервер с Linux (протестировано на Ubuntu)
- Доступ к серверу по SSH
- Установленный Docker и Docker Compose на сервере

## Файлы для развертывания

В репозитории подготовлены следующие файлы для развертывания:

- `Dockerfile` - файл для сборки Docker-образа
- `docker-compose.yml` - файл для оркестрации контейнеров
- `supervisord.conf` - файл для управления процессами внутри контейнера
- `config/config.yaml` - конфигурационный файл для сервера
- `update-sirius.sh` - скрипт для автоматического обновления системы

## Шаги по развертыванию

### 1. Установка Docker и Docker Compose (если не установлены)

```bash
# Обновляем пакеты
sudo apt-get update

# Устанавливаем необходимые пакеты
sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common

# Устанавливаем Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Добавляем текущего пользователя в группу docker
sudo usermod -aG docker $USER

# Устанавливаем Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.18.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Выходим и снова входим для применения изменений группы
exit
```

### 2. Клонирование репозитория

```bash
# Создаем директорию для проекта
mkdir -p ~/sirius-loyality-system

# Клонируем репозиторий
git clone <URL_РЕПОЗИТОРИЯ> ~/sirius-loyality-system
cd ~/sirius-loyality-system
```

### 3. Настройка конфигурационных файлов

```bash
# Создаем директории для данных, логов и конфигурации
mkdir -p ~/sirius-loyality-system/data ~/sirius-loyality-system/logs ~/sirius-loyality-system/config

# Создаем директории для токенов
mkdir -p ~/sirius-loyality-system/server/cmd/telegrambot/user ~/sirius-loyality-system/server/cmd/telegrambot/admin

# Создаем файл с токеном для пользовательского бота
echo "YOUR_USER_BOT_TOKEN" > ~/sirius-loyality-system/config/token.txt

# Создаем файл с токеном для административного бота
echo "YOUR_ADMIN_BOT_TOKEN" > ~/sirius-loyality-system/config/admin_token.txt


# Создаем файл со списком администраторов
echo '{"admins":[{"id":ADMIN_ID_1,"name":"Admin Name"}]}' > ~/sirius-loyality-system/config/admins.json


# Создаем файл с API-токеном для ботов
echo "sirius-rating-system-api-token-2025" > ~/sirius-loyality-system/server/cmd/telegrambot/api_token.txt
```

Не забудьте заменить:

- `YOUR_USER_BOT_TOKEN` - токен пользовательского бота, полученный от @BotFather
- `YOUR_ADMIN_BOT_TOKEN` - токен административного бота, полученный от @BotFather
- `ADMIN_ID_1` - ID администратора в Telegram (числовое значение, без кавычек)
- `Admin Name` - имя администратора (опционально)

### 3.1. Настройка путей и доступа извне

Для корректной работы приложения в Docker и доступа извне, необходимо изменить конфигурационные файлы:

1. Отредактируйте файл `docker-compose.yml`, чтобы указать внешний порт:

```bash
nano ~/sirius-loyality-system/docker-compose.yml
```

Убедитесь, что в секции `ports` указан маппинг портов, например:

```yaml
ports:
  - "80:80"  # Внешний порт : Внутренний порт
```

2. Отредактируйте файл `supervisord.conf`, чтобы указать правильный адрес сервера для ботов:

```bash
nano ~/sirius-loyality-system/supervisord.conf
```

Найдите строки с `environment=` для `userbot` и `adminbot` и измените `SERVER_URL` на внешний IP-адрес или домен вашего сервера:

```
environment=TOKEN_PATH="/app/cmd/telegrambot/user/token.txt",SERVER_URL="http://YOUR_SERVER_IP_OR_DOMAIN",API_TOKEN_PATH="/app/cmd/telegrambot/api_token.txt"
```

```
environment=TOKEN_PATH="/app/cmd/telegrambot/admin/token.txt",ADMINS_PATH="/app/cmd/telegrambot/admin/admins.json",SERVER_URL="http://YOUR_SERVER_IP_OR_DOMAIN",API_TOKEN_PATH="/app/cmd/telegrambot/api_token.txt"
```

Замените `YOUR_SERVER_IP_OR_DOMAIN` на внешний IP-адрес или домен вашего сервера.

3. Приложение использует следующие переменные окружения для путей к файлам:

- `CONFIG_PATH` - путь к конфигурационному файлу (по умолчанию: `/app/config/config.yaml`)
- `MIGRATIONS_PATH` - путь к миграциям SQLite (по умолчанию: `/app/migrations/sqlite`)
- `DB_PATH` - путь к файлу базы данных SQLite (по умолчанию: `/app/data/loyality_system.db`)
- `ADMIN_STATIC_DIR` - путь к директории со статическими файлами админки (по умолчанию: `/app/static/admin`)
- `ADMIN_ADMINS_PATH` - путь к файлу со списком администраторов (по умолчанию: `/app/cmd/telegrambot/admin/admins.json`)
- `TOKEN_PATH` - путь к файлу с токеном пользовательского бота (по умолчанию: `/app/cmd/telegrambot/user/token.txt`)
- `API_TOKEN_PATH` - путь к файлу с API-токеном (по умолчанию: `/app/cmd/telegrambot/api_token.txt`)

Эти переменные уже настроены в Dockerfile и не требуют изменения, если вы не меняете структуру директорий в контейнере.

### 4. Сборка и запуск Docker-контейнера

```bash
# Делаем скрипт обновления исполняемым
chmod +x ~/sirius-loyality-system/update-sirius.sh
```

```bash
# Собираем и запускаем контейнер
cd ~/sirius-loyality-system
docker-compose up -d
```

### 5. Проверка работоспособности

```bash
# Проверяем, что контейнер запущен
docker ps

# Проверяем логи
docker-compose logs

# Проверяем логи отдельных компонентов
docker-compose exec sirius-system cat /app/logs/server.log
docker-compose exec sirius-system cat /app/logs/userbot.log
docker-compose exec sirius-system cat /app/logs/adminbot.log
```

Теперь ваша система должна быть доступна по адресу `http://your-server-ip/`.

## Управление системой

### Основные команды для управления Docker-контейнером

```bash
# Остановка контейнера
docker-compose stop

# Запуск контейнера
docker-compose start

# Перезапуск контейнера
docker-compose restart

# Просмотр логов
docker-compose logs -f

# Остановка и удаление контейнера
docker-compose down
```

### Обновление системы

Для обновления системы можно использовать подготовленный скрипт:

```bash
~/sirius-loyality-system/update-sirius.sh
```

Или выполнить обновление вручную:

```bash
cd ~/sirius-loyality-system
git pull
docker-compose build
docker-compose down
docker-compose up -d
```

### Настройка автоматического обновления

Для автоматического обновления системы можно добавить задание в cron:

```bash
# Добавляем задание в cron для запуска каждый час
(crontab -l 2>/dev/null; echo "0 * * * * ~/sirius-loyality-system/update-sirius.sh >> ~/update-sirius.log 2>&1") | crontab -
```

### Резервное копирование данных

```bash
# Создаем резервную копию базы данных
docker-compose exec sirius-system sqlite3 /app/data/loyality_system.db .dump > ~/backup_$(date +%Y%m%d).sql

# Архивируем резервную копию
tar -czf ~/backup_$(date +%Y%m%d).tar.gz ~/backup_$(date +%Y%m%d).sql

# Удаляем SQL-файл
rm ~/backup_$(date +%Y%m%d).sql
```

### Восстановление из резервной копии

```bash
# Распаковываем архив
tar -xzf ~/backup_YYYYMMDD.tar.gz

# Останавливаем контейнер
docker-compose stop

# Восстанавливаем базу данных
cat ~/backup_YYYYMMDD.sql | docker-compose exec -T sirius-system sqlite3 /app/data/loyality_system.db

# Запускаем контейнер
docker-compose start
```

## Настройка HTTPS (опционально)

Для настройки HTTPS можно использовать Nginx в качестве обратного прокси:

```bash
# Устанавливаем Nginx
sudo apt-get install -y nginx

# Создаем самоподписанный сертификат
sudo mkdir -p /etc/nginx/ssl
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /etc/nginx/ssl/nginx.key -out /etc/nginx/ssl/nginx.crt

# Настраиваем Nginx как обратный прокси
sudo bash -c 'cat > /etc/nginx/sites-available/sirius << EOF
server {
    listen 80;
    server_name _;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name _;

    ssl_certificate /etc/nginx/ssl/nginx.crt;
    ssl_certificate_key /etc/nginx/ssl/nginx.key;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF'

# Активируем конфигурацию
sudo ln -s /etc/nginx/sites-available/sirius /etc/nginx/sites-enabled/
sudo rm /etc/nginx/sites-enabled/default

# Проверяем конфигурацию Nginx
sudo nginx -t

# Перезапускаем Nginx
sudo systemctl restart nginx

# Обновляем конфигурацию Docker-контейнера
docker-compose down
# Изменяем порт в docker-compose.yml с 80:80 на 8080:80
docker-compose up -d
```

## Устранение неполадок

### Проблема: Контейнер не запускается

Проверьте логи контейнера:

```bash
docker-compose logs
```

### Проблема: Боты не подключаются к Telegram API

Проверьте, что токены ботов указаны правильно:

```bash
cat ~/sirius-loyality-system/server/cmd/telegrambot/user/token.txt
cat ~/sirius-loyality-system/server/cmd/telegrambot/admin/token.txt
```

### Проблема: Не работает автоматическое обновление

Проверьте логи обновления:

```bash
cat ~/update-sirius.log
```

Убедитесь, что скрипт обновления имеет права на выполнение:

```bash
chmod +x ~/sirius-loyality-system/update-sirius.sh
```
