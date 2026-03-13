# UI Design

## Overview

This document specifies the layout, navigation, page structure, and interaction patterns
for every screen in Specto's web app. Each page includes an ASCII wireframe, responsive
behavior, and HTMX integration notes.

The visual language is defined in [styling.md](./styling.md) (themes, colors, typography).
The data contract is defined in [api.md](./api.md).

---

## App Shell

Top-nav layout. No sidebar — the app is content-focused, not tool-heavy. The navbar
persists across all pages via `vt-navbar` (no animation during view transitions).

```
┌─────────────────────────────────────────────────────────────────┐
│ navbar  (sticky, bg-base-200/60, backdrop-blur)                 │
│  ┌──────────┐                    ┌────┐ ┌────────┐ ┌────────┐  │
│  │ Specto   │                    │ TL │ │ Plugins│ │ avatar │  │
│  │ (logo)   │                    │    │ │        │ │   ▼    │  │
│  └──────────┘                    └────┘ └────────┘ └────────┘  │
│                                                     dropdown:  │
│                                                     Settings   │
│                                                     Share      │
│                                                     Sign out   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  main  (vt-main, max-w-7xl, centered)                           │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                                                         │    │
│  │              page content lives here                    │    │
│  │                                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  #page-loader  (fixed top, 2px amber bar, hidden until active)  │
└─────────────────────────────────────────────────────────────────┘
```

### Navbar structure

| Element          | Position   | Behavior                              |
| ---------------- | ---------- | ------------------------------------- |
| Logo "Specto"    | Left       | Links to `/` (dashboard when logged in) |
| Timeline link    | Right      | Links to `/timeline`                  |
| Plugins link     | Right      | Links to `/plugins`                   |
| Avatar dropdown  | Far right  | DaisyUI dropdown: Settings, Share, Sign out |

**Unauthenticated navbar** shows only the logo and a "Sign in" button.

### Responsive

- `>= lg` (1024px): Full horizontal nav, all links visible
- `>= sm` (640px): Same layout, tighter spacing
- `< sm` (mobile): Hamburger menu (DaisyUI drawer), links collapse into slide-out

---

## Pages

### 1. Landing / Home (unauthenticated)

**Route:** `/`  
**Auth:** None

```
┌─────────────────────────────────────────────────────────────┐
│ navbar: [Specto]                              [Sign in]     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│  Know what you consume.          (text-display, 5xl-7xl)    │
│  ─────────────────────                                      │
│  "consume." in text-primary                                 │
│                                                             │
│  Your media diet across every    (text-base-content/60)     │
│  platform — unified, analyzed,                              │
│  and shareable.                                             │
│                                                             │
│  ┌──────────────┐  ┌──────────┐                             │
│  │ Get started  │  │ Sign in  │  (btn-primary, btn-ghost)   │
│  └──────────────┘  └──────────┘                             │
│                                                             │
│                    ~ 60% viewport height ~                   │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │  Unified │  │ Insights │  │ Shareable│  (3-col grid)    │
│  │   Feed   │  │  & Tags  │  │ Profiles │                  │
│  │          │  │          │  │          │                  │
│  │  Pull    │  │ See what │  │ Curate   │                  │
│  │  history │  │ patterns │  │ your     │                  │
│  │  from    │  │ dominate │  │ media    │                  │
│  │  every   │  │ your     │  │ identity │                  │
│  │  platform│  │ attention│  │ page     │                  │
│  └──────────┘  └──────────┘  └──────────┘                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**HTMX:** Static page. `hx-boost` links to `/register` and `/login`.

**Animation:** Staggered `animate-fade-in-up` on heading, subtitle, buttons (0s, 0.1s, 0.2s).

---

### 2. Login

**Route:** `/login`  
**Auth:** None (redirects to `/` if already logged in)

```
┌─────────────────────────────────────────────────────────────┐
│ navbar: [Specto]                                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│              ┌─────────────────────────────┐                │
│              │                             │                │
│              │  Sign in to Specto          │  (card)        │
│              │  ─────────────────          │                │
│              │                             │                │
│              │  ┌─────────────────────┐    │                │
│              │  │  ○ Continue with    │    │  (btn, full-w) │
│              │  │    Google           │    │                │
│              │  └─────────────────────┘    │                │
│              │                             │                │
│              │  ┌─────────────────────┐    │                │
│              │  │  ○ Continue with    │    │  (btn, full-w) │
│              │  │    GitHub           │    │                │
│              │  └─────────────────────┘    │                │
│              │                             │                │
│              │  ─────── or ───────         │  (divider)     │
│              │                             │                │
│              │  Email                      │                │
│              │  ┌─────────────────────┐    │                │
│              │  │                     │    │  (input)       │
│              │  └─────────────────────┘    │                │
│              │  Password                   │                │
│              │  ┌─────────────────────┐    │                │
│              │  │                     │    │  (input)       │
│              │  └─────────────────────┘    │                │
│              │                             │                │
│              │  ┌─────────────────────┐    │                │
│              │  │     Sign in         │    │  (btn-primary) │
│              │  └─────────────────────┘    │                │
│              │                             │                │
│              │  Don't have an account?     │                │
│              │  Register →                 │  (link)        │
│              │                             │                │
│              └─────────────────────────────┘                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Layout:** Centered card, max-w-sm. Vertically centered on screen.

