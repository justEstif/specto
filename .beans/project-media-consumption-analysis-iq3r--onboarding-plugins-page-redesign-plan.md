---
# project-media-consumption-analysis-iq3r
title: Onboarding & Plugins Page Redesign Plan
status: in-progress
type: epic
priority: normal
created_at: 2026-03-14T22:06:38Z
updated_at: 2026-03-14T22:15:15Z
---

UX research-backed plan for improving the new user onboarding experience and plugins connection page to motivate users to connect their services and import data.


## Research Findings

### Current State Problems
1. **No onboarding flow exists.** After signup, users land on an empty dashboard with zero guidance.
2. **Empty states are text-only** — no illustrations, no CTAs, no links to the plugins page.
3. **The plugins page is buried** — it's a navbar link, not surfaced in empty states or post-signup.
4. **No "why" communicated** — users don't see what they'll get before going through the effort of connecting.
5. **All empty states look identical** — generic gray boxes with faint text. No personality, no motivation.

### Key UX Research Insights
- **80% of users abandon apps** where onboarding feels like work with no visible payoff.
- **"Show the output before asking for input"** is the single highest-converting pattern for data-import apps.
- **Endowed progress effect** — starting a progress bar at 25% (account already created!) boosts completion by 55% (LinkedIn data).
- **Zeigarnik effect** — people are driven to complete unfinished tasks. A visible checklist creates this tension.
- **Loss aversion is 2x stronger than gains** — showing blurred previews of what they're "missing" outperforms describing what they'll "get."
- **One connection = usable product** — never require multiple connections before showing value.
- **Celebrate the import itself** — live-counting stats during import ("Found 3,247 songs...") turns waiting into a reveal.

---

## Onboarding Improvement Plan

### Phase 1: Post-Signup Welcome Flow (High Priority)

**Problem**: Users register → land on empty dashboard → don't know what to do.

**Solution**: A 3-step welcome flow triggered on first login only.

#### Step 1: "What Do You Track?" (Personalization)
- Full-page welcome screen after first signup/login
- "What media do you consume most?" — visual cards for: Music, Movies & TV, Anime, Everything
- Single selection auto-highlights the recommended first plugin
- Skip option always visible ("I'll explore on my own")
- **Design direction**: Editorial/magazine feel — large typography, generous whitespace, no cards-on-cards

#### Step 2: "Connect Your First Source" (Guided Connection)
- Based on selection, show ONE recommended plugin prominently
- **Blurred preview dashboard** behind the connection prompt — "This is what your profile will look like"
- For OAuth plugins: single "Connect with Spotify" button with security reassurance ("We only read your listening history")
- For file-import plugins: condensed version of the existing export guide, inline
- Progress indicator: "Step 2 of 3"

#### Step 3: "Your First Insights" (Immediate Value)
- During import: animated counter showing live stats ("Importing... 1,247 songs found... 89 artists... Your top genre: Electronic")
- After import completes: celebration moment (subtle, not patronizing — e.g., the dashboard fading in with their real data, stats animating up from 0)
- Redirect to dashboard with real data populated
- Persistent but dismissible "Getting Started" checklist appears in sidebar or dashboard

---

### Phase 2: Empty States Redesign (High Priority)

**Problem**: Every empty state is identical gray text. No motivation, no CTAs.

**Solution**: Context-aware empty states that teach and motivate.

#### Dashboard Empty States
- **Activity chart**: Blurred preview of a sample chart + "Connect a source to see your listening patterns" + [Connect a source] button
- **Recent items**: Show 2-3 sample items (faded/blurred) with overlay "Your recent media will appear here" + CTA
- **Tags section**: Blurred tag cloud preview + "Tags are automatically generated from your media"
- **Platform breakdown**: Blurred pie chart + "See how your consumption splits across platforms"

#### Design Principles for Empty States
- Each empty state shows a **blurred or faded preview** of what it will look like with data (loss aversion)
- Each includes a **specific CTA button** that links directly to `/plugins`
- Use the plugin's logo/icon in the CTA when possible ("Connect Spotify →")
- Keep copy concise: what will be here + why it matters + how to get it (3 lines max)

---

### Phase 3: Plugins Page Redesign (High Priority)

**Problem**: The plugins page is functional but doesn't motivate. It's a list of cards without communicating the value of each connection.

