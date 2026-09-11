# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public site, OAuth, access queue, roster, articles, YouTube poller.
- Admin home at `/admin` (counts + last sync). Custom pages at `/{slug}`; CRUD at `/admin/pages`.
- Article/page editors load Monaco from `/static/monaco/vs` when present; otherwise the textarea stays. Drop `monaco-editor` `min/vs` there to enable it.

## Next

1. Tailwind build / Turso when those are actually needed.

## Log

- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
- 2026-09-12 — Discord/YouTube OAuth + guild RBAC. Access-request UI still next.
- 2026-09-12 — Access-request queue. Approve grants member only.
- 2026-09-12 — Skater roster CRUD and self-serve profile.
- 2026-09-12 — Articles CMS with sanitized Markdown.
- 2026-09-12 — YouTube ingest poller and video gallery.
- 2026-09-12 — Admin dashboard, custom pages, optional Monaco loader.
