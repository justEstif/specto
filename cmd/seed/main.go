// Command seed creates a test user with realistic media data so you can
// develop and test the UI without hitting external APIs or burning LLM tokens.
//
// Usage: go run cmd/seed/main.go
// Or:    mise run seed
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/justestif/specto/internal/database"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	db := database.New(pool)

	// --- 1. Create or fetch the seed user ---
	const email = "test@email.com"
	const password = "password123"
	const displayName = "Test User"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	// Delete existing seed user data (idempotent re-run)
	existing, _ := db.GetUserByEmail(ctx, email)
	if existing.ID.Valid {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", existing.ID)
		fmt.Println("Deleted existing seed user, re-creating...")
	}

	user, err := db.CreateUserWithPassword(ctx, database.CreateUserWithPasswordParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: pgText(string(hash)),
	})
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	fmt.Printf("Created user: %s (%s)\n", user.Email, pgUUID(user.ID))

	// Mark onboarded
	_ = db.MarkUserOnboarded(ctx, user.ID)

	// --- 2. Create plugin states (simulate connected platforms) ---
	for _, plugin := range []string{"spotify", "youtube", "lastfm", "netflix", "tiktok", "goodreads", "anilist"} {
		_, err := db.UpsertPluginState(ctx, database.UpsertPluginStateParams{
			UserID:  user.ID,
			Plugin:  plugin,
			Status:  "connected",
			Enabled: true,
		})
		if err != nil {
			log.Fatalf("plugin state %s: %v", plugin, err)
		}
	}
	fmt.Println("Created plugin states: spotify, youtube, lastfm, netflix, tiktok, goodreads, anilist")

	// --- 3. Seed media items ---
	now := time.Now()
	items := buildMediaItems(now)

	var createdIDs []pgtype.UUID
	for _, item := range items {
		created, err := db.CreateMediaItem(ctx, database.CreateMediaItemParams{
			UserID:      user.ID,
			Platform:    item.platform,
			Type:        item.mediaType,
			Title:       item.title,
			Creator:     pgText(item.creator),
			ConsumedAt:  pgTimestamp(item.consumedAt),
			Duration:    pgInterval(item.duration),
			TimeSpent:   pgInterval(item.timeSpent),
			Url:         pgText(item.url),
			ExternalID:  item.externalID,
			RawMetadata: mustJSON(item.rawMetadata),
		})
		if err != nil {
			log.Fatalf("create media item %q: %v", item.title, err)
		}
		createdIDs = append(createdIDs, created.ID)

		// Mark as enriched
		_ = db.UpdateEnrichmentStatus(ctx, database.UpdateEnrichmentStatusParams{
			ID:               created.ID,
			EnrichmentStatus: "enriched",
		})
	}
	fmt.Printf("Created %d media items\n", len(createdIDs))

	// --- 4. Seed tags and attach to items ---
	tagMap := make(map[string]pgtype.UUID) // name -> tag ID
	getOrCreateTag := func(name, category string) pgtype.UUID {
		if id, ok := tagMap[name]; ok {
			return id
		}
		tag, err := db.GetOrCreateTag(ctx, database.GetOrCreateTagParams{
			Name:     name,
			Category: pgText(category),
		})
		if err != nil {
			log.Fatalf("tag %q: %v", name, err)
		}
		tagMap[name] = tag.ID
		return tag.ID
	}

	tagAssignments := buildTagAssignments()
	tagCount := 0
	for i, item := range items {
		if i >= len(createdIDs) {
			break
		}
		tags, ok := tagAssignments[item.externalID]
		if !ok {
			continue
		}
		for _, t := range tags {
			tagID := getOrCreateTag(t.name, t.category)
			_ = db.AddMediaItemTag(ctx, database.AddMediaItemTagParams{
				MediaItemID: createdIDs[i],
				TagID:       tagID,
				Source:      t.source,
				Confidence:  pgFloat(t.confidence),
			})
			tagCount++
		}
	}
	fmt.Printf("Created %d tags, attached %d tag assignments\n", len(tagMap), tagCount)

	// --- 5. Seed sync logs ---
	for _, plugin := range []string{"spotify", "youtube", "lastfm", "netflix", "tiktok", "goodreads", "anilist"} {
		sl, _ := db.CreateSyncLog(ctx, database.CreateSyncLogParams{
			UserID: user.ID,
			Plugin: plugin,
		})
		added := int32(rand.IntN(20) + 5)
		_, _ = db.CompleteSyncLog(ctx, database.CompleteSyncLogParams{
			ID:           sl.ID,
			ItemsAdded:   pgInt4(added),
			ItemsSkipped: pgInt4(int32(rand.IntN(3))),
			ItemsUpdated: pgInt4(int32(rand.IntN(5))),
			Status:       "success",
			DurationMs:   pgInt4(int32(rand.IntN(3000) + 500)),
		})
	}
	fmt.Println("Created sync logs")

	// --- 6. Seed share profile ---
	_, _ = db.UpsertShareProfile(ctx, database.UpsertShareProfileParams{
		UserID:            user.ID,
		Blocks:            []byte(`[{"type":"top_genres","enabled":true,"time_range":"all"},{"type":"mood_profile","enabled":true,"time_range":"all"},{"type":"top_creators","enabled":true,"time_range":"all","count":10},{"type":"platform_mix","enabled":true,"time_range":"all"},{"type":"currently_into","enabled":true,"text":"Frank Ocean, philosophy videos, and existentialist podcasts"},{"type":"listening_stats","enabled":true,"time_range":"all"}]`),
		ExcludedPlatforms: []string{},
		ExcludedTags:      []string{},
		Published:         true,
		Slug:              pgText("testuser"),
	})
	fmt.Println("Created share profile (slug: testuser)")

	// Update user profile slug to match
	_, _ = db.UpdateUserProfile(ctx, database.UpdateUserProfileParams{
		ID:          user.ID,
		DisplayName: displayName,
		ProfileSlug: pgText("testuser"),
	})

	fmt.Println("\n--- Seed complete ---")
	fmt.Printf("  Email:    %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	fmt.Printf("  Profile:  /u/testuser\n")
}

// --- Media item builder ---

type seedItem struct {
	platform    string
	mediaType   string
	title       string
	creator     string
	consumedAt  time.Time
	duration    *time.Duration
	timeSpent   *time.Duration
	url         string
	externalID  string
	rawMetadata map[string]any
}

