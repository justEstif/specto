CREATE TABLE plugin_credentials (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plugin         TEXT NOT NULL,
    auth_type      TEXT NOT NULL,
    encrypted_data BYTEA NOT NULL,
    expires_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, plugin)
);

CREATE INDEX idx_plugin_credentials_user ON plugin_credentials(user_id);
