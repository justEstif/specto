# Specto Use Cases Research Report

**Researcher:** UX Research Agent
**Date:** March 14, 2026
**Method:** Analytical use case derivation from data model capabilities, competitive landscape analysis, and behavioral pattern research
**Scope:** Cross-platform media consumption tracker (Spotify, YouTube, Netflix, TikTok; normalized into platform/type/title/creator/consumed_at/duration/time_spent/tags/raw_metadata)

---

## User Personas

### Persona 1: Maya — The Intentional Consumer

**Age:** 28 | **Occupation:** UX designer | **Location:** Portland, OR
**Tech proficiency:** High | **Platforms used:** Spotify (daily), YouTube (daily), Netflix (weekends), podcasts (commute)

**Behavioral patterns:**

- Consumes 4-6 hours of media daily across platforms
- Has tried screen time limits but finds them too blunt ("I don't care about _time_, I care about _what_ I'm spending it on")
- Journals sporadically about what she's reading/watching but can't sustain the habit
- Feels guilt about "junk food" media but has no data to quantify it vs. learning content

**Goals:** Shift her media diet toward more intentional, growth-oriented content without becoming a puritan about entertainment. Wants a feedback loop, not a guilt machine.

**Core tension:** "I know I watched something great on YouTube last month about design systems, but I also know I spent three hours watching reaction videos. I just don't know the ratio."

> "Spotify Wrapped makes me feel seen once a year. I want that feeling continuously — but for everything, not just music."

---

### Persona 2: Dario — The Taste Curator

**Age:** 24 | **Occupation:** Grad student (film studies) | **Location:** Chicago, IL
**Tech proficiency:** Medium-high | **Platforms used:** YouTube (daily), Netflix (3x/week), Spotify (daily), Letterboxd (weekly)

**Behavioral patterns:**

- Curates his media taste as a core part of his identity
- Manually maintains a Notion page of "media I consumed this month" — tedious, always falls behind
- Shares Spotify Wrapped on Instagram stories every year; wishes he could do this for all media
- Uses Letterboxd for movies but frustrated that it doesn't connect to his music or YouTube habits

**Goals:** A living, shareable "media identity page" that auto-updates. Wants others to see his taste profile without him having to manually curate lists.

**Core tension:** "My Letterboxd shows my film taste. My Spotify shows my music taste. But nobody can see the whole picture — that I'm into Japanese cinema AND ambient electronic AND physics YouTube. That combination _is_ me."

> "I want my share page to be like a dating profile but for taste. If someone looks at it and likes the same stuff, we'd probably get along."

---

### Persona 3: Kenji — The Quantified Self Analyst

**Age:** 35 | **Occupation:** Data engineer | **Location:** Seattle, WA
**Tech proficiency:** Very high | **Platforms used:** Spotify (heavy), YouTube (heavy), Netflix, TikTok, podcasts, RSS

**Behavioral patterns:**

- Tracks sleep, exercise, diet, and productivity with various apps and custom dashboards
- Exports Spotify data annually and writes Python scripts to analyze it
- Frustrated that media consumption is the one area of his life he can't instrument properly
- Would use an API or raw data export if available

**Goals:** Complete, structured data about his media consumption. Wants to run his own analyses, spot correlations with other life data (sleep, mood, productivity), and build personal dashboards.

**Core tension:** "I have a Grafana dashboard for my home network, my fitness, my finances. But the most hours-heavy activity in my life — consuming media — is a black box distributed across five apps that won't talk to each other."

> "Just give me the normalized data. I'll do the analysis. But right now I can't even get the data."

---

### Persona 4: Priya — The Content Creator

**Age:** 31 | **Occupation:** YouTuber (tech/science niche, 200K subscribers) | **Location:** Austin, TX
**Tech proficiency:** High | **Platforms used:** YouTube (creator + consumer), Spotify (daily), Netflix (occasional), TikTok (research)

**Behavioral patterns:**

- Watches 2-3 hours of YouTube daily as research for her own content
- Tracks competitor channels manually in a spreadsheet
- Notices that her best video ideas come from cross-pollinating topics (e.g., a physics concept explained through a cooking metaphor she saw on Netflix)
- Consumes TikTok specifically to understand trending formats and topics

**Goals:** Understand her own "input diet" to make better creative output. Spot the cross-platform inspiration chains that lead to her best ideas. Ensure she's not trapped in a content bubble.

