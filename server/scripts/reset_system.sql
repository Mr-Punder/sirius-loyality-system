-- Скрипт сброса системы к начальному состоянию
-- Сохраняет: пазлы, детали (коды), уведомления, администраторов
-- Сбрасывает: пользователей, статусы деталей, статусы пазлов

-- ВНИМАНИЕ: Этот скрипт удалит всех пользователей!

BEGIN;

-- 1. Сбрасываем статусы деталей (убираем владельцев)
UPDATE puzzle_pieces
SET owner_id = NULL,
    registered_at = NULL;

-- 2. Сбрасываем статусы пазлов
UPDATE puzzles
SET is_completed = FALSE,
    completed_at = NULL;

-- 3. Удаляем всех пользователей
DELETE FROM users;

COMMIT;

-- Проверка результата
SELECT 'Пользователей:' as label, COUNT(*) as count FROM users
UNION ALL
SELECT 'Деталей с владельцем:', COUNT(*) FROM puzzle_pieces WHERE owner_id IS NOT NULL
UNION ALL
SELECT 'Засчитанных пазлов:', COUNT(*) FROM puzzles WHERE is_completed = TRUE;