func buildMediaItems(now time.Time) []seedItem {
	d := func(minutes int) *time.Duration {
		dur := time.Duration(minutes) * time.Minute
		return &dur
	}
	ago := func(days int, hours int) time.Time {
		return now.Add(-time.Duration(days)*24*time.Hour - time.Duration(hours)*time.Hour)
	}

	return []seedItem{
		// ============================================================
		// MUSIC ERA 1: Indie/Dreamy (months 24-18 ago, days ~730-540)
		// Tags: indie, alternative, ambient, dreamy, peaceful, chill
		// ~25 items across spotify + lastfm
		// ============================================================

		// --- Spotify: Indie/Dreamy era ---
		{platform: "spotify", mediaType: "music", title: "Myth", creator: "Beach House", consumedAt: ago(725, 14), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/1", externalID: "sp-001", rawMetadata: map[string]any{"album": "Bloom", "popularity": 72}},
		{platform: "spotify", mediaType: "music", title: "Space Song", creator: "Beach House", consumedAt: ago(718, 21), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/2", externalID: "sp-002", rawMetadata: map[string]any{"album": "Depression Cherry", "popularity": 82}},
		{platform: "spotify", mediaType: "music", title: "The Less I Know The Better", creator: "Tame Impala", consumedAt: ago(710, 9), duration: d(4), timeSpent: d(3), url: "https://open.spotify.com/track/3", externalID: "sp-003", rawMetadata: map[string]any{"album": "Currents", "popularity": 90}},
		{platform: "spotify", mediaType: "music", title: "Electric Feel", creator: "MGMT", consumedAt: ago(703, 16), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/4", externalID: "sp-004", rawMetadata: map[string]any{"album": "Oracular Spectacular", "popularity": 78}},
		{platform: "spotify", mediaType: "music", title: "Midnight City", creator: "M83", consumedAt: ago(695, 3), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/5", externalID: "sp-005", rawMetadata: map[string]any{"album": "Hurry Up, We're Dreaming", "popularity": 82}},
		{platform: "spotify", mediaType: "music", title: "Intro", creator: "The xx", consumedAt: ago(688, 11), duration: d(2), timeSpent: d(2), url: "https://open.spotify.com/track/6", externalID: "sp-006", rawMetadata: map[string]any{"album": "xx", "popularity": 75}},
		{platform: "spotify", mediaType: "music", title: "Skinny Love", creator: "Bon Iver", consumedAt: ago(680, 20), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/7", externalID: "sp-007", rawMetadata: map[string]any{"album": "For Emma, Forever Ago", "popularity": 79}},
		{platform: "spotify", mediaType: "music", title: "Two Weeks", creator: "Grizzly Bear", consumedAt: ago(672, 7), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/8", externalID: "sp-008", rawMetadata: map[string]any{"album": "Veckatimest", "popularity": 65}},
		{platform: "spotify", mediaType: "music", title: "Holocene", creator: "Bon Iver", consumedAt: ago(661, 15), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/9", externalID: "sp-009", rawMetadata: map[string]any{"album": "Bon Iver", "popularity": 76}},
		{platform: "spotify", mediaType: "music", title: "Re: Stacks", creator: "Bon Iver", consumedAt: ago(650, 2), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/10", externalID: "sp-010", rawMetadata: map[string]any{"album": "For Emma, Forever Ago", "popularity": 73}},
		{platform: "spotify", mediaType: "music", title: "Flume", creator: "Bon Iver", consumedAt: ago(640, 19), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/11", externalID: "sp-011", rawMetadata: map[string]any{"album": "For Emma, Forever Ago", "popularity": 68}},
		{platform: "spotify", mediaType: "music", title: "On Melancholy Hill", creator: "Gorillaz", consumedAt: ago(628, 10), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/12", externalID: "sp-012", rawMetadata: map[string]any{"album": "Plastic Beach", "popularity": 80}},
		{platform: "spotify", mediaType: "music", title: "Dissolve Me", creator: "alt-J", consumedAt: ago(618, 22), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/13", externalID: "sp-013", rawMetadata: map[string]any{"album": "An Awesome Wave", "popularity": 67}},
		{platform: "spotify", mediaType: "music", title: "Youth", creator: "Daughter", consumedAt: ago(605, 8), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/14", externalID: "sp-014", rawMetadata: map[string]any{"album": "If You Leave", "popularity": 71}},
		{platform: "spotify", mediaType: "music", title: "Fitzpleasure", creator: "alt-J", consumedAt: ago(590, 13), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/15", externalID: "sp-015", rawMetadata: map[string]any{"album": "An Awesome Wave", "popularity": 69}},
		{platform: "spotify", mediaType: "music", title: "Breezeblocks", creator: "alt-J", consumedAt: ago(575, 5), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/16", externalID: "sp-016", rawMetadata: map[string]any{"album": "An Awesome Wave", "popularity": 74}},
		{platform: "spotify", mediaType: "music", title: "Tongue", creator: "MNEK", consumedAt: ago(560, 17), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/17", externalID: "sp-017", rawMetadata: map[string]any{"album": "Small Talk", "popularity": 55}},
		{platform: "spotify", mediaType: "music", title: "Oblivion", creator: "Grimes", consumedAt: ago(548, 1), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/18", externalID: "sp-018", rawMetadata: map[string]any{"album": "Visions", "popularity": 70}},

		// --- Last.fm: Indie/Dreamy era ---
		{platform: "lastfm", mediaType: "music", title: "Svefn-g-englar", creator: "Sigur Ros", consumedAt: ago(720, 6), duration: d(10), timeSpent: d(10), url: "https://www.last.fm/music/Sigur+Ros/_/Svefn-g-englar", externalID: "lf-001", rawMetadata: map[string]any{"album": "Agaetis byrjun"}},
		{platform: "lastfm", mediaType: "music", title: "Breathe (In the Air)", creator: "Pink Floyd", consumedAt: ago(705, 12), duration: d(3), timeSpent: d(3), url: "https://www.last.fm/music/Pink+Floyd/_/Breathe", externalID: "lf-002", rawMetadata: map[string]any{"album": "The Dark Side of the Moon"}},
		{platform: "lastfm", mediaType: "music", title: "Teardrop", creator: "Massive Attack", consumedAt: ago(690, 23), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Massive+Attack/_/Teardrop", externalID: "lf-003", rawMetadata: map[string]any{"album": "Mezzanine"}},
		{platform: "lastfm", mediaType: "music", title: "Everything In Its Right Place", creator: "Radiohead", consumedAt: ago(670, 4), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Radiohead", externalID: "lf-004", rawMetadata: map[string]any{"album": "Kid A"}},
		{platform: "lastfm", mediaType: "music", title: "Karma Police", creator: "Radiohead", consumedAt: ago(655, 18), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Radiohead/_/Karma+Police", externalID: "lf-005", rawMetadata: map[string]any{"album": "OK Computer"}},
		{platform: "lastfm", mediaType: "music", title: "All I Need", creator: "Radiohead", consumedAt: ago(635, 9), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Radiohead/_/All+I+Need", externalID: "lf-006", rawMetadata: map[string]any{"album": "In Rainbows"}},
		{platform: "lastfm", mediaType: "music", title: "Cigarettes After Sex", creator: "Apocalypse", consumedAt: ago(615, 0), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Cigarettes+After+Sex/_/Apocalypse", externalID: "lf-007", rawMetadata: map[string]any{"album": "Cigarettes After Sex"}},

		// ============================================================
		// MUSIC ERA 2: Hip-Hop/Intense (months 18-12 ago, days ~540-365)
		// Tags: hip-hop, r-and-b, intense, raw, aggressive, dark
		// ~25 items across spotify + lastfm
		// ============================================================

		// --- Spotify: Hip-Hop/Intense era ---
		{platform: "spotify", mediaType: "music", title: "Runaway", creator: "Kanye West", consumedAt: ago(535, 3), duration: d(9), timeSpent: d(9), url: "https://open.spotify.com/track/19", externalID: "sp-019", rawMetadata: map[string]any{"album": "My Beautiful Dark Twisted Fantasy", "popularity": 85}},
		{platform: "spotify", mediaType: "music", title: "m.A.A.d city", creator: "Kendrick Lamar", consumedAt: ago(525, 20), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/20", externalID: "sp-020", rawMetadata: map[string]any{"album": "good kid, m.A.A.d city", "popularity": 82}},
		{platform: "spotify", mediaType: "music", title: "Alright", creator: "Kendrick Lamar", consumedAt: ago(518, 7), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/21", externalID: "sp-021", rawMetadata: map[string]any{"album": "To Pimp a Butterfly", "popularity": 83}},
		{platform: "spotify", mediaType: "music", title: "Sing About Me, I'm Dying of Thirst", creator: "Kendrick Lamar", consumedAt: ago(508, 14), duration: d(12), timeSpent: d(12), url: "https://open.spotify.com/track/22", externalID: "sp-022", rawMetadata: map[string]any{"album": "good kid, m.A.A.d city", "popularity": 78}},
		{platform: "spotify", mediaType: "music", title: "Power", creator: "Kanye West", consumedAt: ago(500, 22), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/23", externalID: "sp-023", rawMetadata: map[string]any{"album": "My Beautiful Dark Twisted Fantasy", "popularity": 84}},
		{platform: "spotify", mediaType: "music", title: "Backseat Freestyle", creator: "Kendrick Lamar", consumedAt: ago(492, 1), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/24", externalID: "sp-024", rawMetadata: map[string]any{"album": "good kid, m.A.A.d city", "popularity": 80}},
		{platform: "spotify", mediaType: "music", title: "Devil in a New Dress", creator: "Kanye West", consumedAt: ago(483, 16), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/25", externalID: "sp-025", rawMetadata: map[string]any{"album": "My Beautiful Dark Twisted Fantasy", "popularity": 79}},
		{platform: "spotify", mediaType: "music", title: "DNA.", creator: "Kendrick Lamar", consumedAt: ago(475, 10), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/26", externalID: "sp-026", rawMetadata: map[string]any{"album": "DAMN.", "popularity": 88}},
		{platform: "spotify", mediaType: "music", title: "HUMBLE.", creator: "Kendrick Lamar", consumedAt: ago(465, 5), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/27", externalID: "sp-027", rawMetadata: map[string]any{"album": "DAMN.", "popularity": 92}},
		{platform: "spotify", mediaType: "music", title: "Gorgeous", creator: "Kanye West", consumedAt: ago(455, 19), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/28", externalID: "sp-028", rawMetadata: map[string]any{"album": "My Beautiful Dark Twisted Fantasy", "popularity": 76}},
		{platform: "spotify", mediaType: "music", title: "Money Trees", creator: "Kendrick Lamar", consumedAt: ago(445, 2), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/29", externalID: "sp-029", rawMetadata: map[string]any{"album": "good kid, m.A.A.d city", "popularity": 86}},
		{platform: "spotify", mediaType: "music", title: "N.Y. State of Mind", creator: "Nas", consumedAt: ago(435, 13), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/30", externalID: "sp-030", rawMetadata: map[string]any{"album": "Illmatic", "popularity": 77}},
		{platform: "spotify", mediaType: "music", title: "Shook Ones Part II", creator: "Mobb Deep", consumedAt: ago(425, 8), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/31", externalID: "sp-031", rawMetadata: map[string]any{"album": "The Infamous", "popularity": 75}},
		{platform: "spotify", mediaType: "music", title: "C.R.E.A.M.", creator: "Wu-Tang Clan", consumedAt: ago(415, 21), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/32", externalID: "sp-032", rawMetadata: map[string]any{"album": "Enter the Wu-Tang", "popularity": 78}},
		{platform: "spotify", mediaType: "music", title: "93 'Til Infinity", creator: "Souls of Mischief", consumedAt: ago(405, 6), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/33", externalID: "sp-033", rawMetadata: map[string]any{"album": "93 'Til Infinity", "popularity": 68}},
		{platform: "spotify", mediaType: "music", title: "Dead Presidents II", creator: "Jay-Z", consumedAt: ago(395, 11), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/34", externalID: "sp-034", rawMetadata: map[string]any{"album": "Reasonable Doubt", "popularity": 72}},
		{platform: "spotify", mediaType: "music", title: "Wesley's Theory", creator: "Kendrick Lamar", consumedAt: ago(385, 18), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/35", externalID: "sp-035", rawMetadata: map[string]any{"album": "To Pimp a Butterfly", "popularity": 74}},
		{platform: "spotify", mediaType: "music", title: "All of the Lights", creator: "Kanye West", consumedAt: ago(375, 4), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/36", externalID: "sp-036", rawMetadata: map[string]any{"album": "My Beautiful Dark Twisted Fantasy", "popularity": 86}},

		// --- Last.fm: Hip-Hop/Intense era ---
		{platform: "lastfm", mediaType: "music", title: "Protect Ya Neck", creator: "Wu-Tang Clan", consumedAt: ago(530, 15), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Wu-Tang+Clan/_/Protect+Ya+Neck", externalID: "lf-008", rawMetadata: map[string]any{"album": "Enter the Wu-Tang"}},
		{platform: "lastfm", mediaType: "music", title: "The World Is Yours", creator: "Nas", consumedAt: ago(510, 9), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Nas/_/The+World+Is+Yours", externalID: "lf-009", rawMetadata: map[string]any{"album": "Illmatic"}},
		{platform: "lastfm", mediaType: "music", title: "Juicy", creator: "The Notorious B.I.G.", consumedAt: ago(490, 0), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Notorious+B.I.G./_/Juicy", externalID: "lf-010", rawMetadata: map[string]any{"album": "Ready to Die"}},
		{platform: "lastfm", mediaType: "music", title: "Mathematics", creator: "Mos Def", consumedAt: ago(470, 17), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Mos+Def/_/Mathematics", externalID: "lf-011", rawMetadata: map[string]any{"album": "Black on Both Sides"}},
		{platform: "lastfm", mediaType: "music", title: "Accordion", creator: "Madvillain", consumedAt: ago(450, 7), duration: d(2), timeSpent: d(2), url: "https://www.last.fm/music/Madvillain/_/Accordion", externalID: "lf-012", rawMetadata: map[string]any{"album": "Madvillainy"}},
		{platform: "lastfm", mediaType: "music", title: "Electric Relaxation", creator: "A Tribe Called Quest", consumedAt: ago(430, 22), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/A+Tribe+Called+Quest/_/Electric+Relaxation", externalID: "lf-013", rawMetadata: map[string]any{"album": "Midnight Marauders"}},
		{platform: "lastfm", mediaType: "music", title: "Passin' Me By", creator: "The Pharcyde", consumedAt: ago(410, 3), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/The+Pharcyde/_/Passin'+Me+By", externalID: "lf-014", rawMetadata: map[string]any{"album": "Bizarre Ride II the Pharcyde"}},

		// ============================================================
		// MUSIC ERA 3: Electronic/Energetic (months 12-6 ago, days ~365-180)
		// Tags: electronic, pop, energetic, playful, uplifting
		// ~25 items across spotify + lastfm
		// ============================================================

		// --- Spotify: Electronic/Energetic era ---
		{platform: "spotify", mediaType: "music", title: "One More Time", creator: "Daft Punk", consumedAt: ago(360, 10), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/37", externalID: "sp-037", rawMetadata: map[string]any{"album": "Discovery", "popularity": 88}},
		{platform: "spotify", mediaType: "music", title: "Around the World", creator: "Daft Punk", consumedAt: ago(352, 2), duration: d(7), timeSpent: d(7), url: "https://open.spotify.com/track/38", externalID: "sp-038", rawMetadata: map[string]any{"album": "Homework", "popularity": 82}},
		{platform: "spotify", mediaType: "music", title: "Strobe", creator: "deadmau5", consumedAt: ago(343, 17), duration: d(10), timeSpent: d(10), url: "https://open.spotify.com/track/39", externalID: "sp-039", rawMetadata: map[string]any{"album": "For Lack of a Better Name", "popularity": 79}},
		{platform: "spotify", mediaType: "music", title: "Clarity", creator: "Zedd", consumedAt: ago(335, 8), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/40", externalID: "sp-040", rawMetadata: map[string]any{"album": "Clarity", "popularity": 83}},
		{platform: "spotify", mediaType: "music", title: "Levels", creator: "Avicii", consumedAt: ago(328, 23), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/41", externalID: "sp-041", rawMetadata: map[string]any{"album": "True", "popularity": 86}},
		{platform: "spotify", mediaType: "music", title: "Get Lucky", creator: "Daft Punk", consumedAt: ago(318, 6), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/42", externalID: "sp-042", rawMetadata: map[string]any{"album": "Random Access Memories", "popularity": 90}},
		{platform: "spotify", mediaType: "music", title: "Blinding Lights", creator: "The Weeknd", consumedAt: ago(308, 14), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/43", externalID: "sp-043", rawMetadata: map[string]any{"album": "After Hours", "popularity": 92}},
		{platform: "spotify", mediaType: "music", title: "Starboy", creator: "The Weeknd", consumedAt: ago(300, 0), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/44", externalID: "sp-044", rawMetadata: map[string]any{"album": "Starboy", "popularity": 91}},
		{platform: "spotify", mediaType: "music", title: "Feel Good Inc.", creator: "Gorillaz", consumedAt: ago(290, 19), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/45", externalID: "sp-045", rawMetadata: map[string]any{"album": "Demon Days", "popularity": 87}},
		{platform: "spotify", mediaType: "music", title: "Digital Love", creator: "Daft Punk", consumedAt: ago(282, 11), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/46", externalID: "sp-046", rawMetadata: map[string]any{"album": "Discovery", "popularity": 76}},
		{platform: "spotify", mediaType: "music", title: "Midnight City", creator: "M83", consumedAt: ago(273, 4), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/47", externalID: "sp-047", rawMetadata: map[string]any{"album": "Hurry Up, We're Dreaming", "popularity": 82}},
		{platform: "spotify", mediaType: "music", title: "D.A.N.C.E.", creator: "Justice", consumedAt: ago(262, 16), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/48", externalID: "sp-048", rawMetadata: map[string]any{"album": "Cross", "popularity": 73}},
		{platform: "spotify", mediaType: "music", title: "Opus", creator: "Eric Prydz", consumedAt: ago(250, 9), duration: d(9), timeSpent: d(9), url: "https://open.spotify.com/track/49", externalID: "sp-049", rawMetadata: map[string]any{"album": "Opus", "popularity": 71}},
		{platform: "spotify", mediaType: "music", title: "Lean On", creator: "Major Lazer", consumedAt: ago(240, 21), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/50", externalID: "sp-050", rawMetadata: map[string]any{"album": "Peace Is the Mission", "popularity": 85}},
		{platform: "spotify", mediaType: "music", title: "Innerbloom", creator: "RUFUS DU SOL", consumedAt: ago(230, 7), duration: d(9), timeSpent: d(9), url: "https://open.spotify.com/track/51", externalID: "sp-051", rawMetadata: map[string]any{"album": "Bloom", "popularity": 77}},
		{platform: "spotify", mediaType: "music", title: "Sad Machine", creator: "Porter Robinson", consumedAt: ago(220, 13), duration: d(6), timeSpent: d(6), url: "https://open.spotify.com/track/52", externalID: "sp-052", rawMetadata: map[string]any{"album": "Worlds", "popularity": 74}},
		{platform: "spotify", mediaType: "music", title: "Shelter", creator: "Porter Robinson & Madeon", consumedAt: ago(210, 1), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/53", externalID: "sp-053", rawMetadata: map[string]any{"album": "Shelter", "popularity": 78}},
		{platform: "spotify", mediaType: "music", title: "Something About Us", creator: "Daft Punk", consumedAt: ago(195, 18), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/54", externalID: "sp-054", rawMetadata: map[string]any{"album": "Discovery", "popularity": 80}},

		// --- Last.fm: Electronic/Energetic era ---
		{platform: "lastfm", mediaType: "music", title: "Windowlicker", creator: "Aphex Twin", consumedAt: ago(355, 5), duration: d(6), timeSpent: d(6), url: "https://www.last.fm/music/Aphex+Twin/_/Windowlicker", externalID: "lf-015", rawMetadata: map[string]any{"album": "Windowlicker"}},
		{platform: "lastfm", mediaType: "music", title: "Born Slippy", creator: "Underworld", consumedAt: ago(340, 12), duration: d(10), timeSpent: d(10), url: "https://www.last.fm/music/Underworld/_/Born+Slippy", externalID: "lf-016", rawMetadata: map[string]any{"album": "Second Toughest in the Infants"}},
		{platform: "lastfm", mediaType: "music", title: "Porcelain", creator: "Moby", consumedAt: ago(320, 20), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Moby/_/Porcelain", externalID: "lf-017", rawMetadata: map[string]any{"album": "Play"}},
		{platform: "lastfm", mediaType: "music", title: "Go", creator: "Chemical Brothers", consumedAt: ago(298, 3), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Chemical+Brothers/_/Go", externalID: "lf-018", rawMetadata: map[string]any{"album": "Further"}},
		{platform: "lastfm", mediaType: "music", title: "Firestarter", creator: "The Prodigy", consumedAt: ago(275, 15), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/The+Prodigy/_/Firestarter", externalID: "lf-019", rawMetadata: map[string]any{"album": "The Fat of the Land"}},
		{platform: "lastfm", mediaType: "music", title: "Halcyon and On and On", creator: "Orbital", consumedAt: ago(255, 8), duration: d(10), timeSpent: d(10), url: "https://www.last.fm/music/Orbital/_/Halcyon", externalID: "lf-020", rawMetadata: map[string]any{"album": "Orbital 2"}},
		{platform: "lastfm", mediaType: "music", title: "Xtal", creator: "Aphex Twin", consumedAt: ago(235, 0), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Aphex+Twin/_/Xtal", externalID: "lf-021", rawMetadata: map[string]any{"album": "Selected Ambient Works 85-92"}},

		// ============================================================
		// MUSIC ERA 4: R&B/Melancholic (months 6-0 ago, days ~180-0)
		// Tags: r-and-b, soul, funk, melancholic, nostalgic, contemplative
		// ~25 items across spotify + lastfm
		// ============================================================

		// --- Spotify: R&B/Melancholic era ---
		{platform: "spotify", mediaType: "music", title: "Nights", creator: "Frank Ocean", consumedAt: ago(175, 9), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/55", externalID: "sp-055", rawMetadata: map[string]any{"album": "Blonde", "popularity": 84}},
		{platform: "spotify", mediaType: "music", title: "Self Control", creator: "Frank Ocean", consumedAt: ago(168, 22), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/56", externalID: "sp-056", rawMetadata: map[string]any{"album": "Blonde", "popularity": 80}},
		{platform: "spotify", mediaType: "music", title: "Pink + White", creator: "Frank Ocean", consumedAt: ago(160, 3), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/57", externalID: "sp-057", rawMetadata: map[string]any{"album": "Blonde", "popularity": 79}},
		{platform: "spotify", mediaType: "music", title: "Ivy", creator: "Frank Ocean", consumedAt: ago(152, 15), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/58", externalID: "sp-058", rawMetadata: map[string]any{"album": "Blonde", "popularity": 81}},
		{platform: "spotify", mediaType: "music", title: "Pyramids", creator: "Frank Ocean", consumedAt: ago(143, 7), duration: d(10), timeSpent: d(10), url: "https://open.spotify.com/track/59", externalID: "sp-059", rawMetadata: map[string]any{"album": "Channel Orange", "popularity": 76}},
		{platform: "spotify", mediaType: "music", title: "Redbone", creator: "Childish Gambino", consumedAt: ago(135, 20), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/60", externalID: "sp-060", rawMetadata: map[string]any{"album": "Awaken, My Love!", "popularity": 88}},
		{platform: "spotify", mediaType: "music", title: "Best Part", creator: "Daniel Caesar", consumedAt: ago(128, 1), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/61", externalID: "sp-061", rawMetadata: map[string]any{"album": "Freudian", "popularity": 85}},
		{platform: "spotify", mediaType: "music", title: "Untitled 02", creator: "Kendrick Lamar", consumedAt: ago(120, 14), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/62", externalID: "sp-062", rawMetadata: map[string]any{"album": "untitled unmastered.", "popularity": 70}},
		{platform: "spotify", mediaType: "music", title: "Come Through and Chill", creator: "Miguel", consumedAt: ago(112, 6), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/63", externalID: "sp-063", rawMetadata: map[string]any{"album": "War & Leisure", "popularity": 72}},
		{platform: "spotify", mediaType: "music", title: "Cranes in the Sky", creator: "Solange", consumedAt: ago(103, 19), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/64", externalID: "sp-064", rawMetadata: map[string]any{"album": "A Seat at the Table", "popularity": 74}},
		{platform: "spotify", mediaType: "music", title: "Girl Like Me", creator: "Jazmine Sullivan", consumedAt: ago(95, 11), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/65", externalID: "sp-065", rawMetadata: map[string]any{"album": "Heaux Tales", "popularity": 71}},
		{platform: "spotify", mediaType: "music", title: "Good Days", creator: "SZA", consumedAt: ago(85, 2), duration: d(5), timeSpent: d(5), url: "https://open.spotify.com/track/66", externalID: "sp-066", rawMetadata: map[string]any{"album": "Good Days", "popularity": 83}},
		{platform: "spotify", mediaType: "music", title: "Doo Wop (That Thing)", creator: "Lauryn Hill", consumedAt: ago(75, 16), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/67", externalID: "sp-067", rawMetadata: map[string]any{"album": "The Miseducation of Lauryn Hill", "popularity": 80}},
		{platform: "spotify", mediaType: "music", title: "Electric", creator: "Khalid", consumedAt: ago(65, 8), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/68", externalID: "sp-068", rawMetadata: map[string]any{"album": "Free Spirit", "popularity": 75}},
		{platform: "spotify", mediaType: "music", title: "Superposition", creator: "Daniel Caesar", consumedAt: ago(50, 21), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/69", externalID: "sp-069", rawMetadata: map[string]any{"album": "Never Enough", "popularity": 73}},
		{platform: "spotify", mediaType: "music", title: "Lost", creator: "Frank Ocean", consumedAt: ago(38, 5), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/70", externalID: "sp-070", rawMetadata: map[string]any{"album": "Channel Orange", "popularity": 77}},
		{platform: "spotify", mediaType: "music", title: "Thinkin Bout You", creator: "Frank Ocean", consumedAt: ago(22, 13), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/71", externalID: "sp-071", rawMetadata: map[string]any{"album": "Channel Orange", "popularity": 82}},
		{platform: "spotify", mediaType: "music", title: "Kiss of Life", creator: "Sade", consumedAt: ago(10, 0), duration: d(4), timeSpent: d(4), url: "https://open.spotify.com/track/72", externalID: "sp-072", rawMetadata: map[string]any{"album": "Love Deluxe", "popularity": 70}},
		{platform: "spotify", mediaType: "music", title: "Killed Before", creator: "SZA", consumedAt: ago(3, 18), duration: d(3), timeSpent: d(3), url: "https://open.spotify.com/track/73", externalID: "sp-073", rawMetadata: map[string]any{"album": "SOS", "popularity": 81}},

		// --- Last.fm: R&B/Melancholic era ---
		{platform: "lastfm", mediaType: "music", title: "Untitled (How Does It Feel)", creator: "D'Angelo", consumedAt: ago(170, 4), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/D'Angelo/_/Untitled", externalID: "lf-022", rawMetadata: map[string]any{"album": "Voodoo"}},
		{platform: "lastfm", mediaType: "music", title: "Golden", creator: "Jill Scott", consumedAt: ago(148, 12), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Jill+Scott/_/Golden", externalID: "lf-023", rawMetadata: map[string]any{"album": "Beautifully Human"}},
		{platform: "lastfm", mediaType: "music", title: "Brown Skin Girl", creator: "Beyonce", consumedAt: ago(125, 23), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Beyonce/_/Brown+Skin+Girl", externalID: "lf-024", rawMetadata: map[string]any{"album": "The Lion King: The Gift"}},
		{platform: "lastfm", mediaType: "music", title: "I Want You", creator: "Erykah Badu", consumedAt: ago(100, 7), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Erykah+Badu/_/I+Want+You", externalID: "lf-025", rawMetadata: map[string]any{"album": "Mama's Gun"}},
		{platform: "lastfm", mediaType: "music", title: "On & On", creator: "Erykah Badu", consumedAt: ago(72, 16), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/Erykah+Badu/_/On+%26+On", externalID: "lf-026", rawMetadata: map[string]any{"album": "Baduizm"}},
		{platform: "lastfm", mediaType: "music", title: "Me & Those Dreamin' Eyes of Mine", creator: "D'Angelo", consumedAt: ago(42, 10), duration: d(5), timeSpent: d(5), url: "https://www.last.fm/music/D'Angelo/_/Dreamin+Eyes", externalID: "lf-027", rawMetadata: map[string]any{"album": "Brown Sugar"}},
		{platform: "lastfm", mediaType: "music", title: "Say My Name", creator: "Destiny's Child", consumedAt: ago(15, 3), duration: d(4), timeSpent: d(4), url: "https://www.last.fm/music/Destiny's+Child/_/Say+My+Name", externalID: "lf-028", rawMetadata: map[string]any{"album": "The Writing's on the Wall"}},

		// ============================================================
		// VIDEO ERA 1: Science/Math (months 24-16 ago, days ~730-480)
		// Tags: science, mathematics, education, serious, contemplative
		// ~20 items
		// ============================================================
		{platform: "youtube", mediaType: "video", title: "But what is a neural network? | Deep learning, chapter 1", creator: "3Blue1Brown", consumedAt: ago(728, 10), duration: d(19), timeSpent: d(19), url: "https://youtube.com/watch?v=v001", externalID: "yt-001", rawMetadata: map[string]any{"channel_id": "UCYO_jab_esuFRV4b17AJtAw", "view_count": 18000000}},
		{platform: "youtube", mediaType: "video", title: "The Unreasonable Effectiveness of Mathematics", creator: "Veritasium", consumedAt: ago(718, 22), duration: d(22), timeSpent: d(22), url: "https://youtube.com/watch?v=v002", externalID: "yt-002", rawMetadata: map[string]any{"channel_id": "UCHnyfMqiRRG1u-2MsSQLbXA", "view_count": 9500000}},
		{platform: "youtube", mediaType: "video", title: "The Banach-Tarski Paradox", creator: "Vsauce", consumedAt: ago(705, 3), duration: d(24), timeSpent: d(24), url: "https://youtube.com/watch?v=v003", externalID: "yt-003", rawMetadata: map[string]any{"channel_id": "UC6nSFpj9HTCZ5t-N3Rm3-HA", "view_count": 42000000}},
		{platform: "youtube", mediaType: "video", title: "Why Gravity is NOT a Force", creator: "Veritasium", consumedAt: ago(695, 14), duration: d(17), timeSpent: d(17), url: "https://youtube.com/watch?v=v004", externalID: "yt-004", rawMetadata: map[string]any{"channel_id": "UCHnyfMqiRRG1u-2MsSQLbXA", "view_count": 14000000}},
		{platform: "youtube", mediaType: "video", title: "The Map of Mathematics", creator: "Domain of Science", consumedAt: ago(682, 8), duration: d(11), timeSpent: d(11), url: "https://youtube.com/watch?v=v005", externalID: "yt-005", rawMetadata: map[string]any{"channel_id": "UCxqAWLTk1CmBvZFPzeYP3dA", "view_count": 15000000}},
		{platform: "youtube", mediaType: "video", title: "How To Speak", creator: "MIT OpenCourseWare", consumedAt: ago(670, 1), duration: d(60), timeSpent: d(45), url: "https://youtube.com/watch?v=v006", externalID: "yt-006", rawMetadata: map[string]any{"channel_id": "UCEBb1b_L6zDS3xTUrIALZOw", "view_count": 12000000}},
		{platform: "youtube", mediaType: "video", title: "But what is the Fourier Transform?", creator: "3Blue1Brown", consumedAt: ago(658, 19), duration: d(20), timeSpent: d(20), url: "https://youtube.com/watch?v=v007", externalID: "yt-007", rawMetadata: map[string]any{"channel_id": "UCYO_jab_esuFRV4b17AJtAw", "view_count": 12000000}},
		{platform: "youtube", mediaType: "video", title: "The Essence of Calculus", creator: "3Blue1Brown", consumedAt: ago(645, 6), duration: d(17), timeSpent: d(17), url: "https://youtube.com/watch?v=v008", externalID: "yt-008", rawMetadata: map[string]any{"channel_id": "UCYO_jab_esuFRV4b17AJtAw", "view_count": 10000000}},
		{platform: "youtube", mediaType: "video", title: "How Imaginary Numbers Were Invented", creator: "Veritasium", consumedAt: ago(632, 15), duration: d(23), timeSpent: d(23), url: "https://youtube.com/watch?v=v009", externalID: "yt-009", rawMetadata: map[string]any{"channel_id": "UCHnyfMqiRRG1u-2MsSQLbXA", "view_count": 11000000}},
		{platform: "youtube", mediaType: "video", title: "What is NOT Random?", creator: "Vsauce", consumedAt: ago(618, 0), duration: d(16), timeSpent: d(16), url: "https://youtube.com/watch?v=v010", externalID: "yt-010", rawMetadata: map[string]any{"channel_id": "UC6nSFpj9HTCZ5t-N3Rm3-HA", "view_count": 23000000}},
		{platform: "youtube", mediaType: "video", title: "Godel's Incompleteness Theorem", creator: "Numberphile", consumedAt: ago(605, 11), duration: d(14), timeSpent: d(14), url: "https://youtube.com/watch?v=v011", externalID: "yt-011", rawMetadata: map[string]any{"channel_id": "UCoxcjq-8xIDTYp3uz647V5A", "view_count": 5000000}},
		{platform: "youtube", mediaType: "video", title: "Quantum Entanglement Explained", creator: "Fermilab", consumedAt: ago(592, 4), duration: d(10), timeSpent: d(10), url: "https://youtube.com/watch?v=v012", externalID: "yt-012", rawMetadata: map[string]any{"channel_id": "UCD5B6VoXvoOYR-WJhJVribQ", "view_count": 4500000}},
		{platform: "youtube", mediaType: "video", title: "The Riemann Hypothesis, Explained", creator: "Quanta Magazine", consumedAt: ago(578, 17), duration: d(15), timeSpent: d(15), url: "https://youtube.com/watch?v=v013", externalID: "yt-013", rawMetadata: map[string]any{"view_count": 3800000}},
		{platform: "youtube", mediaType: "video", title: "Chaos Theory: The Language of Fractals", creator: "Veritasium", consumedAt: ago(560, 7), duration: d(18), timeSpent: d(18), url: "https://youtube.com/watch?v=v014", externalID: "yt-014", rawMetadata: map[string]any{"channel_id": "UCHnyfMqiRRG1u-2MsSQLbXA", "view_count": 8000000}},
		{platform: "youtube", mediaType: "video", title: "How Big is Infinity?", creator: "TED-Ed", consumedAt: ago(545, 20), duration: d(7), timeSpent: d(7), url: "https://youtube.com/watch?v=v015", externalID: "yt-015", rawMetadata: map[string]any{"view_count": 6000000}},
		{platform: "youtube", mediaType: "video", title: "The Map of Physics", creator: "Domain of Science", consumedAt: ago(530, 2), duration: d(9), timeSpent: d(9), url: "https://youtube.com/watch?v=v016", externalID: "yt-016", rawMetadata: map[string]any{"channel_id": "UCxqAWLTk1CmBvZFPzeYP3dA", "view_count": 12000000}},
		{platform: "youtube", mediaType: "video", title: "The Simplest Math Problem No One Can Solve", creator: "Veritasium", consumedAt: ago(515, 13), duration: d(22), timeSpent: d(22), url: "https://youtube.com/watch?v=v017", externalID: "yt-017", rawMetadata: map[string]any{"channel_id": "UCHnyfMqiRRG1u-2MsSQLbXA", "view_count": 30000000}},
		{platform: "youtube", mediaType: "video", title: "Linear Algebra - Full Course", creator: "3Blue1Brown", consumedAt: ago(500, 5), duration: d(30), timeSpent: d(25), url: "https://youtube.com/watch?v=v018", externalID: "yt-018", rawMetadata: map[string]any{"channel_id": "UCYO_jab_esuFRV4b17AJtAw", "view_count": 7000000}},
		{platform: "youtube", mediaType: "video", title: "The Beauty of Euler's Formula", creator: "Mathologer", consumedAt: ago(488, 18), duration: d(20), timeSpent: d(20), url: "https://youtube.com/watch?v=v019", externalID: "yt-019", rawMetadata: map[string]any{"view_count": 2500000}},

		// ============================================================
		// VIDEO ERA 2: Programming/Design (months 16-8 ago, days ~480-240)
		// Tags: programming, design, technology, inspirational
		// ~20 items
		// ============================================================
		{platform: "youtube", mediaType: "video", title: "The Art of Code - Dylan Beattie", creator: "NDC Conferences", consumedAt: ago(478, 3), duration: d(60), timeSpent: d(60), url: "https://youtube.com/watch?v=v020", externalID: "yt-020", rawMetadata: map[string]any{"channel_id": "UCTdw38Cw6jcm0atBPA39a0Q", "view_count": 2500000}},
		{platform: "youtube", mediaType: "video", title: "Inventing on Principle - Bret Victor", creator: "CUSEC", consumedAt: ago(465, 14), duration: d(54), timeSpent: d(54), url: "https://youtube.com/watch?v=v021", externalID: "yt-021", rawMetadata: map[string]any{"view_count": 1800000}},
		{platform: "youtube", mediaType: "video", title: "Simple Made Easy - Rich Hickey", creator: "Strange Loop", consumedAt: ago(455, 8), duration: d(60), timeSpent: d(60), url: "https://youtube.com/watch?v=v022", externalID: "yt-022", rawMetadata: map[string]any{"view_count": 900000}},
		{platform: "youtube", mediaType: "video", title: "The Future of Programming - Bret Victor", creator: "DBX Conference", consumedAt: ago(442, 20), duration: d(32), timeSpent: d(32), url: "https://youtube.com/watch?v=v023", externalID: "yt-023", rawMetadata: map[string]any{"view_count": 1200000}},
		{platform: "youtube", mediaType: "video", title: "How to Design a Good API and Why It Matters", creator: "Google Tech Talks", consumedAt: ago(430, 1), duration: d(60), timeSpent: d(50), url: "https://youtube.com/watch?v=v024", externalID: "yt-024", rawMetadata: map[string]any{"view_count": 600000}},
		{platform: "youtube", mediaType: "video", title: "Hammock Driven Development - Rich Hickey", creator: "Clojure", consumedAt: ago(418, 16), duration: d(40), timeSpent: d(40), url: "https://youtube.com/watch?v=v025", externalID: "yt-025", rawMetadata: map[string]any{"view_count": 350000}},
		{platform: "youtube", mediaType: "video", title: "What Makes Great UI?", creator: "Juxtopposed", consumedAt: ago(405, 7), duration: d(12), timeSpent: d(12), url: "https://youtube.com/watch?v=v026", externalID: "yt-026", rawMetadata: map[string]any{"view_count": 2100000}},
		{platform: "youtube", mediaType: "video", title: "The Grug Brained Developer", creator: "ThePrimeagen", consumedAt: ago(395, 22), duration: d(45), timeSpent: d(45), url: "https://youtube.com/watch?v=v027", externalID: "yt-027", rawMetadata: map[string]any{"view_count": 800000}},
		{platform: "youtube", mediaType: "video", title: "Responsive Design Made Easy", creator: "Kevin Powell", consumedAt: ago(382, 4), duration: d(25), timeSpent: d(25), url: "https://youtube.com/watch?v=v028", externalID: "yt-028", rawMetadata: map[string]any{"view_count": 500000}},
		{platform: "youtube", mediaType: "video", title: "Building a Modern CLI in Go", creator: "Charm", consumedAt: ago(370, 11), duration: d(18), timeSpent: d(18), url: "https://youtube.com/watch?v=v029", externalID: "yt-029", rawMetadata: map[string]any{"view_count": 250000}},
		{platform: "youtube", mediaType: "video", title: "Concurrency Is Not Parallelism - Rob Pike", creator: "Gopher Academy", consumedAt: ago(358, 19), duration: d(31), timeSpent: d(31), url: "https://youtube.com/watch?v=v030", externalID: "yt-030", rawMetadata: map[string]any{"view_count": 700000}},
		{platform: "youtube", mediaType: "video", title: "Design Systems in Figma", creator: "Figma", consumedAt: ago(345, 6), duration: d(35), timeSpent: d(35), url: "https://youtube.com/watch?v=v031", externalID: "yt-031", rawMetadata: map[string]any{"view_count": 450000}},
		{platform: "youtube", mediaType: "video", title: "HTMX: The Good Parts", creator: "Fireship", consumedAt: ago(332, 15), duration: d(8), timeSpent: d(8), url: "https://youtube.com/watch?v=v032", externalID: "yt-032", rawMetadata: map[string]any{"view_count": 1500000}},
		{platform: "youtube", mediaType: "video", title: "Every React Concept Explained in 12 Minutes", creator: "Fireship", consumedAt: ago(318, 0), duration: d(12), timeSpent: d(12), url: "https://youtube.com/watch?v=v033", externalID: "yt-033", rawMetadata: map[string]any{"view_count": 3000000}},
		{platform: "youtube", mediaType: "video", title: "How I'd Learn Web Dev in 2024", creator: "Theo", consumedAt: ago(305, 13), duration: d(20), timeSpent: d(20), url: "https://youtube.com/watch?v=v034", externalID: "yt-034", rawMetadata: map[string]any{"view_count": 900000}},
		{platform: "youtube", mediaType: "video", title: "Systems Design Explained in 15 Minutes", creator: "Fireship", consumedAt: ago(292, 8), duration: d(15), timeSpent: d(15), url: "https://youtube.com/watch?v=v035", externalID: "yt-035", rawMetadata: map[string]any{"view_count": 2000000}},
		{platform: "youtube", mediaType: "video", title: "CSS Container Queries Are Finally Here", creator: "Kevin Powell", consumedAt: ago(278, 21), duration: d(22), timeSpent: d(22), url: "https://youtube.com/watch?v=v036", externalID: "yt-036", rawMetadata: map[string]any{"view_count": 400000}},
		{platform: "youtube", mediaType: "video", title: "Refactoring a Go Codebase", creator: "Go Team", consumedAt: ago(265, 5), duration: d(40), timeSpent: d(35), url: "https://youtube.com/watch?v=v037", externalID: "yt-037", rawMetadata: map[string]any{"view_count": 200000}},
		{platform: "youtube", mediaType: "video", title: "Typography for Developers", creator: "Google Design", consumedAt: ago(252, 17), duration: d(28), timeSpent: d(28), url: "https://youtube.com/watch?v=v038", externalID: "yt-038", rawMetadata: map[string]any{"view_count": 600000}},
		{platform: "youtube", mediaType: "video", title: "Every Noise at Once - Genre Map Explorer", creator: "The Pudding", consumedAt: ago(245, 2), duration: d(12), timeSpent: d(12), url: "https://youtube.com/watch?v=v039", externalID: "yt-039", rawMetadata: map[string]any{"view_count": 650000}},

		// ============================================================
		// VIDEO ERA 3: Philosophy/Contemplative (months 8-0 ago, days ~240-0)
		// Tags: philosophy, psychology, contemplative, dreamy, history
		// ~20 items
		// ============================================================
		{platform: "youtube", mediaType: "video", title: "Justice: What's The Right Thing To Do? Episode 1", creator: "Harvard", consumedAt: ago(235, 10), duration: d(55), timeSpent: d(55), url: "https://youtube.com/watch?v=v040", externalID: "yt-040", rawMetadata: map[string]any{"view_count": 15000000}},
		{platform: "youtube", mediaType: "video", title: "The Philosophy of Stoicism", creator: "TED-Ed", consumedAt: ago(222, 3), duration: d(6), timeSpent: d(6), url: "https://youtube.com/watch?v=v041", externalID: "yt-041", rawMetadata: map[string]any{"view_count": 8000000}},
		{platform: "youtube", mediaType: "video", title: "Albert Camus - The Absurd", creator: "Academy of Ideas", consumedAt: ago(210, 18), duration: d(12), timeSpent: d(12), url: "https://youtube.com/watch?v=v042", externalID: "yt-042", rawMetadata: map[string]any{"view_count": 3000000}},
		{platform: "youtube", mediaType: "video", title: "Nietzsche - How To Find Yourself", creator: "Academy of Ideas", consumedAt: ago(198, 7), duration: d(14), timeSpent: d(14), url: "https://youtube.com/watch?v=v043", externalID: "yt-043", rawMetadata: map[string]any{"view_count": 4000000}},
		{platform: "youtube", mediaType: "video", title: "The Paradox of Choice", creator: "TED", consumedAt: ago(185, 22), duration: d(20), timeSpent: d(20), url: "https://youtube.com/watch?v=v044", externalID: "yt-044", rawMetadata: map[string]any{"view_count": 7000000}},
		{platform: "youtube", mediaType: "video", title: "Kierkegaard: The First Existentialist", creator: "Einzelganger", consumedAt: ago(172, 1), duration: d(15), timeSpent: d(15), url: "https://youtube.com/watch?v=v045", externalID: "yt-045", rawMetadata: map[string]any{"view_count": 1200000}},
		{platform: "youtube", mediaType: "video", title: "How Your Brain Is Getting Hacked", creator: "After Skool", consumedAt: ago(160, 14), duration: d(12), timeSpent: d(12), url: "https://youtube.com/watch?v=v046", externalID: "yt-046", rawMetadata: map[string]any{"view_count": 5000000}},
		{platform: "youtube", mediaType: "video", title: "Plato's Allegory of the Cave", creator: "TED-Ed", consumedAt: ago(148, 8), duration: d(5), timeSpent: d(5), url: "https://youtube.com/watch?v=v047", externalID: "yt-047", rawMetadata: map[string]any{"view_count": 9000000}},
		{platform: "youtube", mediaType: "video", title: "The Psychology of Persuasion", creator: "Psych2Go", consumedAt: ago(135, 20), duration: d(10), timeSpent: d(10), url: "https://youtube.com/watch?v=v048", externalID: "yt-048", rawMetadata: map[string]any{"view_count": 2500000}},
		{platform: "youtube", mediaType: "video", title: "What Makes Life Meaningful", creator: "Kurzgesagt", consumedAt: ago(122, 5), duration: d(10), timeSpent: d(10), url: "https://youtube.com/watch?v=v049", externalID: "yt-049", rawMetadata: map[string]any{"channel_id": "UCsXVk37bltHxD1rDPwtNM8Q", "view_count": 18000000}},
		{platform: "youtube", mediaType: "video", title: "The Fall of Rome Explained", creator: "OverSimplified", consumedAt: ago(110, 16), duration: d(20), timeSpent: d(20), url: "https://youtube.com/watch?v=v050", externalID: "yt-050", rawMetadata: map[string]any{"view_count": 25000000}},
		{platform: "youtube", mediaType: "video", title: "Simone de Beauvoir and The Ethics of Ambiguity", creator: "Einzelganger", consumedAt: ago(98, 0), duration: d(14), timeSpent: d(14), url: "https://youtube.com/watch?v=v051", externalID: "yt-051", rawMetadata: map[string]any{"view_count": 800000}},
		{platform: "youtube", mediaType: "video", title: "The Meaning of Knowledge", creator: "Crash Course", consumedAt: ago(85, 12), duration: d(10), timeSpent: d(10), url: "https://youtube.com/watch?v=v052", externalID: "yt-052", rawMetadata: map[string]any{"view_count": 3500000}},
		{platform: "youtube", mediaType: "video", title: "Why Socrates Hated Democracy", creator: "Academy of Ideas", consumedAt: ago(72, 19), duration: d(10), timeSpent: d(10), url: "https://youtube.com/watch?v=v053", externalID: "yt-053", rawMetadata: map[string]any{"view_count": 6000000}},
		{platform: "youtube", mediaType: "video", title: "Consciousness: Crash Course Philosophy", creator: "Crash Course", consumedAt: ago(58, 4), duration: d(10), timeSpent: d(10), url: "https://youtube.com/watch?v=v054", externalID: "yt-054", rawMetadata: map[string]any{"view_count": 4000000}},
		{platform: "youtube", mediaType: "video", title: "Wittgenstein on Language Games", creator: "Academy of Ideas", consumedAt: ago(45, 15), duration: d(12), timeSpent: d(12), url: "https://youtube.com/watch?v=v055", externalID: "yt-055", rawMetadata: map[string]any{"view_count": 1500000}},
		{platform: "youtube", mediaType: "video", title: "The Ancient Greeks and Western Civilization", creator: "Yale Courses", consumedAt: ago(32, 7), duration: d(50), timeSpent: d(40), url: "https://youtube.com/watch?v=v056", externalID: "yt-056", rawMetadata: map[string]any{"view_count": 900000}},
		{platform: "youtube", mediaType: "video", title: "How to Think Like a Philosopher", creator: "TED", consumedAt: ago(18, 21), duration: d(15), timeSpent: d(15), url: "https://youtube.com/watch?v=v057", externalID: "yt-057", rawMetadata: map[string]any{"view_count": 2000000}},
		{platform: "youtube", mediaType: "video", title: "The History of Philosophy in 16 Questions", creator: "Einzelganger", consumedAt: ago(5, 2), duration: d(18), timeSpent: d(18), url: "https://youtube.com/watch?v=v058", externalID: "yt-058", rawMetadata: map[string]any{"view_count": 700000}},

		// ============================================================
		// PODCAST ERA 1: Tech/Business (months 24-12 ago, days ~730-365)
		// Tags: technology, business, ai, serious, interview
		// ~13 items
		// ============================================================
		{platform: "youtube", mediaType: "podcast", title: "Lex Fridman Podcast #400 - Elon Musk", creator: "Lex Fridman", consumedAt: ago(720, 0), duration: d(180), timeSpent: d(120), url: "https://youtube.com/watch?v=p001", externalID: "yt-059", rawMetadata: map[string]any{"channel_id": "UCSHZKJJfhK61718lbao9Z2g"}},
		{platform: "youtube", mediaType: "podcast", title: "Lex Fridman Podcast #367 - Sam Altman", creator: "Lex Fridman", consumedAt: ago(700, 10), duration: d(150), timeSpent: d(150), url: "https://youtube.com/watch?v=p002", externalID: "yt-060", rawMetadata: map[string]any{"channel_id": "UCSHZKJJfhK61718lbao9Z2g"}},
		{platform: "youtube", mediaType: "podcast", title: "All-In Podcast E145: AI Revolution", creator: "All-In Podcast", consumedAt: ago(680, 5), duration: d(90), timeSpent: d(90), url: "https://youtube.com/watch?v=p003", externalID: "yt-061", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Dwarkesh Podcast: Patrick Collison", creator: "Dwarkesh Patel", consumedAt: ago(660, 19), duration: d(120), timeSpent: d(120), url: "https://youtube.com/watch?v=p004", externalID: "yt-062", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Acquired: NVIDIA", creator: "Acquired Podcast", consumedAt: ago(640, 3), duration: d(180), timeSpent: d(150), url: "https://youtube.com/watch?v=p005", externalID: "yt-063", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "My First Million: How to Build a SaaS", creator: "My First Million", consumedAt: ago(618, 14), duration: d(60), timeSpent: d(60), url: "https://youtube.com/watch?v=p006", externalID: "yt-064", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Lex Fridman Podcast #383 - Mark Zuckerberg", creator: "Lex Fridman", consumedAt: ago(598, 8), duration: d(160), timeSpent: d(130), url: "https://youtube.com/watch?v=p007", externalID: "yt-065", rawMetadata: map[string]any{"channel_id": "UCSHZKJJfhK61718lbao9Z2g"}},
		{platform: "youtube", mediaType: "podcast", title: "Dwarkesh Podcast: Tyler Cowen", creator: "Dwarkesh Patel", consumedAt: ago(575, 22), duration: d(90), timeSpent: d(90), url: "https://youtube.com/watch?v=p008", externalID: "yt-066", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "All-In Podcast E160: State of Startups", creator: "All-In Podcast", consumedAt: ago(555, 6), duration: d(100), timeSpent: d(100), url: "https://youtube.com/watch?v=p009", externalID: "yt-067", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Acquired: TSMC", creator: "Acquired Podcast", consumedAt: ago(530, 17), duration: d(200), timeSpent: d(170), url: "https://youtube.com/watch?v=p010", externalID: "yt-068", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Y Combinator: How to Start a Startup", creator: "Y Combinator", consumedAt: ago(505, 1), duration: d(60), timeSpent: d(60), url: "https://youtube.com/watch?v=p011", externalID: "yt-069", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "a]6z Podcast: The AI Infrastructure Stack", creator: "a16z", consumedAt: ago(480, 12), duration: d(45), timeSpent: d(45), url: "https://youtube.com/watch?v=p012", externalID: "yt-070", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Lex Fridman Podcast #418 - Demis Hassabis", creator: "Lex Fridman", consumedAt: ago(455, 0), duration: d(140), timeSpent: d(140), url: "https://youtube.com/watch?v=p013", externalID: "yt-071", rawMetadata: map[string]any{"channel_id": "UCSHZKJJfhK61718lbao9Z2g"}},

		// ============================================================
		// PODCAST ERA 2: Culture/Philosophy (months 12-0 ago, days ~365-0)
		// Tags: philosophy, psychology, spirituality, contemplative, history
		// ~12 items
		// ============================================================
		{platform: "youtube", mediaType: "podcast", title: "Huberman Lab: Science of Meditation", creator: "Andrew Huberman", consumedAt: ago(355, 15), duration: d(120), timeSpent: d(100), url: "https://youtube.com/watch?v=p014", externalID: "yt-072", rawMetadata: map[string]any{"channel_id": "UC2D2CMWXMOVWx7giW1n3LIg"}},
		{platform: "youtube", mediaType: "podcast", title: "Making Sense #300: The Nature of Consciousness", creator: "Sam Harris", consumedAt: ago(335, 7), duration: d(90), timeSpent: d(90), url: "https://youtube.com/watch?v=p015", externalID: "yt-073", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Philosophize This! Simone Weil", creator: "Stephen West", consumedAt: ago(310, 20), duration: d(35), timeSpent: d(35), url: "https://youtube.com/watch?v=p016", externalID: "yt-074", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Huberman Lab: Sleep Toolkit", creator: "Andrew Huberman", consumedAt: ago(288, 2), duration: d(120), timeSpent: d(90), url: "https://youtube.com/watch?v=p017", externalID: "yt-075", rawMetadata: map[string]any{"channel_id": "UC2D2CMWXMOVWx7giW1n3LIg"}},
		{platform: "youtube", mediaType: "podcast", title: "Making Sense #312: Free Will Revisited", creator: "Sam Harris", consumedAt: ago(265, 14), duration: d(80), timeSpent: d(80), url: "https://youtube.com/watch?v=p018", externalID: "yt-076", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "On Being: Krista Tippett with Esther Perel", creator: "On Being", consumedAt: ago(242, 8), duration: d(60), timeSpent: d(60), url: "https://youtube.com/watch?v=p019", externalID: "yt-077", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Philosophize This! Hannah Arendt", creator: "Stephen West", consumedAt: ago(218, 21), duration: d(40), timeSpent: d(40), url: "https://youtube.com/watch?v=p020", externalID: "yt-078", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Hardcore History: Wrath of the Khans", creator: "Dan Carlin", consumedAt: ago(192, 5), duration: d(240), timeSpent: d(180), url: "https://youtube.com/watch?v=p021", externalID: "yt-079", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Making Sense #325: Moral Landscapes", creator: "Sam Harris", consumedAt: ago(165, 16), duration: d(70), timeSpent: d(70), url: "https://youtube.com/watch?v=p022", externalID: "yt-080", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Huberman Lab: How to Focus", creator: "Andrew Huberman", consumedAt: ago(130, 3), duration: d(110), timeSpent: d(90), url: "https://youtube.com/watch?v=p023", externalID: "yt-081", rawMetadata: map[string]any{"channel_id": "UC2D2CMWXMOVWx7giW1n3LIg"}},
		{platform: "youtube", mediaType: "podcast", title: "Hardcore History: The Celtic Holocaust", creator: "Dan Carlin", consumedAt: ago(90, 11), duration: d(360), timeSpent: d(200), url: "https://youtube.com/watch?v=p024", externalID: "yt-082", rawMetadata: map[string]any{}},
		{platform: "youtube", mediaType: "podcast", title: "Philosophize This! Foucault on Power", creator: "Stephen West", consumedAt: ago(55, 0), duration: d(45), timeSpent: d(45), url: "https://youtube.com/watch?v=p025", externalID: "yt-083", rawMetadata: map[string]any{}},

		// ============================================================
		// NETFLIX: Movies & TV Shows (~20 items, type "video")
		// Spread across the full 2-year range (days ~730-0)
		// ============================================================
		{platform: "netflix", mediaType: "video", title: "Breaking Bad - S1E1: Pilot", creator: "Vince Gilligan", consumedAt: ago(722, 20), duration: d(58), timeSpent: d(58), url: "https://www.netflix.com/title/70143836", externalID: "nf-001", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Breaking Bad - S1E2: Cat's in the Bag...", creator: "Vince Gilligan", consumedAt: ago(721, 21), duration: d(48), timeSpent: d(48), url: "https://www.netflix.com/title/70143836", externalID: "nf-002", rawMetadata: map[string]any{"season": 1, "episode": 2, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Stranger Things - S1E1: The Vanishing of Will Byers", creator: "The Duffer Brothers", consumedAt: ago(690, 18), duration: d(49), timeSpent: d(49), url: "https://www.netflix.com/title/80057281", externalID: "nf-003", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-14"}},
		{platform: "netflix", mediaType: "video", title: "Stranger Things - S1E2: The Weirdo on Maple Street", creator: "The Duffer Brothers", consumedAt: ago(689, 22), duration: d(56), timeSpent: d(56), url: "https://www.netflix.com/title/80057281", externalID: "nf-004", rawMetadata: map[string]any{"season": 1, "episode": 2, "rating": "TV-14"}},
		{platform: "netflix", mediaType: "video", title: "The Crown - S1E1: Wolferton Splash", creator: "Peter Morgan", consumedAt: ago(635, 10), duration: d(57), timeSpent: d(57), url: "https://www.netflix.com/title/80025678", externalID: "nf-005", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Black Mirror - S1E1: The National Anthem", creator: "Charlie Brooker", consumedAt: ago(580, 14), duration: d(44), timeSpent: d(44), url: "https://www.netflix.com/title/70264888", externalID: "nf-006", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Black Mirror - S3E4: San Junipero", creator: "Charlie Brooker", consumedAt: ago(578, 20), duration: d(62), timeSpent: d(62), url: "https://www.netflix.com/title/70264888", externalID: "nf-007", rawMetadata: map[string]any{"season": 3, "episode": 4, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "The Queen's Gambit - E1: Openings", creator: "Scott Frank", consumedAt: ago(520, 9), duration: d(56), timeSpent: d(56), url: "https://www.netflix.com/title/80234304", externalID: "nf-008", rawMetadata: map[string]any{"episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Dark - S1E1: Secrets", creator: "Baran bo Odar", consumedAt: ago(470, 3), duration: d(52), timeSpent: d(52), url: "https://www.netflix.com/title/80100172", externalID: "nf-009", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Mindhunter - S1E1: Episode 1", creator: "David Fincher", consumedAt: ago(432, 16), duration: d(50), timeSpent: d(50), url: "https://www.netflix.com/title/80114855", externalID: "nf-010", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Narcos - S1E1: Descenso", creator: "Chris Brancato", consumedAt: ago(388, 7), duration: d(49), timeSpent: d(49), url: "https://www.netflix.com/title/80025172", externalID: "nf-011", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "The Social Dilemma", creator: "Jeff Orlowski", consumedAt: ago(345, 12), duration: d(94), timeSpent: d(94), url: "https://www.netflix.com/title/81254224", externalID: "nf-012", rawMetadata: map[string]any{"rating": "PG-13", "type": "documentary"}},
		{platform: "netflix", mediaType: "video", title: "Don't Look Up", creator: "Adam McKay", consumedAt: ago(310, 21), duration: d(138), timeSpent: d(138), url: "https://www.netflix.com/title/81252357", externalID: "nf-013", rawMetadata: map[string]any{"rating": "R", "type": "film"}},
		{platform: "netflix", mediaType: "video", title: "Squid Game - S1E1: Red Light, Green Light", creator: "Hwang Dong-hyuk", consumedAt: ago(265, 4), duration: d(60), timeSpent: d(60), url: "https://www.netflix.com/title/81040344", externalID: "nf-014", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Ozark - S1E1: Sugarwood", creator: "Bill Dubuque", consumedAt: ago(218, 15), duration: d(60), timeSpent: d(60), url: "https://www.netflix.com/title/80117552", externalID: "nf-015", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Glass Onion: A Knives Out Mystery", creator: "Rian Johnson", consumedAt: ago(175, 19), duration: d(139), timeSpent: d(139), url: "https://www.netflix.com/title/81458416", externalID: "nf-016", rawMetadata: map[string]any{"rating": "PG-13", "type": "film"}},
		{platform: "netflix", mediaType: "video", title: "Beef - S1E1: The Birds Don't Sing, They Screech in Pain", creator: "Lee Sung Jin", consumedAt: ago(130, 8), duration: d(36), timeSpent: d(36), url: "https://www.netflix.com/title/81447461", externalID: "nf-017", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "The Watcher - S1E1: Welcome, Friends", creator: "Ryan Murphy", consumedAt: ago(82, 2), duration: d(48), timeSpent: d(48), url: "https://www.netflix.com/title/81008806", externalID: "nf-018", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-MA"}},
		{platform: "netflix", mediaType: "video", title: "Wednesday - S1E1: Wednesday's Child Is Full of Woe", creator: "Tim Burton", consumedAt: ago(42, 11), duration: d(50), timeSpent: d(50), url: "https://www.netflix.com/title/81231974", externalID: "nf-019", rawMetadata: map[string]any{"season": 1, "episode": 1, "rating": "TV-14"}},
		{platform: "netflix", mediaType: "video", title: "All Quiet on the Western Front", creator: "Edward Berger", consumedAt: ago(8, 17), duration: d(148), timeSpent: d(148), url: "https://www.netflix.com/title/81260280", externalID: "nf-020", rawMetadata: map[string]any{"rating": "R", "type": "film"}},

		// ============================================================
		// TIKTOK: Short Videos (~15 items, type "video")
		// Durations 1-3 minutes. Spread across the 2-year range.
		// ============================================================
		{platform: "tiktok", mediaType: "video", title: "POV: When the beat drops", creator: "@musicvibes", consumedAt: ago(715, 8), duration: d(1), timeSpent: d(1), url: "https://tiktok.com/@musicvibes/video/7001", externalID: "tt-001", rawMetadata: map[string]any{"likes": 450000, "shares": 12000}},
		{platform: "tiktok", mediaType: "video", title: "3 coding tricks you didn't know", creator: "@devtips", consumedAt: ago(665, 13), duration: d(2), timeSpent: d(2), url: "https://tiktok.com/@devtips/video/7002", externalID: "tt-002", rawMetadata: map[string]any{"likes": 320000, "shares": 8500}},
		{platform: "tiktok", mediaType: "video", title: "This philosophy will change your life", creator: "@deepthoughts", consumedAt: ago(620, 5), duration: d(3), timeSpent: d(3), url: "https://tiktok.com/@deepthoughts/video/7003", externalID: "tt-003", rawMetadata: map[string]any{"likes": 890000, "shares": 45000}},
		{platform: "tiktok", mediaType: "video", title: "Making ramen from scratch", creator: "@chefkai", consumedAt: ago(585, 19), duration: d(2), timeSpent: d(2), url: "https://tiktok.com/@chefkai/video/7004", externalID: "tt-004", rawMetadata: map[string]any{"likes": 1200000, "shares": 67000}},
		{platform: "tiktok", mediaType: "video", title: "The math behind music explained", creator: "@sciencefacts", consumedAt: ago(540, 0), duration: d(3), timeSpent: d(3), url: "https://tiktok.com/@sciencefacts/video/7005", externalID: "tt-005", rawMetadata: map[string]any{"likes": 560000, "shares": 23000}},
		{platform: "tiktok", mediaType: "video", title: "Interior design tips for small spaces", creator: "@homestyle", consumedAt: ago(495, 16), duration: d(1), timeSpent: d(1), url: "https://tiktok.com/@homestyle/video/7006", externalID: "tt-006", rawMetadata: map[string]any{"likes": 780000, "shares": 34000}},
		{platform: "tiktok", mediaType: "video", title: "Day in the life of a software engineer", creator: "@techlife", consumedAt: ago(450, 10), duration: d(3), timeSpent: d(3), url: "https://tiktok.com/@techlife/video/7007", externalID: "tt-007", rawMetadata: map[string]any{"likes": 920000, "shares": 41000}},
		{platform: "tiktok", mediaType: "video", title: "Why this song is a masterpiece", creator: "@musicbreakdown", consumedAt: ago(398, 22), duration: d(2), timeSpent: d(2), url: "https://tiktok.com/@musicbreakdown/video/7008", externalID: "tt-008", rawMetadata: map[string]any{"likes": 670000, "shares": 28000}},
		{platform: "tiktok", mediaType: "video", title: "Historical facts they don't teach you", creator: "@historynerd", consumedAt: ago(350, 6), duration: d(2), timeSpent: d(2), url: "https://tiktok.com/@historynerd/video/7009", externalID: "tt-009", rawMetadata: map[string]any{"likes": 1500000, "shares": 89000}},
		{platform: "tiktok", mediaType: "video", title: "Sunset timelapse in Iceland", creator: "@travelgram", consumedAt: ago(295, 14), duration: d(1), timeSpent: d(1), url: "https://tiktok.com/@travelgram/video/7010", externalID: "tt-010", rawMetadata: map[string]any{"likes": 2100000, "shares": 120000}},
		{platform: "tiktok", mediaType: "video", title: "How to learn any language in 6 months", creator: "@polyglotlife", consumedAt: ago(248, 3), duration: d(3), timeSpent: d(3), url: "https://tiktok.com/@polyglotlife/video/7011", externalID: "tt-011", rawMetadata: map[string]any{"likes": 430000, "shares": 19000}},
		{platform: "tiktok", mediaType: "video", title: "AI art is getting insane", creator: "@futuretech", consumedAt: ago(195, 17), duration: d(1), timeSpent: d(1), url: "https://tiktok.com/@futuretech/video/7012", externalID: "tt-012", rawMetadata: map[string]any{"likes": 1800000, "shares": 95000}},
		{platform: "tiktok", mediaType: "video", title: "Workout routine that actually works", creator: "@fitcoach", consumedAt: ago(140, 9), duration: d(2), timeSpent: d(2), url: "https://tiktok.com/@fitcoach/video/7013", externalID: "tt-013", rawMetadata: map[string]any{"likes": 340000, "shares": 15000}},
		{platform: "tiktok", mediaType: "video", title: "The psychology of color in film", creator: "@filmanalysis", consumedAt: ago(78, 20), duration: d(3), timeSpent: d(3), url: "https://tiktok.com/@filmanalysis/video/7014", externalID: "tt-014", rawMetadata: map[string]any{"likes": 710000, "shares": 32000}},
		{platform: "tiktok", mediaType: "video", title: "Making espresso the Italian way", creator: "@coffeeculture", consumedAt: ago(20, 12), duration: d(2), timeSpent: d(2), url: "https://tiktok.com/@coffeeculture/video/7015", externalID: "tt-015", rawMetadata: map[string]any{"likes": 520000, "shares": 21000}},

		// ============================================================
		// GOODREADS: Books (~15 items, type "book")
		// Spread across the 2-year range.
		// ============================================================
		{platform: "goodreads", mediaType: "book", title: "Dune", creator: "Frank Herbert", consumedAt: ago(720, 11), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/44767458", externalID: "gr-001", rawMetadata: map[string]any{"isbn": "9780441172719", "pages": 688, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Project Hail Mary", creator: "Andy Weir", consumedAt: ago(680, 6), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/54493401", externalID: "gr-002", rawMetadata: map[string]any{"isbn": "9780593135204", "pages": 476, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "The Stranger", creator: "Albert Camus", consumedAt: ago(645, 17), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/49552", externalID: "gr-003", rawMetadata: map[string]any{"isbn": "9780679720201", "pages": 123, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Sapiens: A Brief History of Humankind", creator: "Yuval Noah Harari", consumedAt: ago(605, 2), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/23692271", externalID: "gr-004", rawMetadata: map[string]any{"isbn": "9780062316097", "pages": 443, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Neuromancer", creator: "William Gibson", consumedAt: ago(562, 21), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/6088007", externalID: "gr-005", rawMetadata: map[string]any{"isbn": "9780441569595", "pages": 271, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Man's Search for Meaning", creator: "Viktor E. Frankl", consumedAt: ago(525, 8), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/4069", externalID: "gr-006", rawMetadata: map[string]any{"isbn": "9780807014295", "pages": 184, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "The Left Hand of Darkness", creator: "Ursula K. Le Guin", consumedAt: ago(488, 15), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/18423", externalID: "gr-007", rawMetadata: map[string]any{"isbn": "9780441478125", "pages": 304, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Thinking, Fast and Slow", creator: "Daniel Kahneman", consumedAt: ago(440, 4), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/11468377", externalID: "gr-008", rawMetadata: map[string]any{"isbn": "9780374533557", "pages": 499, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "The Brothers Karamazov", creator: "Fyodor Dostoevsky", consumedAt: ago(392, 19), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/4934", externalID: "gr-009", rawMetadata: map[string]any{"isbn": "9780374528379", "pages": 796, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "A Philosophy of Software Design", creator: "John Ousterhout", consumedAt: ago(348, 7), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/39996759", externalID: "gr-010", rawMetadata: map[string]any{"isbn": "9781732102200", "pages": 190, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "The Myth of Sisyphus", creator: "Albert Camus", consumedAt: ago(298, 13), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/91950", externalID: "gr-011", rawMetadata: map[string]any{"isbn": "9780525564454", "pages": 212, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Designing Data-Intensive Applications", creator: "Martin Kleppmann", consumedAt: ago(245, 0), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/23463279", externalID: "gr-012", rawMetadata: map[string]any{"isbn": "9781449373320", "pages": 616, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Norwegian Wood", creator: "Haruki Murakami", consumedAt: ago(188, 16), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/11297", externalID: "gr-013", rawMetadata: map[string]any{"isbn": "9780375704024", "pages": 296, "shelf": "read"}},
		{platform: "goodreads", mediaType: "book", title: "Meditations", creator: "Marcus Aurelius", consumedAt: ago(115, 5), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/30659", externalID: "gr-014", rawMetadata: map[string]any{"isbn": "9780140449334", "pages": 254, "shelf": "currently-reading"}},
		{platform: "goodreads", mediaType: "book", title: "Kafka on the Shore", creator: "Haruki Murakami", consumedAt: ago(28, 22), duration: nil, timeSpent: nil, url: "https://www.goodreads.com/book/show/4929", externalID: "gr-015", rawMetadata: map[string]any{"isbn": "9781400079278", "pages": 467, "shelf": "currently-reading"}},

		// ============================================================
		// ANILIST: Anime (~10 items, type "video") & Manga (~5 items, type "book")
		// Spread across the 2-year range.
		// ============================================================

		// --- AniList: Anime (video) ---
		{platform: "anilist", mediaType: "video", title: "Steins;Gate - Episode 1: Turning Point", creator: "White Fox", consumedAt: ago(710, 7), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/9253", externalID: "al-001", rawMetadata: map[string]any{"mal_id": 9253, "format": "TV", "episodes": 24}},
		{platform: "anilist", mediaType: "video", title: "Attack on Titan - S1E1: To You, in 2000 Years", creator: "Wit Studio", consumedAt: ago(668, 18), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/16498", externalID: "al-002", rawMetadata: map[string]any{"mal_id": 16498, "format": "TV", "episodes": 25}},
		{platform: "anilist", mediaType: "video", title: "Neon Genesis Evangelion - Episode 1: Angel Attack", creator: "Gainax", consumedAt: ago(625, 4), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/30", externalID: "al-003", rawMetadata: map[string]any{"mal_id": 30, "format": "TV", "episodes": 26}},
		{platform: "anilist", mediaType: "video", title: "Spirited Away", creator: "Studio Ghibli", consumedAt: ago(572, 12), duration: d(125), timeSpent: d(125), url: "https://anilist.co/anime/199", externalID: "al-004", rawMetadata: map[string]any{"mal_id": 199, "format": "MOVIE"}},
		{platform: "anilist", mediaType: "video", title: "Cowboy Bebop - Session 1: Asteroid Blues", creator: "Sunrise", consumedAt: ago(518, 23), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/1", externalID: "al-005", rawMetadata: map[string]any{"mal_id": 1, "format": "TV", "episodes": 26}},
		{platform: "anilist", mediaType: "video", title: "Mob Psycho 100 - Episode 1: Self-Proclaimed Psychic", creator: "Bones", consumedAt: ago(462, 6), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/21507", externalID: "al-006", rawMetadata: map[string]any{"mal_id": 21507, "format": "TV", "episodes": 12}},
		{platform: "anilist", mediaType: "video", title: "Your Name", creator: "CoMix Wave Films", consumedAt: ago(408, 15), duration: d(106), timeSpent: d(106), url: "https://anilist.co/anime/21519", externalID: "al-007", rawMetadata: map[string]any{"mal_id": 21519, "format": "MOVIE"}},
		{platform: "anilist", mediaType: "video", title: "Vinland Saga - Episode 1: Somewhere Not Here", creator: "Wit Studio", consumedAt: ago(335, 1), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/101348", externalID: "al-008", rawMetadata: map[string]any{"mal_id": 37521, "format": "TV", "episodes": 24}},
		{platform: "anilist", mediaType: "video", title: "Perfect Blue", creator: "Madhouse", consumedAt: ago(228, 20), duration: d(81), timeSpent: d(81), url: "https://anilist.co/anime/437", externalID: "al-009", rawMetadata: map[string]any{"mal_id": 437, "format": "MOVIE"}},
		{platform: "anilist", mediaType: "video", title: "Monster - Episode 1: Herr Dr. Tenma", creator: "Madhouse", consumedAt: ago(155, 9), duration: d(24), timeSpent: d(24), url: "https://anilist.co/anime/19", externalID: "al-010", rawMetadata: map[string]any{"mal_id": 19, "format": "TV", "episodes": 74}},

		// --- AniList: Manga (book) ---
		{platform: "anilist", mediaType: "book", title: "Berserk - Chapter 1: The Black Swordsman", creator: "Kentaro Miura", consumedAt: ago(700, 3), duration: nil, timeSpent: nil, url: "https://anilist.co/manga/30002", externalID: "al-011", rawMetadata: map[string]any{"mal_id": 2, "format": "MANGA", "chapters": 364}},
		{platform: "anilist", mediaType: "book", title: "Vagabond - Chapter 1: Shinmen Takezo", creator: "Takehiko Inoue", consumedAt: ago(558, 11), duration: nil, timeSpent: nil, url: "https://anilist.co/manga/30656", externalID: "al-012", rawMetadata: map[string]any{"mal_id": 656, "format": "MANGA", "chapters": 327}},
		{platform: "anilist", mediaType: "book", title: "Chainsaw Man - Chapter 1: Dog & Chainsaw", creator: "Tatsuki Fujimoto", consumedAt: ago(415, 14), duration: nil, timeSpent: nil, url: "https://anilist.co/manga/105778", externalID: "al-013", rawMetadata: map[string]any{"mal_id": 116778, "format": "MANGA", "chapters": 97}},
		{platform: "anilist", mediaType: "book", title: "Oyasumi Punpun - Chapter 1", creator: "Inio Asano", consumedAt: ago(280, 2), duration: nil, timeSpent: nil, url: "https://anilist.co/manga/34632", externalID: "al-014", rawMetadata: map[string]any{"mal_id": 4632, "format": "MANGA", "chapters": 147}},
		{platform: "anilist", mediaType: "book", title: "Pluto - Chapter 1: Act 01", creator: "Naoki Urasawa", consumedAt: ago(105, 18), duration: nil, timeSpent: nil, url: "https://anilist.co/manga/30745", externalID: "al-015", rawMetadata: map[string]any{"mal_id": 745, "format": "MANGA", "chapters": 65}},
	}
}

type seedTag struct {
	name       string
	category   string
	source     string
	confidence float32
}

func buildTagAssignments() map[string][]seedTag {
	llm := func(name, cat string, conf float32) seedTag {
		return seedTag{name: name, category: cat, source: "llm", confidence: conf}
	}
	api := func(name, cat string) seedTag {
		return seedTag{name: name, category: cat, source: "api", confidence: 0}
	}

	return map[string][]seedTag{
		// ============================================================
		// MUSIC ERA 1: Indie/Dreamy (sp-001 to sp-018, lf-001 to lf-007)
		// ============================================================
		"sp-001": {api("indie", "genre"), llm("indie", "genre", 0.92), llm("dreamy", "mood", 0.95), llm("nostalgic", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-002": {api("indie", "genre"), llm("indie", "genre", 0.90), llm("dreamy", "mood", 0.92), llm("melancholic", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-003": {api("indie", "genre"), api("pop", "genre"), llm("indie", "genre", 0.92), llm("dreamy", "mood", 0.88), llm("romantic", "mood", 0.75)},
		"sp-004": {api("indie", "genre"), api("electronic", "genre"), llm("indie", "genre", 0.90), llm("electronic", "genre", 0.85), llm("dreamy", "mood", 0.88), llm("playful", "mood", 0.75)},
		"sp-005": {api("electronic", "genre"), llm("electronic", "genre", 0.93), llm("indie", "genre", 0.80), llm("energetic", "mood", 0.85), llm("dreamy", "mood", 0.78)},
		"sp-006": {api("indie", "genre"), api("electronic", "genre"), llm("indie", "genre", 0.88), llm("chill", "mood", 0.90), llm("peaceful", "mood", 0.82)},
		"sp-007": {api("indie", "genre"), api("folk", "genre"), llm("indie", "genre", 0.92), llm("melancholic", "mood", 0.90), llm("peaceful", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-008": {api("indie", "genre"), llm("indie", "genre", 0.88), llm("alternative", "genre", 0.80), llm("dreamy", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-009": {api("indie", "genre"), api("folk", "genre"), llm("indie", "genre", 0.90), llm("peaceful", "mood", 0.88), llm("contemplative", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-010": {api("indie", "genre"), api("folk", "genre"), llm("indie", "genre", 0.90), llm("melancholic", "mood", 0.92), llm("peaceful", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-011": {api("indie", "genre"), api("folk", "genre"), llm("folk", "genre", 0.88), llm("peaceful", "mood", 0.85), llm("chill", "mood", 0.80)},
		"sp-012": {api("alternative", "genre"), api("indie", "genre"), llm("alternative", "genre", 0.88), llm("dreamy", "mood", 0.85), llm("chill", "mood", 0.82)},
		"sp-013": {api("indie", "genre"), api("alternative", "genre"), llm("indie", "genre", 0.90), llm("dreamy", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-014": {api("indie", "genre"), llm("indie", "genre", 0.92), llm("melancholic", "mood", 0.88), llm("dreamy", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-015": {api("indie", "genre"), api("alternative", "genre"), llm("alternative", "genre", 0.88), llm("playful", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-016": {api("indie", "genre"), api("alternative", "genre"), llm("indie", "genre", 0.90), llm("dreamy", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-017": {api("pop", "genre"), api("indie", "genre"), llm("indie", "genre", 0.85), llm("chill", "mood", 0.82), llm("single", "format", 0.9)},
		"sp-018": {api("indie", "genre"), api("electronic", "genre"), llm("indie", "genre", 0.90), llm("dreamy", "mood", 0.92), llm("eerie", "mood", 0.78), llm("album-track", "format", 0.9)},

		"lf-001": {api("ambient", "genre"), llm("ambient", "genre", 0.95), llm("dreamy", "mood", 0.92), llm("peaceful", "mood", 0.88), llm("eerie", "mood", 0.72)},
		"lf-002": {api("rock", "genre"), llm("rock", "genre", 0.90), llm("ambient", "genre", 0.78), llm("peaceful", "mood", 0.92), llm("dreamy", "mood", 0.85)},
		"lf-003": {api("electronic", "genre"), llm("electronic", "genre", 0.92), llm("dark", "mood", 0.88), llm("dreamy", "mood", 0.82), llm("eerie", "mood", 0.78)},
		"lf-004": {api("alternative", "genre"), api("electronic", "genre"), llm("alternative", "genre", 0.90), llm("contemplative", "mood", 0.85), llm("dreamy", "mood", 0.80)},
		"lf-005": {api("alternative", "genre"), api("rock", "genre"), llm("alternative", "genre", 0.92), llm("melancholic", "mood", 0.85), llm("contemplative", "mood", 0.80)},
		"lf-006": {api("alternative", "genre"), llm("alternative", "genre", 0.90), llm("contemplative", "mood", 0.88), llm("peaceful", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-007": {api("indie", "genre"), llm("indie", "genre", 0.90), llm("dreamy", "mood", 0.92), llm("melancholic", "mood", 0.85), llm("album-track", "format", 0.9)},

		// ============================================================
		// MUSIC ERA 2: Hip-Hop/Intense (sp-019 to sp-036, lf-008 to lf-014)
		// ============================================================
		"sp-019": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("contemplative", "mood", 0.82), llm("raw", "mood", 0.78), llm("album-track", "format", 0.9)},
		"sp-020": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("intense", "mood", 0.90), llm("dark", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-021": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("uplifting", "mood", 0.88), llm("triumphant", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-022": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("raw", "mood", 0.90), llm("contemplative", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-023": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.93), llm("aggressive", "mood", 0.88), llm("intense", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-024": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("aggressive", "mood", 0.90), llm("energetic", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-025": {api("hip-hop", "genre"), api("r-and-b", "genre"), llm("hip-hop", "genre", 0.90), llm("r-and-b", "genre", 0.78), llm("contemplative", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-026": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("aggressive", "mood", 0.92), llm("intense", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-027": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("aggressive", "mood", 0.90), llm("intense", "mood", 0.85), llm("single", "format", 0.95)},
		"sp-028": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.92), llm("raw", "mood", 0.85), llm("contemplative", "mood", 0.80), llm("album-track", "format", 0.9)},
		"sp-029": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("contemplative", "mood", 0.80), llm("chill", "mood", 0.78), llm("album-track", "format", 0.9)},
		"sp-030": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("intense", "mood", 0.90), llm("raw", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-031": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.93), llm("dark", "mood", 0.90), llm("intense", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-032": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.92), llm("raw", "mood", 0.85), llm("intense", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-033": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.90), llm("chill", "mood", 0.82), llm("nostalgic", "mood", 0.78), llm("album-track", "format", 0.9)},
		"sp-034": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.92), llm("dark", "mood", 0.85), llm("contemplative", "mood", 0.80), llm("album-track", "format", 0.9)},
		"sp-035": {api("hip-hop", "genre"), api("funk", "genre"), llm("hip-hop", "genre", 0.93), llm("funk", "genre", 0.80), llm("contemplative", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-036": {api("hip-hop", "genre"), api("pop", "genre"), llm("hip-hop", "genre", 0.90), llm("intense", "mood", 0.85), llm("energetic", "mood", 0.82), llm("album-track", "format", 0.9)},

		"lf-008": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.93), llm("aggressive", "mood", 0.90), llm("intense", "mood", 0.85), llm("album-track", "format", 0.9)},
		"lf-009": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("contemplative", "mood", 0.85), llm("raw", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-010": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.92), llm("uplifting", "mood", 0.85), llm("nostalgic", "mood", 0.80), llm("album-track", "format", 0.9)},
		"lf-011": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.93), llm("intense", "mood", 0.88), llm("raw", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-012": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.95), llm("dark", "mood", 0.85), llm("playful", "mood", 0.78), llm("album-track", "format", 0.9)},
		"lf-013": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.90), llm("chill", "mood", 0.88), llm("nostalgic", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-014": {api("hip-hop", "genre"), llm("hip-hop", "genre", 0.90), llm("chill", "mood", 0.85), llm("playful", "mood", 0.80), llm("album-track", "format", 0.9)},

		// ============================================================
		// MUSIC ERA 3: Electronic/Energetic (sp-037 to sp-054, lf-015 to lf-021)
		// ============================================================
		"sp-037": {api("electronic", "genre"), llm("electronic", "genre", 0.95), llm("energetic", "mood", 0.92), llm("uplifting", "mood", 0.88), llm("album-track", "format", 0.9)},
		"sp-038": {api("electronic", "genre"), llm("electronic", "genre", 0.95), llm("energetic", "mood", 0.90), llm("playful", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-039": {api("electronic", "genre"), llm("electronic", "genre", 0.93), llm("dreamy", "mood", 0.85), llm("contemplative", "mood", 0.78), llm("album-track", "format", 0.9)},
		"sp-040": {api("electronic", "genre"), api("pop", "genre"), llm("electronic", "genre", 0.90), llm("pop", "genre", 0.82), llm("energetic", "mood", 0.88), llm("single", "format", 0.95)},
		"sp-041": {api("electronic", "genre"), llm("electronic", "genre", 0.92), llm("uplifting", "mood", 0.92), llm("energetic", "mood", 0.90), llm("single", "format", 0.95)},
		"sp-042": {api("electronic", "genre"), api("funk", "genre"), llm("electronic", "genre", 0.90), llm("funk", "genre", 0.82), llm("energetic", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-043": {api("pop", "genre"), api("electronic", "genre"), llm("pop", "genre", 0.92), llm("electronic", "genre", 0.88), llm("energetic", "mood", 0.90), llm("single", "format", 0.95)},
		"sp-044": {api("pop", "genre"), api("electronic", "genre"), llm("pop", "genre", 0.90), llm("energetic", "mood", 0.88), llm("dark", "mood", 0.75)},
		"sp-045": {api("alternative", "genre"), api("electronic", "genre"), llm("alternative", "genre", 0.90), llm("dark", "mood", 0.85), llm("energetic", "mood", 0.80)},
		"sp-046": {api("electronic", "genre"), llm("electronic", "genre", 0.92), llm("romantic", "mood", 0.85), llm("chill", "mood", 0.80), llm("album-track", "format", 0.9)},
		"sp-047": {api("electronic", "genre"), llm("electronic", "genre", 0.93), llm("indie", "genre", 0.78), llm("energetic", "mood", 0.85), llm("dreamy", "mood", 0.75)},
		"sp-048": {api("electronic", "genre"), llm("electronic", "genre", 0.95), llm("energetic", "mood", 0.92), llm("playful", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-049": {api("electronic", "genre"), llm("electronic", "genre", 0.93), llm("uplifting", "mood", 0.88), llm("contemplative", "mood", 0.75), llm("album-track", "format", 0.9)},
		"sp-050": {api("electronic", "genre"), api("pop", "genre"), llm("electronic", "genre", 0.90), llm("energetic", "mood", 0.92), llm("playful", "mood", 0.85), llm("single", "format", 0.95)},
		"sp-051": {api("electronic", "genre"), llm("electronic", "genre", 0.92), llm("dreamy", "mood", 0.88), llm("contemplative", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-052": {api("electronic", "genre"), llm("electronic", "genre", 0.90), llm("uplifting", "mood", 0.88), llm("dreamy", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-053": {api("electronic", "genre"), api("pop", "genre"), llm("electronic", "genre", 0.92), llm("uplifting", "mood", 0.90), llm("energetic", "mood", 0.85), llm("single", "format", 0.95)},
		"sp-054": {api("electronic", "genre"), llm("electronic", "genre", 0.90), llm("romantic", "mood", 0.88), llm("chill", "mood", 0.82), llm("album-track", "format", 0.9)},

		"lf-015": {api("electronic", "genre"), llm("electronic", "genre", 0.92), llm("playful", "mood", 0.85), llm("energetic", "mood", 0.80), llm("album-track", "format", 0.9)},
		"lf-016": {api("electronic", "genre"), llm("electronic", "genre", 0.93), llm("energetic", "mood", 0.90), llm("intense", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-017": {api("electronic", "genre"), llm("electronic", "genre", 0.90), llm("chill", "mood", 0.88), llm("peaceful", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-018": {api("electronic", "genre"), llm("electronic", "genre", 0.92), llm("energetic", "mood", 0.88), llm("playful", "mood", 0.80), llm("album-track", "format", 0.9)},
		"lf-019": {api("electronic", "genre"), llm("electronic", "genre", 0.95), llm("aggressive", "mood", 0.90), llm("energetic", "mood", 0.88), llm("album-track", "format", 0.9)},
		"lf-020": {api("electronic", "genre"), llm("electronic", "genre", 0.90), llm("peaceful", "mood", 0.85), llm("dreamy", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-021": {api("electronic", "genre"), api("ambient", "genre"), llm("electronic", "genre", 0.90), llm("ambient", "genre", 0.82), llm("chill", "mood", 0.88), llm("dreamy", "mood", 0.82)},

		// ============================================================
		// MUSIC ERA 4: R&B/Melancholic (sp-055 to sp-073, lf-022 to lf-028)
		// ============================================================
		"sp-055": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.93), llm("contemplative", "mood", 0.90), llm("melancholic", "mood", 0.82), llm("album-track", "format", 0.95)},
		"sp-056": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.95), llm("melancholic", "mood", 0.92), llm("contemplative", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-057": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.90), llm("peaceful", "mood", 0.85), llm("dreamy", "mood", 0.80)},
		"sp-058": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.90), llm("nostalgic", "mood", 0.88), llm("romantic", "mood", 0.82)},
		"sp-059": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.93), llm("electronic", "genre", 0.75), llm("intense", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-060": {api("r-and-b", "genre"), api("funk", "genre"), llm("r-and-b", "genre", 0.92), llm("soul", "genre", 0.88), llm("nostalgic", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-061": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.92), llm("romantic", "mood", 0.90), llm("peaceful", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-062": {api("hip-hop", "genre"), api("r-and-b", "genre"), llm("r-and-b", "genre", 0.85), llm("contemplative", "mood", 0.88), llm("raw", "mood", 0.80), llm("album-track", "format", 0.9)},
		"sp-063": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.90), llm("romantic", "mood", 0.85), llm("chill", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-064": {api("r-and-b", "genre"), api("soul", "genre"), llm("r-and-b", "genre", 0.92), llm("soul", "genre", 0.85), llm("melancholic", "mood", 0.88), llm("contemplative", "mood", 0.82)},
		"sp-065": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.90), llm("raw", "mood", 0.85), llm("contemplative", "mood", 0.80), llm("album-track", "format", 0.9)},
		"sp-066": {api("r-and-b", "genre"), api("pop", "genre"), llm("r-and-b", "genre", 0.88), llm("uplifting", "mood", 0.85), llm("dreamy", "mood", 0.80), llm("single", "format", 0.95)},
		"sp-067": {api("r-and-b", "genre"), api("soul", "genre"), llm("r-and-b", "genre", 0.92), llm("soul", "genre", 0.88), llm("nostalgic", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-068": {api("r-and-b", "genre"), api("pop", "genre"), llm("r-and-b", "genre", 0.88), llm("chill", "mood", 0.82), llm("romantic", "mood", 0.78), llm("album-track", "format", 0.9)},
		"sp-069": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.90), llm("melancholic", "mood", 0.85), llm("contemplative", "mood", 0.80), llm("album-track", "format", 0.9)},
		"sp-070": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.92), llm("nostalgic", "mood", 0.88), llm("chill", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-071": {api("r-and-b", "genre"), llm("r-and-b", "genre", 0.93), llm("romantic", "mood", 0.90), llm("melancholic", "mood", 0.85), llm("album-track", "format", 0.9)},
		"sp-072": {api("r-and-b", "genre"), api("soul", "genre"), llm("soul", "genre", 0.92), llm("romantic", "mood", 0.88), llm("peaceful", "mood", 0.82), llm("album-track", "format", 0.9)},
		"sp-073": {api("r-and-b", "genre"), api("pop", "genre"), llm("r-and-b", "genre", 0.90), llm("melancholic", "mood", 0.88), llm("contemplative", "mood", 0.82), llm("album-track", "format", 0.9)},

		"lf-022": {api("r-and-b", "genre"), api("soul", "genre"), llm("soul", "genre", 0.93), llm("nostalgic", "mood", 0.90), llm("romantic", "mood", 0.85), llm("album-track", "format", 0.9)},
		"lf-023": {api("r-and-b", "genre"), api("soul", "genre"), llm("r-and-b", "genre", 0.90), llm("uplifting", "mood", 0.85), llm("peaceful", "mood", 0.80), llm("album-track", "format", 0.9)},
		"lf-024": {api("r-and-b", "genre"), api("pop", "genre"), llm("r-and-b", "genre", 0.88), llm("uplifting", "mood", 0.82), llm("nostalgic", "mood", 0.78), llm("album-track", "format", 0.9)},
		"lf-025": {api("r-and-b", "genre"), api("soul", "genre"), llm("soul", "genre", 0.92), llm("romantic", "mood", 0.90), llm("contemplative", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-026": {api("r-and-b", "genre"), api("soul", "genre"), llm("r-and-b", "genre", 0.92), llm("soul", "genre", 0.85), llm("chill", "mood", 0.82), llm("contemplative", "mood", 0.78)},
		"lf-027": {api("r-and-b", "genre"), api("soul", "genre"), llm("soul", "genre", 0.90), llm("romantic", "mood", 0.88), llm("nostalgic", "mood", 0.82), llm("album-track", "format", 0.9)},
		"lf-028": {api("r-and-b", "genre"), api("pop", "genre"), llm("r-and-b", "genre", 0.90), llm("energetic", "mood", 0.85), llm("nostalgic", "mood", 0.80), llm("album-track", "format", 0.9)},

		// ============================================================
		// VIDEO ERA 1: Science/Math (yt-001 to yt-019)
		// ============================================================
		"yt-001": {llm("science", "topic", 0.95), llm("technology", "topic", 0.90), llm("ai", "topic", 0.92), llm("education", "topic", 0.88), llm("serious", "mood", 0.80)},
		"yt-002": {llm("science", "topic", 0.92), llm("mathematics", "topic", 0.95), llm("philosophy", "topic", 0.78), llm("contemplative", "mood", 0.85), llm("inspirational", "mood", 0.80)},
		"yt-003": {llm("science", "topic", 0.92), llm("mathematics", "topic", 0.95), llm("contemplative", "mood", 0.82), llm("playful", "mood", 0.78)},
		"yt-004": {llm("science", "topic", 0.95), llm("space", "topic", 0.88), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.78)},
		"yt-005": {llm("science", "topic", 0.90), llm("mathematics", "topic", 0.92), llm("education", "topic", 0.85), llm("serious", "mood", 0.82)},
		"yt-006": {llm("education", "topic", 0.95), llm("science", "topic", 0.72), llm("serious", "mood", 0.82), llm("inspirational", "mood", 0.78)},
		"yt-007": {llm("science", "topic", 0.92), llm("mathematics", "topic", 0.95), llm("education", "topic", 0.85), llm("contemplative", "mood", 0.82)},
		"yt-008": {llm("science", "topic", 0.90), llm("mathematics", "topic", 0.93), llm("education", "topic", 0.88), llm("serious", "mood", 0.80)},
		"yt-009": {llm("science", "topic", 0.92), llm("mathematics", "topic", 0.90), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.80)},
		"yt-010": {llm("science", "topic", 0.93), llm("mathematics", "topic", 0.88), llm("playful", "mood", 0.82), llm("contemplative", "mood", 0.78)},
		"yt-011": {llm("mathematics", "topic", 0.95), llm("science", "topic", 0.85), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.82)},
		"yt-012": {llm("science", "topic", 0.95), llm("space", "topic", 0.90), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.80)},
		"yt-013": {llm("mathematics", "topic", 0.95), llm("science", "topic", 0.88), llm("serious", "mood", 0.85), llm("contemplative", "mood", 0.82)},
		"yt-014": {llm("science", "topic", 0.93), llm("mathematics", "topic", 0.90), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.78)},
		"yt-015": {llm("mathematics", "topic", 0.95), llm("education", "topic", 0.88), llm("playful", "mood", 0.82), llm("contemplative", "mood", 0.78)},
		"yt-016": {llm("science", "topic", 0.95), llm("space", "topic", 0.88), llm("education", "topic", 0.82), llm("serious", "mood", 0.80)},
		"yt-017": {llm("science", "topic", 0.90), llm("mathematics", "topic", 0.95), llm("contemplative", "mood", 0.85), llm("playful", "mood", 0.78)},
		"yt-018": {llm("mathematics", "topic", 0.95), llm("education", "topic", 0.90), llm("serious", "mood", 0.82), llm("contemplative", "mood", 0.78)},
		"yt-019": {llm("mathematics", "topic", 0.95), llm("science", "topic", 0.88), llm("contemplative", "mood", 0.85), llm("inspirational", "mood", 0.78)},

		// ============================================================
		// VIDEO ERA 2: Programming/Design (yt-020 to yt-039)
		// ============================================================
		"yt-020": {llm("programming", "topic", 0.95), llm("technology", "topic", 0.88), llm("art", "topic", 0.75), llm("funny", "mood", 0.85), llm("inspirational", "mood", 0.80)},
		"yt-021": {llm("programming", "topic", 0.92), llm("design", "topic", 0.88), llm("technology", "topic", 0.85), llm("inspirational", "mood", 0.90)},
		"yt-022": {llm("programming", "topic", 0.95), llm("design", "topic", 0.85), llm("contemplative", "mood", 0.82), llm("inspirational", "mood", 0.78)},
		"yt-023": {llm("programming", "topic", 0.92), llm("design", "topic", 0.90), llm("technology", "topic", 0.82), llm("inspirational", "mood", 0.88)},
		"yt-024": {llm("programming", "topic", 0.95), llm("design", "topic", 0.88), llm("technology", "topic", 0.85), llm("serious", "mood", 0.78)},
		"yt-025": {llm("programming", "topic", 0.90), llm("design", "topic", 0.82), llm("contemplative", "mood", 0.88), llm("inspirational", "mood", 0.82)},
		"yt-026": {llm("design", "topic", 0.95), llm("technology", "topic", 0.82), llm("inspirational", "mood", 0.85), llm("playful", "mood", 0.78)},
		"yt-027": {llm("programming", "topic", 0.95), llm("technology", "topic", 0.88), llm("funny", "mood", 0.82), llm("inspirational", "mood", 0.78)},
		"yt-028": {llm("design", "topic", 0.93), llm("programming", "topic", 0.85), llm("technology", "topic", 0.82), llm("inspirational", "mood", 0.80)},
		"yt-029": {llm("programming", "topic", 0.95), llm("technology", "topic", 0.90), llm("inspirational", "mood", 0.82), llm("serious", "mood", 0.78)},
		"yt-030": {llm("programming", "topic", 0.95), llm("technology", "topic", 0.90), llm("serious", "mood", 0.82), llm("inspirational", "mood", 0.78)},
		"yt-031": {llm("design", "topic", 0.95), llm("technology", "topic", 0.85), llm("inspirational", "mood", 0.82), llm("serious", "mood", 0.78)},
		"yt-032": {llm("programming", "topic", 0.92), llm("technology", "topic", 0.90), llm("playful", "mood", 0.82), llm("energetic", "mood", 0.78)},
		"yt-033": {llm("programming", "topic", 0.95), llm("technology", "topic", 0.90), llm("energetic", "mood", 0.82), llm("inspirational", "mood", 0.78)},
		"yt-034": {llm("programming", "topic", 0.90), llm("technology", "topic", 0.88), llm("education", "topic", 0.82), llm("inspirational", "mood", 0.80)},
		"yt-035": {llm("programming", "topic", 0.88), llm("technology", "topic", 0.92), llm("design", "topic", 0.80), llm("serious", "mood", 0.82)},
		"yt-036": {llm("design", "topic", 0.92), llm("programming", "topic", 0.88), llm("technology", "topic", 0.85), llm("inspirational", "mood", 0.80)},
		"yt-037": {llm("programming", "topic", 0.95), llm("technology", "topic", 0.88), llm("serious", "mood", 0.82), llm("contemplative", "mood", 0.75)},
		"yt-038": {llm("design", "topic", 0.95), llm("art", "topic", 0.82), llm("education", "topic", 0.78), llm("inspirational", "mood", 0.80)},
		"yt-039": {llm("music-theory", "topic", 0.88), llm("data", "topic", 0.82), llm("technology", "topic", 0.78), llm("playful", "mood", 0.80)},

		// ============================================================
		// VIDEO ERA 3: Philosophy/Contemplative (yt-040 to yt-058)
		// ============================================================
		"yt-040": {llm("philosophy", "topic", 0.95), llm("education", "topic", 0.88), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.82)},
		"yt-041": {llm("philosophy", "topic", 0.95), llm("history", "topic", 0.82), llm("contemplative", "mood", 0.88), llm("peaceful", "mood", 0.80)},
		"yt-042": {llm("philosophy", "topic", 0.95), llm("contemplative", "mood", 0.92), llm("melancholic", "mood", 0.82), llm("serious", "mood", 0.78)},
		"yt-043": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.82), llm("contemplative", "mood", 0.90), llm("inspirational", "mood", 0.78)},
		"yt-044": {llm("psychology", "topic", 0.92), llm("philosophy", "topic", 0.85), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.80)},
		"yt-045": {llm("philosophy", "topic", 0.95), llm("history", "topic", 0.80), llm("contemplative", "mood", 0.90), llm("melancholic", "mood", 0.78)},
		"yt-046": {llm("psychology", "topic", 0.95), llm("technology", "topic", 0.82), llm("serious", "mood", 0.88), llm("dark", "mood", 0.78)},
		"yt-047": {llm("philosophy", "topic", 0.95), llm("education", "topic", 0.85), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.82)},
		"yt-048": {llm("psychology", "topic", 0.93), llm("philosophy", "topic", 0.80), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.78)},
		"yt-049": {llm("philosophy", "topic", 0.92), llm("science", "topic", 0.82), llm("contemplative", "mood", 0.90), llm("uplifting", "mood", 0.82)},
		"yt-050": {llm("history", "topic", 0.95), llm("philosophy", "topic", 0.75), llm("serious", "mood", 0.88), llm("contemplative", "mood", 0.80)},
		"yt-051": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.82), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.78)},
		"yt-052": {llm("philosophy", "topic", 0.92), llm("education", "topic", 0.88), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.80)},
		"yt-053": {llm("philosophy", "topic", 0.95), llm("history", "topic", 0.85), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.82)},
		"yt-054": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.88), llm("contemplative", "mood", 0.90), llm("dreamy", "mood", 0.78)},
		"yt-055": {llm("philosophy", "topic", 0.95), llm("language", "topic", 0.82), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.78)},
		"yt-056": {llm("history", "topic", 0.95), llm("philosophy", "topic", 0.88), llm("serious", "mood", 0.85), llm("contemplative", "mood", 0.82)},
		"yt-057": {llm("philosophy", "topic", 0.95), llm("education", "topic", 0.82), llm("contemplative", "mood", 0.88), llm("inspirational", "mood", 0.80)},
		"yt-058": {llm("philosophy", "topic", 0.95), llm("history", "topic", 0.88), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.80)},

		// ============================================================
		// PODCAST ERA 1: Tech/Business (yt-059 to yt-071)
		// ============================================================
		"yt-059": {llm("technology", "topic", 0.92), llm("business", "topic", 0.85), llm("ai", "topic", 0.88), llm("serious", "mood", 0.80), llm("interview", "format", 0.95)},
		"yt-060": {llm("technology", "topic", 0.95), llm("ai", "topic", 0.92), llm("business", "topic", 0.82), llm("serious", "mood", 0.82), llm("interview", "format", 0.95)},
		"yt-061": {llm("technology", "topic", 0.90), llm("ai", "topic", 0.88), llm("business", "topic", 0.85), llm("serious", "mood", 0.78), llm("podcast-episode", "format", 0.95)},
		"yt-062": {llm("business", "topic", 0.92), llm("entrepreneurship", "topic", 0.88), llm("technology", "topic", 0.82), llm("serious", "mood", 0.80), llm("interview", "format", 0.95)},
		"yt-063": {llm("technology", "topic", 0.95), llm("business", "topic", 0.90), llm("engineering", "topic", 0.82), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-064": {llm("business", "topic", 0.95), llm("entrepreneurship", "topic", 0.90), llm("finance", "topic", 0.78), llm("inspirational", "mood", 0.80), llm("podcast-episode", "format", 0.95)},
		"yt-065": {llm("technology", "topic", 0.92), llm("business", "topic", 0.85), llm("social-media", "topic", 0.80), llm("serious", "mood", 0.78), llm("interview", "format", 0.95)},
		"yt-066": {llm("economics", "topic", 0.90), llm("philosophy", "topic", 0.82), llm("contemplative", "mood", 0.85), llm("interview", "format", 0.95)},
		"yt-067": {llm("business", "topic", 0.90), llm("entrepreneurship", "topic", 0.85), llm("technology", "topic", 0.80), llm("serious", "mood", 0.78), llm("podcast-episode", "format", 0.95)},
		"yt-068": {llm("technology", "topic", 0.95), llm("business", "topic", 0.90), llm("engineering", "topic", 0.85), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-069": {llm("business", "topic", 0.95), llm("entrepreneurship", "topic", 0.92), llm("technology", "topic", 0.80), llm("inspirational", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-070": {llm("technology", "topic", 0.95), llm("ai", "topic", 0.92), llm("engineering", "topic", 0.85), llm("serious", "mood", 0.80), llm("podcast-episode", "format", 0.95)},
		"yt-071": {llm("technology", "topic", 0.92), llm("ai", "topic", 0.95), llm("science", "topic", 0.82), llm("serious", "mood", 0.82), llm("interview", "format", 0.95)},

		// ============================================================
		// PODCAST ERA 2: Culture/Philosophy (yt-072 to yt-083)
		// ============================================================
		"yt-072": {llm("health", "topic", 0.92), llm("science", "topic", 0.85), llm("spirituality", "topic", 0.80), llm("contemplative", "mood", 0.85), llm("podcast-episode", "format", 0.95)},
		"yt-073": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.88), llm("contemplative", "mood", 0.92), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-074": {llm("philosophy", "topic", 0.95), llm("history", "topic", 0.80), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.78), llm("podcast-episode", "format", 0.95)},
		"yt-075": {llm("health", "topic", 0.95), llm("science", "topic", 0.88), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-076": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.85), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-077": {llm("psychology", "topic", 0.92), llm("relationships", "topic", 0.88), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.78), llm("interview", "format", 0.95)},
		"yt-078": {llm("philosophy", "topic", 0.95), llm("history", "topic", 0.82), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.80), llm("podcast-episode", "format", 0.95)},
		"yt-079": {llm("history", "topic", 0.95), llm("philosophy", "topic", 0.78), llm("serious", "mood", 0.90), llm("intense", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-080": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.82), llm("contemplative", "mood", 0.90), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-081": {llm("health", "topic", 0.92), llm("science", "topic", 0.88), llm("psychology", "topic", 0.80), llm("serious", "mood", 0.82), llm("podcast-episode", "format", 0.95)},
		"yt-082": {llm("history", "topic", 0.95), llm("philosophy", "topic", 0.75), llm("serious", "mood", 0.90), llm("intense", "mood", 0.85), llm("podcast-episode", "format", 0.95)},
		"yt-083": {llm("philosophy", "topic", 0.95), llm("psychology", "topic", 0.82), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.80), llm("podcast-episode", "format", 0.95)},

		// ============================================================
		// NETFLIX (nf-001 to nf-020)
		// ============================================================
		"nf-001": {llm("drama", "genre", 0.95), llm("crime", "genre", 0.92), llm("thriller", "genre", 0.85), llm("intense", "mood", 0.90), llm("series", "format", 0.95)},
		"nf-002": {llm("drama", "genre", 0.95), llm("crime", "genre", 0.90), llm("dark", "mood", 0.88), llm("intense", "mood", 0.85), llm("series", "format", 0.95)},
		"nf-003": {llm("sci-fi", "genre", 0.93), llm("horror", "genre", 0.80), llm("mystery", "genre", 0.85), llm("eerie", "mood", 0.88), llm("series", "format", 0.95)},
		"nf-004": {llm("sci-fi", "genre", 0.92), llm("mystery", "genre", 0.88), llm("eerie", "mood", 0.85), llm("nostalgic", "mood", 0.80), llm("series", "format", 0.95)},
		"nf-005": {llm("drama", "genre", 0.95), llm("documentary", "genre", 0.78), llm("serious", "mood", 0.90), llm("contemplative", "mood", 0.82), llm("series", "format", 0.95)},
		"nf-006": {llm("sci-fi", "genre", 0.88), llm("thriller", "genre", 0.90), llm("dark", "mood", 0.92), llm("eerie", "mood", 0.85), llm("series", "format", 0.95)},
		"nf-007": {llm("sci-fi", "genre", 0.90), llm("romance", "genre", 0.85), llm("nostalgic", "mood", 0.90), llm("romantic", "mood", 0.88), llm("series", "format", 0.95)},
		"nf-008": {llm("drama", "genre", 0.93), llm("intense", "mood", 0.88), llm("contemplative", "mood", 0.82), llm("inspirational", "mood", 0.78), llm("mini-series", "format", 0.92)},
		"nf-009": {llm("sci-fi", "genre", 0.92), llm("mystery", "genre", 0.90), llm("thriller", "genre", 0.85), llm("dark", "mood", 0.92), llm("series", "format", 0.95)},
		"nf-010": {llm("crime", "genre", 0.95), llm("thriller", "genre", 0.90), llm("dark", "mood", 0.90), llm("intense", "mood", 0.88), llm("series", "format", 0.95)},
		"nf-011": {llm("crime", "genre", 0.95), llm("drama", "genre", 0.90), llm("intense", "mood", 0.88), llm("dark", "mood", 0.82), llm("series", "format", 0.95)},
		"nf-012": {llm("documentary", "genre", 0.95), llm("serious", "mood", 0.90), llm("dark", "mood", 0.82), llm("contemplative", "mood", 0.78), llm("film", "format", 0.92)},
		"nf-013": {llm("comedy", "genre", 0.85), llm("sci-fi", "genre", 0.80), llm("drama", "genre", 0.78), llm("funny", "mood", 0.82), llm("film", "format", 0.95)},
		"nf-014": {llm("thriller", "genre", 0.93), llm("drama", "genre", 0.88), llm("intense", "mood", 0.92), llm("dark", "mood", 0.88), llm("series", "format", 0.95)},
		"nf-015": {llm("crime", "genre", 0.93), llm("thriller", "genre", 0.90), llm("drama", "genre", 0.85), llm("dark", "mood", 0.90), llm("series", "format", 0.95)},
		"nf-016": {llm("mystery", "genre", 0.95), llm("comedy", "genre", 0.82), llm("playful", "mood", 0.85), llm("funny", "mood", 0.80), llm("film", "format", 0.95)},
		"nf-017": {llm("comedy", "genre", 0.88), llm("drama", "genre", 0.92), llm("dark", "mood", 0.85), llm("intense", "mood", 0.80), llm("series", "format", 0.95)},
		"nf-018": {llm("thriller", "genre", 0.90), llm("mystery", "genre", 0.88), llm("horror", "genre", 0.78), llm("eerie", "mood", 0.90), llm("series", "format", 0.95)},
		"nf-019": {llm("comedy", "genre", 0.85), llm("mystery", "genre", 0.82), llm("fantasy", "genre", 0.78), llm("playful", "mood", 0.85), llm("series", "format", 0.95)},
		"nf-020": {llm("drama", "genre", 0.95), llm("action", "genre", 0.88), llm("dark", "mood", 0.92), llm("intense", "mood", 0.90), llm("film", "format", 0.95)},

		// ============================================================
		// TIKTOK (tt-001 to tt-015)
		// ============================================================
		"tt-001": {llm("pop", "genre", 0.82), llm("electronic", "genre", 0.78), llm("energetic", "mood", 0.92), llm("playful", "mood", 0.85), llm("clip", "format", 0.95)},
		"tt-002": {llm("comedy", "genre", 0.75), llm("energetic", "mood", 0.82), llm("inspirational", "mood", 0.78), llm("playful", "mood", 0.80), llm("clip", "format", 0.95)},
		"tt-003": {llm("drama", "genre", 0.72), llm("contemplative", "mood", 0.88), llm("inspirational", "mood", 0.85), llm("serious", "mood", 0.78), llm("clip", "format", 0.95)},
		"tt-004": {llm("comedy", "genre", 0.70), llm("chill", "mood", 0.82), llm("peaceful", "mood", 0.78), llm("playful", "mood", 0.85), llm("clip", "format", 0.95)},
		"tt-005": {llm("documentary", "genre", 0.78), llm("contemplative", "mood", 0.82), llm("playful", "mood", 0.78), llm("inspirational", "mood", 0.75), llm("clip", "format", 0.95)},
		"tt-006": {llm("comedy", "genre", 0.72), llm("chill", "mood", 0.85), llm("inspirational", "mood", 0.80), llm("playful", "mood", 0.78), llm("clip", "format", 0.95)},
		"tt-007": {llm("comedy", "genre", 0.75), llm("chill", "mood", 0.82), llm("inspirational", "mood", 0.78), llm("playful", "mood", 0.80), llm("clip", "format", 0.95)},
		"tt-008": {llm("documentary", "genre", 0.75), llm("contemplative", "mood", 0.85), llm("inspirational", "mood", 0.82), llm("nostalgic", "mood", 0.78), llm("clip", "format", 0.95)},
		"tt-009": {llm("documentary", "genre", 0.80), llm("serious", "mood", 0.82), llm("playful", "mood", 0.78), llm("nostalgic", "mood", 0.80), llm("clip", "format", 0.95)},
		"tt-010": {llm("documentary", "genre", 0.78), llm("peaceful", "mood", 0.92), llm("dreamy", "mood", 0.88), llm("contemplative", "mood", 0.80), llm("clip", "format", 0.95)},
		"tt-011": {llm("documentary", "genre", 0.75), llm("inspirational", "mood", 0.85), llm("energetic", "mood", 0.80), llm("playful", "mood", 0.78), llm("clip", "format", 0.95)},
		"tt-012": {llm("sci-fi", "genre", 0.78), llm("energetic", "mood", 0.85), llm("playful", "mood", 0.82), llm("dreamy", "mood", 0.78), llm("clip", "format", 0.95)},
		"tt-013": {llm("documentary", "genre", 0.72), llm("energetic", "mood", 0.90), llm("inspirational", "mood", 0.85), llm("uplifting", "mood", 0.82), llm("clip", "format", 0.95)},
		"tt-014": {llm("documentary", "genre", 0.82), llm("contemplative", "mood", 0.88), llm("serious", "mood", 0.82), llm("dreamy", "mood", 0.78), llm("clip", "format", 0.95)},
		"tt-015": {llm("comedy", "genre", 0.72), llm("chill", "mood", 0.85), llm("peaceful", "mood", 0.82), llm("nostalgic", "mood", 0.78), llm("clip", "format", 0.95)},

		// ============================================================
		// GOODREADS (gr-001 to gr-015)
		// ============================================================
		"gr-001": {llm("sci-fi", "genre", 0.95), llm("adventure", "genre", 0.85), llm("contemplative", "mood", 0.82), llm("intense", "mood", 0.78), llm("novel", "format", 0.95)},
		"gr-002": {llm("sci-fi", "genre", 0.93), llm("adventure", "genre", 0.88), llm("science", "topic", 0.82), llm("uplifting", "mood", 0.85), llm("novel", "format", 0.95)},
		"gr-003": {llm("literary-fiction", "genre", 0.95), llm("philosophy", "topic", 0.90), llm("contemplative", "mood", 0.92), llm("melancholic", "mood", 0.85), llm("novel", "format", 0.95)},
		"gr-004": {llm("non-fiction", "genre", 0.95), llm("history", "topic", 0.92), llm("science", "topic", 0.82), llm("contemplative", "mood", 0.85), llm("novel", "format", 0.90)},
		"gr-005": {llm("sci-fi", "genre", 0.95), llm("technology", "topic", 0.88), llm("dark", "mood", 0.85), llm("eerie", "mood", 0.78), llm("novel", "format", 0.95)},
		"gr-006": {llm("memoir", "genre", 0.90), llm("non-fiction", "genre", 0.88), llm("philosophy", "topic", 0.92), llm("contemplative", "mood", 0.90), llm("novel", "format", 0.85)},
		"gr-007": {llm("sci-fi", "genre", 0.93), llm("literary-fiction", "genre", 0.82), llm("philosophy", "topic", 0.78), llm("contemplative", "mood", 0.88), llm("novel", "format", 0.95)},
		"gr-008": {llm("non-fiction", "genre", 0.95), llm("psychology", "topic", 0.95), llm("science", "topic", 0.85), llm("serious", "mood", 0.82), llm("novel", "format", 0.85)},
		"gr-009": {llm("literary-fiction", "genre", 0.95), llm("philosophy", "topic", 0.92), llm("contemplative", "mood", 0.92), llm("dark", "mood", 0.82), llm("novel", "format", 0.95)},
		"gr-010": {llm("non-fiction", "genre", 0.92), llm("programming", "topic", 0.95), llm("design", "topic", 0.88), llm("serious", "mood", 0.82), llm("novel", "format", 0.78)},
		"gr-011": {llm("non-fiction", "genre", 0.90), llm("philosophy", "topic", 0.95), llm("contemplative", "mood", 0.92), llm("melancholic", "mood", 0.82), llm("essay", "format", 0.90)},
		"gr-012": {llm("non-fiction", "genre", 0.95), llm("programming", "topic", 0.92), llm("technology", "topic", 0.90), llm("serious", "mood", 0.85), llm("novel", "format", 0.78)},
		"gr-013": {llm("literary-fiction", "genre", 0.95), llm("romance", "genre", 0.82), llm("melancholic", "mood", 0.92), llm("nostalgic", "mood", 0.88), llm("novel", "format", 0.95)},
		"gr-014": {llm("non-fiction", "genre", 0.88), llm("philosophy", "topic", 0.95), llm("contemplative", "mood", 0.92), llm("peaceful", "mood", 0.85), llm("novel", "format", 0.80)},
		"gr-015": {llm("literary-fiction", "genre", 0.93), llm("fantasy", "genre", 0.78), llm("dreamy", "mood", 0.90), llm("contemplative", "mood", 0.85), llm("novel", "format", 0.95)},

		// ============================================================
		// ANILIST: Anime (al-001 to al-010) & Manga (al-011 to al-015)
		// ============================================================
		"al-001": {llm("sci-fi", "genre", 0.95), llm("thriller", "genre", 0.90), llm("intense", "mood", 0.88), llm("contemplative", "mood", 0.82), llm("series", "format", 0.95)},
		"al-002": {llm("action", "genre", 0.95), llm("fantasy", "genre", 0.85), llm("intense", "mood", 0.92), llm("dark", "mood", 0.85), llm("series", "format", 0.95)},
		"al-003": {llm("sci-fi", "genre", 0.92), llm("drama", "genre", 0.88), llm("dark", "mood", 0.90), llm("contemplative", "mood", 0.88), llm("series", "format", 0.95)},
		"al-004": {llm("fantasy", "genre", 0.95), llm("adventure", "genre", 0.90), llm("animation", "genre", 0.88), llm("dreamy", "mood", 0.92), llm("film", "format", 0.95)},
		"al-005": {llm("sci-fi", "genre", 0.90), llm("action", "genre", 0.88), llm("chill", "mood", 0.82), llm("nostalgic", "mood", 0.85), llm("series", "format", 0.95)},
		"al-006": {llm("action", "genre", 0.88), llm("comedy", "genre", 0.85), llm("playful", "mood", 0.85), llm("uplifting", "mood", 0.80), llm("series", "format", 0.95)},
		"al-007": {llm("romance", "genre", 0.92), llm("fantasy", "genre", 0.85), llm("animation", "genre", 0.88), llm("romantic", "mood", 0.90), llm("film", "format", 0.95)},
		"al-008": {llm("action", "genre", 0.90), llm("adventure", "genre", 0.92), llm("drama", "genre", 0.85), llm("intense", "mood", 0.88), llm("series", "format", 0.95)},
		"al-009": {llm("thriller", "genre", 0.93), llm("horror", "genre", 0.82), llm("dark", "mood", 0.92), llm("eerie", "mood", 0.88), llm("film", "format", 0.95)},
		"al-010": {llm("thriller", "genre", 0.92), llm("mystery", "genre", 0.90), llm("drama", "genre", 0.85), llm("dark", "mood", 0.90), llm("series", "format", 0.95)},
		"al-011": {llm("fantasy", "genre", 0.92), llm("action", "genre", 0.90), llm("dark", "mood", 0.92), llm("intense", "mood", 0.90), llm("graphic-novel", "format", 0.90)},
		"al-012": {llm("action", "genre", 0.90), llm("drama", "genre", 0.88), llm("contemplative", "mood", 0.85), llm("intense", "mood", 0.82), llm("graphic-novel", "format", 0.90)},
		"al-013": {llm("action", "genre", 0.93), llm("horror", "genre", 0.82), llm("dark", "mood", 0.85), llm("energetic", "mood", 0.82), llm("graphic-novel", "format", 0.90)},
		"al-014": {llm("drama", "genre", 0.95), llm("melancholic", "mood", 0.92), llm("dark", "mood", 0.88), llm("contemplative", "mood", 0.85), llm("graphic-novel", "format", 0.90)},
		"al-015": {llm("sci-fi", "genre", 0.90), llm("mystery", "genre", 0.88), llm("contemplative", "mood", 0.85), llm("serious", "mood", 0.82), llm("graphic-novel", "format", 0.90)},
	}
}

// --- Helpers ---

func pgUUID(id pgtype.UUID) string {
	if !id.Valid {
		return "<nil>"
	}
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func pgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func pgInterval(d *time.Duration) pgtype.Interval {
	if d == nil {
		return pgtype.Interval{}
	}
	return pgtype.Interval{Microseconds: d.Microseconds(), Valid: true}
}

func pgInt4(n int32) pgtype.Int4 {
	return pgtype.Int4{Int32: n, Valid: true}
}

func pgFloat(f float32) pgtype.Float4 {
	if f == 0 {
		return pgtype.Float4{}
	}
	return pgtype.Float4{Float32: f, Valid: true}
}

func mustJSON(v map[string]any) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("json: %v", err)
	}
	return b
}