**Core tension:** "I watch YouTube for work, but I also watch it for fun. I listen to podcasts for research and music for focus. I need to see the patterns in what I'm absorbing to understand what's shaping my creative output."

> "My best video came from a Netflix documentary I watched the same week I was deep in a Spotify jazz phase. I want to see those collisions, not just the individual streams."

---

## Use Cases

---

### Use Case 1: Attention Audit

**User need / Job to be done:**
"Show me where my attention actually goes — not where I _think_ it goes."

Most people have a distorted self-image of their media consumption. They overestimate time spent on "worthy" content and underestimate passive consumption. This use case provides an honest, automated accounting of attention allocation across platforms, content types, topics, and time periods.

**What data from Specto enables it:**

- `media_items.consumed_at` + `media_items.duration` / `time_spent` — total time by platform, by type
- `media_items.type` — breakdown of music vs. video vs. podcast vs. article
- `tags` (category: `topic`) — what subjects dominate attention
- `tags` (category: `genre`) — entertainment genre distribution
- Temporal aggregation by day/week/month via `consumed_at`

**Concrete example scenario:**
Maya opens her Specto dashboard on a Sunday evening. She selects "last 7 days" and sees: 14 hours on Spotify (mostly ambient/electronic during work), 8 hours on YouTube (split between design tutorials and commentary channels), 4 hours Netflix (one series binge on Saturday). The topic breakdown shows 35% technology, 25% entertainment, 15% design, 25% other. She realizes her YouTube time is evenly split between learning and passive entertainment — she thought it was 80/20 toward learning. She decides to be more deliberate next week.

**Primary persona:** Maya (The Intentional Consumer)

**Feasibility:** Simple queries

- Platform/type/time breakdowns are direct `GROUP BY` aggregations on `media_items`
- Topic distribution uses the existing `media_item_tags` join with `tags.category = 'topic'`
- Already partially implemented in the dashboard's stat cards and platform breakdown blocks
- No LLM inference or embeddings needed

---

### Use Case 2: Cross-Platform Taste DNA

**User need / Job to be done:**
"Show me the invisible threads that connect what I consume across different platforms."

People don't think of their media habits per-platform — they think in terms of interests, moods, and phases. But every platform silos its data. This use case reveals the unified taste profile that emerges when you overlay Spotify listening, YouTube watching, Netflix viewing, and reading habits. What genres, topics, and moods cut across all your platforms?

**What data from Specto enables it:**

- `tags` shared across platforms (same tag taxonomy applied to all items regardless of source)
- `tags.category` = `genre`, `topic`, `mood` — each normalized across platforms
- `media_items.platform` — to show which platforms each tag appears on
- `media_item_tags.confidence` — to filter to high-confidence cross-platform tags

**Concrete example scenario:**
Dario checks his Taste DNA view. Specto shows that "science" appears as a tag across his YouTube (physics channels), his Spotify (science podcasts), and his Netflix (documentaries). His "contemplative" mood tag spans both his ambient music on Spotify and his arthouse film selections on Netflix. He sees a Venn diagram-style view: Spotify-only tags (prog-rock, jazz), YouTube-only tags (gaming, commentary), and the overlap zone (science, technology, contemplative). The overlap zone is his "core identity" — the stuff he gravitates to regardless of medium.

**Primary persona:** Dario (The Taste Curator)

**Feasibility:** Moderate queries

- Requires cross-platform tag aggregation: `GROUP BY tag_id` across multiple platform values
- Overlap detection: tags that appear in items from 2+ distinct platforms
- Straightforward SQL with `HAVING COUNT(DISTINCT platform) >= 2`
- Visual representation (Venn/radar chart) is the UX challenge, not the data query

---

### Use Case 3: Era Detection / Life Chapters

**User need / Job to be done:**
"Label the phases of my life by what I was consuming during them."

People naturally think in "eras" — "my hip-hop phase," "when I was obsessed with true crime," "my breakup playlist period." But these eras are only visible in retrospect and across platforms. This use case automatically detects shifts in consumption patterns and labels them, creating a biographical timeline of media eras.

**What data from Specto enables it:**

- `media_items.consumed_at` — temporal ordering of all consumption
- `tags` (genre, topic, mood) — the "fingerprint" of each time period
- Cross-platform data — eras often span platforms (a "dark" era shows up in music mood, film genre, and YouTube topics simultaneously)
- `media_items.creator` — repeated engagement with specific creators can anchor an era

