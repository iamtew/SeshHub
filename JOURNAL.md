# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public site, OAuth, access queue, roster, articles, YouTube poller, admin dashboard, custom pages.
- Listen address is `PORT` or `-port` (`53054` or `127.0.0.1:53054`). `just dev -port 53054`. Default stays 53053 for the Meat Bag.

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
