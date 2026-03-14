-- Add enrichment_retries column for tracking retry attempts.
-- The worker increments this on each failure and stops after 3 retries.
ALTER TABLE media_items ADD COLUMN enrichment_retries INTEGER NOT NULL DEFAULT 0;