**Concrete example scenario:**
Kenji views his "Eras" timeline, which Specto has segmented into blocks. He sees: Jan-Mar 2025 labeled "Deep Focus" (dominated by ambient music, programming tutorials, and science documentaries), Apr-Jun 2025 labeled "Social Discovery" (shift to hip-hop, comedy specials, pop culture YouTube), Jul-Sep 2025 labeled "Sci-Fi Immersion" (Netflix sci-fi binge, synth-heavy Spotify, physics YouTube). He clicks into the "Sci-Fi Immersion" era and sees the top items, creators, and tags from that period. He shares the era summary to his profile.

**Primary persona:** Dario (The Taste Curator) / Kenji (The Quantified Self Analyst)

**Feasibility:** LLM inference + algorithmic clustering

- Requires time-windowed tag distributions (sliding window or fixed buckets)
- Change-point detection to find when the tag distribution shifts significantly (statistical — can use simple KL divergence or cosine similarity between adjacent windows)
- Era labeling: either top-tag concatenation (simple) or LLM-generated human-readable era names (richer)
- Medium complexity — the core query is tag distributions over time windows; the clustering and naming add sophistication
- Could start simple (monthly tag summaries) and add algorithmic era detection later

---

### Use Case 4: Media Diet Scorecard

**User need / Job to be done:**
"Am I consuming a balanced diet of media, or am I stuck in a rut?"

Analogous to a nutritional scorecard, this use case evaluates the diversity, balance, and intentionality of someone's media consumption. It answers: How diverse are your topics? How concentrated is your creator set? Are you in an echo chamber? What's the ratio of passive entertainment to active learning?

**What data from Specto enables it:**

- `tags` (topic) — diversity measured via unique topic count and Shannon entropy
- `media_items.creator` — creator concentration (Herfindahl index or top-N share)
- `tags` (genre, mood) — genre diversity and mood range
- `media_items.type` — content type distribution (music/video/podcast/article)
- Classification of content as "passive" vs. "active" using topic tags (entertainment vs. education/science/technology)

**Concrete example scenario:**
Maya opens her weekly scorecard. She sees five dimensions rated on a simple scale:

- **Topic diversity:** 7/10 (she consumed content spanning 12 distinct topics this week)
- **Creator diversity:** 4/10 (60% of her YouTube time went to just 3 channels)
- **Platform balance:** 6/10 (heavily Spotify-weighted, minimal reading)
- **Mood range:** 8/10 (good mix of energetic, chill, contemplative, and funny)
- **Learning ratio:** 5/10 (roughly half entertainment, half educational)

She taps "Creator diversity" to see which 3 channels are dominating her YouTube and decides to intentionally explore new creators this week.

**Primary persona:** Maya (The Intentional Consumer)

**Feasibility:** Moderate queries + light computation

- Diversity metrics (Shannon entropy, Herfindahl index) are computed from standard `GROUP BY` aggregations
- "Learning ratio" requires a mapping from topic tags to a passive/active classification — could be a hardcoded mapping or an LLM-derived one
- Scorecard rendering is the main design challenge
- All data already available; this is primarily an aggregation and scoring layer

---

### Use Case 5: "On This Day" / Nostalgia Machine

**User need / Job to be done:**
"Show me what I was consuming exactly one year ago (or two, or five)."

People are powerfully moved by nostalgia triggers. Facebook's "On This Day" feature is one of its highest-engagement features. For media consumption, the emotional resonance is even stronger — hearing a song or seeing a show title instantly transports you back to that period.

**What data from Specto enables it:**

- `media_items.consumed_at` — exact historical dates across all platforms
- `media_items.title`, `creator`, `platform` — the content itself
- `tags` (mood, genre) — the "flavor" of that past day
- `media_items.url` — deep link back to the original content for re-consumption

**Concrete example scenario:**
Kenji gets a notification (or sees a dashboard card): "1 year ago today, you were deep into jazz. You listened to 'Kind of Blue' by Miles Davis 3 times, watched a documentary about Coltrane on YouTube, and started the Netflix series 'The Eddy.'" He clicks the Miles Davis link and relives the moment. The tag summary for that day shows "jazz, contemplative, nostalgic" — a mood cluster he hasn't visited in months. He decides to revisit it.