**HTMX:** OAuth buttons are normal `<a>` links (redirect to `/auth/{provider}/login`).
Email/password form uses `hx-post="/login"` with `hx-swap="outerHTML"` on the card
to show validation errors inline without a full page reload.

**Responsive:** Card stretches to full width on mobile with `px-4` padding.

---

### 3. Register

**Route:** `/register`  
**Auth:** None

Same layout as Login. Fields: display name, email, password, confirm password.
"Already have an account? Sign in" link at bottom.

---

### 4. Dashboard (authenticated home)

**Route:** `/`  
**Auth:** Required

The main screen after login. Summary stats + recent activity + quick actions.

```
┌─────────────────────────────────────────────────────────────┐
│ navbar: [Specto]          [Timeline] [Plugins] [▼ avatar]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Dashboard                         (vt-heading, text-display)│
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Filters:                                            │    │
│  │ ┌──────────┐ ┌──────────┐          ┌─────────────┐ │    │
│  │ │ Platform▼│ │  Type  ▼ │          │ 7d 30d 90d  │ │    │
│  │ └──────────┘ └──────────┘          │ (range tabs) │ │    │
│  │                                    └─────────────┘ │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐ │
│  │ 4,218      │ │ 263 hrs    │ │ spotify    │ │ music    │ │
│  │ items      │ │ total time │ │ top source │ │ top type │ │
│  │ (stat)     │ │ (stat)     │ │ (stat)     │ │ (stat)   │ │
│  └────────────┘ └────────────┘ └────────────┘ └──────────┘ │
│                                                             │
│  Activity                                                   │
│  ─────────                                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  ▁▃▅▇▆▄▃▅▇█▆▅▃▂▁▃▅▆▇█▆▅▃▂▁▃▅▇▆▄                  │    │
│  │  consumption over time (bar chart)                   │    │
│  │  M  T  W  T  F  S  S  M  T  W  T                   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Recent                                                     │
│  ──────                                                     │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ ♫  Breathe · Pink Floyd          spotify  · 3m ago  │    │
│  │ ▶  The Art of X · Channel Y      youtube  · 1h ago  │    │
│  │ ♫  Shine On · Pink Floyd         spotify  · 2h ago  │    │
│  │ ▶  Go Doc · Justforfunc          youtube  · 5h ago  │    │
│  │ ♫  Dogs · Pink Floyd             spotify  · 6h ago  │    │
│  └─────────────────────────────────────────────────────┘    │
│  Show more → (hx-get, append)                               │
│                                                             │
│  Top Tags                     Platform Breakdown            │
│  ────────                     ──────────────────            │
│  ┌────────────────────┐       ┌────────────────────┐       │
│  │ rock         184   │       │ spotify   ████░ 62%│       │
│  │ electronic    91   │       │ youtube   ██░░░ 28%│       │
│  │ science       67   │       │ netflix   █░░░░ 10%│       │
│  │ prog-rock     54   │       │                    │       │
│  │ ambient       42   │       │                    │       │
│  └────────────────────┘       └────────────────────┘       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Filters:**

A global filter bar sits between the heading and the stats row. All dashboard
sections respond to the same filters — changing any filter re-renders everything
below the filter bar as a single partial swap.

| Filter     | Control                 | Values                                         |
| ---------- | ----------------------- | ---------------------------------------------- |
| Platform   | `<select>` dropdown     | All platforms, spotify, youtube, netflix        |
| Type       | `<select>` dropdown     | All types, music, video, podcast, article       |
| Date range | Tab group (DaisyUI tabs)| 7d (default), 30d, 90d                         |

Platform and type dropdowns follow the same pattern as the timeline page filters.
The date range tabs replace the previous per-section activity chart tabs — the
range now applies globally to stats, activity chart, tags, and platform breakdown.

**Sections:**

| Section            | Data source                      | DaisyUI component     |
| ------------------ | -------------------------------- | --------------------- |
| Filter bar         | (client-side controls)           | `select` + `tabs`     |
| Summary stats      | `GET /api/v1/insights/summary`   | `stat` (4-col grid)   |
| Activity chart     | `GET /api/v1/insights/timeline`  | Custom bar chart      |
| Recent items       | `GET /api/v1/timeline?limit=5`   | `list`                |
| Top Tags           | `GET /api/v1/insights/tags`      | `list` with counts    |
| Platform Breakdown | `GET /api/v1/insights/platform-breakdown` | Horizontal bars |

All data endpoints accept optional `?platform=`, `?type=`, and `?range=` query
params. When filters are active, stats/tags/breakdown reflect only the filtered
subset.

**HTMX interactions:**

- Filter change (any control): `hx-get="/partials/dashboard?platform=X&type=Y&range=Z"` targeting `#dashboard-content`, `hx-swap="innerHTML"`, `hx-push-url="true"`. Each filter control uses `hx-include` to send sibling filter values.
- "Show more" on recent items: `hx-get="/partials/timeline?offset=5&limit=5&platform=X&type=Y"` with `hx-swap="beforeend"` to append rows (carries active filters)
- All use `htmx-indicator` spinner on the target area

