-- Share profiles: user-curated public profile configuration.
-- Each user has at most one share profile with ordered display blocks.
CREATE TABLE share_profiles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    blocks      JSONB NOT NULL DEFAULT '[]',
    excluded_platforms TEXT[] NOT NULL DEFAULT '{}',
    excluded_tags      TEXT[] NOT NULL DEFAULT '{}',
    published   BOOLEAN NOT NULL DEFAULT false,
    slug        TEXT UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_share_profiles_slug ON share_profiles(slug) WHERE slug IS NOT NULL;
