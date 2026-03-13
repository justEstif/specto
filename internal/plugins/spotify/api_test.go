package spotify

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/justestif/specto/internal/core"
)

// sampleRecentlyPlayed returns a valid Spotify recently-played JSON response.
func sampleRecentlyPlayed() string {
	return `{
		"items": [
			{
				"track": {
					"id": "4u7EnebtmKWzUH433cf5Qv",
					"name": "Bohemian Rhapsody",
					"duration_ms": 354947,
					"popularity": 89,
					"explicit": false,
					"uri": "spotify:track:4u7EnebtmKWzUH433cf5Qv",
					"external_urls": {"spotify": "https://open.spotify.com/track/4u7EnebtmKWzUH433cf5Qv"},
					"artists": [
						{"id": "1dfeR4HaWDbWqFHLkxsg1d", "name": "Queen"}
					],
					"album": {"id": "6i6folBtxKV28WX3msQ4FE", "name": "A Night at the Opera"}
				},
				"played_at": "2026-03-12T08:32:17.000Z",
				"context": {"type": "album", "uri": "spotify:album:6i6folBtxKV28WX3msQ4FE"}
			},
			{
				"track": {
					"id": "5CQ30WqJwcep0pYcV4AMNc",
					"name": "Stairway to Heaven",
					"duration_ms": 482830,
					"popularity": 82,
					"explicit": false,
					"uri": "spotify:track:5CQ30WqJwcep0pYcV4AMNc",
					"external_urls": {"spotify": "https://open.spotify.com/track/5CQ30WqJwcep0pYcV4AMNc"},
					"artists": [
						{"id": "36QJpDe2go2KgaRleHCDTp", "name": "Led Zeppelin"}
					],
					"album": {"id": "70lQYZtypdCALtjDlLWWjh", "name": "Led Zeppelin IV"}
				},
				"played_at": "2026-03-12T07:45:00.000Z",
				"context": null
			}
		],
		"cursors": {"after": "1741768337000", "before": "1741765500000"},
		"limit": 50,
		"next": null
	}`
}

func TestAPIPluginName(t *testing.T) {
	p := NewAPI()
	if got := p.Name(); got != "spotify-api" {
		t.Errorf("Name() = %q, want %q", got, "spotify-api")
	}
}

func TestAPIPluginAuthType(t *testing.T) {
	p := NewAPI()
	if got := p.AuthType(); got != core.AuthOAuth {
		t.Errorf("AuthType() = %v, want AuthOAuth (%v)", got, core.AuthOAuth)
	}
}

func TestAPIPluginAuthConfig(t *testing.T) {
	p := NewAPI()
	cfg := p.AuthConfig()
	if cfg == nil {
		t.Fatal("AuthConfig() returned nil")
	}
	if cfg.ProviderName != "Spotify" {
		t.Errorf("ProviderName = %q, want %q", cfg.ProviderName, "Spotify")
	}
	if cfg.AuthURL != "https://accounts.spotify.com/authorize" {
		t.Errorf("AuthURL = %q", cfg.AuthURL)
	}
	if cfg.TokenURL != "https://accounts.spotify.com/api/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if len(cfg.Scopes) != 1 || cfg.Scopes[0] != "user-read-recently-played" {
		t.Errorf("Scopes = %v", cfg.Scopes)
	}
}

func TestAPISyncSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/me/player/recently-played" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("limit = %q, want %q", got, "50")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, sampleRecentlyPlayed())
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	// Verify first item (later played_at = first in response)
	item := result.Items[0]
	if item.Platform != "spotify" {
		t.Errorf("Platform = %q, want %q", item.Platform, "spotify")
	}
	if item.Type != core.MediaMusic {
		t.Errorf("Type = %q, want %q", item.Type, core.MediaMusic)
	}
	if item.Title != "Bohemian Rhapsody" {
		t.Errorf("Title = %q, want %q", item.Title, "Bohemian Rhapsody")
	}
	if item.Creator != "Queen" {
		t.Errorf("Creator = %q, want %q", item.Creator, "Queen")
	}
	if item.ExternalID != "spotify:track:4u7EnebtmKWzUH433cf5Qv" {
		t.Errorf("ExternalID = %q", item.ExternalID)
	}
	if item.URL != "https://open.spotify.com/track/4u7EnebtmKWzUH433cf5Qv" {
		t.Errorf("URL = %q", item.URL)
	}

	wantDuration := 354947 * time.Millisecond
	if item.Duration == nil {
		t.Fatal("Duration is nil")
	}
	if *item.Duration != wantDuration {
		t.Errorf("Duration = %v, want %v", *item.Duration, wantDuration)
	}

	wantTime, _ := time.Parse(time.RFC3339, "2026-03-12T08:32:17Z")
	if !item.ConsumedAt.Equal(wantTime) {
		t.Errorf("ConsumedAt = %v, want %v", item.ConsumedAt, wantTime)
	}

	// Check raw metadata
	if item.RawMetadata["album"] != "A Night at the Opera" {
		t.Errorf("RawMetadata[album] = %v", item.RawMetadata["album"])
	}
	if item.RawMetadata["popularity"] != 89 {
		t.Errorf("RawMetadata[popularity] = %v", item.RawMetadata["popularity"])
	}
	if item.RawMetadata["context_type"] != "album" {
		t.Errorf("RawMetadata[context_type] = %v", item.RawMetadata["context_type"])
	}

	// Second item should have no context
	item2 := result.Items[1]
	if item2.Title != "Stairway to Heaven" {
		t.Errorf("Items[1].Title = %q", item2.Title)
	}
	if _, ok := item2.RawMetadata["context_type"]; ok {
		t.Error("Items[1] should not have context_type in RawMetadata")
	}

	// Cursor should be the latest played_at in Unix ms
	if result.NextCursor == "" {
		t.Fatal("NextCursor is empty")
	}
	// The later played_at is 2026-03-12T08:32:17.000Z
	wantCursorTime, _ := time.Parse(time.RFC3339Nano, "2026-03-12T08:32:17.000Z")
	wantCursor := fmt.Sprintf("%d", wantCursorTime.UnixMilli())
	if result.NextCursor != wantCursor {
		t.Errorf("NextCursor = %q, want %q", result.NextCursor, wantCursor)
	}

	// No next page
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestAPISyncWithCursor(t *testing.T) {
	var receivedAfter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAfter = r.URL.Query().Get("after")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items": [], "cursors": null, "limit": 50, "next": null}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "test-token"}, "1741768337000")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if receivedAfter != "1741768337000" {
		t.Errorf("after param = %q, want %q", receivedAfter, "1741768337000")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
}

