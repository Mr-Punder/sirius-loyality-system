-- Добавляем колонку user_ids для указания конкретных получателей
ALTER TABLE notifications ADD COLUMN user_ids TEXT DEFAULT '';
