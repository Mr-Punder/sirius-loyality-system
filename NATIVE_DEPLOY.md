# Деплой

## Первый раз на удаленной машине

```bash
ssh USER@HOST

sudo mkdir -p /opt/sirius/{bin,config,logs,static,migrations,data}
sudo chmod -R 777 /opt/sirius

nano /opt/sirius/production.yaml
nano /opt/sirius/config/token.txt
nano /opt/sirius/config/admin_token.txt
nano /opt/sirius/config/api_token.txt
nano /opt/sirius/config/admins.json
```

Скопировать миграции (один раз):

```bash
scp -r server/migrations/* USER@HOST:/opt/sirius/migrations/
```

## Деплой с локальной машины

```bash
./deploy.sh USER@HOST
```

После первого деплоя установить пароль для веб-интерфейса:

```bash
# Локально собрать утилиту
cd server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o setadminpassword ./cmd/setadminpassword
cd ..

# Загрузить на сервер
scp server/setadminpassword USER@HOST:/opt/sirius/bin/

# Установить пароль
ssh USER@HOST 'CONFIG_PATH=/opt/sirius/production.yaml /opt/sirius/bin/setadminpassword -password=YOUR_PASSWORD'
```

## Обновление

```bash
git pull
./deploy.sh USER@HOST
```
