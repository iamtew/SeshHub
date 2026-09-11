# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public empty-state pages, SQLite, Discord/YouTube OAuth, guild RBAC.
- Pending users can request member access. Admins review at `/admin/access` (approve = member, not skater/admin). Discord re-login does not wipe an approved member back to pending.

## Next

1. Skater roster CRUD.
2. Articles (goldmark + bluemonday).
3. YouTube poller.
4. Admin UI + Monaco.
5. Tailwind build / Turso when those are actually needed.

## Log

- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
- 2026-09-12 — Discord/YouTube OAuth + guild RBAC. Access-request UI still next.
- 2026-09-12 — Access-request queue. Approve grants member only.