**Responsive:**

- `>= lg`: Filters in one row (selects + tabs), 4-col stat grid, 2-col bottom row (tags + platforms side by side)
- `>= sm`: Filters in one row, 2-col stat grid, stacked bottom row
- `< sm`: Filters stack vertically (selects full-width, tabs below), 1-col everything, stats stack vertically

---

### 5. Timeline

**Route:** `/timeline`  
**Auth:** Required

Full chronological feed with filters.

```
┌─────────────────────────────────────────────────────────────┐
│ navbar                                                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Timeline                          (vt-heading, text-display)│
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Filters:                                            │    │
│  │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ │    │
│  │ │ Platform▼│ │  Type  ▼ │ │  Search  │ │ From-To │ │    │
│  │ └──────────┘ └──────────┘ └──────────┘ └─────────┘ │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  March 13, 2026                                     │    │
│  │  ──────────────                                     │    │
│  │                                                     │    │
│  │  ┌─────────────────────────────────────────────┐    │    │
│  │  │ ♫  Breathe                     3:12         │    │    │
│  │  │    Pink Floyd · spotify                     │    │    │
│  │  │    ┌──────┐ ┌────────────┐ ┌─────────┐     │    │    │
│  │  │    │ rock │ │prog-rock   │ │ dreamy  │     │    │    │
│  │  │    └──────┘ └────────────┘ └─────────┘     │    │    │
│  │  │                               10:04 PM  🔒 │    │    │
│  │  └─────────────────────────────────────────────┘    │    │
│  │                                                     │    │
│  │  ┌─────────────────────────────────────────────┐    │    │
│  │  │ ▶  The Art of Code                 42:18    │    │    │
│  │  │    Dylan Beattie · youtube                  │    │    │
│  │  │    ┌────────────┐ ┌────────────┐            │    │    │
│  │  │    │ programming│ │ conference │            │    │    │
│  │  │    └────────────┘ └────────────┘            │    │    │
│  │  │                                8:30 PM      │    │    │
│  │  └─────────────────────────────────────────────┘    │    │
│  │                                                     │    │
│  │  March 12, 2026                                     │    │
│  │  ──────────────                                     │    │
│  │  ...                                                │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │           Load more (hx-get, click-to-load)         │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Item row details:**

- Media type icon (music note, play triangle, book, etc.)
- Title (bold) + duration
- Creator + platform badge (`badge badge-sm`)
- Tags as small badges (`badge badge-outline badge-xs`)
- Timestamp (right-aligned, `text-base-content/40`)
- Privacy lock icon: toggle via `hx-post="/api/v1/items/{id}/privacy"`

**HTMX interactions:**

- Filters: `hx-get="/partials/timeline?platform=spotify&type=music"` on change, targeting the item list, with `hx-push-url` to update the URL
- Search: `hx-trigger="keyup changed delay:400ms"` (active search pattern)
- Load more: `hx-get` with offset, `hx-swap="beforeend"` on the item list
- Privacy toggle: `hx-post` on the lock icon, swaps just that row

**Responsive:**

- `>= sm`: Filters in a horizontal row, timestamp right-aligned
- `< sm`: Filters stack vertically, timestamp below tags

---

### 6. Plugins

**Route:** `/plugins`  
**Auth:** Required

Manage connected data sources.

```
┌─────────────────────────────────────────────────────────────┐
│ navbar                                                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Plugins                           (vt-heading, text-display)│
│                                                             │
│  Connected                                                  │
│  ─────────                                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                                                     │    │
│  │  ┌────────────────────────────────────────────┐     │    │
│  │  │ ♫ Spotify                     Connected ●  │     │    │
│  │  │   OAuth · Last synced 3m ago               │     │    │
│  │  │   1,880 items                              │     │    │
│  │  │                                            │     │    │
│  │  │   ┌──────────┐  ┌────────────┐             │     │    │
│  │  │   │ Sync now │  │ Disconnect │             │     │    │
│  │  │   └──────────┘  └────────────┘             │     │    │
│  │  └────────────────────────────────────────────┘     │    │
│  │                                                     │    │
│  │  ┌────────────────────────────────────────────┐     │    │
│  │  │ ▶ YouTube                     Connected ●  │     │    │
│  │  │   OAuth · Last synced 1h ago               │     │    │
│  │  │   740 items                                │     │    │
│  │  │                                            │     │    │
│  │  │   ┌──────────┐  ┌────────────┐             │     │    │
│  │  │   │ Sync now │  │ Disconnect │             │     │    │
│  │  │   └──────────┘  └────────────┘             │     │    │
│  │  └────────────────────────────────────────────┘     │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Available                                                  │
│  ─────────                                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                                                     │    │
│  │  ┌────────────────────────────────────────────┐     │    │
│  │  │ 🎬 Netflix                  Not connected  │     │    │
│  │  │   File import                              │     │    │
│  │  │                                            │     │    │
│  │  │   ┌───────────────────────────────────┐    │     │    │
│  │  │   │ Upload viewing history (.csv)     │    │     │    │
│  │  │   └───────────────────────────────────┘    │     │    │
│  │  └────────────────────────────────────────────┘     │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Plugin card details:**

