# Feature Spec: Message Board (Forum) — SeshHub

**Version:** 2.0  
**Target:** [iamtew/SeshHub](https://github.com/iamtew/SeshHub)  
**Site:** https://seshsofa.nl / hub  
**Status:** Ready for Cursor agent  
**Style:** Follow `AGENTS.md` and existing README conventions. YAGNI. Reuse in-repo patterns. Shortest correct diff.

---

## 1. Goal

Add a private **Message Board** (classic forum) for logged-in users only.

- Sections (e.g. General, Skateboarding, Off topic) → Threads → Posts.
- Menu item only in the **logged-in** navigation (not the public main menu).
- Menu item shows an **unread badge** when the current user has new posts to see.
- Fully responsive (desktop + mobile) using the existing dark-neon CSS and layout patterns.
- No new dependencies. No real-time websockets. No attachments. No email notifications.

---

## 2. Access & Navigation

### Who can use it
- Any **authenticated** user (session present). Roles `admin`, `skater`, `friend` (and any future logged-in role) can read and post.
- Guests / unauthenticated → redirect to `/login` (preserve return URL the same way other protected routes already do).
- Section management (create / edit / reorder / deactivate) → **admin** only (existing `RequireRole` / admin middleware pattern).

### Menu
- Add “Message Board” (or “Forum”) **only** to the logged-in portion of the nav (see existing `partials/nav.html` / logged-in block).
- Do **not** put it in the public main menu.
- Badge on the menu item:
  - Small high-contrast pill/circle (match existing accent / neon style in `app.css`).
  - Shows count of unread posts for the current user.
  - Hidden when count is 0.
  - Must stay visible and non-colliding in mobile nav.

---

## 3. Data Model (libSQL / SQLite)

Follow existing style: `TEXT` primary keys, `DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`, parameterized queries, migrations as `.sql` files under `internal/db/migrations/`.

```sql
-- Sections (top-level categories)
CREATE TABLE IF NOT EXISTS forum_sections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,   -- 1 = active
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Threads
CREATE TABLE IF NOT EXISTS forum_threads (
    id           TEXT PRIMARY KEY,
    section_id   TEXT NOT NULL REFERENCES forum_sections(id),
    user_id      TEXT NOT NULL REFERENCES users(id),
    title        TEXT NOT NULL,
    slug         TEXT NOT NULL,
    is_locked    INTEGER NOT NULL DEFAULT 0,
    is_sticky    INTEGER NOT NULL DEFAULT 0,   -- optional, can ignore in v1 UI
    last_post_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(section_id, slug)
);

-- Posts
CREATE TABLE IF NOT EXISTS forum_posts (
    id            TEXT PRIMARY KEY,
    thread_id     TEXT NOT NULL REFERENCES forum_threads(id),
    user_id       TEXT NOT NULL REFERENCES users(id),
    body_raw      TEXT NOT NULL,              -- markdown or plain
    body_html     TEXT NOT NULL,              -- sanitized HTML (bluemonday)
    is_first_post INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Per-user read tracking (for badge + "unread" state)
CREATE TABLE IF NOT EXISTS forum_thread_reads (
    user_id      TEXT NOT NULL REFERENCES users(id),
    thread_id    TEXT NOT NULL REFERENCES forum_threads(id),
    last_read_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_forum_threads_section_last
    ON forum_threads(section_id, last_post_at DESC);
CREATE INDEX IF NOT EXISTS idx_forum_posts_thread_created
    ON forum_posts(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_forum_thread_reads_user
    ON forum_thread_reads(user_id);
```

**Unread definition**  
A post is unread for user U if there is no `forum_thread_reads` row for (U, thread) **or** `post.created_at > last_read_at`.  
Badge count = total number of such posts across all threads (exclude the user’s own posts if you want; either is fine, pick one and stick to it).

When a user opens a thread view → upsert `forum_thread_reads` with `last_read_at = now` (or = max post time in that thread).

---

## 4. Routes (sketch — match existing router style)

All under an auth-required group (same middleware used for other logged-in / internal pages).

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/forum` | Index – list active sections + stats |
| GET | `/forum/{sectionSlug}` | Section – list threads |
| GET | `/forum/{sectionSlug}/{threadSlug}` | Thread – list posts + reply form |
| GET | `/forum/{sectionSlug}/new` | New thread form |
| POST | `/forum/{sectionSlug}/new` | Create thread + first post |
| POST | `/forum/{sectionSlug}/{threadSlug}/reply` | Add reply |
| GET/POST | `/admin/forum/sections` … | Admin CRUD + reorder for sections |

Use the same ID generation, slugify, and form-CSRF patterns already present in the codebase.

---

## 5. Behaviour

### Index (`/forum`)
- List active sections ordered by `sort_order`.
- Per section: name, description, thread count, last activity (optional).
- Unread indicator per section is nice-to-have, not required for v1.

### Section view
- Threads ordered: sticky first (if used), then `last_post_at DESC`.
- Show title, author, reply count, last activity.
- “New Thread” button.

### Create thread
- Title + body (markdown).
- Create thread + first post (`is_first_post = 1`).
- Render body through the existing goldmark + bluemonday pipeline (same as articles/pages).
- Redirect to the new thread.

### Thread view
- Posts oldest → newest.
- Author (link to `/team/{slug}` or `/friends/{slug}` if they have a profile, otherwise plain name).
- Timestamps, “edited” if `updated_at > created_at`.
- Reply form at bottom (disabled when `is_locked`).
- On GET of the thread → mark as read for current user.

### Reply
- Body only.
- Update `forum_threads.last_post_at`.
- Redirect back to thread (bottom / new post).

### Admin sections
- List, create, edit, deactivate, reorder (`sort_order`).
- Keep it simple; match the style of other admin list + form pages.

### Edit / delete (minimum)
- Author can edit/delete own posts (and own thread title).
- Admin can edit/delete anything.
- Soft-delete optional; hard delete is acceptable for v1 if simpler. Prefer soft if the rest of the CMS already uses it.

---

## 6. UI / Templates / CSS

- Templates live under `web/templates/` (pages + partials). Reuse `layouts/base.html`.
- Add a partial for the unread badge so the nav can include it cleanly.
- Style with the existing hand-written dark-neon CSS in `web/static/css/app.css`. No new CSS framework.
- **Responsive is mandatory**:
  - Desktop: comfortable reading width, normal nav.
  - Mobile: single column, large enough tap targets, badge still visible in hamburger/collapsed nav, forms usable without horizontal scroll.
  - Follow the same breakpoints and patterns already used on the site.
- Empty states: “No threads yet — start one”, etc.
- Pagination if lists get long (reuse any existing pager pattern, or simple `?page=`).

Prefer HTMX partials only where it clearly helps (e.g. reply without full reload); otherwise plain form POSTs + redirect are fine and match the “only if a page actually needs them” rule.

---

## 7. Implementation notes (match the repo)

- **Package layout** (follow existing modules):
  - `internal/forum/` or split into models + handlers under `internal/web/handlers/`.
  - New migration file(s) in `internal/db/migrations/`.
  - Handlers registered in the existing router with the auth middleware already used for internal pages.
- **IDs**: same TEXT/UUID style as `users`, `articles`, etc.
- **Markdown**: reuse the goldmark + bluemonday path used for articles/pages. Never mark unsanitized HTML as `template.HTML`.
- **Auth**: existing session middleware + role helpers. Do not invent a new auth system.
- **No new deps**. No Node, no extra Go modules unless already present.
- Keep the diff small. YAGNI. Prefer the simplest thing that works and matches surrounding code.
- GDPR / privacy: do not log IPs, do not store extra PII. Forum posts are user content; existing account-deletion / export paths should eventually consider them (can be a follow-up).

---

## 8. Acceptance criteria

1. Unauthenticated users cannot reach any `/forum` URL (redirect to login).
2. “Message Board” appears only in the logged-in nav.
3. Unread badge shows the correct count and disappears when count is 0; viewing a thread marks it read and lowers the count.
4. Admins can manage sections (create / edit / reorder / deactivate).
5. Logged-in users can create threads and reply.
6. Threads ordered by latest activity.
7. Post bodies are safely rendered (markdown → sanitized HTML).
8. Works well on both desktop and mobile (no horizontal scroll, usable forms, badge visible).
9. No new dependencies; follows existing patterns and `AGENTS.md`.

---

## 9. Out of scope (do not implement now)

- Public / guest access
- Real-time updates / websockets
- File or image attachments
- Full-text search
- Email or push notifications
- Advanced moderation (reports, bans, etc.)
- Sticky/locked UI polish beyond the columns already in the schema

---

**End of spec.**  
Feed this file to the Cursor agent as the source of truth for the Message Board feature in SeshHub.
