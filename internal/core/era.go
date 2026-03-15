package core

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Era detection guardrails.
const (
	// MinEraDuration is the shortest an era can be.
	MinEraDuration = 3 * 7 * 24 * time.Hour // 3 weeks

	// MaxErasPerYear caps the number of eras detected per calendar year.
	MaxErasPerYear = 5

	// MinItemsPerEra requires enough data for a meaningful taste vector.
	MinItemsPerEra = 15

	// MinDistinctiveness is the minimum cosine distance between adjacent
	// eras for a boundary to be valid.
	MinDistinctiveness = 0.25

	// MinWindowItems is the minimum items in a biweekly window to produce
	// a reliable tag vector. Windows below this are skipped.
	MinWindowItems = 8

	// InactivityGap is the minimum gap in consumption that acts as a
	// natural era boundary regardless of similarity.
	InactivityGap = 2 * 7 * 24 * time.Hour // 2 weeks

	// WindowSize is the biweekly window duration for tag vectors.
	WindowSize = 2 * 7 * 24 * time.Hour // 2 weeks
)

// Era status values.
const (
	EraStatusSuggested = "suggested"
	EraStatusConfirmed = "confirmed"
	EraStatusDismissed = "dismissed"
)

// Era represents a detected consumption era for a user.
type Era struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	MediaType       *string // nil = cross-media (future)
	Title           *string // user-confirmed name
	SuggestedTitle  *string // LLM-suggested name
	StartedAt       time.Time
	EndedAt         *time.Time // nil = ongoing
	ItemCount       int32
	Distinctiveness float32 // cosine distance from adjacent era (0-1)
	Status          string  // suggested | confirmed | dismissed
	Tags            []EraTag
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// EraTag represents a tag that characterizes an era, with its relative weight.
type EraTag struct {
	TagID    uuid.UUID
	TagName  string
	Category string
	Weight   float32 // relative prominence (0-1)
}

// EraStore manages era persistence.
type EraStore interface {
	// Create inserts a new era.
	Create(ctx context.Context, era Era) (*Era, error)

	// List returns non-dismissed eras for a user, optionally filtered by media type.
	List(ctx context.Context, userID uuid.UUID, mediaType *string) ([]Era, error)

	// Get retrieves a single era by ID for a user.
	Get(ctx context.Context, userID, eraID uuid.UUID) (*Era, error)

	// UpdateTitle confirms an era with a user-provided title.
	UpdateTitle(ctx context.Context, eraID uuid.UUID, title string) (*Era, error)

	// UpdateSuggestedTitle sets the LLM-generated suggested title without
	// changing the era's status.
	UpdateSuggestedTitle(ctx context.Context, eraID uuid.UUID, title string) (*Era, error)

	// Dismiss marks an era as dismissed so it won't resurface.
	Dismiss(ctx context.Context, userID, eraID uuid.UUID) error

	// DeleteSuggested removes all suggested eras for a user/media_type,
	// preserving confirmed and dismissed eras. Used before recomputation.
	DeleteSuggested(ctx context.Context, userID uuid.UUID, mediaType string) error

	// UpsertTag creates or updates a tag weight for an era.
	UpsertTag(ctx context.Context, eraID, tagID uuid.UUID, weight float32) error

	// GetTags returns the characterizing tags for an era.
	GetTags(ctx context.Context, eraID uuid.UUID) ([]EraTag, error)

	// TagVectorByWindow returns tag frequency data bucketed into biweekly
	// windows for a user/media_type within a time range. Used by the
	// detection algorithm.
	TagVectorByWindow(ctx context.Context, userID uuid.UUID, from, to time.Time, mediaType string) ([]WindowTagEntry, error)

	// CountItemsInRange returns the number of items in a time range for
	// a user/media_type.
	CountItemsInRange(ctx context.Context, userID uuid.UUID, mediaType string, from, to time.Time) (int64, error)
}

// EraNamer generates suggested titles for detected eras using LLM.
type EraNamer interface {
	// NameEra generates a short, evocative title for an era based on its
	// characterizing tags. Returns empty string if naming fails.
	NameEra(ctx context.Context, mediaType string, tags []EraTag) (string, error)
}

// WindowTagEntry is a single tag count within a biweekly window,
// returned by EraStore.TagVectorByWindow.
type WindowTagEntry struct {
	WindowStart time.Time
	TagID       uuid.UUID
	TagName     string
	Category    string
	TagCount    int64
	WindowTotal int64 // total tag occurrences in this window
}

// eraRange is a contiguous range of windows forming a candidate era.
type eraRange struct {
	startIdx int
	endIdx   int     // inclusive
	distance float32 // cosine distance at the boundary preceding this era
}

