-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    telegramm TEXT UNIQUE,
    first_name TEXT,
    last_name TEXT,
    middle_name TEXT,
    points INTEGER DEFAULT 0,
    "group" TEXT,
    registration_time TEXT,
    deleted INTEGER DEFAULT 0
);

-- Создание таблицы кодов
CREATE TABLE IF NOT EXISTS codes (
    code TEXT PRIMARY KEY,
    amount INTEGER,
    per_user INTEGER,
    total INTEGER,
    applied_count INTEGER DEFAULT 0,
    is_active INTEGER DEFAULT 1,
    "group" TEXT,
    error_code TEXT
);

-- Создание таблицы транзакций
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    code TEXT,
    diff INTEGER,
    time TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (code) REFERENCES codes(code)
);

-- Создание таблицы использований кодов
CREATE TABLE IF NOT EXISTS code_usages (
    id TEXT PRIMARY KEY,
    code TEXT,
    user_id TEXT,
    count INTEGER DEFAULT 0,
    FOREIGN KEY (code) REFERENCES codes(code),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(code, user_id)
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_users_telegramm ON users(telegramm);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_code_usages_code ON code_usages(code);
CREATE INDEX IF NOT EXISTS idx_code_usages_user_id ON code_usages(user_id);