**Primary persona:** Dario (The Taste Curator) / Kenji (The Quantified Self Analyst)

**Feasibility:** Simple queries

- `WHERE consumed_at::date = (CURRENT_DATE - INTERVAL '1 year')` (and 2 years, etc.)
- Group items by day, calculate tag summaries for that historical day
- Simple to implement, high emotional payoff
- Optional: LLM-generated narrative summary of the historical day ("You were in a jazz phase...")

---

### Use Case 6: Consumption Routines & Rhythm Mapping

**User need / Job to be done:**
"Show me my daily and weekly media rhythms — what I consume when."

People have unconscious routines: morning podcasts, lunchtime YouTube, evening Netflix, late-night music. Making these routines visible helps with self-understanding and intentional scheduling. This use case creates a "media clock" or heatmap showing what content type/genre/platform maps to what time of day and day of week.

**What data from Specto enables it:**

- `media_items.consumed_at` — hour of day and day of week extraction
- `media_items.type` — what kind of content at each time slot
- `media_items.platform` — which platform at each time slot
- `tags` (mood) — what emotional register dominates each time slot
- `tags` (genre, topic) — what subjects map to which times

**Concrete example scenario:**
Priya opens her Rhythm Map. A heatmap grid (24 hours x 7 days) shows her media patterns:

- **6-8 AM weekdays:** Podcasts (science/technology) — her research routine
- **9 AM-12 PM weekdays:** Spotify (ambient/electronic) — focus music while editing
- **12-1 PM weekdays:** YouTube (commentary/pop-culture) — lunch break consumption
- **8-11 PM weekdays:** Netflix (drama, thriller) — wind-down
- **Saturday afternoon:** YouTube deep-dives (technology/science) — creative research sessions
- **Sunday morning:** Spotify (jazz/classical) — slow morning

She notices that her Friday evening consumption shifts heavily toward "energetic" and "funny" mood tags — an unconscious pattern she wasn't aware of. She also spots that her most productive creative weeks correlate with heavier morning podcast consumption.

**Primary persona:** Kenji (The Quantified Self Analyst) / Maya (The Intentional Consumer)

**Feasibility:** Simple-to-moderate queries

- `EXTRACT(DOW FROM consumed_at)` and `EXTRACT(HOUR FROM consumed_at)` with `GROUP BY`
- Heatmap cells colored by dominant type, platform, or mood
- All data available in `media_items` + tags
- The visualization (heatmap) is the main challenge, not the data

---

### Use Case 7: Input-Output Correlation (for Creators)

**User need / Job to be done:**
"What am I consuming that feeds my best creative work?"

Content creators know their output is shaped by their input, but they can't trace the connection. This use case helps creators see what they were consuming in the days and weeks before they produced their best work — looking for patterns between media diet and creative output.

**What data from Specto enables it:**

- `media_items.consumed_at` + tags — the full media diet for any time window
- `tags` (topic) — cross-platform topic clustering that reveals "research binges"
- External signal: the user's own content output (YouTube upload dates, blog post dates) — either manually entered or imported
- Temporal proximity: what was consumed in the 7-14 days before a creative output

**Concrete example scenario:**
Priya marks her last 10 YouTube uploads in Specto with their publish dates and performance metrics (views in first 48 hours). Specto correlates her consumption in the 2 weeks before each upload. Pattern: her top 3 performing videos were all preceded by weeks where she consumed content across 3+ platforms on the same topic (e.g., quantum computing appeared in YouTube videos, a Spotify podcast, and a Netflix documentary). Her worst-performing videos were preceded by narrow, single-platform research. Insight: cross-pollination in her consumption predicts creative quality.

**Primary persona:** Priya (The Content Creator)

**Feasibility:** Moderate queries + manual input (MVP), LLM inference (advanced)

- Core query: tag distribution in a time window before a marked event
- Requires a way to mark "output events" (could be as simple as a manual log or future integration)
- Correlation analysis is straightforward once both data sets exist
- Advanced: LLM-generated insight summaries ("Your best content followed diverse research periods")

---

### Use Case 8: Topic Obsession Tracker

**User need / Job to be done:**
"Show me when I get obsessed with something — how deep I go, how long it lasts, and when it fades."