#### 3a: Value-First Plugin Cards
Current cards show: logo, name, auth type, connect button.
Redesigned cards show:
- **What you'll unlock**: "Track your listening history, discover genre patterns, see your most-played artists"
- **Data preview**: "Most users import 5,000+ songs" or "Typically imports 2 years of watch history"
- **Time estimate**: "Takes about 3 minutes" (for file import) or "One click" (for OAuth)
- **Social proof**: "Connected by 73% of Specto users" (once there's data, or omit for now)

#### 3b: Guided Export Instructions (Improve Existing)
The current file import guides are good but buried in a collapsible. Improvements:
- Make the export guide a **dedicated expandable panel** per platform, not nested inside a unified upload card
- Add **estimated time** for each step ("This usually takes 5-10 minutes")
- Add **screenshot placeholders** or icons for each step to break up the wall of text
- Add a **"What you'll get"** section showing sample insights from that data source
- Consider a **video walkthrough link** for complex exports (Spotify GDPR, YouTube Takeout)

#### 3c: Connection Progress & Cross-Platform Hooks
- After connecting first plugin, show a **"Your media profile is X% complete"** bar
- Suggest next connection with cross-platform value: "You're tracking music — connect YouTube to discover how your video taste overlaps"
- Each new connection should visibly **unlock new dashboard sections** or insights

---

### Phase 4: Progressive Discovery (Medium Priority)

**Problem**: After initial setup, there's no guidance toward deeper features.

#### Contextual Feature Discovery
- **First time viewing dashboard**: Subtle tooltip on the time-range filter ("Try switching to 'This Year' to see annual trends")
- **After 50+ items imported**: "Did you know? You can share your profile publicly" — link to settings
- **After first week**: "Your Weekly Digest is ready" (if/when implemented)

#### Persistent Getting-Started Checklist
A non-intrusive checklist widget (collapsible, lives on dashboard):
- [x] Create account (pre-checked — endowed progress)
- [ ] Connect your first source
- [ ] Import your media history
- [ ] Explore your dashboard
- [ ] Share your profile
- Dismissable after completing 3+ items or via "x" button
- Stored in localStorage or user preferences

---

### Phase 5: Import Experience Enhancement (Medium Priority)

**Problem**: File imports show a generic spinner/redirect. OAuth syncs are opaque.

#### Live Import Stats
- During file upload processing: show a progress indicator with live counters
- "Processing your Spotify history... 2,341 of ~8,000 songs imported"
- After completion: summary card — "Import complete! 8,247 songs, 412 artists, spanning Jan 2019 - Mar 2026"

#### Sync Status Feedback
- After OAuth sync: "Synced 47 recently played tracks from Spotify"
- Show what's new since last sync
- If sync found nothing new: "Already up to date! Last synced 2 hours ago"

---

## Implementation Priorities

### Batch 1 (Highest Impact, Do First)
- [ ] Post-signup welcome flow (Steps 1-3)
- [ ] Empty states redesign with blurred previews and CTAs
- [ ] Plugin cards value-first redesign

### Batch 2 (High Impact)
- [ ] Getting-started checklist widget
- [ ] Export guide improvements (time estimates, better layout)
- [ ] Live import stats during file upload

### Batch 3 (Medium Impact, Progressive)
- [ ] Cross-platform connection hooks
- [ ] Contextual feature discovery tooltips
- [ ] Profile completion percentage

---

## Design Direction (Frontend Design Skill)

### Aesthetic
- **Editorial/magazine** feel — Specto already has custom themes (specto-dark, specto-light)
- Large, confident typography for welcome flow headings
- Generous whitespace — let the value props breathe
- Blurred previews use CSS `filter: blur(8px)` on real component renders with sample data
- No cards-inside-cards. Flat hierarchy with strong typographic contrast
- Subtle motion: stats counting up (CSS counter animation or JS), dashboard sections fading in with staggered timing

### Anti-Patterns to Avoid
- No generic "Welcome! 👋" screens with confetti
- No mandatory tutorials that block product access
- No identical card grids for plugin selection — vary visual weight by recommendation
- No modals for onboarding steps — use full-page or inline panels
- No dark-mode-with-neon AI aesthetic — respect existing Specto theme system

### Empty State Design Language
- Blurred preview of actual component (not illustration/icon)
- Single line of purposeful copy
- One CTA button using `btn-primary`
- Consistent but not monotonous — each empty state previews its specific content type


---

## Wireframes (aligned with ui-design.md conventions)

### Welcome Flow — Step 1: "What Do You Track?"

**Route:** `/welcome` (redirect here on first login, stored in user row `onboarded` bool)
**Auth:** Required

```
┌─────────────────────────────────────────────────────────────┐
│ navbar: [Specto]                                [▼ avatar]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│  Welcome to Specto                   (text-display, 5xl)    │
│  ─────────────────                                          │
│  What do you consume most?           (text-base-content/60) │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   ♫          │  │   ▶          │  │   ◈          │      │
│  │              │  │              │  │              │      │
│  │   Music      │  │  Movies &    │  │   Anime &    │      │
│  │              │  │  TV          │  │   Manga      │      │
│  │  Spotify,    │  │  YouTube,    │  │  AniList,    │      │
│  │  Last.fm     │  │  Netflix     │  │  Crunchyroll │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                             │
│  ┌────────────────────────────────────────────────────┐     │
│  │                    Everything                       │     │
│  │        I track across all media types               │     │
│  └────────────────────────────────────────────────────┘     │
│                                                             │
│                          ~ or ~                             │
│                                                             │
│       I'll explore on my own →       (text-link, muted)     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Interaction:** Clicking a card sets preference + navigates to Step 2 with recommended plugin pre-selected. "I'll explore on my own" skips to dashboard, sets `onboarded=true`.

### Welcome Flow — Step 2: "Connect Your First Source"

**Route:** `/welcome/connect`

```
┌─────────────────────────────────────────────────────────────┐
│ navbar: [Specto]                                [▼ avatar]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Step 2 of 3                         (text-base-content/40) │
│                                                             │
│  Connect your first source           (text-display, 3xl)    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                                                     │    │
│  │  Recommended for you:                               │    │
│  │                                                     │    │
│  │  ┌─────────────────────────────────────────────┐    │    │
│  │  │ ♫ Spotify                                   │    │    │
│  │  │                                             │    │    │
│  │  │ What you'll get:                            │    │    │
│  │  │ • Your full listening history               │    │    │
│  │  │ • Genre patterns & top artists              │    │    │
│  │  │ • Cross-platform insights                   │    │    │
│  │  │                                             │    │    │
│  │  │ ┌───────────────────────┐   Takes ~5 min    │    │    │
│  │  │ │  Upload history file  │   (file import)   │    │    │
│  │  │ └───────────────────────┘                   │    │    │
│  │  │                                             │    │    │
│  │  │ ── or ──                                    │    │    │
│  │  │                                             │    │    │
│  │  │ ┌───────────────────────┐   One click       │    │    │
│  │  │ │  Connect with OAuth   │   (last 50 only)  │    │    │
│  │  │ └───────────────────────┘                   │    │    │
│  │  │                                             │    │    │
│  │  │ 🔒 We only read your history. Never posts.  │    │    │
│  │  └─────────────────────────────────────────────┘    │    │
│  │                                                     │    │
│  │  Other sources:                                     │    │
│  │  YouTube · AniList · (show as text links)           │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│       Skip for now →                 (text-link, muted)     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Welcome Flow — Step 3: "Your First Insights" (post-import)

**Route:** `/welcome/complete` (after successful import/OAuth callback)

```
┌─────────────────────────────────────────────────────────────┐
│ navbar                                                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│  You're all set.                     (text-display, 5xl)    │
│                                                             │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐              │
│  │ 3,247      │ │ 89         │ │ 14 hrs     │              │
│  │ songs      │ │ artists    │ │ total time │              │
│  │ imported   │ │ discovered │ │ tracked    │              │
│  └────────────┘ └────────────┘ └────────────┘              │
│  (stat counters animate up from 0)                          │
│                                                             │
│  Your top genre: Electronic          (text-primary)         │
│  Spanning: Jan 2019 – Mar 2026                              │
│                                                             │
│  ┌────────────────────────────┐                             │
│  │  Go to your dashboard →   │  (btn-primary, lg)           │
│  └────────────────────────────┘                             │
│                                                             │
│  Connect another source →            (text-link)            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Empty State Pattern (Dashboard example)

```
┌─────────────────────────────────────────────────────────────┐
│  Activity                                                   │
│  ─────────                                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                                                     │    │
│  │  ▁▃▅▇▆▄▃▅▇█▆▅▃▂▁▃▅▆▇█▆▅▃▂▁▃▅▇▆▄  ← filter:      │    │
│  │  M  T  W  T  F  S  S  M  T  W  T     blur(8px)    │    │
│  │                                       opacity-40   │    │
│  │  ┌──────────────────────────────────────────────┐   │    │
│  │  │  Your listening patterns will appear here.   │   │    │
│  │  │  ┌─────────────────────────┐                 │   │    │
│  │  │  │  Connect a source →    │  (btn-primary)   │   │    │
│  │  │  └─────────────────────────┘                 │   │    │
│  │  └──────────────────────────────────────────────┘   │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
```

**Implementation:** Render the actual chart component with hardcoded sample data, apply `filter: blur(8px) opacity: 0.4` via CSS, overlay the CTA using `absolute inset-0` positioning.

### Plugins Page — Value-First Card Redesign

```
┌─────────────────────────────────────────────────────────────┐
│  Plugins                               (vt-heading)         │
│                                                             │
│  Your media profile              ████████████░░░░  62%      │
│  ──────────────────              (progress bar)             │
│  Connect more sources to unlock deeper insights.            │
│                                                             │
│  Connected (2)                                              │
│  ─────────────                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │ ♫ Spotify                          Connected ●     │     │
│  │   1,880 items · Last synced 3m ago                 │     │
│  │   ┌──────────┐  ┌────────────┐                     │     │
│  │   │ Sync now │  │ Disconnect │                     │     │
│  │   └──────────┘  └────────────┘                     │     │
│  └────────────────────────────────────────────────────┘     │
│                                                             │
│  Available                                                  │
│  ─────────                                                  │
│  ┌────────────────────────────────────────────────────┐     │
│  │ ▶ YouTube                                          │     │
│  │                                                    │     │
│  │ What you'll unlock:                                │     │
│  │ Watch history, video genres, creator patterns      │     │
│  │                                                    │     │
│  │ ┌───────────────────┐  ~5 min via Google Takeout   │     │
│  │ │ Upload history    │                              │     │
│  │ └───────────────────┘                              │     │
│  │ ┌───────────────────┐  One click (recent only)     │     │
│  │ │ Connect with OAuth│                              │     │
│  │ └───────────────────┘                              │     │
│  │                                                    │     │
│  │ ▼ How to export your YouTube data                  │     │
│  │   (collapsible guide with time estimates per step) │     │
│  └────────────────────────────────────────────────────┘     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Getting-Started Checklist (Dashboard widget)

```
┌─────────────────────────────────────────────────────────────┐
│  Getting started               ████████░░░░  40%    [×]     │
│  ─────────────────                                          │
│  ✓ Create your account                                      │
│  ✓ Connect your first source                                │
│  ○ Explore your dashboard                                   │
│  ○ Share your profile                                       │
│  ○ Connect a second source                                  │
└─────────────────────────────────────────────────────────────┘
```

Position: Top of dashboard, above filters. Uses `collapse` component (collapsible). Dismissible via × button. State tracked in `localStorage` key `specto-onboarding-checklist`.

---

## HTMX Integration for Onboarding

| Interaction                  | Method                                           | Notes                                    |
| ---------------------------- | ------------------------------------------------ | ---------------------------------------- |
| Welcome step navigation      | `hx-boost` links                                 | View Transition between steps            |
| Media type selection          | `hx-post="/welcome/preference"` + redirect       | Sets preference, redirects to step 2     |
| File upload in welcome        | `hx-post` with `multipart/form-data`             | Same as existing import endpoint         |
| Import progress               | `hx-get` polling on `/partials/import-status`     | Every 2s until complete, shows counters  |
| Skip/dismiss                  | `hx-get="/"` with `hx-push-url`                  | Marks onboarded, goes to dashboard       |
| Checklist dismiss             | Client-side JS, `localStorage`                   | No server call needed                    |
| Empty state CTA               | `hx-boost` link to `/plugins`                    | Standard navigation                      |

## New Routes Required

| Route                        | Method | Purpose                                      |
| ---------------------------- | ------ | -------------------------------------------- |
| `/welcome`                   | GET    | Welcome step 1 (media type selection)        |
| `/welcome/connect`           | GET    | Welcome step 2 (guided first connection)     |
| `/welcome/complete`          | GET    | Welcome step 3 (import summary)              |
| `/welcome/preference`        | POST   | Store media type preference                  |
| `/welcome/skip`              | POST   | Mark user as onboarded, redirect to `/`      |
| `/partials/import-status`    | GET    | Polling endpoint for live import progress    |

## DB Changes

| Table   | Column       | Type    | Purpose                           |
| ------- | ------------ | ------- | --------------------------------- |
| `users` | `onboarded`  | `bool`  | Skip welcome flow on repeat login |
| `users` | `media_pref` | `text`  | Preferred media type (optional)   |
