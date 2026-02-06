-- Добавляем колонку name в таблицу puzzles
ALTER TABLE puzzles ADD COLUMN IF NOT EXISTS name VARCHAR(255) DEFAULT '';

-- Обновляем названия пазлов (по умолчанию "Пазл N")
UPDATE puzzles SET name = 'Пазл ' || id WHERE name = '' OR name IS NULL;
