-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    telegramm VARCHAR(255) UNIQUE,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    middle_name VARCHAR(255),
    "group" VARCHAR(50),
    registration_time TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_users_telegramm ON users(telegramm);

-- Таблица администраторов
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255),
    username VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE
);

-- Таблица пазлов
CREATE TABLE IF NOT EXISTS puzzles (
    id INTEGER PRIMARY KEY,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP
);

-- Таблица деталей пазлов
CREATE TABLE IF NOT EXISTS puzzle_pieces (
    code VARCHAR(7) PRIMARY KEY,
    puzzle_id INTEGER NOT NULL REFERENCES puzzles(id),
    piece_number INTEGER NOT NULL CHECK (piece_number BETWEEN 1 AND 6),
    owner_id UUID REFERENCES users(id),
    registered_at TIMESTAMP,
    UNIQUE(puzzle_id, piece_number)
);

CREATE INDEX IF NOT EXISTS idx_puzzle_pieces_puzzle_id ON puzzle_pieces(puzzle_id);
CREATE INDEX IF NOT EXISTS idx_puzzle_pieces_owner_id ON puzzle_pieces(owner_id);

-- Инициализируем 30 пазлов
INSERT INTO puzzles (id, is_completed, completed_at)
SELECT generate_series(1, 30), FALSE, NULL
ON CONFLICT (id) DO NOTHING;