func TestAPISyncHasMore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// next is non-empty → HasMore should be true
		fmt.Fprint(w, `{
			"items": [{
				"track": {
					"id": "abc",
					"name": "Song",
					"duration_ms": 200000,
					"popularity": 50,
					"explicit": false,
					"uri": "spotify:track:abc",
					"external_urls": {"spotify": "https://open.spotify.com/track/abc"},
					"artists": [{"id": "a1", "name": "Artist"}],
					"album": {"id": "al1", "name": "Album"}
				},
				"played_at": "2026-03-12T10:00:00.000Z",
				"context": null
			}],
			"cursors": {"after": "123", "before": "456"},
			"limit": 50,
			"next": "https://api.spotify.com/v1/me/player/recently-played?limit=50&before=456"
		}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true")
	}
}

func TestAPISyncAuthExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": {"status": 401, "message": "The access token expired"}}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "expired-token"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 401 response")
	}
	if result.Err.Code != core.ErrAuthExpired {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrAuthExpired)
	}
}

func TestAPISyncNoAccessToken(t *testing.T) {
	p := NewAPI()
	result := p.Sync(context.Background(), core.Credentials{}, "")

	if result.Err == nil {
		t.Fatal("expected error for empty access token")
	}
	if result.Err.Code != core.ErrAuthExpired {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrAuthExpired)
	}
}

func TestAPISyncRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error": {"status": 429, "message": "rate limit"}}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 429 response")
	}
	if result.Err.Code != core.ErrRateLimit {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrRateLimit)
	}
	if !result.Err.Retry {
		t.Error("Retry = false, want true")
	}
	if result.Err.After != 60*time.Second {
		t.Errorf("After = %v, want %v", result.Err.After, 60*time.Second)
	}
}

func TestAPISyncRateLimitNoRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Retry-After header
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 429 response")
	}
	// Should default to 30 seconds
	if result.Err.After != 30*time.Second {
		t.Errorf("After = %v, want %v (default)", result.Err.After, 30*time.Second)
	}
}

func TestAPISyncForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error": {"status": 403, "message": "Insufficient client scope"}}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 403 response")
	}
	if result.Err.Code != core.ErrPermissionDenied {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrPermissionDenied)
	}
}

func TestAPISyncServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": {"status": 500, "message": "Internal Server Error"}}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 500 response")
	}
	if result.Err.Code != core.ErrUpstream {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrUpstream)
	}
}

func TestAPISyncBadGateway(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "bad gateway")
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for 502 response")
	}
	if result.Err.Code != core.ErrUpstream {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrUpstream)
	}
}

func TestAPISyncInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{not valid json}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if result.Err.Code != core.ErrInvalidData {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrInvalidData)
	}
}

func TestAPISyncEmptyItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items": [], "cursors": null, "limit": 50, "next": null}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", result.NextCursor)
	}
	if result.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestAPISyncMultipleArtists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"items": [{
				"track": {
					"id": "collab1",
					"name": "Collab Song",
					"duration_ms": 240000,
					"popularity": 70,
					"explicit": true,
					"uri": "spotify:track:collab1",
					"external_urls": {"spotify": "https://open.spotify.com/track/collab1"},
					"artists": [
						{"id": "a1", "name": "Artist A"},
						{"id": "a2", "name": "Artist B"},
						{"id": "a3", "name": "Artist C"}
					],
					"album": {"id": "al1", "name": "Collab Album"}
				},
				"played_at": "2026-03-12T12:00:00.000Z",
				"context": null
			}],
			"cursors": {"after": "123"},
			"limit": 50,
			"next": null
		}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	// Creator should be the first artist
	if item.Creator != "Artist A" {
		t.Errorf("Creator = %q, want %q", item.Creator, "Artist A")
	}
	// All artist IDs should be in raw metadata
	ids, ok := item.RawMetadata["artist_ids"].([]string)
	if !ok {
		t.Fatalf("artist_ids not []string: %T", item.RawMetadata["artist_ids"])
	}
	if len(ids) != 3 {
		t.Errorf("artist_ids length = %d, want 3", len(ids))
	}
	if item.RawMetadata["explicit"] != true {
		t.Errorf("explicit = %v, want true", item.RawMetadata["explicit"])
	}
}

func TestAPISyncURLFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Track with empty external_urls — should fall back to constructed URL
		fmt.Fprint(w, `{
			"items": [{
				"track": {
					"id": "nourl123",
					"name": "No URL Track",
					"duration_ms": 180000,
					"popularity": 50,
					"explicit": false,
					"uri": "spotify:track:nourl123",
					"external_urls": {},
					"artists": [{"id": "a1", "name": "Artist"}],
					"album": {"id": "al1", "name": "Album"}
				},
				"played_at": "2026-03-12T14:00:00.000Z",
				"context": null
			}],
			"cursors": null,
			"limit": 50,
			"next": null
		}`)
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Items[0].URL != "https://open.spotify.com/track/nourl123" {
		t.Errorf("URL = %q, want fallback URL", result.Items[0].URL)
	}
}

func TestAPIEnrich(t *testing.T) {
	p := NewAPI()
	input := []core.MediaItem{
		{Platform: "spotify", Title: "Song", Type: core.MediaMusic},
	}

	got, err := p.Enrich(context.Background(), core.Credentials{}, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].Title != "Song" {
		t.Errorf("Title = %q, want %q", got[0].Title, "Song")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
	}{
		{"60", 60 * time.Second},
		{"1", 1 * time.Second},
		{"", 30 * time.Second},
		{"abc", 30 * time.Second},
		{"-5", 30 * time.Second},
		{"0", 30 * time.Second},
	}

	for _, tt := range tests {
		got := parseRetryAfter(tt.header)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate(short, 10) = %q", got)
	}
	if got := truncate("a long string here", 6); got != "a long..." {
		t.Errorf("truncate(long, 6) = %q", got)
	}
}

func TestToUnixMs(t *testing.T) {
	// RFC3339Nano
	got := toUnixMs("2026-03-12T08:32:17.000Z")
	if got == 0 {
		t.Fatal("toUnixMs returned 0 for valid timestamp")
	}
	wantTime, _ := time.Parse(time.RFC3339Nano, "2026-03-12T08:32:17.000Z")
	if got != wantTime.UnixMilli() {
		t.Errorf("toUnixMs = %d, want %d", got, wantTime.UnixMilli())
	}

	// RFC3339 without nanos
	got2 := toUnixMs("2026-03-12T08:32:17Z")
	if got2 == 0 {
		t.Fatal("toUnixMs returned 0 for RFC3339 timestamp")
	}

	// Invalid
	if toUnixMs("not-a-date") != 0 {
		t.Error("toUnixMs should return 0 for invalid timestamp")
	}
}

func TestAPISyncUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
		fmt.Fprint(w, "I'm a teapot")
	}))
	defer srv.Close()

	p := NewAPIWithBaseURL(srv.URL)
	result := p.Sync(context.Background(), core.Credentials{AccessToken: "tok"}, "")

	if result.Err == nil {
		t.Fatal("expected error for unexpected status code")
	}
	if result.Err.Code != core.ErrUpstream {
		t.Errorf("error code = %q, want %q", result.Err.Code, core.ErrUpstream)
	}
}
