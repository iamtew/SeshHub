# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public pages, OAuth, access queue, skater roster.
- Articles: goldmark + bluemonday. Published at `/news` and `/news/{slug}`. Admins CRUD/publish at `/admin/articles`. Skaters draft at `/dashboard/articles`.

## Next

1. YouTube poller.
2. Admin UI + Monaco.
3. Tailwind build / Turso when those are actually needed.

## Log

- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
- 2026-09-12 — Discord/YouTube OAuth + guild RBAC. Access-request UI still next.
- 2026-09-12 — Access-request queue. Approve grants member only.
- 2026-09-12 — Skater roster CRUD and self-serve profile.
- 2026-09-12 — Articles CMS with sanitized Markdown.
