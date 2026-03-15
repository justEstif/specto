package core

import (
	"testing"
	"time"
)

func TestCosineDistance(t *testing.T) {
	tests := []struct {
		name string
		a, b TagVector
		want float64
		tol  float64 // tolerance for floating point comparison
	}{
		{
			name: "identical vectors",
			a:    TagVector{"rock": 0.5, "pop": 0.5},
			b:    TagVector{"rock": 0.5, "pop": 0.5},
			want: 0.0,
			tol:  1e-10,
		},
		{
			name: "orthogonal vectors",
			a:    TagVector{"rock": 1.0},
			b:    TagVector{"pop": 1.0},
			want: 1.0,
			tol:  1e-10,
		},
		{
			name: "partially overlapping",
			a:    TagVector{"rock": 0.7, "pop": 0.3},
			b:    TagVector{"rock": 0.3, "jazz": 0.7},
			want: 0.6379, // approximate
			tol:  0.01,
		},
		{
			name: "empty vector a",
			a:    TagVector{},
			b:    TagVector{"rock": 1.0},
			want: 1.0,
			tol:  1e-10,
		},
		{
			name: "both empty",
			a:    TagVector{},
			b:    TagVector{},
			want: 1.0,
			tol:  1e-10,
		},
		{
			name: "same direction different magnitude",
			a:    TagVector{"rock": 1.0, "pop": 1.0},
			b:    TagVector{"rock": 5.0, "pop": 5.0},
			want: 0.0,
			tol:  1e-10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineDistance(tt.a, tt.b)
			if diff := got - tt.want; diff > tt.tol || diff < -tt.tol {
				t.Errorf("CosineDistance() = %v, want %v (±%v)", got, tt.want, tt.tol)
			}
		})
	}
}

