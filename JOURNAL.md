# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public pages, OAuth, access queue, skater roster, articles.
- YouTube poller: Data API v3, first 50 uploads, ticker + `/admin/youtube/sync`. Public list at `/videos`. Needs `YOUTUBE_API_KEY` and `YOUTUBE_CHANNEL_ID`.

## Next

1. Admin UI + Monaco.
2. Tailwind build / Turso when those are actually needed.

## Log

- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
- 2026-09-12 — Discord/YouTube OAuth + guild RBAC. Access-request UI still next.
- 2026-09-12 — Access-request queue. Approve grants member only.
- 2026-09-12 — Skater roster CRUD and self-serve profile.
- 2026-09-12 — Articles CMS with sanitized Markdown.
- 2026-09-12 — YouTube ingest poller and video gallery.