- Platform icon + name
- Auth type label (OAuth / File import)
- Connection status: `status status-success` (connected) or `status status-neutral` (not connected)
- Item count, last sync time
- Action buttons vary by state and auth type:
  - Connected OAuth: "Sync now" + "Disconnect"
  - Disconnected OAuth: "Connect" (redirects to OAuth)
  - File import: File upload input

**HTMX interactions:**

- "Sync now": `hx-post="/api/v1/plugins/{plugin}/sync"` — button shows spinner via `htmx-request` auto-styling, then swaps the card with updated sync time/count
- "Disconnect": `hx-delete="/api/v1/plugins/{plugin}/disconnect"` with `hx-confirm`
- "Connect": Normal `<a>` link to `/api/v1/plugins/{plugin}/connect` (redirect)
- File upload: `hx-post="/api/v1/plugins/{plugin}/import"` with `hx-encoding="multipart/form-data"`, progress shown via `htmx-indicator`

**Responsive:**

- `>= sm`: Cards in a single column with generous padding
- `< sm`: Cards edge-to-edge, reduced padding

---

### 7. Settings

**Route:** `/settings`  
**Auth:** Required

Account settings and preferences.

```
┌─────────────────────────────────────────────────────────────┐
│ navbar                                                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Settings                          (vt-heading, text-display)│
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ tabs: [Account]  [Appearance]  [Sharing]            │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ═══ Account tab ══════════════════════════════════════      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Profile                                (fieldset)  │    │
│  │                                                     │    │
│  │  Display name                                       │    │
│  │  ┌──────────────────────────────┐                   │    │
│  │  │ Estifanos                    │                   │    │
│  │  └──────────────────────────────┘                   │    │
│  │                                                     │    │
│  │  Email                                              │    │
│  │  ┌──────────────────────────────┐                   │    │
│  │  │ user@example.com             │  (disabled)       │    │
│  │  └──────────────────────────────┘                   │    │
│  │                                                     │    │
│  │  Profile slug (for share URL)                       │    │
│  │  ┌──────────────────────────────┐                   │    │
│  │  │ estifanos                    │                   │    │
│  │  └──────────────────────────────┘                   │    │
│  │  /share/estifanos                                   │    │
│  │                                                     │    │
│  │  ┌───────────────┐                                  │    │
│  │  │ Save changes  │  (btn-primary)                   │    │
│  │  └───────────────┘                                  │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Connected accounts                     (fieldset)  │    │
│  │                                                     │    │
│  │  Google   ● Connected                               │    │
│  │  GitHub   ○ Not connected   [Link account]          │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Danger zone                            (fieldset)  │    │
│  │                                                     │    │
│  │  ┌──────────────────┐                               │    │
│  │  │ Delete account   │  (btn-error, btn-outline)     │    │
│  │  └──────────────────┘                               │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ═══ Appearance tab ═══════════════════════════════════      │
│                                                             │
│  Theme                                                      │
│  ┌──────────┐  ┌──────────┐                                 │
│  │ ● Dark   │  │ ○ Light  │  (radio, theme-controller)     │
│  └──────────┘  └──────────┘                                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**HTMX interactions:**

- Tabs: `hx-get="/settings/account"`, `hx-get="/settings/appearance"`, etc. with `hx-push-url` — swap just the tab content area, not the full page
- Save changes: `hx-put` on the fieldset, inline validation errors via `hx-swap="outerHTML"` on the form
- Theme toggle: Uses DaisyUI `theme-controller` radio inputs that set `data-theme` on `<html>` — pure client-side, no server call needed

---

### 8. Share Settings

**Route:** `/settings/sharing` (tab within settings)  
**Auth:** Required

```
┌─────────────────────────────────────────────────────────────┐
│  ═══ Sharing tab ══════════════════════════════════════      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Public profile                                     │    │
│  │                                                     │    │
│  │  ┌────────────────────────┐                         │    │
│  │  │ ● Enable public profile│  (toggle)               │    │
│  │  └────────────────────────┘                         │    │
│  │  URL: /share/estifanos  [copy]                      │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Blocks                                                     │
│  ──────                                                     │
│  Drag to reorder. Toggle to enable/disable.                 │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ ≡  ┌──┐ Top Genres          ┌──────┐ ┌───────────┐ │    │
│  │    │✓ │                     │ 30d ▼│ │ All platf.│ │    │
│  │    └──┘                     └──────┘ └───────────┘ │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ≡  ┌──┐ Mood Profile        ┌──────┐ ┌───────────┐ │    │
│  │    │✓ │                     │ 30d ▼│ │ All platf.│ │    │
│  │    └──┘                     └──────┘ └───────────┘ │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ≡  ┌──┐ Top Creators        ┌──────┐ ┌───────────┐ │    │
│  │    │✓ │   Show: [10] ▼      │ 30d ▼│ │ spotify ▼ │ │    │
│  │    └──┘                     └──────┘ └───────────┘ │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ≡  ┌──┐ Platform Mix        ┌──────┐               │    │
│  │    │  │                     │ 30d ▼│               │    │
│  │    └──┘                     └──────┘               │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ≡  ┌──┐ Currently Into                              │    │
│  │    │✓ │                                             │    │
│  │    └──┘                                             │    │
│  │    ┌───────────────────────────────────────────┐    │    │
│  │    │ Deep into 70s prog rock and Korean        │    │    │
│  │    │ cinema right now.                         │    │    │
│  │    └───────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Exclusions                                                 │
│  ──────────                                                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Exclude platforms:  [Netflix ×]  [+ add]           │    │
│  │  Exclude tags:       [romance ×]  [+ add]           │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌───────────────┐  ┌───────────────────┐                   │
│  │ Save & publish│  │ Preview profile → │                   │
│  └───────────────┘  └───────────────────┘                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**HTMX interactions:**

