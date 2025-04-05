#!/bin/bash
set -e

cd ~/sirius-loyality-system

# Сохраняем текущую версию
CURRENT_COMMIT=$(git rev-parse HEAD)

# Получаем изменения из репозитория
git pull

# Проверяем, есть ли изменения
NEW_COMMIT=$(git rev-parse HEAD)
if [ "$CURRENT_COMMIT" == "$NEW_COMMIT" ]; then
    echo "No changes detected. Exiting."
    exit 0
fi

# Пересобираем Docker-образ
docker-compose build

# Перезапускаем контейнер
docker-compose down
docker-compose up -d

echo "Update completed successfully at $(date)"
