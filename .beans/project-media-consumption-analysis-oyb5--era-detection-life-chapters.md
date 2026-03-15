---
# project-media-consumption-analysis-oyb5
title: Era detection / life chapters
status: in-progress
type: feature
priority: low
created_at: 2026-03-14T21:04:53Z
updated_at: 2026-03-15T00:16:00Z
parent: project-media-consumption-analysis-doqg
---

Automatically detect shifts in consumption patterns and label them as 'eras'. Requires time-windowed tag distributions, change-point detection (KL divergence or cosine similarity), and LLM-generated era names. Highest emotional payoff but highest complexity. Persona: Dario/Kenji. Ref: docs/research/ux-use-cases.md #3


## Research Findings (2026-03-14)

### Key Insight: Nobody Has Solved This

No product does cross-platform media era detection. Spotify is the closest with Music Evolution (2024) — up to 3 phases/year with AI labels — but they **partially retreated in 2025** after user backlash against meaningless labels like "Pink Pilates Princess Strut Pop."

### The Core Tension

**Algorithmic precision vs. human meaning.** Statistically rigorous approaches (PELT, Bayesian CPD) produce defensible boundaries but unintuitive labels. Fixed calendar windows are predictable but miss the point of what an "era" is.

### Spotify's Cautionary Tale

- 2024: Launched Music Evolution with AI-generated era names → backlash (labels felt random/patronizing)
- 2025: Pulled back from era labels, pivoted to social features (Wrapped Party)
- Lesson: **Users want to recognize themselves in the label. Imposed labels destroy trust.**

### Guardrails That Make Eras Meaningful

| Constraint | Value | Rationale |
|---|---|---|
| Min era duration | 3 weeks | Shorter = moods, not eras |
| Max eras/year | 4-5 | More = noise |
| Min items consumed | 15-20 per era | Below this, taste vector too sparse |
| Min distinctiveness | Cosine distance > 0.25 from adjacent | Must *feel* different |
| Min data history | 8 weeks | Don't attempt on new users |
| Gap handling | 2+ weeks inactive = natural boundary | |

### Three Promising Positioning Angles

1. **"Your Media Autobiography"** — Always-on cross-platform timeline where eras emerge visually. Users name their own eras with AI suggestions. Avoids the Spotify trap.

2. **"Shift Detection"** — Real-time alerts when taste changes ("you've shifted away from anime this quarter for the first time in 2 years"). Eras accumulate naturally from detected shifts. Forward-looking, not retrospective.

3. **"Media Seasons"** — Cyclical pattern detection across years ("every October you pivot to horror across all media"). More robust than arbitrary eras. Requires 12+ months of data.

### Recommended Implementation Phases

- **Phase 1 (MVP)**: Fixed monthly windows with taste profile characterization. No era detection yet — just show what you consumed when.
- **Phase 2**: Sliding-window cosine similarity to detect boundaries. Apply hard guardrails. Propose eras for user confirmation/naming.
- **Phase 3**: LLM-generated era names/narratives + cross-media era convergence.

### Competitive White Space

Cross-platform era detection has **no direct competitor**. Trakt/Simkl track media but offer only stats. Exist.io correlates but doesn't name eras. The cross-media signal ("anime + J-pop + Persona 5 in the same month") is Specto's unique moat.

### Key Risks

1. **Meaningless labels** (Spotify's mistake) — mitigate with user-driven naming
2. **Sparse data** — fall back to calendar stats for low-data users
3. **Privacy/creepiness** — "we noticed you were going through something in March" crosses a line
4. **Retroactive instability** — new data shifts old era boundaries; snapshot/freeze eras once a period closes

### Decision Needed

- [x] ~~Choose positioning angle~~ → **Media Autobiography** (primary) + **Shift Detection** (Phase 3 layer). Media Seasons deferred to post-v1.
- [x] Decide: per-media-type eras or holistic cross-media eras for MVP? → **Per-media-type** for MVP. Design data model with enrichment tags so cross-media eras and seasonal detection are future queries over the same data.
- [x] Decide: user-named eras, AI-named, or hybrid? → **Hybrid** (AI suggests, user confirms/edits)


## Summary of Changes

### Schema (migration 014)
- `eras` table: stores detected eras per user/media_type with suggested/confirmed/dismissed status
- `era_tags` table: weighted tags that characterize each era (enables future cross-media queries)

### Algorithm (`internal/core/era.go`)
- Sliding-window cosine similarity on biweekly tag vectors
- Guardrails: min 3 weeks, max 5/year, min 15 items, min 0.25 distinctiveness
- Inactivity gaps (2+ weeks) create automatic boundaries
- Short eras merged into most similar neighbor
- Excess eras pruned by keeping most distinctive boundaries

### Store (`internal/core/store/era_store.go`)
- Full EraStore implementation with pgx/sqlc
- TagVectorByWindow query for biweekly tag distributions
- CRUD for eras and era_tags

### Worker (`internal/core/era_worker.go`)
- Background goroutine (1-hour interval) following EnrichmentWorker pattern
- `DetectUserEras()` method for on-demand detection
- Wired into App struct and started in main.go

### Tests (`internal/core/era_test.go`)
- CosineDistance: identical, orthogonal, partial overlap, empty vectors
- BuildWindows: sparse filtering, normalization
- DetectEras: boundaries, merging, inactivity gaps, per-year cap
