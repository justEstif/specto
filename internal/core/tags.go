package core

// TagCategory represents a category of tags in the fixed taxonomy.
type TagCategory string

const (
	TagCategoryGenre  TagCategory = "genre"
	TagCategoryTopic  TagCategory = "topic"
	TagCategoryMood   TagCategory = "mood"
	TagCategoryFormat TagCategory = "format"
)

// ValidTagCategories contains all recognized tag categories.
var ValidTagCategories = map[TagCategory]bool{
	TagCategoryGenre:  true,
	TagCategoryTopic:  true,
	TagCategoryMood:   true,
	TagCategoryFormat: true,
}

// GenreTags is the fixed set of genre tags.
var GenreTags = []string{
	"rock", "pop", "hip-hop", "r-and-b", "jazz", "classical", "electronic",
	"country", "folk", "metal", "punk", "indie", "latin", "reggae", "blues",
	"soul", "funk", "ambient", "alternative",
	"comedy", "drama", "thriller", "horror", "sci-fi", "fantasy", "romance",
	"documentary", "animation", "action", "adventure", "mystery", "crime",
	"western", "musical",
	"literary-fiction", "non-fiction", "memoir", "self-help", "biography",
	"poetry",
	"true-crime", "news", "interview", "panel", "narrative",
}

// TopicTags is the fixed set of topic tags.
var TopicTags = []string{
	"science", "technology", "politics", "history", "philosophy", "psychology",
	"economics", "health", "fitness", "cooking", "travel", "nature",
	"environment", "education", "art", "music-theory", "film-criticism",
	"gaming", "sports", "business", "entrepreneurship", "relationships",
	"parenting", "spirituality", "mathematics", "engineering", "design",
	"social-media", "pop-culture", "current-events", "language", "literature",
	"space", "ai", "programming", "data", "security", "finance",
	"real-estate", "fashion", "automotive",
}

// MoodTags is the fixed set of mood tags.
var MoodTags = []string{
	"energetic", "melancholic", "chill", "dark", "uplifting", "aggressive",
	"romantic", "nostalgic", "anxious", "peaceful", "funny", "serious",
	"inspirational", "eerie", "intense", "dreamy", "playful", "raw",
	"contemplative", "triumphant",
}

// FormatTags is the fixed set of format tags.
var FormatTags = []string{
	"album-track", "single", "ep-track", "live-recording", "remix", "cover",
	"film", "series", "mini-series", "short-film", "music-video", "livestream",
	"episode", "clip", "trailer", "compilation",
	"novel", "short-story", "essay", "collection", "graphic-novel",
	"podcast-episode", "podcast-series", "audiobook",
}

// validTags is a precomputed lookup map of all valid tags across all categories.
// Keyed by tag name, value is the category it belongs to.
var validTags map[string]TagCategory

func init() {
	validTags = make(map[string]TagCategory,
		len(GenreTags)+len(TopicTags)+len(MoodTags)+len(FormatTags))

	for _, t := range GenreTags {
		validTags[t] = TagCategoryGenre
	}
	for _, t := range TopicTags {
		validTags[t] = TagCategoryTopic
	}
	for _, t := range MoodTags {
		validTags[t] = TagCategoryMood
	}
	for _, t := range FormatTags {
		validTags[t] = TagCategoryFormat
	}
}

// IsValidTag returns true if the tag is in the fixed tag set.
func IsValidTag(tag string) bool {
	_, ok := validTags[tag]
	return ok
}

// TagCategoryOf returns the category of a tag, or empty string if unknown.
func TagCategoryOf(tag string) TagCategory {
	return validTags[tag]
}

// ValidateTagResult filters a TagResult to only include tags from the fixed set.
// Unknown tags are silently dropped. Returns a new TagResult.
func ValidateTagResult(tr *TagResult) *TagResult {
	return &TagResult{
		Genre:  filterTagScores(tr.Genre, TagCategoryGenre),
		Topic:  filterTagScores(tr.Topic, TagCategoryTopic),
		Mood:   filterTagScores(tr.Mood, TagCategoryMood),
		Format: filterTagScores(tr.Format, TagCategoryFormat),
	}
}

// filterTagScores returns only the tag scores where the tag exists in the
// fixed set and belongs to the expected category.
func filterTagScores(scores []TagScore, expected TagCategory) []TagScore {
	if len(scores) == 0 {
		return nil
	}
	var filtered []TagScore
	for _, ts := range scores {
		if TagCategoryOf(ts.Tag) == expected {
			filtered = append(filtered, ts)
		}
	}
	return filtered
}

// AllFixedTags returns all tags in the fixed set as a map keyed by tag name.
func AllFixedTags() map[string]TagCategory {
	// Return a copy to prevent mutation.
	copy := make(map[string]TagCategory, len(validTags))
	for k, v := range validTags {
		copy[k] = v
	}
	return copy
}

// TagsByCategory returns all tags for a given category.
func TagsByCategory(cat TagCategory) []string {
	switch cat {
	case TagCategoryGenre:
		return GenreTags
	case TagCategoryTopic:
		return TopicTags
	case TagCategoryMood:
		return MoodTags
	case TagCategoryFormat:
		return FormatTags
	default:
		return nil
	}
}
