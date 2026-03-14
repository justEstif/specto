---
# project-media-consumption-analysis-l1gi
title: Design website UI/UX
status: completed
type: task
priority: normal
created_at: 2026-03-13T12:51:13Z
updated_at: 2026-03-13T15:18:37Z
parent: project-media-consumption-analysis-nn0q
---

Create the visual and UX design for Specto's web app before implementation begins. This includes:

- Overall layout and navigation structure (app shell, sidebar vs top-nav)
- Page designs for: login/register, dashboard/home, timeline, plugin management, settings, sharing/public profile
- Component library decisions (DaisyUI theme, color palette, typography)
- Responsive breakpoints and mobile considerations
- HTMX interaction patterns (what loads inline vs full page)
- Wireframes or mockups for each key screen

This should produce a design document in docs/ that the team references during implementation.

## Work Done\n\n### Custom DaisyUI Theme (styles/input.css)\n- Created `specto-dark` theme (default) — dark & cinematic aesthetic\n- Created `specto-light` theme — warm editorial counterpart\n- Both use oklch color space with warm amber primary, desaturated teal secondary, dusty copper accent\n- Sharp edges (0.25rem/0.5rem radii), fine 1px borders, depth enabled\n\n### Typography\n- Display: Playfair Display (serif) — cinematic headings\n- Body: DM Sans (geometric sans) — clean, warm\n- Mono: JetBrains Mono — data/code readability\n- Loaded via Google Fonts\n\n### Custom Utilities\n- `text-display` — applies display font\n- `grain` — SVG noise overlay for hero sections\n- `shimmer` — loading state animation\n- `animate-fade-in-up` — staggered entrance animation\n- `glow-primary` — amber glow for CTAs\n\n### Updated Templates\n- `layout.templ` — added data-theme, base styles, font, container\n- `navbar.templ` — DaisyUI navbar with blur backdrop, sticky, styled links\n- `home.templ` — hero section with display typography, staggered animation

\n### Styling Doc\n- Created `docs/styling.md` covering themes, typography, custom utilities, HTMX CSS state classes, View Transition API, conventions\n- Added reference to AGENTS.md documentation list

\n### UI Design Doc\n- Created `docs/ui-design.md` with ASCII wireframes for all 9 pages\n- Covers: landing, login, register, dashboard, timeline, plugins, settings, share settings, public profile\n- Responsive breakpoints and mobile patterns\n- HTMX interaction patterns (inline vs full page, partials convention)\n- Full DaisyUI component inventory\n\n## Summary of Changes\n\nDesign task complete. Produced three artifacts:\n1. `styles/input.css` — custom themes, typography, utilities, View Transitions, HTMX state styling\n2. `docs/styling.md` — design system reference (colors, fonts, animations, conventions)\n3. `docs/ui-design.md` — page-by-page wireframes, responsive rules, HTMX patterns
