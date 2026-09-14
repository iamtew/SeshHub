# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public site, OAuth (login + link Discord/YouTube), access queue, Discord-role team + skater profiles, articles, per-skater YouTube clips, admin users (merge/unlink/delete), custom pages, Spot (`/` markdown + Subotto `{{placeholders}}`), `-port`.
- Dark neon CSS in `web/static/css/app.css`. Visitor: sofa hero. Sesh Hub (logged-in): folded bar + avatar. `/login` is the OAuth picker.
- Halftone-over-gradient recipe in `sesh_halftone.md` (local lift-out; not always in git).

## Next

1. Turso only when a real need shows up.

## Log

- 2026-09-14 — 12h/24h TZ label from the timestamp offset (CET), not the browser.
- 2026-09-13 — Discord CDN avatars: no-referrer (stops 403).
- 2026-09-13 — Log out lives at the bottom of `/account`.
- 2026-09-13 — Fold PFP height = seshhub.png (12rem × 453/1280).
- 2026-09-13 — Fold signin: links column + who column (no overlap).
- 2026-09-13 — Fold avatar height matches seshhub.png.
- 2026-09-13 — Visitor vs Sesh Hub chrome in README; fold menu: bigger type, avatar+name right.
- 2026-09-13 — Folded bar: panel bg, seshhub.png only, click plays intro.
- 2026-09-13 — `/login` picker; logged-in folded hero with sofa/hub crossfade.
- 2026-09-13 — Well max-width 76rem → 68rem.
- 2026-09-13 — Halftone checker restored (every-other); cells fill the slot, rx 9.
- 2026-09-13 — Halftone checker: 1px gutters, rx 6→9.
- 2026-09-13 — `just build` on Unix writes `bin/seshhub`; Windows still `dist/seshhub.exe`.
- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
- 2026-09-12 — Discord/YouTube OAuth + guild RBAC. Access-request UI still next.
- 2026-09-12 — Access-request queue. Approve grants member only.
- 2026-09-12 — Skater roster CRUD and self-serve profile.
- 2026-09-12 — Articles CMS with sanitized Markdown.
- 2026-09-12 — YouTube ingest poller and video gallery.
- 2026-09-12 — Admin dashboard, custom pages, optional Monaco loader.
- 2026-09-12 — `-port` / `-db` / `-web` flags. Tailwind and Turso parked.
- 2026-09-12 — YouTube paging, featured skater video, Open Graph tags.
- 2026-09-12 — README: blurb, Meat Bag runbook (incl. OAuth), then spec.
- 2026-09-12 — Dropped site-wide YouTube poller. Account linking + logged-in skater clip sync.
- 2026-09-12 — Admin users list with merge, unlink, delete.
- 2026-09-12 — Team from Discord skater role; dropped roster CRUD; profile display name.
- 2026-09-12 — Ensure skater profile on each request so an existing login still lands on /team.
- 2026-09-12 — Own Skater profile vs admin Team Skaters (status only).
- 2026-09-12 — Dark high-tech restyle: palette tokens, space, sticky nav, burger.
- 2026-09-12 — Dropped Tailwind from the spec. Plain `app.css` is the styling layer.
- 2026-09-12 — Restyle toward seshsofa.nl: fire field, dark rounded well, sofa mark, show fonts.
- 2026-09-12 — Monster Chiller nav only; headings use Turbo Jungle (digits).
- 2026-09-12 — Halftone dots slightly larger, grid rotated 45°.
- 2026-09-12 — Halftone dots 4× size.
- 2026-09-12 — Halftone: dot diameter = gap.
- 2026-09-12 — Halftone: rounded squares, gap = half the cell.
- 2026-09-12 — Halftone: checkerboard, not brick.
- 2026-09-12 — Halftone checkerboard gutters tightened.
- 2026-09-12 — Gradient −23°, halftone +58°, rounder cells.
- 2026-09-12 — Halftone multiply-blend, slightly quieter.
- 2026-09-12 — Halftone opacity a smidge lower.
- 2026-09-12 — Wrote `halftone_background.md` (gradient + checker SVG multiply).
- 2026-09-12 — Full-width app shell; fire/halftone frame unchanged.
- 2026-09-12 — Header scrolls with the page (not sticky).
- 2026-09-12 — Narrower 76rem well. Home hero overlay plays latest YouTube clip.
- 2026-09-12 — Hero plays local `seshsofa.mp4` like the show site, not YouTube.
- 2026-09-12 — Hero overlay is transparent; fire/halftone shows through the PNG.
- 2026-09-12 — Track `web/static/vid/seshsofa.mp4` in git.
- 2026-09-12 — Dropped sofa mark from the header; hero owns it.
- 2026-09-12 — Hero on public menu pages (home/team/news/videos); not on admin/dashboard.
- 2026-09-12 — Logged-in .signin sits on the row under the public menu.
