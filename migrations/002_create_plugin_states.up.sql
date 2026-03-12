CREATE TABLE plugin_states (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plugin         TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'disconnected',
    enabled        BOOLEAN NOT NULL DEFAULT true,
    cursor         TEXT,
    last_synced_at TIMESTAMPTZ,
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, plugin)
);

CREATE INDEX idx_plugin_states_user ON plugin_states(user_id);