- Block reorder: Sortable.js + `hx-post` on drop to persist order
- Block toggle/settings: Each block row is its own form target — toggle/select changes trigger `hx-put` to update that block's config
- Preview: `hx-get="/api/v1/share-profile/preview"` opens a modal with the rendered preview
- Save: `hx-put="/api/v1/share-profile"` sends the full blocks config

---

### 9. Public Share Profile

**Route:** `/share/{slug}`  
**Auth:** None

```
┌─────────────────────────────────────────────────────────────┐
│  (no navbar — standalone page)                              │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                                                     │    │
│  │          Estifanos                (display name)    │    │
│  │          ──────────               (text-display)    │    │
│  │          @estifanos               (slug, muted)     │    │
│  │                                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Top Genres                             (block)     │    │
│  │                                                     │    │
│  │  rock          ██████████████████░░░  45%           │    │
│  │  electronic    █████████░░░░░░░░░░░  22%           │    │
│  │  hip-hop       ██████░░░░░░░░░░░░░░  15%           │    │
│  │  jazz          ████░░░░░░░░░░░░░░░░  10%           │    │
│  │  ambient       ██░░░░░░░░░░░░░░░░░░   8%           │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Mood Profile                           (block)     │    │
│  │                                                     │    │
│  │  mostly chill and contemplative                     │    │
│  │                                                     │    │
│  │  chill    ████████████░░░  38%                      │    │
│  │  upbeat   ████████░░░░░░  25%                      │    │
│  │  intense  █████░░░░░░░░░  18%                      │    │
│  │  mellow   ████░░░░░░░░░░  12%                      │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Top Creators                           (block)     │    │
│  │                                                     │    │
│  │  1. Pink Floyd          ♫  spotify                  │    │
│  │  2. Radiohead           ♫  spotify                  │    │
│  │  3. Fireship            ▶  youtube                  │    │
│  │  4. Boards of Canada    ♫  spotify                  │    │
│  │  5. 3Blue1Brown         ▶  youtube                  │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Currently Into                         (block)     │    │
│  │                                                     │    │
│  │  "Deep into 70s prog rock and Korean cinema         │    │
│  │   right now."                                       │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  powered by Specto              (footer, muted)     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Rendering:** Fully server-rendered. No HTMX, no client-side data fetching. The blocks
render top-to-bottom in the order the user configured.

**Styling:** Same theme as the rest of the app, but could use `grain` on the header
section for texture. Each block is a `card bg-neutral` for slight elevation.

**Responsive:**

- `>= sm`: Centered, max-w-2xl for readability
- `< sm`: Edge-to-edge cards, full width

---

## Responsive Breakpoints

Using Tailwind's default breakpoints:

| Breakpoint | Width    | Layout changes                                       |
| ---------- | -------- | ---------------------------------------------------- |
| `< sm`     | < 640px  | Single column. Navbar collapses to hamburger. Cards edge-to-edge. Filters stack. |
| `sm`       | >= 640px | 2-col stat grids. Horizontal filter rows. Card padding increases. |
| `md`       | >= 768px | No major changes (transitional).                     |
| `lg`       | >= 1024px| 4-col stat grids. Side-by-side layouts (tags + platforms). Full navbar. |
| `xl`       | >= 1280px| Max content width (max-w-7xl = 80rem). Extra breathing room. |

### Mobile-first patterns

- **Drawer for mobile nav:** DaisyUI `drawer` component, triggered by hamburger button visible `< sm`
- **Touch targets:** All interactive elements minimum 44px touch target
- **Cards:** Use `rounded-none sm:rounded-box` pattern — edge-to-edge on mobile, rounded on larger screens
- **Stat grid:** `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4`

---

## HTMX Interaction Patterns

### What loads inline (partial swap) vs full page

| Interaction                | Method            | Why                                    |
| -------------------------- | ----------------- | -------------------------------------- |
| Page navigation            | `hx-boost` link   | View Transition between pages          |
| Tab switching (settings)   | Partial `hx-get`  | Only tab content changes               |
| Filter change (timeline)   | Partial `hx-get`  | Only the item list re-renders          |
| "Load more" pagination     | Partial `hx-get`  | Append rows to existing list           |
| Sync button                | Partial `hx-post` | Swap just the plugin card              |
| Privacy toggle             | Partial `hx-post` | Swap just that timeline row            |
| Delete/disconnect          | Partial `hx-delete` | Swap the card to disconnected state  |
| OAuth connect              | Full redirect      | Leaves the app for external OAuth flow |
| Theme switch               | Client-side only   | `theme-controller` sets `data-theme`  |

### Partial response convention

Server endpoints that return HTML fragments (not full pages) live under `/partials/`:

```
/partials/timeline?offset=5&limit=5    → timeline item rows only
/partials/activity-chart?range=30d     → chart HTML only
/partials/settings/account             → account tab content only
/partials/plugin-card/{plugin}         → single plugin card
```

These return bare HTML fragments (no `<html>`, no `<head>`, no layout wrapper).
The server checks for the `HX-Request` header to distinguish partial requests from
full page loads. If `HX-Request` is absent, redirect to the full page.

### Loading feedback

Three tiers of loading feedback, all CSS-only:

1. **Global:** `#page-loader` bar at top — always visible during `hx-boost` navigations
2. **Element:** `htmx-request` class dims the triggering button/element + shows spinner
3. **Target:** `htmx-indicator` on a skeleton/spinner inside the target area for content loads

