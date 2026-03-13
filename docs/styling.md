# Styling & HTMX Integration

## Overview

Specto's frontend is built with:

- **Tailwind CSS v4** (standalone binary, no npm)
- **DaisyUI v5** (Tailwind plugin, standalone `.mjs` bundle)
- **templ** (type-safe Go HTML templates)
- **HTMX 2.0** (HTML-driven AJAX, loaded from CDN)

All styling lives in one file: `styles/input.css`. There is no `tailwind.config.js` —
Tailwind v4 uses CSS-first configuration.

Related docs:
- [architecture.md](./architecture.md) — system layers
- [sharing.md](./sharing.md) — public profile page design
- [MVP.md](./MVP.md) — product scope

---

## Design Direction

**Aesthetic: dark & cinematic.** Deep blacks with a warm undertone, muted jewel-tone
accents, sharp edges, fine borders. The feel of a premium streaming app — Letterboxd,
Plex, or a high-end editorial publication.

Not bubbly, not flat, not generic. Intentional warmth in a dark palette.

---

## Themes

Two custom DaisyUI themes are defined in `styles/input.css`:

| Theme          | Mode  | Role                                         |
| -------------- | ----- | -------------------------------------------- |
| `specto-dark`  | dark  | Default. Set as both `default` and `prefersdark`. |
| `specto-light` | light | Optional light mode for users who prefer it. |

The active theme is set via `data-theme` on `<html>`:

```html
<html lang="en" data-theme="specto-dark">
```

### Color palette

All colors use the `oklch()` color space for perceptual uniformity.

| Token       | Purpose                          | Dark value                          | Light value                         |
| ----------- | -------------------------------- | ----------------------------------- | ----------------------------------- |
| `primary`   | Hero accent — warm amber/gold    | `oklch(72% 0.155 70)`              | `oklch(52% 0.17 55)`               |
| `secondary` | Cool counterpoint — muted teal   | `oklch(62% 0.09 190)`              | `oklch(45% 0.1 190)`               |
| `accent`    | Badges, tags — dusty copper/rose | `oklch(62% 0.12 25)`               | `oklch(50% 0.14 25)`               |
| `neutral`   | Cards, elevated surfaces         | `oklch(22% 0.01 70)`               | `oklch(35% 0.015 70)`              |
| `base-100`  | Page background                  | `oklch(14% 0.008 70)` (near-black) | `oklch(97% 0.008 80)` (warm cream) |
| `base-200`  | Recessed surface                 | `oklch(11% 0.006 70)`              | `oklch(94% 0.01 80)`               |
| `base-300`  | Borders, dividers                | `oklch(8% 0.004 70)`               | `oklch(90% 0.014 80)`              |

All bases share a warm ~70° hue angle, giving them a subtle warmth rather than cold
neutral gray.

Status colors (`info`, `success`, `warning`, `error`) are desaturated to stay cinematic.
They do not dominate — they inform.

### Shape tokens

Sharp, not rounded. Minimal rounding to maintain the editorial feel:

| Token             | Value      | Used for                     |
| ----------------- | ---------- | ---------------------------- |
| `--radius-box`    | `0.5rem`   | Cards, modals, alerts        |
| `--radius-field`  | `0.25rem`  | Buttons, inputs, tabs        |
| `--radius-selector` | `0.25rem` | Checkboxes, toggles, badges |
| `--border`        | `1px`      | Fine structural borders      |
| `--depth`         | `1`        | Subtle shadows enabled       |
| `--noise`         | `0`        | No noise texture on components |

### Using DaisyUI semantic colors

Always use semantic color names (`bg-primary`, `text-base-content`, `border-base-300`)
instead of hardcoded Tailwind colors (`bg-amber-500`, `text-gray-200`). This ensures
colors respond to theme changes automatically.

```html
<!-- Good -->
<div class="bg-base-200 text-base-content border border-base-300">

<!-- Bad — hardcoded, won't respond to theme switch -->
<div class="bg-zinc-900 text-zinc-200 border border-zinc-800">
```

