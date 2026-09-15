# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- Public site, OAuth (login + link Discord/YouTube), access queue, Discord-role team + skater profiles, articles, per-skater YouTube clips, admin users (merge/unlink/delete), custom pages (nested slugs; seeded `/about`, `/about/privacy`, `/about/tos`), Spot (`/` markdown + Subotto `{{placeholders}}`), Episodes (`/episodes` archive + Subotto stub; hosts-role edit), `-port`. `DISCORD_HUB_ADMIN_ROLE_ID` + `DISCORD_HOSTS_ROLE_ID`. `SUBOTTO_INSTANCE` (Core env) is the Subotto host. Nav About after Videos; footer Privacy / ToS / SeshHub.
- Dark neon CSS in `web/static/css/app.css`. Site-wide `background.png`. Visitor: sofa hero + Discord/Twitch/YouTube + Hub square. Sesh Hub (logged-in): folded bar + avatar; same three social icons, no Hub square. `/login` is the OAuth picker. Well is 70% / min 720px. Queue list actions sit on the right. Public nav and admin skaters link are **FS Team**.
- Halftone-over-gradient recipe in `sesh_halftone.md` (local lift-out; not always in git).
- `docs/privacy-policy.md` is the source of truth for `/about/privacy`. The CMS row is still the `# Privacy Policy` stub from `006_about_pages.sql` — paste the markdown into `/admin/pages` to publish. Seed's `WHERE NOT EXISTS` never overwrites it. Remediation plan for the disclosed gaps: `docs/privacy-remediation.md`.

## Next

Privacy backlog, ordered by legal exposure. `docs/privacy-policy.md` §9 discloses every one of these, so shipping a fix means deleting a line from §9. The how-to lives in `docs/privacy-remediation.md` (P0-P3); this list stays the status. **Read its "three traps" section before starting items 4 or 5** — the obvious implementation is wrong in all three cases (vanishing articles, `DROP COLUMN` bricking the second boot, half-deleted accounts).

1. **P0-1** Consent gate before `gtag.js`. Only item with live legal exposure.
2. **P0-2** Fill the policy's two `[TODO before publishing]` markers (contact address, hosting country) before pasting it into `/admin/pages`.
3. **P1-1** Purge expired sessions at boot.
4. **P1-2** Stop storing `sessions.ip_address` / `user_agent`. Write-only columns; deleting the collection is a smaller diff than keeping it.
5. **P2-2** Finish `DeleteUser`: profile + cached clips, tombstone author, wrap in a tx. Needs a test — it's the one path where a bug means we lied about erasing someone.
6. **P2-1** Self-service unlink / delete / export on `/account`.
7. **P3** Self-host the cdnfonts faces (check licensing), delete dead `SESSION_SECRET`.
8. Turso only when a real need shows up.

Erasure runbook until P2-2 lands: delete the **account first** (kills sessions), **then** the skater profile. `maybeEnsureProfile` runs on every request, so removing the profile alone just recreates it on the next page load. Also clear `youtube_videos` by channel ID, and note an author cannot be deleted at all yet (`ErrHasArticles`).

## Log

- 2026-09-15 — Docs moved to `docs/`; added `docs/privacy-remediation.md` (P0-P3 fix plan for the §9 gaps).
- 2026-09-15 — Privacy policy rewritten against the actual schema; old "we store nothing" text was false. Known gaps disclosed in §9, code fixes queued in Next.
- 2026-09-15 — Google tag from `GTAG_ID` env (empty = omit).
- 2026-09-15 — `just clean` drops `bin/` and `dist/`.
- 2026-09-15 — Markdown GFM tables (goldmark Table); CMS pages render from raw.
- 2026-09-15 — About CMS pages (`/about`, `/about/privacy`, `/about/tos`); nested page slugs; footer copyright.
- 2026-09-15 — Nav Team and admin Team Skaters renamed FS Team.
- 2026-09-15 — Queue list Edit/Save/Delete sit on the right.
- 2026-09-15 — Stop tracking `seshsofa.mp4`; copy it onto the VPS by hand.
- 2026-09-15 — Discord/Twitch/YouTube icons stay in nav when logged in.
- 2026-09-15 — Site-wide `background.png`; dropped fire-field page overlay.
- 2026-09-15 — Login: bold labels; louder back link; team/friends copy.
- 2026-09-15 — Visitor: PNG bg, social icons, no Sesh Sofa brand; well 70%/720px.
- 2026-09-15 — Rerun `ALTER` migrations ignore duplicate columns (hosts `host`).
- 2026-09-15 — `SUBOTTO_INSTANCE` in Core env; Spot/Episodes fetch that host.
- 2026-09-15 — Spot/Episodes edit is `DISCORD_HOSTS_ROLE_ID`; hub admin renamed.
- 2026-09-15 — Show-site gradient `<hr>` is site-wide.
- 2026-09-15 — Episode sections split by the show-site gradient `<hr>`.
- 2026-09-15 — YouTube cards: no purple border; episode links on one row.
- 2026-09-15 — Episode thumbs are two-up Videos-style cards.
- 2026-09-14 — `/episodes` archive: auto TOC, configurable rows, Subotto stub, EP1–19 backfill.
- 2026-09-14 — Fire-field top color sine-cycles red↔violet, 30s.
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