People go through intense interest phases — a week of binge-watching Korean drama, a month of obsessing over a new music genre, a deep dive into AI research across every platform. This use case visualizes these obsession arcs: onset, peak, duration, and fadeout.

**What data from Specto enables it:**

- `tags` (topic, genre) — tracking specific tag frequency over time
- `media_items.consumed_at` — time-series data for tag frequency
- Cross-platform data — obsessions often span platforms (watching K-drama on Netflix, listening to K-pop on Spotify, watching K-drama analysis on YouTube)
- `media_items.duration` / `time_spent` — intensity measurement

**Concrete example scenario:**
Dario selects the tag "sci-fi" and sees a timeline chart. In March 2025, sci-fi consumption spiked from near-zero to 8 items/week — triggered by starting "Severance" on Netflix. The obsession spread to Spotify (synthwave playlists), YouTube (sci-fi analysis channels), and even podcasts. Peak was in April (15 items/week across 3 platforms). By June, it tapered to 2 items/week. Total obsession duration: ~3 months. Specto labels it "Sci-Fi Arc: March-June 2025" and shows the cascade across platforms.

**Primary persona:** Dario (The Taste Curator) / Kenji (The Quantified Self Analyst)

**Feasibility:** Moderate queries

- Time-series aggregation of specific tag counts: `COUNT(*) GROUP BY date_trunc('week', consumed_at)` filtered by tag
- Cross-platform spread: same query but also `GROUP BY platform`
- Spike/peak detection: simple statistical methods (rolling average, threshold)
- Mostly SQL + light application logic

---

### Use Case 9: Curated Media Identity Page

**User need / Job to be done:**
"Let me show the world who I am through what I consume — not a raw data dump, but a curated identity."

This goes beyond stats sharing. It's about self-expression through consumption patterns. The share page becomes a "taste resume" — a way to signal cultural identity, intellectual interests, and aesthetic sensibility to friends, dates, or professional contacts.

**What data from Specto enables it:**

- All `share_profiles` block types (Top Genres, Mood Profile, Top Creators, Currently Into, etc.)
- `tags` — the aggregated taste profile that powers each block
- `media_items` marked as "Recent Favorites" — user-curated highlights
- Exclusion controls — platforms, tags, and items the user chooses to hide

