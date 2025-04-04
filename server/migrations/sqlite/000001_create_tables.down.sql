-- Удаление индексов
DROP INDEX IF EXISTS idx_users_telegramm;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP INDEX IF EXISTS idx_code_usages_code;
DROP INDEX IF EXISTS idx_code_usages_user_id;

-- Удаление таблиц
DROP TABLE IF EXISTS code_usages;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS codes;
DROP TABLE IF EXISTS users;
