# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public empty-state pages on `:53053` with SQLite schema on start.
- Discord and YouTube OAuth (PKCE, session cookie `seshhub_session`). Guild roles map to admin/skater/member; everyone else (including YouTube) is `pending`.
- Login buttons stay disabled until client id/secret are set in env.

## Next

1. Access-request queue.
2. Skater roster CRUD.
3. Articles (goldmark + bluemonday).
4. YouTube poller.
5. Admin UI + Monaco.
6. Tailwind build / Turso when those are actually needed.

## Log

- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
- 2026-09-12 — Discord/YouTube OAuth + guild RBAC. Access-request UI still next.
