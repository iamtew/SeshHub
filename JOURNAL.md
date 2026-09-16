# SeshHub journal

Status, not spec. Spec lives in `README.md`.

## Now

- `/videos` paginates 9/18/27 (default 9) via `?n=&p=`. Per-skater clip publish filter at the bottom of `/dashboard/profile` (AND rules; Test Keep/Hidden); public `/videos` and team pages show only that user's matching clips. Public site, OAuth (login + link Discord/YouTube), access queue, Discord-role team + skater profiles, articles, per-skater YouTube clips, admin users (merge/unlink/delete), custom pages (nested slugs; seeded `/about`, `/about/privacy`, `/about/tos`), Spot (`/` markdown + Subotto `{{placeholders}}`), Episodes (`/episodes` archive + Subotto stub; hosts-role edit), `-port`. Markdown: basic textarea + live goldmark preview; Advanced overlay is CDN Monaco. `DISCORD_HUB_ADMIN_ROLE_ID` + `DISCORD_HOSTS_ROLE_ID`. `SUBOTTO_INSTANCE` (Core env) is the Subotto host. Nav About after Videos; footer Privacy / ToS / SeshHub.
- Dark neon CSS in `web/static/css/app.css`. Site-wide `background.png`. Visitor: sofa hero + Discord/Twitch/YouTube + Hub square. Sesh Hub (logged-in): folded bar + avatar; same three social icons, no Hub square. `/login` is the OAuth picker. Well is 70% / min 720px. Queue list actions sit on the right. Public nav and admin skaters link are **FS Team**.
- Halftone-over-gradient recipe in `sesh_halftone.md` (local lift-out; not always in git).
- `docs/privacy-policy.md` + `docs/tos.md` are the source of truth for `/about/privacy` and `/about/tos` (YouTube API Services + Limited Use + Google PP/revoke links). Contact mailbox is `tewmten@gmail.com`. Paste both into `/admin/pages` (seed never overwrites). OAuth consent screen URLs: `https://hub.seshsofa.nl/about/privacy` and `/about/tos`. `/login` states YouTube sign-in agrees to those. Consent gate, no session IP/UA, complete erasure + `/account` self-service. Fonts stay on cdnfonts. Dead `SESSION_SECRET` gone. GDPR is a hard agent rule (`.cursor/rules/privacy-gdpr.mdc` + `AGENTS.md`); code must not grow past the policy. Optional `YOUTUBE_DATA_API_KEY` hourly stats poll on cached clip IDs (title, channel name, views/likes/comments counts; zeros hidden).

## Next

- Turso only when a real need shows up.

## Log

- 2026-09-16 — Privacy/ToS contact mailbox is `tewmten@gmail.com`. Paste CMS.
- 2026-09-16 — Clip publish filter sits at the bottom of Skater profile (`/dashboard/profile`).
- 2026-09-16 — Clip publish filter lives on `/account` per user; public `/videos` and team pages show only that owner's matching clips.
- 2026-09-16 — `/videos` allowlist is site-wide: visitors see the same filtered gallery; login still required to edit.
- 2026-09-16 — Logged-in `/videos` allowlist (title/channel/category/tags) + Test Keep/Hidden; privacy §4.1/§9. Paste CMS privacy.
- 2026-09-16 — YouTube stats poller (`YOUTUBE_DATA_API_KEY`): hourly public counts on cached clip IDs; hide zeros.
- 2026-09-16 — `/videos` pager: 9/18/27 default 9; louder chartreuse bar.
- 2026-09-16 — `/videos` pager: 10/30/50, range, first/prev/next.
- 2026-09-16 — News editor wired from the form itself (preview + Advanced); admin menu says News, not Articles. Bounce `just dev` for 401-on-unauth `/preview`.
- 2026-09-16 — Spot edit: Subotto fields sit under markdown/preview.
- 2026-09-16 — Public nav always shows; no mobile Menu burger.
- 2026-09-16 — Markdown editor: live goldmark preview, CDN Monaco overlay, Save / Save and close / Discard. Custom CSS folded. Bounce `just dev` for `POST /preview`.
- 2026-09-16 — Privacy/ToS rewritten for Google OAuth publish (YouTube API Services, Limited Use, Google PP + revoke URLs). Paste CMS; point Cloud Console at hub.seshsofa.nl/about/privacy.
- 2026-09-16 — `GoatOps@stupid.systems` confirmed live; Meat Bag pastes the policy into `/admin/pages`.
- 2026-09-16 — Privacy contact mailbox is `GoatOps@stupid.systems`.
- 2026-09-16 — Privacy P0–P3: consent before gtag, session IP/UA gone, DeleteUser in a tx with tombstone + profile/clips, `/account` unlink/export/delete, logout all sessions, drop `SESSION_SECRET`. Fonts stay remote (Pill Gothic is commercial). Paste the policy into the CMS.
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
