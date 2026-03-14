-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByAuth :one
SELECT * FROM users WHERE auth_provider = $1 AND auth_subject = $2;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByProfileSlug :one
SELECT * FROM users WHERE profile_slug = $1;

-- name: CreateUser :one
INSERT INTO users (email, display_name, avatar_url, auth_provider, auth_subject)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateUserWithPassword :one
INSERT INTO users (email, display_name, auth_provider, auth_subject, password_hash)
VALUES ($1, $2, 'email', $1, $3)
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users SET display_name = $2, avatar_url = $3, profile_slug = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetPluginState :one
SELECT * FROM plugin_states WHERE user_id = $1 AND plugin = $2;

-- name: ListPluginStates :many
SELECT * FROM plugin_states WHERE user_id = $1 ORDER BY plugin;

-- name: UpsertPluginState :one
INSERT INTO plugin_states (user_id, plugin, status, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, plugin) DO UPDATE SET
    status = EXCLUDED.status,
    enabled = EXCLUDED.enabled,
    updated_at = now()
RETURNING *;

-- name: UpdatePluginStateStatus :one
UPDATE plugin_states SET status = $3, error_message = $4, updated_at = now()
WHERE user_id = $1 AND plugin = $2
RETURNING *;

-- name: UpdatePluginStateSynced :one
UPDATE plugin_states SET
    status = 'connected',
    last_synced_at = now(),
    cursor = $3,
    error_message = NULL,
    updated_at = now()
WHERE user_id = $1 AND plugin = $2
RETURNING *;

-- name: UpsertPluginCredentials :one
INSERT INTO plugin_credentials (user_id, plugin, auth_type, encrypted_data, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, plugin) DO UPDATE SET
    auth_type = EXCLUDED.auth_type,
    encrypted_data = EXCLUDED.encrypted_data,
    expires_at = EXCLUDED.expires_at,
    updated_at = now()
RETURNING *;

-- name: GetPluginCredentials :one
SELECT * FROM plugin_credentials WHERE user_id = $1 AND plugin = $2;

-- name: DeletePluginCredentials :exec
DELETE FROM plugin_credentials WHERE user_id = $1 AND plugin = $2;

