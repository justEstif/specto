# Reddit Plugin

## Overview

Reddit has a well-documented OAuth2 API that provides access to a user's saved posts,
upvoted content, comment history, and subscribed subreddits. The API is free for
non-commercial use with reasonable rate limits. Reddit also offers a GDPR data export
that includes full voting and comment history.

## Access Method

- **Primary**: OAuth2 API (free tier)
- **Fallback**: GDPR data export (Reddit settings → Request Your Data)

## API Details

- **Base URL**: `https://oauth.reddit.com`
- **Auth**: OAuth 2.0 (Authorization Code flow with PKCE)
- **Token URL**: `https://www.reddit.com/api/v1/access_token`
- **Authorization URL**: `https://www.reddit.com/api/v1/authorize`

### Key Endpoints

| Endpoint | Description | Pagination |
|----------|-------------|------------|
| `GET /user/{username}/saved` | Saved posts and comments | `after`/`before` fullname cursors, 100 items/page |
| `GET /user/{username}/upvoted` | Upvoted posts | Same cursor pagination |
| `GET /user/{username}/comments` | User's comment history | Same cursor pagination |
| `GET /subreddits/mine/subscriber` | Subscribed subreddits | Same cursor pagination |
| `GET /api/v1/me` | User profile info | N/A |

All listing endpoints return a `Listing` wrapper with `data.after` for the next page cursor.

### Rate Limits

- **100 requests per minute** per OAuth client (with valid OAuth token)
- Rate limit headers: `X-Ratelimit-Used`, `X-Ratelimit-Remaining`, `X-Ratelimit-Reset`
- Free tier: no daily quota, just per-minute rate limit
- Must set a unique `User-Agent` header (e.g., `media-tracker:v1.0.0 (by /u/yourname)`)

### Scopes Required

| Scope | Purpose |
|-------|---------|
| `history` | Access saved/upvoted/hidden content |
| `identity` | Read username and account info |
| `mysubreddits` | Read subscribed subreddits |
| `read` | Read posts and comments |

## Data Export

- **How**: Settings → Safety & Privacy → Request Your Data (or `reddit.com/settings/data-request`)
- **Format**: ZIP containing CSV files
- **Key files**:
  - `saved_posts.csv` — saved posts with title, URL, subreddit, date
  - `saved_comments.csv` — saved comments
  - `post_votes.csv` — upvoted/downvoted posts with ID and direction
  - `comment_votes.csv` — upvoted/downvoted comments
  - `comments.csv` — user's own comments
  - `subscribed_subreddits.csv` — subreddit list
- **Delivery**: Usually within a few days
- **Limitation**: No "viewed posts" history — Reddit doesn't track or export what you scrolled past

## Available Data Fields

| Platform Field | MediaItem Field | Notes |
|----------------|-----------------|-------|
| `title` | `title` | Post title |
| `author` | `creator` | Post author |
| `created_utc` | `consumed_at` | When the post was created (not when you viewed it) |
| `subreddit` | `tags[]` | Subreddit as a topic tag |
| `permalink` | `url` | Reddit permalink |
| `name` (fullname) | `external_id` | e.g., `t3_abc123` for posts |
| `link_flair_text` | `tags[]` | Post flair as additional tag |
| `over_18` | `raw_metadata` | NSFW flag |
| `score` | `raw_metadata` | Post score at time of fetch |
| `num_comments` | `raw_metadata` | Comment count |
| `selftext` / `url` | `raw_metadata` | Post body or linked URL |

## Gotchas & Limitations

- **No view history** — Reddit does not track or expose what posts you actually viewed/read. Only saved, upvoted, and commented posts are accessible. This is a fundamental gap.
- **Upvoted/saved caps at ~1000** — Reddit's listing endpoints return at most ~1000 items even with pagination. For heavy users, older saved/upvoted posts are inaccessible via API.
- **`consumed_at` is imprecise** — We can use the save/upvote timestamp, but Reddit doesn't expose *when* you interacted, only the post's creation time. The save/vote time isn't in the API response (only in GDPR export).
- **Content type ambiguity** — Reddit posts can be text, images, videos, or links to articles. The `type` field mapping requires inspecting `post_hint`, `is_video`, and `domain`.
- **Rate limit enforcement changed in 2023** — After the API pricing controversy, Reddit tightened enforcement. Respect the 100 req/min limit strictly.
- **GDPR export has timestamps** — The data export includes interaction timestamps (when you voted/saved), making it more useful than the API for `consumed_at` accuracy.

## Plugin Classification

- **Auth Type**: OAuth
- **Sync Strategy**: Incremental (cursor-based using `after` fullname). Note the ~1000 item cap.
- **Difficulty**: Medium (API is clean but the 1000-item cap and missing view history limit usefulness)
- **MVP Priority**: No — Reddit consumption is hard to track meaningfully since there's no view history. Saved/upvoted posts represent a small fraction of actual consumption. Consider for post-MVP as a "curated content" source rather than a consumption tracker.
