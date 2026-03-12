CREATE TABLE media_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform          TEXT NOT NULL,
    type              TEXT NOT NULL,
    title             TEXT NOT NULL,
    creator           TEXT,
    consumed_at       TIMESTAMPTZ NOT NULL,
    duration          INTERVAL,
    time_spent        INTERVAL,
    url               TEXT,
    external_id       TEXT NOT NULL,
    enrichment_status TEXT NOT NULL DEFAULT 'pending',
    raw_metadata      JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, platform, external_id)
);

CREATE INDEX idx_media_items_user_consumed ON media_items(user_id, consumed_at DESC);
CREATE INDEX idx_media_items_user_platform ON media_items(user_id, platform);
CREATE INDEX idx_media_items_user_type ON media_items(user_id, type);
CREATE INDEX idx_media_items_enrichment ON media_items(enrichment_status) WHERE enrichment_status = 'pending';
CREATE INDEX idx_media_items_search ON media_items USING gin(
    to_tsvector('english', coalesce(title, '') || ' ' || coalesce(creator, ''))
);
