-- Создание таблицы администраторов
CREATE TABLE IF NOT EXISTS admins (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    username TEXT,
    is_active INTEGER DEFAULT 1
);

-- Создание индекса
CREATE INDEX IF NOT EXISTS idx_admins_username ON admins(username);
