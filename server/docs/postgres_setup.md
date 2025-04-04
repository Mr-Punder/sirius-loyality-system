# Настройка PostgreSQL для системы лояльности

## Установка PostgreSQL

### macOS

```bash
# Установка через Homebrew
brew install postgresql@15

# Запуск PostgreSQL
brew services start postgresql@15

# Если команда psql не найдена, добавьте PostgreSQL в PATH
echo 'export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Если после этого команда `psql` все еще не найдена, попробуйте:

```bash
# Создание символической ссылки
ln -s /opt/homebrew/opt/postgresql@15/bin/psql /opt/homebrew/bin/psql

# Или используйте полный путь
/opt/homebrew/opt/postgresql@15/bin/psql
```

#### Первое подключение к PostgreSQL

При установке PostgreSQL через Homebrew на macOS, по умолчанию создается роль с именем текущего пользователя системы, а не "postgres". Однако, база данных с именем пользователя может не существовать. Для первого подключения используйте:

```bash
# Подключение к базе данных postgres (существует по умолчанию)
psql postgres

# Или явно указать имя вашего пользователя и базу данных postgres
psql -U $(whoami) postgres
```

#### Решение проблемы с ролью "postgres"

После подключения к PostgreSQL, есть два способа настроить работу:

1. Использовать существующую роль (имя вашего пользователя):

```bash
# Вы уже подключены к PostgreSQL, просто продолжайте работу
```

2. Создать роль "postgres":

```bash
# Подключение к PostgreSQL
psql postgres

# Создание роли postgres с правами суперпользователя
# ВАЖНО: не забудьте точку с запятой в конце команды!
CREATE ROLE postgres WITH SUPERUSER LOGIN PASSWORD 'postgres';

# Проверьте, что роль создана
\du

# Выход из psql
\q

# Перезапустите PostgreSQL для применения изменений
brew services restart postgresql@15

# Теперь можно подключиться как postgres
psql -U postgres
```

Если после этих шагов все еще возникает ошибка, попробуйте альтернативный способ:

```bash
# Создание роли postgres через createuser
createuser -s postgres

# Установка пароля для роли postgres
psql postgres -c "ALTER USER postgres WITH PASSWORD 'postgres';"

# Перезапустите PostgreSQL
brew services restart postgresql@15

# Теперь можно подключиться как postgres
psql -U postgres
```

После создания роли "postgres", обновите строку подключения в конфигурации:

```yaml
connection_string: "postgres://postgres:postgres@localhost:5432/loyality_system?sslmode=disable"
```

### Linux (Ubuntu/Debian)

```bash
# Установка PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Запуск PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### Windows

1. Скачайте установщик с [официального сайта](https://www.postgresql.org/download/windows/)
2. Запустите установщик и следуйте инструкциям
3. Запомните пароль для пользователя postgres, который вы зададите при установке

## Создание базы данных

Если вы используете роль с именем вашего пользователя:

```bash
# Подключение к PostgreSQL
psql

# Создание базы данных
CREATE DATABASE loyality_system;

# Выход из psql
\q
```

Если вы создали роль "postgres":

```bash
# Подключение к PostgreSQL
psql -U postgres

# Создание базы данных
CREATE DATABASE loyality_system;

# Выход из psql
\q
```

## Настройка сервера для работы с PostgreSQL

1. Отредактируйте файл `server/cmd/loyalityserver/config.yaml`, чтобы использовать PostgreSQL:

```yaml
storage:
  # Тип хранилища: "file", "postgres", "sqlite"
  type: "postgres"

  # Параметры для PostgreSQL
  # Используйте эту строку подключения, если вы создали роль postgres
  # connection_string: "postgres://postgres:postgres@localhost:5432/loyality_system?sslmode=disable"
  # Или эту строку подключения, если вы используете роль с именем вашего пользователя (без пароля)
  connection_string: "postgres:///loyality_system?sslmode=disable"
  migrations_path: "/Users/goomer125/Documents/sirius-rating-system/server/migrations/postgres"
```

По умолчанию используется строка подключения для роли с именем вашего пользователя (без пароля). Если вы создали роль postgres, раскомментируйте соответствующую строку и закомментируйте другую.

2. Запустите сервер:

```bash
cd server
go run cmd/loyalityserver/main.go
```

## Проверка работы

После запуска сервера, вы можете проверить, что таблицы были созданы в базе данных:

Если вы используете роль с именем вашего пользователя:

```bash
psql -d loyality_system -c "\dt"
```

Если вы создали роль "postgres":

```bash
psql -U postgres -d loyality_system -c "\dt"
```

Вы должны увидеть список таблиц: `users`, `transactions`, `codes`, `code_usages`.

## Миграции

### Как работают миграции

Миграции в нашей системе используют библиотеку `golang-migrate/migrate`. Они работают следующим образом:

1. **Миграции "up"** (файлы `.up.sql`) применяются автоматически при запуске сервера, если они еще не были применены. Они создают или изменяют структуру базы данных.

2. **Миграции "down"** (файлы `.down.sql`) используются только для ручного отката миграций, если это необходимо. Они **не выполняются автоматически при закрытии сервера**. Таблицы не удаляются при остановке сервера.

3. Библиотека миграций отслеживает, какие миграции уже были применены, в специальной таблице `schema_migrations` в базе данных. Это гарантирует, что каждая миграция применяется только один раз.

### Создание новых миграций

Если вы хотите добавить новую миграцию, создайте файлы в формате:

```
server/migrations/postgres/000002_название_миграции.up.sql
server/migrations/postgres/000002_название_миграции.down.sql
```

Где:

- `000002` - порядковый номер миграции (увеличивается на 1 для каждой новой миграции)
- `название_миграции` - краткое описание изменений
- `.up.sql` - SQL-запросы для применения миграции (создание/изменение таблиц)
- `.down.sql` - SQL-запросы для отката миграции (используются только при ручном откате)

### Ручное управление миграциями

Если вам нужно вручную управлять миграциями, вы можете использовать инструмент `migrate`:

```bash
# Установка инструмента migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Применение всех миграций
migrate -database "postgres:///loyality_system?sslmode=disable" -path server/migrations/postgres up

# Откат последней миграции
migrate -database "postgres:///loyality_system?sslmode=disable" -path server/migrations/postgres down 1

# Откат всех миграций (удаление всех таблиц)
migrate -database "postgres:///loyality_system?sslmode=disable" -path server/migrations/postgres down
```
