# Команды

## Деплой

```bash
./deploy.sh USER@HOST
```

## Локальная разработка

```bash
make run-server
make run-userbot
make run-adminbot
```

## Удаленное управление

```bash
ssh USER@HOST 'sudo systemctl restart sirius-server.service'
ssh USER@HOST 'sudo journalctl -u sirius-server.service -f'
```