-- name: CreateMediaItem :one
INSERT INTO media_items (user_id, platform, type, title, creator, consumed_at, duration, time_spent, url, external_id, raw_metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (user_id, platform, external_id) DO UPDATE SET
    title = EXCLUDED.title,
    creator = EXCLUDED.creator,
    duration = EXCLUDED.duration,
    time_spent = EXCLUDED.time_spent,
    url = EXCLUDED.url,
    raw_metadata = EXCLUDED.raw_metadata,
    updated_at = now()
RETURNING *;

-- name: ListMediaItems :many
SELECT * FROM media_items
WHERE user_id = $1
    AND consumed_at >= $2
    AND consumed_at <= $3
ORDER BY consumed_at DESC
LIMIT $4 OFFSET $5;

-- name: ListMediaItemsFiltered :many
SELECT * FROM media_items
WHERE user_id = $1
    AND consumed_at >= $2
    AND consumed_at <= $3
    AND (sqlc.narg('platform')::TEXT IS NULL OR platform = sqlc.narg('platform'))
    AND (sqlc.narg('media_type')::TEXT IS NULL OR type = sqlc.narg('media_type'))
    AND (sqlc.narg('search')::TEXT IS NULL OR (
        title ILIKE '%' || sqlc.narg('search') || '%'
        OR creator ILIKE '%' || sqlc.narg('search') || '%'
    ))
ORDER BY consumed_at DESC
LIMIT $4 OFFSET $5;

-- name: GetMediaItemByID :one
SELECT * FROM media_items WHERE id = $1 AND user_id = $2;

-- name: GetMediaItemByExternalID :one
SELECT * FROM media_items WHERE user_id = $1 AND platform = $2 AND external_id = $3;

-- name: ListPendingEnrichment :many
SELECT * FROM media_items
WHERE enrichment_status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: ClaimPendingItems :many
SELECT * FROM media_items
WHERE enrichment_status = 'pending'
    AND enrichment_retries < $2
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: UpdateEnrichmentStatus :exec
UPDATE media_items SET enrichment_status = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateEnrichmentStatusWithRetries :exec
UPDATE media_items SET
    enrichment_status = $2,
    enrichment_retries = $3,
    updated_at = now()
WHERE id = $1;

-- name: ResetEnrichmentByUser :execrows
UPDATE media_items SET
    enrichment_status = 'pending',
    enrichment_retries = 0,
    updated_at = now()
WHERE user_id = $1
    AND enrichment_status IN ('enriched', 'failed');

-- name: ResetEnrichmentByID :exec
UPDATE media_items SET
    enrichment_status = 'pending',
    enrichment_retries = 0,
    updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: EnrichmentStats :one
SELECT
    count(*) FILTER (WHERE enrichment_status = 'pending') AS pending,
    count(*) FILTER (WHERE enrichment_status = 'enriching') AS enriching,
    count(*) FILTER (WHERE enrichment_status = 'enriched') AS enriched,
    count(*) FILTER (WHERE enrichment_status = 'failed') AS failed
FROM media_items
WHERE user_id = $1;

-- name: GetOrCreateTag :one
INSERT INTO tags (name, category)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = $1;

-- name: GetTagByAlias :one
SELECT t.* FROM tags t
JOIN tag_aliases ta ON t.id = ta.tag_id
WHERE ta.alias = $1;

-- name: CreateTagAlias :one
INSERT INTO tag_aliases (alias, tag_id)
VALUES ($1, $2)
ON CONFLICT (alias) DO NOTHING
RETURNING *;

-- name: AddMediaItemTag :exec
INSERT INTO media_item_tags (media_item_id, tag_id, source, confidence)
VALUES ($1, $2, $3, $4)
ON CONFLICT (media_item_id, tag_id, source) DO UPDATE SET
    confidence = EXCLUDED.confidence;

-- name: ListMediaItemTags :many
SELECT t.name, t.category, mit.source, mit.confidence
FROM media_item_tags mit
JOIN tags t ON mit.tag_id = t.id
WHERE mit.media_item_id = $1;

-- name: CreateSyncLog :one
INSERT INTO sync_log (user_id, plugin)
VALUES ($1, $2)
RETURNING *;

-- name: CompleteSyncLog :one
UPDATE sync_log SET
    completed_at = now(),
    items_added = $2,
    items_skipped = $3,
    items_updated = $4,
    status = $5,
    error_code = $6,
    error_message = $7,
    duration_ms = $8
WHERE id = $1
RETURNING *;

-- name: ListSyncLogs :many
SELECT * FROM sync_log
WHERE user_id = $1 AND plugin = $2
ORDER BY started_at DESC
LIMIT $3;

-- name: PlatformBreakdown :many
SELECT platform, type, COUNT(*) AS count,
       COALESCE(SUM(EXTRACT(EPOCH FROM duration)), 0)::BIGINT AS total_duration_sec
FROM media_items
WHERE user_id = $1
    AND consumed_at >= $2
    AND consumed_at <= $3
GROUP BY platform, type
ORDER BY count DESC;

-- name: PlatformBreakdownFiltered :many
SELECT platform, type, COUNT(*) AS count,
       COALESCE(SUM(EXTRACT(EPOCH FROM duration)), 0)::BIGINT AS total_duration_sec
FROM media_items
WHERE user_id = $1
    AND consumed_at >= $2
    AND consumed_at <= $3
    AND (sqlc.narg('platform')::TEXT IS NULL OR platform = sqlc.narg('platform'))
    AND (sqlc.narg('media_type')::TEXT IS NULL OR type = sqlc.narg('media_type'))
GROUP BY platform, type
ORDER BY count DESC;

-- name: TagDistribution :many
SELECT t.name, t.category, COUNT(*) AS count
FROM media_item_tags mit
JOIN tags t ON mit.tag_id = t.id
JOIN media_items mi ON mit.media_item_id = mi.id
WHERE mi.user_id = $1
    AND mi.consumed_at >= $2
    AND mi.consumed_at <= $3
    AND (mit.confidence IS NULL OR mit.confidence >= 0.7)
GROUP BY t.name, t.category
ORDER BY count DESC
LIMIT $4;

-- name: TagDistributionFiltered :many
SELECT t.name, t.category, COUNT(*) AS count
FROM media_item_tags mit
JOIN tags t ON mit.tag_id = t.id
JOIN media_items mi ON mit.media_item_id = mi.id
WHERE mi.user_id = $1
    AND mi.consumed_at >= $2
    AND mi.consumed_at <= $3
    AND (mit.confidence IS NULL OR mit.confidence >= 0.7)
    AND (sqlc.narg('platform')::TEXT IS NULL OR mi.platform = sqlc.narg('platform'))
    AND (sqlc.narg('media_type')::TEXT IS NULL OR mi.type = sqlc.narg('media_type'))
GROUP BY t.name, t.category
ORDER BY count DESC
LIMIT $4;

-- name: TagDistributionByCategory :many
SELECT t.name, t.category, COUNT(*) AS count
FROM media_item_tags mit
JOIN tags t ON mit.tag_id = t.id
JOIN media_items mi ON mit.media_item_id = mi.id
WHERE mi.user_id = $1
    AND mi.consumed_at >= $2
    AND mi.consumed_at <= $3
    AND (mit.confidence IS NULL OR mit.confidence >= 0.7)
    AND (sqlc.narg('platform')::TEXT IS NULL OR mi.platform = sqlc.narg('platform'))
    AND (sqlc.narg('media_type')::TEXT IS NULL OR mi.type = sqlc.narg('media_type'))
    AND t.category = @category
GROUP BY t.name, t.category
ORDER BY count DESC
LIMIT $4;

-- name: AttentionByType :many
SELECT type, COUNT(*) AS count,
       COALESCE(SUM(EXTRACT(EPOCH FROM time_spent)), 0)::BIGINT AS total_time_spent_sec,
       COALESCE(SUM(EXTRACT(EPOCH FROM duration)), 0)::BIGINT AS total_duration_sec
FROM media_items
WHERE user_id = $1
    AND consumed_at >= $2
    AND consumed_at <= $3
    AND (sqlc.narg('platform')::TEXT IS NULL OR platform = sqlc.narg('platform'))
GROUP BY type
ORDER BY total_time_spent_sec DESC;

-- name: DeleteMediaItemsByPlatform :execrows
DELETE FROM media_items WHERE user_id = $1 AND platform = $2;

-- name: DeleteSyncLogsByPlugin :exec
DELETE FROM sync_log WHERE user_id = $1 AND plugin = $2;

-- === Share Profiles ===

-- name: GetShareProfile :one
SELECT * FROM share_profiles WHERE user_id = $1;

-- name: GetShareProfileBySlug :one
SELECT sp.*, u.display_name, u.avatar_url
FROM share_profiles sp
JOIN users u ON sp.user_id = u.id
WHERE sp.slug = $1 AND sp.published = true;

-- name: UpsertShareProfile :one
INSERT INTO share_profiles (user_id, blocks, excluded_platforms, excluded_tags, published, slug)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
    blocks = EXCLUDED.blocks,
    excluded_platforms = EXCLUDED.excluded_platforms,
    excluded_tags = EXCLUDED.excluded_tags,
    published = EXCLUDED.published,
    slug = EXCLUDED.slug,
    updated_at = now()
RETURNING *;

-- name: SetItemPrivacy :one
UPDATE media_items SET private = $3, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, private;

-- name: GetPublicItems :many
SELECT * FROM media_items
WHERE user_id = $1
    AND private = false
    AND consumed_at >= $2
    AND consumed_at <= $3
    AND (sqlc.narg('platform_filter')::TEXT IS NULL OR platform = sqlc.narg('platform_filter'))
ORDER BY consumed_at DESC
LIMIT $4;

-- name: GetPublicTagDistribution :many
SELECT t.name, t.category, COUNT(*) AS count
FROM media_item_tags mit
JOIN tags t ON mit.tag_id = t.id
JOIN media_items mi ON mit.media_item_id = mi.id
WHERE mi.user_id = $1
    AND mi.private = false
    AND mi.consumed_at >= $2
    AND mi.consumed_at <= $3
    AND (mit.confidence IS NULL OR mit.confidence >= 0.7)
    AND NOT (mi.platform = ANY($5::TEXT[]))
    AND NOT (t.name = ANY($6::TEXT[]))
    AND (sqlc.narg('category_filter')::TEXT IS NULL OR t.category = sqlc.narg('category_filter'))
GROUP BY t.name, t.category
ORDER BY count DESC
LIMIT $4;

-- name: GetPublicTopCreators :many
SELECT mi.creator, mi.platform, mi.type, COUNT(*) AS count
FROM media_items mi
WHERE mi.user_id = $1
    AND mi.private = false
    AND mi.consumed_at >= $2
    AND mi.consumed_at <= $3
    AND mi.creator IS NOT NULL AND mi.creator != ''
    AND NOT (mi.platform = ANY($5::TEXT[]))
GROUP BY mi.creator, mi.platform, mi.type
ORDER BY count DESC
LIMIT $4;

-- name: OnThisDay :many
SELECT * FROM media_items
WHERE user_id = $1
    AND EXTRACT(MONTH FROM consumed_at) = @target_month::INT
    AND EXTRACT(DAY FROM consumed_at) = @target_day::INT
    AND consumed_at < $2
ORDER BY consumed_at DESC
LIMIT $3;

-- name: GetPublicPlatformMix :many
SELECT mi.platform, COUNT(*) AS count
FROM media_items mi
WHERE mi.user_id = $1
    AND mi.private = false
    AND mi.consumed_at >= $2
    AND mi.consumed_at <= $3
    AND NOT (mi.platform = ANY($4::TEXT[]))
GROUP BY mi.platform
ORDER BY count DESC;
