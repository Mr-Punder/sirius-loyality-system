-- SQLite не поддерживает DROP COLUMN напрямую
-- Для отката нужно пересоздать таблицу без колонки name
-- Это деструктивная операция, используйте с осторожностью

CREATE TABLE puzzles_backup AS SELECT id, is_completed, completed_at FROM puzzles;
DROP TABLE puzzles;
CREATE TABLE puzzles (
    id INTEGER PRIMARY KEY,
    is_completed INTEGER DEFAULT 0,
    completed_at TEXT
);
INSERT INTO puzzles SELECT * FROM puzzles_backup;
DROP TABLE puzzles_backup;
