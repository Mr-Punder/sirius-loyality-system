CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY,
    filename TEXT NOT NULL,
    store_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attachments_created_at ON attachments(created_at DESC);
