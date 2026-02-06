# Архитектура системы пазлов (Sirius Loyalty System)

## Обзор

Система представляет собой платформу для проведения игры-розыгрыша с пазлами. Участники регистрируются через Telegram-бота, сканируют QR-коды для получения деталей пазлов. Когда пазл собран (все 6 деталей зарегистрированы), администратор может его засчитать и участники получают право на розыгрыш призов.

---

## Компоненты системы

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            КЛИЕНТЫ                                       │
├─────────────────┬─────────────────┬─────────────────────────────────────┤
│   Telegram      │   Telegram      │         Веб-браузер                 │
│   User Bot      │   Admin Bot     │         (Админ-панель)              │
└────────┬────────┴────────┬────────┴──────────────┬──────────────────────┘
         │                 │                       │
         │ HTTP/REST       │ HTTP/REST             │ HTTP
         ▼                 ▼                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         HTTP СЕРВЕР (Chi Router)                         │
│  :8080                                                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  Middleware: Token Auth │ JWT Auth │ GZIP │ Logging                     │
├─────────────────────────────────────────────────────────────────────────┤
│                              HANDLERS                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │   /users     │  │  /puzzles    │  │ /notifications│                  │
│  │   /pieces    │  │  /admins     │  │ /stats       │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
│                                                                          │
│  ┌──────────────────────────────────────────────────┐                   │
│  │            /api/admin/* (JWT-protected)          │                   │
│  │  users, puzzles, pieces, admins, notifications   │                   │
│  └──────────────────────────────────────────────────┘                   │
│                                                                          │
│  ┌──────────────────────────────────────────────────┐                   │
│  │         /admin/* (Static Files)                  │                   │
│  │  login.html, users.html, puzzles.html, etc.      │                   │
│  └──────────────────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           STORAGE LAYER                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │   SQLite     │  │  PostgreSQL  │  │  File-based  │                   │
│  │  (local dev) │  │ (production) │  │   (legacy)   │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          FILE STORAGE                                    │
│  data/                                                                   │
│  ├── attachments/{notification_id}/   # Вложения рассылок               │
│  └── temp/                            # Временные файлы                 │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Структура проекта

```
sirius-loyality-system/
├── server/
│   ├── cmd/
│   │   ├── loyalityserver/      # Основной HTTP-сервер
│   │   │   └── main.go
│   │   ├── telegrambot/
│   │   │   ├── admin/           # Telegram-бот для администраторов
│   │   │   │   └── main.go
│   │   │   └── user/            # Telegram-бот для пользователей
│   │   │       └── main.go
│   │   └── setadminpassword/    # Утилита для установки пароля админа
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── admin/               # Админ-панель (JWT auth, handlers)
│   │   │   ├── auth.go
│   │   │   ├── handlers.go
│   │   │   ├── middleware.go
│   │   │   └── password.go
│   │   │
│   │   ├── config/              # Загрузка конфигурации YAML
│   │   │   └── config.go
│   │   │
│   │   ├── handlers/            # REST API handlers
│   │   │   ├── handlers.go
│   │   │   └── handlers_test.go
│   │   │
│   │   ├── logger/              # Логирование (Zap)
│   │   │   ├── logger.go
│   │   │   └── zaplogger.go
│   │   │
│   │   ├── middleware/          # HTTP middleware
│   │   │   ├── gzipcompressor.go
│   │   │   ├── httplog.go
│   │   │   └── token_auth.go
│   │   │
│   │   ├── models/              # Модели данных
│   │   │   └── models.go
│   │   │
│   │   ├── server/              # HTTP сервер
│   │   │   └── server.go
│   │   │
│   │   ├── storage/             # Слой хранения данных
│   │   │   ├── storage.go       # Интерфейс Storage
│   │   │   ├── pgstorage.go     # PostgreSQL реализация
│   │   │   ├── sqlitestorage.go # SQLite реализация
│   │   │   ├── filestorage.go   # File-based реализация
│   │   │   └── memstorage.go    # In-memory (для тестов)
│   │   │
│   │   └── telegrambot/         # Telegram боты
│   │       ├── adminbot.go      # Админ-бот
│   │       ├── userbot.go       # Юзер-бот
│   │       ├── api_client.go    # HTTP клиент для API
│   │       ├── qrcode.go        # Генерация QR-кодов
│   │       ├── utils.go         # Утилиты (валидация групп)
│   │       └── telegrambot.go   # Общие типы
│   │
│   ├── migrations/
│   │   ├── postgres/            # Миграции PostgreSQL
│   │   └── sqlite/              # Миграции SQLite
│   │
│   └── static/admin/            # Статика админ-панели
│       ├── css/styles.css
│       ├── login.html
│       ├── users.html
│       ├── puzzles.html
│       ├── pieces.html
│       ├── admins.html
│       └── broadcasts.html
│
├── production.yaml              # Конфигурация для production
├── local.yaml                   # Конфигурация для локальной разработки
└── ARCHITECTURE.md              # Этот файл
```

---

## Модели данных

### User (Пользователь)
```go
type User struct {
    Id               uuid.UUID  // Уникальный ID
    Telegramm        string     // Telegram username
    FirstName        string     // Имя
    LastName         string     // Фамилия
    MiddleName       string     // Отчество
    Group            string     // Группа (Н1-Н6)
    RegistrationTime time.Time  // Дата регистрации
    Deleted          bool       // Флаг удаления
}
```

### Puzzle (Пазл)
```go
type Puzzle struct {
    Id          int        // 1-30 (номер пазла)
    Name        string     // Название пазла
    IsCompleted bool       // Засчитан ли администратором
    CompletedAt *time.Time // Когда был засчитан
}
```

### PuzzlePiece (Деталь пазла)
```go
type PuzzlePiece struct {
    Code         string     // Уникальный 7-символьный код (QR)
    PuzzleId     int        // К какому пазлу (1-30)
    PieceNumber  int        // Номер детали (1-6)
    OwnerId      *uuid.UUID // Кто зарегистрировал
    RegisteredAt *time.Time // Когда зарегистрирована
}
```

### Notification (Уведомление/Рассылка)
```go
type Notification struct {
    Id          uuid.UUID          // Уникальный ID
    Message     string             // Текст сообщения
    Group       string             // Группа (пустая = всем или по UserIds)
    UserIds     []uuid.UUID        // Конкретные пользователи
    Attachments []string           // Имена файлов вложений
    Status      NotificationStatus // pending/sent/failed
    CreatedAt   time.Time
    SentAt      *time.Time
    SentCount   int                // Успешно отправлено
    ErrorCount  int                // Ошибок при отправке
}
```

### Admin (Администратор)
```go
type Admin struct {
    ID       int64  // Telegram User ID
    Name     string // Имя (для заметок)
    Username string // Telegram username
    IsActive bool   // Активен ли
}
```

---

## API Endpoints

### Публичные (Token Auth)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/users` | Регистрация пользователя |
| GET | `/users` | Список пользователей |
| GET | `/users/{id}` | Информация о пользователе |
| DELETE | `/users/{id}` | Удаление пользователя |
| GET | `/users/{id}/pieces` | Детали пользователя |
| GET | `/puzzles` | Список пазлов |
| GET | `/puzzles/{id}` | Информация о пазле |
| POST | `/puzzles/{id}/complete` | Засчитать пазл |
| GET | `/pieces` | Все детали |
| POST | `/pieces` | Импорт деталей |
| POST | `/pieces/register` | Регистрация детали |
| GET | `/pieces/{code}` | Информация о детали |
| GET | `/admins` | Список админов |
| POST | `/admins` | Добавить админа |
| GET | `/admins/check/{id}` | Проверить админа |
| DELETE | `/admins/{id}` | Удалить админа |
| POST | `/notifications` | Создать рассылку |
| GET | `/notifications/pending` | Ожидающие рассылки |
| POST | `/notifications/{id}/attachments` | Загрузить вложение |
| GET | `/notifications/{id}/attachments/{filename}` | Скачать вложение |
| PATCH | `/notifications/{id}` | Обновить рассылку |
| GET | `/stats/lottery` | Статистика розыгрыша |

### Админ-панель (JWT Auth)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/admin/login` | Авторизация |
| GET | `/api/admin/users` | Список пользователей |
| GET | `/api/admin/puzzles` | Список пазлов |
| GET | `/api/admin/pieces` | Список деталей |
| GET | `/api/admin/admins` | Список админов |
| POST | `/api/admin/admins/add` | Добавить админа |
| POST | `/api/admin/admins/remove` | Удалить админа |
| GET | `/api/admin/notifications` | Список рассылок |

---

## Telegram боты

### User Bot (Пользовательский)

**Функции:**
- Регистрация пользователей (имя, фамилия, группа)
- Сканирование QR-кодов деталей
- Просмотр своих деталей и прогресса
- Получение уведомлений о рассылках

**Поллинг уведомлений:**
- Каждые 5 секунд проверяет `/notifications/pending`
- Отправляет сообщения + вложения пользователям
- Обновляет статус уведомления

**Команды:**
- `/start` - Регистрация
- `/mypieces` - Мои детали
- `/info` - Информация о профиле
- Текст: код детали → регистрация

### Admin Bot (Административный)

**Функции:**
- Просмотр пользователей (все/по группам)
- Просмотр пазлов и деталей
- Засчитывание пазлов
- Управление админами
- Создание рассылок с вложениями
- Статистика розыгрыша

**Команды:**
- `/start` - Главное меню
- `/users [группа]` - Список пользователей
- `/user <id>` - Инфо о пользователе
- `/puzzles` - Список пазлов
- `/pieces` - Статистика деталей
- `/complete <номер>` - Засчитать пазл
- `/lottery` - Статистика розыгрыша
- `/addadmin <id>` - Добавить админа
- `/listadmins` - Список админов
- `/help` - Справка

**Кнопки меню:**
```
[Пользователи] [Пазлы]
[Администраторы] [Розыгрыш]
[Рассылка] [Помощь]
```

---

## Поток данных: Рассылка с вложениями

```
┌──────────────────┐
│   Admin Bot      │
│   или            │
│   Web Interface  │
└────────┬─────────┘
         │ 1. POST /notifications {message, group}
         ▼
┌──────────────────┐
│   HTTP Server    │
│   (Handler)      │
└────────┬─────────┘
         │ 2. Создать notification в БД
         ▼
┌──────────────────┐
│   Database       │
│   (notifications)│
└────────┬─────────┘
         │ 3. POST /notifications/{id}/attachments
         ▼
┌──────────────────┐
│   File System    │
│   data/          │
│   attachments/   │
│   {id}/file.jpg  │
└────────┬─────────┘
         │
         │ 4. Каждые 5 сек
         ▼
┌──────────────────┐
│   User Bot       │
│   (Poller)       │
│   GET /pending   │
└────────┬─────────┘
         │ 5. Для каждого пользователя:
         │    - Отправить текст
         │    - Отправить файлы из data/attachments/{id}/
         ▼
┌──────────────────┐
│   Telegram API   │
│   → Пользователи │
└──────────────────┘
```

---

## Поток данных: Регистрация детали

```
┌──────────────────┐
│   User Bot       │
│   (Telegram)     │
└────────┬─────────┘
         │ 1. Пользователь отправляет код: "PY1GG7H"
         ▼
┌──────────────────┐
│   User Bot       │
│   (Handler)      │
└────────┬─────────┘
         │ 2. POST /pieces/register {code, owner_id}
         ▼
┌──────────────────┐
│   HTTP Server    │
│   (Handler)      │
└────────┬─────────┘
         │ 3. storage.RegisterPuzzlePiece(code, ownerId)
         ▼
┌──────────────────┐
│   Database       │
│   (puzzle_pieces)│
│   UPDATE owner_id│
└────────┬─────────┘
         │ 4. Результат: {piece, all_collected}
         ▼
┌──────────────────┐
│   User Bot       │
│   → Пользователь │
│   "Деталь #3     │
│   пазла №5       │
│   зарегистр."    │
└──────────────────┘
```

---

## Конфигурация

### production.yaml
```yaml
server:
  runaddress: "0.0.0.0:8080"

logger:
  level: "info"
  path: "/opt/sirius/logs/server.log"
  errorpath: "/opt/sirius/logs/server_error.log"
  maxsize: 10      # МБ
  maxbackups: 5
  maxage: 30       # дней
  compress: true

storage:
  type: "postgres"
  connection_string: "postgres://user:pass@host:5432/db?sslmode=require"
  migrations_path: "/opt/sirius/migrations/postgres"
  datapath: "/opt/sirius/data"

admin:
  jwt_secret: "your-secret-key"
  static_dir: "/opt/sirius/static/admin"
  admins_path: "/opt/sirius/config/admins.json"

api:
  token: "your-api-token"
```

### Переменные окружения
| Переменная | Описание |
|------------|----------|
| `CONFIG_PATH` | Путь к YAML конфигу |
| `MIGRATIONS_PATH` | Путь к миграциям |
| `DB_PATH` | Путь к SQLite БД |
| `ADMIN_STATIC_DIR` | Путь к статике админки |
| `ADMIN_ADMINS_PATH` | Путь к admins.json |
| `POSTGRES_CONNECTION_STRING` | Строка подключения к PostgreSQL |

---

## База данных

### Таблицы

**users**
```sql
id UUID PRIMARY KEY,
telegramm TEXT UNIQUE NOT NULL,
first_name TEXT,
last_name TEXT,
middle_name TEXT,
group_name TEXT,
registration_time TIMESTAMP,
deleted BOOLEAN DEFAULT FALSE
```

**puzzles**
```sql
id INTEGER PRIMARY KEY,  -- 1-30
name TEXT,
is_completed BOOLEAN DEFAULT FALSE,
completed_at TIMESTAMP
```

**puzzle_pieces**
```sql
code TEXT PRIMARY KEY,   -- 7 символов
puzzle_id INTEGER REFERENCES puzzles(id),
piece_number INTEGER,    -- 1-6
owner_id UUID REFERENCES users(id),
registered_at TIMESTAMP
```

**admins**
```sql
id BIGINT PRIMARY KEY,   -- Telegram user ID
name TEXT,
username TEXT,
is_active BOOLEAN DEFAULT TRUE
```

**notifications**
```sql
id UUID PRIMARY KEY,
message TEXT NOT NULL,
group_name TEXT,
user_ids TEXT,           -- JSON array
attachments TEXT,        -- JSON array
status TEXT DEFAULT 'pending',
created_at TIMESTAMP,
sent_at TIMESTAMP,
sent_count INTEGER DEFAULT 0,
error_count INTEGER DEFAULT 0
```

---

## Безопасность

### Аутентификация

1. **API Token** - для ботов и внешних запросов
   - Header: `Authorization: Bearer <token>`
   - Проверяется middleware `TokenAuth`

2. **JWT** - для веб-админки
   - Генерируется при логине с паролем
   - Хранится в localStorage
   - Expires через 24 часа

### Пароль админки

Хранится в файле `data/admin_password.hash` (bcrypt)

Установка:
```bash
./setadminpassword -password "your-password"
```

---

## Запуск

### Development (SQLite)

```bash
# Сервер
cd server && go run cmd/loyalityserver/main.go -config local.yaml

# Admin Bot
cd server && go run cmd/telegrambot/admin/main.go \
  -token "BOT_TOKEN" \
  -server "http://localhost:8080" \
  -api-token "api-token" \
  -admin-id 123456789

# User Bot
cd server && go run cmd/telegrambot/user/main.go \
  -token "BOT_TOKEN" \
  -server "http://localhost:8080" \
  -api-token "api-token"
```

### Production (PostgreSQL + Docker)

```bash
# Сервер
./loyalityserver -config /opt/sirius/production.yaml

# Боты (отдельными процессами)
./adminbot -token "..." -server "http://server:8080" ...
./userbot -token "..." -server "http://server:8080" ...
```

---

## Миграции

```bash
# Применить миграции PostgreSQL
migrate -path ./migrations/postgres -database "postgres://..." up

# Откатить
migrate -path ./migrations/postgres -database "postgres://..." down 1
```

---

## Тестирование

```bash
cd server
go test ./...
```

---

## Мониторинг

- Логи в файлах с ротацией (Zap + Lumberjack)
- HTTP access logs через middleware
- Ошибки в отдельный файл `server_error.log`