Use opacity modifiers for muted text: `text-base-content/60`, `text-base-content/40`.

---

## Typography

Three font families, loaded via Google Fonts in `styles/input.css`:

| Token            | Font             | Role                                      |
| ---------------- | ---------------- | ----------------------------------------- |
| `--font-display` | Playfair Display | Headings, hero text. High-contrast serif.  |
| `--font-sans`    | DM Sans          | Body text, UI. Geometric sans with warmth. |
| `--font-mono`    | JetBrains Mono   | Code, data tables, stats.                  |

### Usage

- `font-sans` is the default body font (set on `<html>`)
- Use `text-display` utility class (or `font-display`) for display headings
- Use `font-mono` for data-heavy content

```html
<h1 class="text-display text-5xl font-bold tracking-tight">
  Your media <span class="text-primary">diet.</span>
</h1>
<p class="text-base-content/60">Body text uses DM Sans by default.</p>
<span class="font-mono text-sm">142 items</span>
```

---

## Custom Utility Classes

Defined in `styles/input.css` via `@utility`:

| Class                 | What it does                                                        |
| --------------------- | ------------------------------------------------------------------- |
| `text-display`        | Sets `font-family` to Playfair Display                              |
| `text-balance`        | `text-wrap: balance` for headings                                   |
| `text-pretty`         | `text-wrap: pretty` for body paragraphs                             |
| `grain`               | Adds a subtle SVG noise overlay (3% opacity) via `::after`         |
| `shimmer`             | Animated gradient for skeleton loading states                       |
| `animate-fade-in-up`  | Entrance animation (fade + translateY). Supports `animation-delay`. |
| `glow-primary`        | Amber box-shadow glow for primary CTAs                              |

### Staggered entrance animations

Use `animate-fade-in-up` with inline `animation-delay` for staggered reveals:

```html
<h1 class="animate-fade-in-up">First</h1>
<p class="animate-fade-in-up" style="animation-delay: 0.1s">Second</p>
<div class="animate-fade-in-up" style="animation-delay: 0.2s">Third</div>
```

### Grain overlay

For hero sections or large surface areas that need cinematic texture:

```html
<section class="grain bg-base-200 p-12">
  <!-- Content sits above the grain overlay -->
</section>
```

The grain is purely decorative — `pointer-events: none`, mixed via `overlay` blend mode.

---

## HTMX Integration

### Configuration

HTMX is configured via a `<meta>` tag in `components/layout.templ`:

```html
<meta name="htmx-config" content='{
  "globalViewTransitions": true,
  "defaultSwapStyle": "innerHTML",
  "defaultSettleDelay": 20
}'/>
```

Key settings:

| Setting                  | Value      | Purpose                                          |
| ------------------------ | ---------- | ------------------------------------------------ |
| `globalViewTransitions`  | `true`     | All htmx swaps use the View Transition API       |
| `defaultSwapStyle`       | `innerHTML`| Default swap replaces inner content of target     |
| `defaultSettleDelay`     | `20`       | 20ms settle delay for CSS transition timing      |

### hx-boost

`hx-boost="true"` is set on `<body>`, which converts all `<a>` and `<form>` elements
within the page into AJAX requests automatically. Links navigate via HTMX instead of
full page loads, enabling:

- View transition animations between pages
- The global loading bar
- Preserved scroll position and DOM state where appropriate

```html
<body hx-boost="true" hx-indicator="#page-loader">
```

### Global loading bar

A 2px amber gradient bar sits fixed at the top of the viewport. It is invisible by
default and appears whenever an htmx request is in flight, using the `hx-indicator`
pattern:

```html
<!-- In layout.templ -->
<body hx-indicator="#page-loader">
  <div id="page-loader"></div>
  ...
</body>
```

The bar animates with a sliding gradient (`page-loader-slide` keyframe). No JavaScript
required — it is entirely CSS-driven via the `htmx-request` class that htmx applies
automatically.

---

## HTMX CSS State Classes

htmx applies CSS classes at each stage of its request lifecycle. We style these globally
in `styles/input.css` to provide visual feedback without writing any JavaScript.

