-- Таблица уведомлений для рассылки
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    message TEXT NOT NULL,
    group_filter VARCHAR(10) DEFAULT '',
    attachments TEXT[] DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP,
    sent_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