func TestBuildWindows(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("filters sparse windows", func(t *testing.T) {
		entries := []WindowTagEntry{
			{WindowStart: base, TagName: "rock", TagCount: 3, WindowTotal: 3},
			{WindowStart: base.Add(WindowSize), TagName: "rock", TagCount: 10, WindowTotal: 10},
		}

		windows := BuildWindows(entries)
		// First window has 3 items (< MinWindowItems=8), should be filtered.
		if len(windows) != 1 {
			t.Fatalf("got %d windows, want 1", len(windows))
		}
		if windows[0].TotalItems != 10 {
			t.Errorf("window total = %d, want 10", windows[0].TotalItems)
		}
	})

	t.Run("normalizes vectors", func(t *testing.T) {
		entries := []WindowTagEntry{
			{WindowStart: base, TagName: "rock", TagCount: 30, WindowTotal: 50},
			{WindowStart: base, TagName: "pop", TagCount: 40, WindowTotal: 50},
		}

		windows := BuildWindows(entries)
		if len(windows) != 1 {
			t.Fatalf("got %d windows, want 1", len(windows))
		}

		// Check normalization: magnitudes should sum to ~1 (unit vector).
		v := windows[0].Vector
		var sumSq float64
		for _, val := range v {
			sumSq += val * val
		}
		if diff := sumSq - 1.0; diff > 0.001 || diff < -0.001 {
			t.Errorf("vector norm² = %v, want ~1.0", sumSq)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		windows := BuildWindows(nil)
		if windows != nil {
			t.Errorf("expected nil, got %v", windows)
		}
	})
}

func TestDetectEras(t *testing.T) {
	t.Run("no windows returns nil", func(t *testing.T) {
		result := DetectEras(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("single window returns single era if enough items", func(t *testing.T) {
		windows := []Window{
			{
				Start:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalItems: 20,
				Vector:     TagVector{"rock": 1.0},
			},
		}
		result := DetectEras(windows)
		if len(result) != 1 {
			t.Fatalf("got %d eras, want 1", len(result))
		}
		if result[0].EndedAt != nil {
			t.Error("single era should have nil EndedAt (ongoing)")
		}
	})

	t.Run("single window with too few items returns nil", func(t *testing.T) {
		windows := []Window{
			{
				Start:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalItems: 5, // below MinItemsPerEra
				Vector:     TagVector{"rock": 1.0},
			},
		}
		result := DetectEras(windows)
		if len(result) != 0 {
			t.Fatalf("got %d eras, want 0", len(result))
		}
	})

	t.Run("detects boundary between distinct windows", func(t *testing.T) {
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		windows := []Window{
			// Era 1: rock-heavy (4 windows = 8 weeks)
			{Start: base, TotalItems: 20, Vector: TagVector{"rock": 0.9, "guitar": 0.1}},
			{Start: base.Add(WindowSize), TotalItems: 20, Vector: TagVector{"rock": 0.85, "guitar": 0.15}},
			{Start: base.Add(2 * WindowSize), TotalItems: 20, Vector: TagVector{"rock": 0.9, "guitar": 0.1}},
			{Start: base.Add(3 * WindowSize), TotalItems: 20, Vector: TagVector{"rock": 0.8, "guitar": 0.2}},
			// Era 2: jazz-heavy (4 windows = 8 weeks)
			{Start: base.Add(4 * WindowSize), TotalItems: 20, Vector: TagVector{"jazz": 0.9, "piano": 0.1}},
			{Start: base.Add(5 * WindowSize), TotalItems: 20, Vector: TagVector{"jazz": 0.85, "piano": 0.15}},
			{Start: base.Add(6 * WindowSize), TotalItems: 20, Vector: TagVector{"jazz": 0.9, "piano": 0.1}},
			{Start: base.Add(7 * WindowSize), TotalItems: 20, Vector: TagVector{"jazz": 0.8, "piano": 0.2}},
		}

		result := DetectEras(windows)
		if len(result) != 2 {
			t.Fatalf("got %d eras, want 2", len(result))
		}

		// First era should have an end time, second should be ongoing.
		if result[0].EndedAt == nil {
			t.Error("first era should have EndedAt set")
		}
		if result[1].EndedAt != nil {
			t.Error("last era should have nil EndedAt (ongoing)")
		}
	})

	t.Run("merges short eras", func(t *testing.T) {
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// One window of jazz between two rock blocks — too short (1 window < 3 weeks).
		windows := []Window{
			{Start: base, TotalItems: 20, Vector: TagVector{"rock": 1.0}},
			{Start: base.Add(WindowSize), TotalItems: 20, Vector: TagVector{"rock": 1.0}},
			{Start: base.Add(2 * WindowSize), TotalItems: 20, Vector: TagVector{"jazz": 1.0}}, // blip
			{Start: base.Add(3 * WindowSize), TotalItems: 20, Vector: TagVector{"rock": 1.0}},
			{Start: base.Add(4 * WindowSize), TotalItems: 20, Vector: TagVector{"rock": 1.0}},
		}

		result := DetectEras(windows)
		// The jazz blip should be merged into a neighbor. The remaining
		// rock-rock boundary (distance ~0) won't cross MinDistinctiveness,
		// so we expect 1 or 2 eras depending on merge direction.
		if len(result) > 2 {
			t.Fatalf("got %d eras, want at most 2 (jazz blip merged)", len(result))
		}
		if len(result) == 0 {
			t.Fatal("got 0 eras, want at least 1")
		}
	})

	t.Run("inactivity gap creates boundary", func(t *testing.T) {
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// Same taste but with a 3-week gap in the middle.
		windows := []Window{
			{Start: base, TotalItems: 20, Vector: TagVector{"rock": 1.0}},
			{Start: base.Add(WindowSize), TotalItems: 20, Vector: TagVector{"rock": 1.0}},
			// Gap of 3 weeks (> InactivityGap)
			{Start: base.Add(2*WindowSize + 3*7*24*time.Hour), TotalItems: 20, Vector: TagVector{"rock": 1.0}},
			{Start: base.Add(3*WindowSize + 3*7*24*time.Hour), TotalItems: 20, Vector: TagVector{"rock": 1.0}},
		}

		result := DetectEras(windows)
		// Despite identical taste, the gap should create 2 eras.
		// But each is only 2 windows (4 weeks) — above MinEraDuration (3 weeks).
		if len(result) != 2 {
			t.Fatalf("got %d eras, want 2 (gap boundary)", len(result))
		}
	})

	t.Run("caps eras per year", func(t *testing.T) {
		base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		// Create 10 distinct eras (alternating rock/jazz every 2 windows).
		var windows []Window
		for i := 0; i < 20; i++ {
			v := TagVector{"rock": 1.0}
			if (i/2)%2 == 1 {
				v = TagVector{"jazz": 1.0}
			}
			windows = append(windows, Window{
				Start:      base.Add(time.Duration(i) * WindowSize),
				TotalItems: 20,
				Vector:     v,
			})
		}

		result := DetectEras(windows)
		if len(result) > MaxErasPerYear {
			t.Errorf("got %d eras, want at most %d", len(result), MaxErasPerYear)
		}
	})
}

func TestTopTagsFromVector(t *testing.T) {
	v := TagVector{"rock": 10, "pop": 5, "jazz": 3, "blues": 1}
	tags := topTagsFromVector(v, 3)

	if len(tags) != 3 {
		t.Fatalf("got %d tags, want 3", len(tags))
	}
	if tags[0].TagName != "rock" {
		t.Errorf("top tag = %q, want rock", tags[0].TagName)
	}
	if tags[0].Weight != 1.0 {
		t.Errorf("top tag weight = %v, want 1.0", tags[0].Weight)
	}
	if tags[1].Weight != 0.5 {
		t.Errorf("second tag weight = %v, want 0.5", tags[1].Weight)
	}
}