### Request lifecycle

```
User interaction
  │
  ▼
┌──────────────────────┐
│ .htmx-request        │  Applied to the triggering element
│ (in-flight)          │  while the HTTP request is pending.
└──────────────────────┘
  │
  ▼  Response received
┌──────────────────────┐
│ .htmx-swapping       │  Applied to the target element
│ (swap phase)         │  during content replacement.
└──────────────────────┘
  │
  ▼  New content inserted
┌──────────────────────┐
│ .htmx-added          │  Applied to each newly inserted
│ (on new elements)    │  element during the settle phase.
└──────────────────────┘
  │
  ▼  Attributes settled
┌──────────────────────┐
│ .htmx-settling       │  Applied to the target element
│ (settle phase)       │  while attributes are being settled.
└──────────────────────┘
  │
  ▼  Complete
```

### What we style

**`.htmx-request`** — Dims the triggering element (opacity 0.65, pointer-events disabled).
Buttons additionally get an inline CSS spinner — the button text becomes transparent and
a rotating ring appears in its center. No need for separate loading elements.

```html
<!-- This button will automatically show a spinner while the request is in flight -->
<button class="btn btn-primary" hx-post="/api/v1/plugins/spotify/sync">
  Sync now
</button>
```

**`.htmx-indicator`** — Standard htmx indicator pattern. Elements with this class are
hidden by default (opacity 0) and revealed when a parent or sibling has `.htmx-request`:

```html
<button hx-get="/api/v1/timeline" hx-target="#feed">
  Load
  <span class="loading loading-spinner htmx-indicator"></span>
</button>
```

**`.htmx-swapping`** — Old content fades out (opacity 0, 150ms transition) before
being replaced. Use the `swap` modifier on `hx-swap` to extend the phase if the
animation needs more time:

```html
<div hx-get="/fragment" hx-swap="innerHTML swap:200ms">
```

**`.htmx-added`** — Newly inserted elements start invisible and shifted down 8px.
Combined with the transition on htmx-attributed elements, this creates a subtle
fade-in-up entrance for new content.

**`.htmx-settling`** — The target element gets transitions on opacity and transform
during the settle phase, enabling smooth attribute changes (e.g., a class addition
from the server response triggers a CSS transition).

### CSS transitions via stable IDs

htmx can drive CSS transitions without any JavaScript by keeping element `id` attributes
stable across swaps. When htmx finds a matching ID between old and new content, it
preserves the old element's attributes momentarily, swaps in the new content, then
settles the new attributes — creating a transition window.

```html
<!-- Server response #1 -->
<div id="status" class="badge badge-warning">Syncing...</div>

<!-- Server response #2 (same ID, different class) -->
<div id="status" class="badge badge-success">Complete</div>
```

With a CSS transition on `.badge`:
```css
.badge { transition: background-color 300ms ease-out; }
```

The badge will smoothly animate from warning to success color.

---

## View Transition API

