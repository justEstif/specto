CREATE TABLE media_item_tags (
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    tag_id        UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    source        TEXT NOT NULL,
    confidence    REAL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (media_item_id, tag_id, source)
);

CREATE INDEX idx_media_item_tags_tag ON media_item_tags(tag_id);
CREATE INDEX idx_media_item_tags_item ON media_item_tags(media_item_id);
