# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public site, OAuth (login + link Discord/YouTube), access queue, roster, articles, per-skater YouTube clips, admin users (merge/unlink/delete), custom pages, `-port`.
- Skater profiles show latest synced clips from a linked YouTube channel while the skater is logged in. `/videos` is that aggregate. No site-wide API-key poller.

## Next

1. Tailwind / Turso only when a real need shows up.

## Log

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