The [View Transition API](https://developer.mozilla.org/en-US/docs/Web/API/View_Transition_API)
provides native browser-level animated transitions between DOM states. htmx integrates
with it directly.

### How it works with htmx

1. `globalViewTransitions: true` in our htmx config enables it for all swaps
2. When htmx does a swap, it calls `document.startViewTransition()` if available
3. The browser snapshots the old state, applies the swap, then animates between old
   and new using `::view-transition-*` pseudo-elements
4. Falls back gracefully — if the browser doesn't support it, swaps happen normally

### Named transition regions

Elements can be assigned a `view-transition-name` to animate independently from the
rest of the page. We use CSS classes prefixed with `vt-`:

| Class        | `view-transition-name` | Behavior                                    |
| ------------ | ---------------------- | ------------------------------------------- |
| `vt-main`    | `main-content`         | Slide + cross-fade on page changes          |
| `vt-navbar`  | `navbar`               | No animation — stays put during transitions |
| `vt-heading` | `page-heading`         | Smooth cross-fade for page titles           |

```html
<nav class="vt-navbar navbar ...">...</nav>
<main class="vt-main ...">{ children... }</main>
<h1 class="vt-heading text-display ...">Page Title</h1>
```

### Transition animations

Defined in `styles/input.css`:

| Region         | Old content                | New content                      |
| -------------- | -------------------------- | -------------------------------- |
| `root`         | 200ms fade-out             | 300ms fade-in (100ms delay)      |
| `main-content` | 200ms fade-out + slide left | 300ms fade-in + slide from right (80ms delay) |
| `navbar`       | No animation               | No animation                     |
| `page-heading` | 200ms fade-out             | 300ms fade-in (50ms delay)       |

The staggered delays create a cinematic layered effect: the old content fades out first,
then the new content slides in slightly after.

### Adding view transition names to new pages

When building a new page, assign `vt-heading` to the page title so it cross-fades
smoothly during navigation:

```go
templ SettingsPage() {
    @Layout("Settings — Specto") {
        <h1 class="vt-heading text-display text-3xl font-bold">Settings</h1>
        // ...
    }
}
```

The `<main>` wrapper already has `vt-main` from the layout, so the page body will
automatically get the slide + fade transition.

### Reduced motion

All view transition animations are suppressed when the user has `prefers-reduced-motion`:

```css
@media (prefers-reduced-motion: reduce) {
    ::view-transition-old(*),
    ::view-transition-new(*) {
        animation-duration: 0.01ms !important;
    }
}
```

### Browser support

The View Transition API is supported in Chrome 111+, Edge 111+, Safari 18+, and
Firefox 126+. In unsupported browsers, htmx swaps happen instantly with no animation.
The HTMX CSS state classes (`htmx-swapping`, `htmx-settling`, `htmx-added`) still
work everywhere regardless.

---

## File Reference

| File                        | Purpose                                            |
| --------------------------- | -------------------------------------------------- |
| `styles/input.css`          | All CSS: themes, fonts, utilities, transitions     |
| `static/css/tailwind.css`   | Compiled output (generated, gitignored)            |
| `daisyui.mjs`               | DaisyUI v5 plugin bundle (gitignored)              |
| `daisyui-theme.mjs`         | DaisyUI theme helper for custom themes (gitignored)|
| `tailwindcss`               | Tailwind v4 standalone binary (gitignored)         |
| `components/layout.templ`   | Base HTML layout, htmx config, page-loader         |
| `components/navbar.templ`   | Navbar component                                   |

### Build

```bash
# Compile CSS (done automatically by air during dev)
./tailwindcss -i styles/input.css -o static/css/tailwind.css --minify

# Or via mise
mise run dev    # starts air, which watches and rebuilds
mise run build  # one-shot build including CSS
```

---

## Conventions

### When building new templ components

1. **Use DaisyUI component classes** (`btn`, `card`, `navbar`, `badge`, etc.) as the
   foundation. Customize with Tailwind utilities on top.

2. **Use semantic colors only.** `bg-primary`, not `bg-amber-600`. `text-base-content/60`,
   not `text-gray-400`.

3. **Add `vt-heading`** to the primary heading of every page for smooth navigation
   transitions.

4. **Use `hx-indicator`** for any action that triggers a server request and needs
   custom loading feedback beyond the global bar.

5. **Keep element IDs stable** across HTMX swaps when you want CSS transitions between
   states. htmx uses ID matching to enable attribute-level transitions.

6. **Use `hx-swap` modifiers** to control timing:
   - `swap:200ms` — extend the swap phase for longer exit animations
   - `settle:100ms` — adjust settle delay for entrance animations
   - `transition:true` — explicitly enable view transitions (redundant if global is on)

7. **Stagger entrance animations** using `animate-fade-in-up` with incremental
   `animation-delay` values for content that loads together.

### When adding new custom styles

- Add to `styles/input.css`, not inline or in separate CSS files
- Use `@utility` for reusable utility classes
- Use `@layer base` for element-level defaults
- Use `@keyframes` at the top level for animations
- Keep the oklch color space for any new color values