---

## Component Inventory

DaisyUI components used across pages:

| Component     | Where used                                             |
| ------------- | ------------------------------------------------------ |
| `navbar`      | App shell                                              |
| `drawer`      | Mobile navigation                                      |
| `btn`         | Everywhere                                             |
| `card`        | Plugin cards, share profile blocks, feature cards      |
| `stat`        | Dashboard summary numbers                              |
| `list`        | Timeline items, recent activity, top tags              |
| `badge`       | Tags, platform labels, status indicators               |
| `status`      | Plugin connection state dots                           |
| `tab`         | Settings page sections, dashboard time range           |
| `dropdown`    | Avatar menu, filter selects                            |
| `fieldset`    | Settings form groups                                   |
| `input`       | All text inputs                                        |
| `select`      | Filter dropdowns                                       |
| `toggle`      | Share block enable/disable, public profile toggle      |
| `checkbox`    | Block selection                                        |
| `modal`       | Share preview, delete confirmation                     |
| `divider`     | Login page "or" separator                              |
| `alert`       | Error/success messages                                 |
| `tooltip`     | Icon-only actions, truncated text                      |
| `loading`     | Inline spinners for htmx indicators                    |
| `skeleton`    | Placeholder content during loads                       |
| `progress`    | File upload progress                                   |
| `collapse`    | FAQ or help sections (future)                          |
