DROP INDEX IF EXISTS idx_media_items_private;
ALTER TABLE media_items DROP COLUMN IF EXISTS private;
