# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- `just dev` serves public empty-state pages on `:53053` (`/`, `/team`, `/news`, `/videos`, `/healthz`).
- Env config, modernc SQLite, README schema applied at start, templates and CSS loaded from `web/`.
- Discord/YouTube login buttons render disabled. No OAuth, admin, articles pipeline, or YouTube sync.

## Next

1. Discord / YouTube OAuth and guild RBAC.
2. Access-request queue.
3. Skater roster CRUD.
4. Articles (goldmark + bluemonday).
5. YouTube poller.
6. Admin UI + Monaco.
7. Tailwind build / Turso when those are actually needed.

## Log

- 2026-09-12 — Started journal. Project is docs-only.
- 2026-09-12 — Bootable server: SQLite + public empty pages. `just test` / `just dev`.
