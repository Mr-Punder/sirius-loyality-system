-- Создание таблицы администраторов
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    username TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

-- Создание индекса
CREATE INDEX IF NOT EXISTS idx_admins_username ON admins(username);
