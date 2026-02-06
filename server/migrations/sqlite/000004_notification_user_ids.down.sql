-- SQLite не поддерживает DROP COLUMN напрямую, создаем новую таблицу
CREATE TABLE notifications_new (
    id TEXT PRIMARY KEY,
    message TEXT NOT NULL,
    group_filter TEXT DEFAULT '',
    attachments TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TEXT NOT NULL,
    sent_at TEXT,
    sent_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0
);

INSERT INTO notifications_new SELECT id, message, group_filter, attachments, status, created_at, sent_at, sent_count, error_count FROM notifications;

DROP TABLE notifications;

ALTER TABLE notifications_new RENAME TO notifications;

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
