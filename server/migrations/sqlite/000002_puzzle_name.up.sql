-- Добавляем колонку name в таблицу puzzles
ALTER TABLE puzzles ADD COLUMN name TEXT DEFAULT '';

-- Обновляем названия пазлов (по умолчанию "Пазл N")
UPDATE puzzles SET name = 'Пазл ' || id WHERE name = '' OR name IS NULL;
