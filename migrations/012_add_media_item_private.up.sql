-- Add private flag to media items for share profile exclusions.
-- Items marked private = true are excluded from all share profile calculations.
ALTER TABLE media_items ADD COLUMN private BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_media_items_private ON media_items(user_id, private) WHERE private = true;