**Concrete example scenario:**
Dario sets up his share page at `specto.app/share/dario`. He enables Top Genres (showing his eclectic mix of arthouse cinema and electronic music), Mood Profile (mostly "contemplative" and "dreamy"), Top Creators (a mix of filmmakers and musicians), and a "Currently Into" blurb: "Revisiting the French New Wave and pairing it with Erik Satie." He excludes Netflix from the platform mix (he doesn't want people to see how much reality TV he watches). He shares the link in his Twitter bio and on dating apps. His share page auto-updates weekly as his consumption shifts.

**Primary persona:** Dario (The Taste Curator)

**Feasibility:** Already designed / simple queries

- The sharing system is already fully designed in `docs/sharing.md`
- Block-based composition with opt-in, exclusions, and preview
- All queries are standard aggregations already needed for the dashboard
- Primary work is UX polish and the "identity page" aesthetic

---

### Use Case 10: Seasonal Pattern Discovery

**User need / Job to be done:**
"Do I consume differently in winter vs. summer? During holidays? On vacations?"

Long-term users accumulate enough data to reveal seasonal patterns they never consciously noticed — heavier music consumption in winter, documentary binges during holidays, mood shifts across seasons, genre preferences that cycle annually.

**What data from Specto enables it:**

- `media_items.consumed_at` — multi-year temporal data
- `tags` (mood, genre, topic) — tag distributions bucketed by season/month
- `media_items.duration` / `time_spent` — volume patterns over seasons
- `media_items.platform` — platform usage seasonality

**Concrete example scenario:**
Kenji, after 18 months on Specto, opens a "Seasons" view. He sees:

- **Winter (Dec-Feb):** Mood skews heavily "melancholic" and "contemplative." Genre: ambient, classical, drama. Total media hours peak — he consumes 30% more in winter.
- **Summer (Jun-Aug):** Mood shifts to "energetic" and "uplifting." Genre: pop, comedy, action. Lower total volume but more social/shareable content.
- **November specifically:** Every year, a spike in "nostalgic" mood tags and classic album revisits — a pattern he never consciously recognized.

He cross-references this with his mood journal data in another app and confirms: his winter media consumption mirrors (and perhaps reinforces) seasonal mood shifts.

**Primary persona:** Kenji (The Quantified Self Analyst)

**Feasibility:** Simple queries (with sufficient data)

- `EXTRACT(MONTH FROM consumed_at)` with `GROUP BY` on tags, platform, type
- Requires 12+ months of data to be meaningful
- Year-over-year comparison: same month across years
- No LLM or embeddings needed — pure aggregation
- The cold start problem (new users don't have enough data yet) is the main UX challenge

---

### Use Case 11: Echo Chamber / Diversity Alert

**User need / Job to be done:**
"Am I trapped in a filter bubble? Are my media inputs becoming more or less diverse over time?"

Platforms optimize for engagement, which tends to narrow the content funnel over time. Users who care about intellectual breadth want a warning system that detects when their consumption is becoming too homogeneous — and credits them when they're genuinely exploring.

**What data from Specto enables it:**

- `tags` (topic, genre) — unique tag count and distribution entropy over time
- `media_items.creator` — creator concentration (are you only watching 5 channels?)
- `media_items.platform` — platform concentration (are you only using one app?)
- Temporal trends — is diversity increasing or decreasing month-over-month?

**Concrete example scenario:**
Maya sees a dashboard alert: "Your topic diversity dropped 25% this month. 70% of your YouTube consumption is from 3 channels covering the same topic (design systems). You haven't consumed any science or politics content in 3 weeks — topics you usually engage with." She also sees a positive signal: "Your music diversity increased — you explored 4 new genres this month through Spotify Discover." She clicks into the alert and sees a trend line of her topic diversity score over the past 6 months, with the current dip highlighted.

**Primary persona:** Maya (The Intentional Consumer) / Priya (The Content Creator)

**Feasibility:** Moderate computation

- Shannon entropy or Simpson's diversity index computed from tag distributions
- Creator concentration via Herfindahl-Hirschman Index (sum of squared market shares)
- Month-over-month comparison of diversity scores
- Alert threshold: configurable (e.g., "notify me if diversity drops >20%")
- All based on existing tag and item data — no new data sources needed

---

### Use Case 12: Time Capsule Generator

**User need / Job to be done:**
"Package my media consumption for a specific period into a shareable, revisitable artifact."

Beyond ongoing profiles, users want to create discrete "time capsule" artifacts — a summary of a trip, a semester, a relationship, a year. Something they can look back on or share that captures "what I was into during that period."

**What data from Specto enables it:**

- `media_items` filtered by arbitrary date range — all items in the capsule period
- `tags` — the taste fingerprint of that period
- `media_items.creator` — top creators of the period
- `media_items.url` — links back to the actual content
- Cross-platform data — the capsule spans all platforms, giving a complete picture

**Concrete example scenario:**
Dario just finished a semester abroad in Tokyo. He creates a time capsule for "Sep 2025 - Jan 2026." Specto generates a summary:

- **Top genres:** anime, jazz, ambient, drama
- **Top creators:** Studio Ghibli (Netflix), Nujabes (Spotify), Abroad in Japan (YouTube)
- **Mood profile:** contemplative (40%), peaceful (25%), nostalgic (20%), playful (15%)
- **Stats:** 342 items consumed, 180 hours total, 4 platforms
- **Standout items:** The 5 most-consumed items (by repeat plays/views)

He gives it a title ("Tokyo Semester"), writes a short personal note, and shares it as a standalone page. His friends can see it and click through to the content.

**Primary persona:** Dario (The Taste Curator)

**Feasibility:** Moderate (queries + new UI surface)

- Data queries are identical to existing dashboard/share queries with a custom date range
- Requires a new "capsule" entity (date range, title, note, shareable URL)
- Could be implemented as a special-case share profile with a fixed date range
- LLM-generated narrative summary optional but compelling ("Your Tokyo semester was dominated by...")

---

## Use Case Summary Matrix

| #   | Use Case                              | Primary Persona | Data Complexity | Technical Feasibility          | Emotional Payoff                |
| --- | ------------------------------------- | --------------- | --------------- | ------------------------------ | ------------------------------- |
| 1   | Attention Audit                       | Maya            | Low             | Simple queries                 | Medium — honest mirror          |
| 2   | Cross-Platform Taste DNA              | Dario           | Medium          | Moderate queries               | High — identity revelation      |
| 3   | Era Detection / Life Chapters         | Dario / Kenji   | Medium          | LLM + clustering               | Very high — biographical        |
| 4   | Media Diet Scorecard                  | Maya            | Medium          | Moderate queries + computation | Medium — actionable             |
| 5   | "On This Day" / Nostalgia Machine     | Dario / Kenji   | Low             | Simple queries                 | Very high — emotional           |
| 6   | Consumption Routines & Rhythm Mapping | Kenji / Maya    | Low-Medium      | Simple-moderate queries        | Medium — self-knowledge         |
| 7   | Input-Output Correlation (Creators)   | Priya           | Medium-High     | Moderate + manual input        | High — professional value       |
| 8   | Topic Obsession Tracker               | Dario / Kenji   | Medium          | Moderate queries               | High — pattern recognition      |
| 9   | Curated Media Identity Page           | Dario           | Low             | Already designed               | Very high — self-expression     |
| 10  | Seasonal Pattern Discovery            | Kenji           | Low             | Simple queries (needs time)    | High — surprise discovery       |
| 11  | Echo Chamber / Diversity Alert        | Maya / Priya    | Medium          | Moderate computation           | High — corrective               |
| 12  | Time Capsule Generator                | Dario           | Medium          | Moderate (new UI surface)      | Very high — nostalgia + sharing |

---

## Prioritization Recommendation

### Tier 1: Ship with MVP (high impact, data already supports it)

1. **Attention Audit** (#1) — The core value proposition. If Specto can't show you where your attention goes, nothing else matters. All data already available through existing dashboard queries.

2. **Curated Media Identity Page** (#9) — Already designed in the sharing system. The differentiation from competitors. Drives word-of-mouth growth (every share page is marketing).

3. **"On This Day" / Nostalgia Machine** (#5) — Trivial to implement, massive emotional engagement. Requires only historical data and a date comparison query. Drives daily return visits.

### Tier 2: Fast follows (moderate effort, strong signal)

4. **Consumption Routines & Rhythm Mapping** (#6) — Makes the invisible visible. Heatmap is a compelling visualization. Drives the "self-awareness" narrative.

5. **Cross-Platform Taste DNA** (#2) — The killer insight that no single-platform tool can offer. Moderate query complexity but high "aha" potential.

6. **Topic Obsession Tracker** (#8) — Engaging, visual, and unique. Good content for sharing ("look at my sci-fi arc").

### Tier 3: Growth features (require more data or complexity)

7. **Media Diet Scorecard** (#4) — Needs careful design to avoid being judgmental. Best after users have enough data for meaningful scores.

8. **Echo Chamber / Diversity Alert** (#11) — Powerful but potentially anxiety-inducing. Needs thoughtful UX to be motivating rather than guilt-tripping.

9. **Seasonal Pattern Discovery** (#10) — Requires 12+ months of data. Ship as a reward for long-term users.

10. **Era Detection / Life Chapters** (#3) — High technical complexity (clustering + naming). Ship after simpler time-based views prove the concept.

### Tier 4: Niche but compelling (specific personas)

11. **Time Capsule Generator** (#12) — New UI surface needed. Best as a natural extension of the share system.

12. **Input-Output Correlation** (#7) — Niche (creator persona only). Requires manual output logging or integration. High value for the right user but small addressable audience initially.

---

## Key Insights from This Analysis

1. **The data model already supports most of these use cases.** The normalized schema with cross-platform tags is the enabling infrastructure. Most use cases are aggregation queries against data that Specto already collects and enriches.

2. **Cross-platform correlation is the unique moat.** No competitor can show how your music listening, video watching, and reading habits interconnect. Every use case that leverages cross-platform data (#2, #3, #6, #7, #8, #10) is differentiated.

3. **Temporal data is underexploited.** The `consumed_at` timestamp is the most powerful field in the schema. Routines, eras, seasons, obsession arcs, and nostalgia features all derive from temporal analysis of the same underlying data.

4. **The "identity" use case may drive adoption more than the "analytics" use case.** Dario (taste curator) is likely a larger persona than Kenji (quantified self). People share identity artifacts; they don't share analytics dashboards. The share page is both the product and the growth engine.

5. **Cold start is the biggest UX risk.** Most compelling use cases require weeks or months of data. The MVP must deliver value from day one (Attention Audit with even a single platform connected) while building toward the richer temporal features over time.
