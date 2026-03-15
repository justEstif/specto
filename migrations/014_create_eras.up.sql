-- Era detection: store detected consumption eras per user per media type
CREATE TABLE eras (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type      TEXT,                                   -- "music", "video", etc. NULL = cross-media (future)
    title           TEXT,                                   -- user-confirmed name
    suggested_title TEXT,                                   -- LLM-suggested name
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,                            -- NULL = ongoing/current era
    item_count      INTEGER NOT NULL,
    distinctiveness REAL NOT NULL,                           -- cosine distance from adjacent era (0-1)
    status          TEXT NOT NULL DEFAULT 'suggested',       -- suggested | confirmed | dismissed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_eras_user_type_status ON eras (user_id, media_type, status);
CREATE INDEX idx_eras_user_started ON eras (user_id, started_at DESC);

-- Top tags that characterize each era
CREATE TABLE era_tags (
    era_id  UUID NOT NULL REFERENCES eras(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    weight  REAL NOT NULL,                                  -- relative prominence (0-1)
    PRIMARY KEY (era_id, tag_id)
);

CREATE INDEX idx_era_tags_tag ON era_tags (tag_id);