// --- Era detection algorithm ---

// TagVector is a sparse vector of tag weights keyed by tag name.
type TagVector map[string]float64

// Window represents a biweekly time window with its tag distribution.
type Window struct {
	Start      time.Time
	TotalItems int64
	Vector     TagVector
}

// EraBoundary is a candidate era with its time range and characterizing tags.
type EraBoundary struct {
	StartedAt       time.Time
	EndedAt         *time.Time // nil = ongoing
	ItemCount       int64
	Distinctiveness float32
	TopTags         []EraTag
}

// DetectEras runs the sliding-window cosine similarity algorithm over
// pre-built windows and returns candidate era boundaries. The caller
// is responsible for fetching windows via EraStore.TagVectorByWindow
// and building them with BuildWindows.
func DetectEras(windows []Window) []EraBoundary {
	if len(windows) == 0 {
		return nil
	}

	// Step 1: Find candidate boundaries using cosine distance.
	type boundary struct {
		index    int     // index of window AFTER the boundary
		distance float32 // cosine distance at this boundary
	}

	var boundaries []boundary
	for i := 1; i < len(windows); i++ {
		// Check for inactivity gap: if the gap between windows exceeds
		// InactivityGap, it's an automatic boundary.
		gap := windows[i].Start.Sub(windows[i-1].End())
		if gap >= InactivityGap {
			boundaries = append(boundaries, boundary{index: i, distance: 1.0})
			continue
		}

		dist := CosineDistance(windows[i-1].Vector, windows[i].Vector)
		if dist >= MinDistinctiveness {
			boundaries = append(boundaries, boundary{index: i, distance: float32(dist)})
		}
	}

	// Step 2: Build eras from boundaries.
	var ranges []eraRange
	prevIdx := 0
	for _, b := range boundaries {
		ranges = append(ranges, eraRange{
			startIdx: prevIdx,
			endIdx:   b.index - 1,
			distance: b.distance,
		})
		prevIdx = b.index
	}
	// Final era from last boundary to end.
	ranges = append(ranges, eraRange{
		startIdx: prevIdx,
		endIdx:   len(windows) - 1,
		distance: 0, // last era has no "next" to compare against
	})

	// Step 3: Apply guardrails — merge short eras.
	ranges = mergeShortEras(ranges, windows)

	// Step 4: Cap eras per year.
	ranges = capErasPerYear(ranges, windows)

	// Step 5: Build output.
	var result []EraBoundary
	for i, r := range ranges {
		var totalItems int64
		merged := make(TagVector)
		for j := r.startIdx; j <= r.endIdx; j++ {
			totalItems += windows[j].TotalItems
			for tag, weight := range windows[j].Vector {
				merged[tag] += weight
			}
		}

		if totalItems < MinItemsPerEra {
			continue
		}

		// Normalize merged vector and pick top tags.
		topTags := topTagsFromVector(merged, 10)

		dist := r.distance
		if i == len(ranges)-1 && len(ranges) > 1 {
			// Last era: use distance from previous boundary.
			dist = ranges[i-1].distance
		}

		var endedAt *time.Time
		endTime := windows[r.endIdx].End()
		// If this is the last era, leave ended_at nil (ongoing).
		if i < len(ranges)-1 {
			endedAt = &endTime
		}

		result = append(result, EraBoundary{
			StartedAt:       windows[r.startIdx].Start,
			EndedAt:         endedAt,
			ItemCount:       totalItems,
			Distinctiveness: dist,
			TopTags:         topTags,
		})
	}

	return result
}

// BuildWindows converts raw WindowTagEntry rows into Window structs,
// normalizing tag counts into unit vectors.
func BuildWindows(entries []WindowTagEntry) []Window {
	if len(entries) == 0 {
		return nil
	}

	// Group by window start time.
	type windowData struct {
		start      time.Time
		totalItems int64
		tags       TagVector
	}

	windowMap := make(map[time.Time]*windowData)
	var windowStarts []time.Time

	for _, e := range entries {
		wd, ok := windowMap[e.WindowStart]
		if !ok {
			wd = &windowData{
				start: e.WindowStart,
				tags:  make(TagVector),
			}
			windowMap[e.WindowStart] = wd
			windowStarts = append(windowStarts, e.WindowStart)
		}
		wd.tags[e.TagName] = float64(e.TagCount)
		if e.WindowTotal > wd.totalItems {
			wd.totalItems = e.WindowTotal
		}
	}

	sort.Slice(windowStarts, func(i, j int) bool {
		return windowStarts[i].Before(windowStarts[j])
	})

	var windows []Window
	for _, start := range windowStarts {
		wd := windowMap[start]
		if wd.totalItems < MinWindowItems {
			continue
		}

		// Normalize to unit vector.
		normalized := normalize(wd.tags)

		windows = append(windows, Window{
			Start:      wd.start,
			TotalItems: wd.totalItems,
			Vector:     normalized,
		})
	}

	return windows
}

