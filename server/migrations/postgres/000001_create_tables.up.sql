-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    telegramm VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    middle_name VARCHAR(255),
    points INTEGER NOT NULL DEFAULT 0,
    "group" VARCHAR(255),
    registration_time TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);

-- Создание индекса для поиска по telegramm
CREATE INDEX IF NOT EXISTS idx_users_telegramm ON users(telegramm);
CREATE INDEX IF NOT EXISTS idx_users_group ON users("group");

-- Создание таблицы QR-кодов
CREATE TABLE IF NOT EXISTS codes (
    code UUID PRIMARY KEY,
    amount INTEGER NOT NULL,
    per_user INTEGER NOT NULL DEFAULT 1,
    total INTEGER NOT NULL DEFAULT 0,
    applied_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    "group" VARCHAR(255),
    error_code INTEGER NOT NULL DEFAULT 0
);

-- Создание индекса для поиска активных кодов
CREATE INDEX IF NOT EXISTS idx_codes_is_active ON codes(is_active);
CREATE INDEX IF NOT EXISTS idx_codes_group ON codes("group");

-- Создание таблицы использования кодов
CREATE TABLE IF NOT EXISTS code_usages (
    id UUID PRIMARY KEY,
    code UUID NOT NULL REFERENCES codes(code),
    user_id UUID NOT NULL REFERENCES users(id),
    count INTEGER NOT NULL DEFAULT 1,
    UNIQUE(code, user_id)
);

-- Создание индексов для поиска использования кодов
CREATE INDEX IF NOT EXISTS idx_code_usages_code ON code_usages(code);
CREATE INDEX IF NOT EXISTS idx_code_usages_user_id ON code_usages(user_id);

-- Создание таблицы транзакций
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    code UUID REFERENCES codes(code),
    diff INTEGER NOT NULL,
    "time" TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создание индексов для поиска транзакций
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_code ON transactions(code);
CREATE INDEX IF NOT EXISTS idx_transactions_time ON transactions("time");
