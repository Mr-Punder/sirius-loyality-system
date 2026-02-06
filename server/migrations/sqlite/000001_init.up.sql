-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    telegramm TEXT UNIQUE,
    first_name TEXT,
    last_name TEXT,
    middle_name TEXT,
    "group" TEXT,
    registration_time TEXT,
    deleted INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_telegramm ON users(telegramm);

-- Таблица администраторов
CREATE TABLE IF NOT EXISTS admins (
    id INTEGER PRIMARY KEY,
    name TEXT,
    username TEXT,
    is_active INTEGER DEFAULT 1
);

-- Таблица пазлов
CREATE TABLE IF NOT EXISTS puzzles (
    id INTEGER PRIMARY KEY,
    is_completed INTEGER DEFAULT 0,
    completed_at TEXT
);

-- Таблица деталей пазлов
CREATE TABLE IF NOT EXISTS puzzle_pieces (
    code TEXT PRIMARY KEY,
    puzzle_id INTEGER NOT NULL,
    piece_number INTEGER NOT NULL,
    owner_id TEXT,
    registered_at TEXT,
    FOREIGN KEY (puzzle_id) REFERENCES puzzles(id),
    FOREIGN KEY (owner_id) REFERENCES users(id),
    UNIQUE(puzzle_id, piece_number)
);

CREATE INDEX IF NOT EXISTS idx_puzzle_pieces_puzzle_id ON puzzle_pieces(puzzle_id);
CREATE INDEX IF NOT EXISTS idx_puzzle_pieces_owner_id ON puzzle_pieces(owner_id);

-- Инициализируем 30 пазлов
INSERT OR IGNORE INTO puzzles (id, is_completed, completed_at) VALUES
(1, 0, NULL), (2, 0, NULL), (3, 0, NULL), (4, 0, NULL), (5, 0, NULL),
(6, 0, NULL), (7, 0, NULL), (8, 0, NULL), (9, 0, NULL), (10, 0, NULL),
(11, 0, NULL), (12, 0, NULL), (13, 0, NULL), (14, 0, NULL), (15, 0, NULL),
(16, 0, NULL), (17, 0, NULL), (18, 0, NULL), (19, 0, NULL), (20, 0, NULL),
(21, 0, NULL), (22, 0, NULL), (23, 0, NULL), (24, 0, NULL), (25, 0, NULL),
(26, 0, NULL), (27, 0, NULL), (28, 0, NULL), (29, 0, NULL), (30, 0, NULL);