// End returns the end time of a window (start + WindowSize).
func (w Window) End() time.Time {
	return w.Start.Add(WindowSize)
}

// CosineDistance returns 1 - cosine_similarity between two sparse vectors.
// Returns 1.0 (maximum distance) if either vector is zero.
func CosineDistance(a, b TagVector) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 1.0
	}

	var dot, normA, normB float64
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
		normA += va * va
	}
	for _, vb := range b {
		normB += vb * vb
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 1.0
	}

	similarity := dot / denom
	// Clamp to [0, 1] due to floating point.
	if similarity > 1.0 {
		similarity = 1.0
	}
	if similarity < 0 {
		similarity = 0
	}

	return 1.0 - similarity
}

// normalize returns a unit vector (L2 norm = 1).
func normalize(v TagVector) TagVector {
	var sumSq float64
	for _, val := range v {
		sumSq += val * val
	}
	norm := math.Sqrt(sumSq)
	if norm == 0 {
		return v
	}

	result := make(TagVector, len(v))
	for k, val := range v {
		result[k] = val / norm
	}
	return result
}

// mergeShortEras absorbs eras shorter than MinEraDuration into their
// most similar neighbor.
func mergeShortEras(ranges []eraRange, windows []Window) []eraRange {
	for {
		merged := false
		for i := 0; i < len(ranges); i++ {
			duration := windows[ranges[i].endIdx].End().Sub(windows[ranges[i].startIdx].Start)
			if duration >= MinEraDuration || len(ranges) <= 1 {
				continue
			}

			// Merge into neighbor with smaller distance (more similar).
			if i == 0 {
				// Merge into next.
				ranges[1].startIdx = ranges[0].startIdx
				ranges = ranges[1:]
			} else if i == len(ranges)-1 {
				// Merge into previous.
				ranges[i-1].endIdx = ranges[i].endIdx
				ranges = ranges[:i]
			} else {
				// Merge into whichever neighbor is more similar.
				leftDist := ranges[i].distance
				rightDist := ranges[i+1].distance
				if leftDist <= rightDist {
					ranges[i-1].endIdx = ranges[i].endIdx
				} else {
					ranges[i+1].startIdx = ranges[i].startIdx
				}
				ranges = append(ranges[:i], ranges[i+1:]...)
			}
			merged = true
			break
		}
		if !merged {
			break
		}
	}
	return ranges
}

// capErasPerYear keeps only the N most distinctive boundaries per calendar year.
func capErasPerYear(ranges []eraRange, windows []Window) []eraRange {
	if len(ranges) <= MaxErasPerYear {
		return ranges
	}

	// For simplicity, cap globally (not per calendar year) for now.
	// Sort boundaries by distinctiveness, keep top N-1 boundaries
	// (which produce N eras).
	type indexedRange struct {
		original int
		distance float32
	}
	var ranked []indexedRange
	for i, r := range ranges {
		if i > 0 { // skip first era (no incoming boundary)
			ranked = append(ranked, indexedRange{original: i, distance: r.distance})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].distance > ranked[j].distance
	})

	// Keep top MaxErasPerYear-1 boundaries.
	keep := make(map[int]bool)
	keep[0] = true // always keep first era
	limit := MaxErasPerYear - 1
	if limit > len(ranked) {
		limit = len(ranked)
	}
	for i := 0; i < limit; i++ {
		keep[ranked[i].original] = true
	}

	// Rebuild ranges, merging non-kept eras into their predecessor.
	var result []eraRange
	for i, r := range ranges {
		if keep[i] {
			result = append(result, r)
		} else if len(result) > 0 {
			result[len(result)-1].endIdx = r.endIdx
		}
	}

	return result
}

// topTagsFromVector returns the top N tags by weight from a merged vector.
func topTagsFromVector(v TagVector, n int) []EraTag {
	type tagWeight struct {
		name   string
		weight float64
	}

	var tags []tagWeight
	for name, weight := range v {
		tags = append(tags, tagWeight{name: name, weight: weight})
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].weight > tags[j].weight
	})

	if n > len(tags) {
		n = len(tags)
	}

	// Normalize weights to 0-1 range relative to the top tag.
	maxWeight := tags[0].weight
	result := make([]EraTag, n)
	for i := 0; i < n; i++ {
		result[i] = EraTag{
			TagName: tags[i].name,
			Weight:  float32(tags[i].weight / maxWeight),
		}
	}
	return result
}
