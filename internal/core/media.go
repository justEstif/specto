package core

import "time"

// MediaItem is the normalized domain representation of any consumed media.
// This is distinct from the database model (internal/database.MediaItem) —
// conversions happen in the store layer.
type MediaItem struct {
	Platform    string    // "spotify", "youtube", "netflix", etc.
	Type        MediaType // Music, Video, Article, Podcast
	Title       string
	Creator     string // artist, channel, author
	ConsumedAt  time.Time
	Duration    *time.Duration // how long the content is (nil if unknown)
	TimeSpent   *time.Duration // how long the user engaged (nil if unknown)
	Tags        []string       // genre, topic, mood — plugin can pre-populate
	URL         string         // link back to original content
	ExternalID  string         // platform-specific ID for dedup
	RawMetadata map[string]any // platform-specific fields, stored as jsonb
}

// MediaType categorizes the kind of media consumed.
type MediaType string

const (
	MediaMusic   MediaType = "music"
	MediaVideo   MediaType = "video"
	MediaArticle MediaType = "article"
	MediaPodcast MediaType = "podcast"
	MediaBook    MediaType = "book"
)

// ValidMediaTypes contains all recognized media types.
var ValidMediaTypes = map[MediaType]bool{
	MediaMusic:   true,
	MediaVideo:   true,
	MediaArticle: true,
	MediaPodcast: true,
	MediaBook:    true,
}

// Valid returns true if the media type is one of the recognized values.
func (mt MediaType) Valid() bool {
	return ValidMediaTypes[mt]
}
