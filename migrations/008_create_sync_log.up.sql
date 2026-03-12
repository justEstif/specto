CREATE TABLE sync_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plugin        TEXT NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ,
    items_added   INTEGER DEFAULT 0,
    items_skipped INTEGER DEFAULT 0,
    items_updated INTEGER DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'running',
    error_code    TEXT,
    error_message TEXT,
    duration_ms   INTEGER
);

CREATE INDEX idx_sync_log_user_plugin ON sync_log(user_id, plugin, started_at DESC);
