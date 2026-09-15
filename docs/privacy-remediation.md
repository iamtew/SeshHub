# Privacy remediation plan

Companion to [`privacy-policy.md`](privacy-policy.md). That document's **§9 Known limitations** is a public admission of eight things our code gets wrong. This is the plan to retire them.

**Rule of thumb:** every fix below ends by deleting a numbered item from §9. If a fix lands and §9 still claims the flaw, the policy is lying again. Update both in the same commit.

Status lives in `JOURNAL.md` **Next**. This file is the *how*, not a second tracker.

## Priorities

| | Theme | Why this order |
|---|---|---|
| **P0** | Google Analytics consent | The only item with live legal exposure. Analytics cookies without consent breach ePrivacy/AVG today, on every page load. |
| **P1** | Stop collecting what we don't use | Cheapest wins in the whole plan. Both are smaller diffs than the status quo. |
| **P2** | Make data subject rights real | Erasure and portability currently depend on an admin doing it by hand, and erasure is incomplete even then. |
| **P3** | Hygiene | Real but minor. Do when passing. |

---

## Read this first: three traps

Three of these fixes have an obvious implementation that is wrong. Each one is fully worked through in its own section below, but they are listed here because getting any of them wrong is worse than not doing the fix at all. **These must be addressed, not discovered.**

### Trap 1 — Anonymising an article author deletes the article

The laziest way to unblock erasure for an author looks like nulling or garbaging `articles.author_id`. Both break the site.

Every article query inner-joins the author: `FROM articles a JOIN users u ON u.id = a.author_id`, in `ListPublished`, `ListAll`, `ListByAuthor` and the single-article get. An `author_id` pointing at a row that no longer exists makes the article **silently vanish from every list and every detail page** — no error, no 404, just gone from `/news`. And `author_id` is `NOT NULL`, so nulling it is not available anyway.

**Fix:** a seeded tombstone user, reassigned on delete. Worked through in **P2-2 (c)**.

### Trap 2 — `DROP COLUMN` in a migration bricks the server on the second boot

[`db.Migrate`](../internal/db/db.go) re-runs **every** `*.sql` file in the migrations directory on **every** start. Migrations here are not tracked as applied, so they must be idempotent.

`ALTER TABLE sessions DROP COLUMN ip_address` succeeds on the first boot and then fails on the next one, and `main.go` exits on a migrate error. The server would come up once, deploy fine, and refuse to start ever again.

**Fix:** `UPDATE sessions SET ip_address = NULL, user_agent = NULL`, which is safe to repeat and scrubs on every deploy. Worked through in **P1-2 step 2**. The same rule applies to any future migration in this repo.

### Trap 3 — Erasure without a transaction can half-delete a person

`DeleteUser` fires four separate `Exec` calls with no transaction. A failure partway through leaves an account that is partly erased and partly not, which is the single worst outcome for a right-to-erasure request: we would have told someone their data was gone while some of it remained.

This gets worse as P2-2 adds more statements to the same function, and it interacts with the profile-recreation race described in P2-2 — `maybeEnsureProfile` runs on every request, so a non-atomic delete leaves a window where a profile can come back.

**Fix:** wrap the whole function in `db.Begin()`, copying the shape `MergeUsers` already uses in the same file. Worked through in **P2-2 (d)**, and it is why every snippet in that section uses `tx.Exec`. This is also the one fix in the plan that gets a real test rather than a manual check.

---

## P0-1. Consent gate before Google Analytics

**Problem.** [`base.html`](../web/templates/layouts/base.html) lines 21-29 load `gtag.js` whenever `GTAG_ID` is non-empty. No banner, no opt-out, no Consent Mode. Retires §9 item 1.

**Approach.** Gate it server-side rather than client-side. `render()` already decides whether to pass `GTag` to the template, so the tag simply never reaches the browser without consent. This is less code than Google Consent Mode and strictly better for the visitor: an unconsented page makes zero requests to Google, rather than a throttled one.

No JavaScript. A plain form with two submit buttons works with JS disabled and needs no client code at all.

**1. Read the choice in `render()`** — [`internal/web/server.go`](../internal/web/server.go) line 107, replacing `data["GTag"] = s.cfg.GTagID`:

```go
if s.cfg.GTagID != "" {
    switch consentFrom(r) {
    case "yes":
        data["GTag"] = s.cfg.GTagID
    case "":
        data["ConsentAsk"] = true // undecided: show the banner
    }
}
```

`render()` already takes `r`, so nothing else needs rethreading.

**2. Handler and cookie** — new `internal/web/consent.go`, roughly 25 lines:

```go
const consentCookie = "seshhub_consent"

func consentFrom(r *http.Request) string {
    c, err := r.Cookie(consentCookie)
    if err != nil {
        return ""
    }
    return c.Value
}

func (s *Server) consent(w http.ResponseWriter, r *http.Request) {
    v := "no"
    if r.FormValue("choice") == "yes" {
        v = "yes"
    }
    http.SetCookie(w, &http.Cookie{
        Name: consentCookie, Value: v, Path: "/",
        HttpOnly: true, SameSite: http.SameSiteLaxMode,
        Secure: s.cfg.CookieSecure(),
        MaxAge: 180 * 24 * 60 * 60,
    })
    // Only bounce back to our own paths; a bare "/" fallback stops open redirects.
    to := r.FormValue("return")
    if !strings.HasPrefix(to, "/") || strings.HasPrefix(to, "//") {
        to = "/"
    }
    http.Redirect(w, r, to, http.StatusFound)
}
```

Note `HttpOnly`: only the server reads this cookie. Default the choice to `"no"` so a malformed submission declines rather than accepts.

**3. Route** — `POST /consent` in [`server.go`](../internal/web/server.go), next to `POST /access/request`.

**4. Banner** — new `web/templates/partials/consent.html`, rendered from `base.html` under `{{if .ConsentAsk}}`. Needs a hidden `return` field carrying the current path. `render()` already puts `Path` in the data map, so `value="{{.Path}}"`.

**5. Style** — a fixed-position bar in [`app.css`](../web/static/css/app.css). Keep it out of the way of the fold.

**Check.** `curl -s localhost:53053/ | rg googletagmanager` returns nothing on a cold request; the same with `-b 'seshhub_consent=yes'` returns the script tag. Do this against the Meat Bag's running instance; a new route needs them to bounce their own `just dev`.

**Also fix §7 of the policy** — the legal basis table currently says "We do not currently ask for it." Once this lands it becomes consent under Art. 6(1)(a), properly obtained.

---

## P0-2. Fill the two publishing placeholders

Not code. [`privacy-policy.md`](privacy-policy.md) carries two `[TODO before publishing]` markers:

1. **A working privacy contact address.** Art. 13 needs a real route to the controller. `privacy@seshsofa.nl` is a placeholder, not a verified mailbox.
2. **Hosting provider and country**, in §13. The international-transfer position depends on where the server actually is.

The policy should not be pasted into `/admin/pages` with these unresolved.

---

## P1-1. Purge expired session rows

**Problem.** Nothing deletes a session row once it expires. `UserByToken` filters on `expires_at > datetime('now')`, so a stale row is harmless for auth, but it keeps an IP address and User-Agent indefinitely. Retires §9 item 2.

**Fix.** One function beside the other session helpers in [`internal/auth/store.go`](../internal/auth/store.go):

```go
// ponytail: boot-time only. If uptime ever exceeds the 30-day session window,
// add a 24h ticker; until then a deploy is the purge.
func PurgeExpiredSessions(db *sql.DB) error {
    _, err := db.Exec(`DELETE FROM sessions WHERE expires_at <= datetime('now')`)
    return err
}
```

Call it in [`cmd/server/main.go`](../cmd/server/main.go) right after `db.Migrate` (line 33-36). Log and continue on error rather than exiting; a failed purge is not worth refusing to boot over.

**Check.** Insert a row with `expires_at` in the past, restart, confirm it is gone. If P1-2 lands first this is even simpler, because there is nothing sensitive left in the row.

---

## P1-2. Stop storing session IP and User-Agent

**Problem.** `sessions.ip_address` and `sessions.user_agent` are **write-only**. The single reference in the whole codebase is the `INSERT` at [`store.go`](../internal/auth/store.go) line 284. No `SELECT`, route, or template reads either one. Retires §9 item 3, and §9 item 2 loses most of its sting.

This is the laziest item in the plan: **deleting the collection is a smaller diff than keeping it.**

**1. Drop the parameters.** `CreateSession(db *sql.DB, userID string)` — remove `ip, ua` and the two columns from the INSERT. One caller to update, `issueSession` in [`internal/web/auth.go`](../internal/web/auth.go) line 212, which stops calling `r.RemoteAddr` and `r.UserAgent()` entirely.

**2. Clear the history.** New `internal/db/migrations/007_clear_session_pii.sql`:

```sql
UPDATE sessions SET ip_address = NULL, user_agent = NULL;
```

**Use `UPDATE`, not `ALTER TABLE ... DROP COLUMN`.** [`db.Migrate`](../internal/db/db.go) re-runs every `*.sql` file on every boot, so migrations must be idempotent. `DROP COLUMN` succeeds once and then fails the second start, taking the server down with it. The `UPDATE` is harmless to repeat and doubles as a belt-and-braces scrub on every deploy.

The two columns stay in the schema as dead weight. That is fine and deliberate: they will hold nothing. Tidy them away only if a table rebuild is ever justified for another reason.

**3. Update the docs.** §4.2 and §8 of the policy both describe storing an IP. [`README.md`](../README.md) lines 328-329 document the schema and can keep the columns listed.

**Check.** Log in, then `SELECT ip_address, user_agent FROM sessions` returns NULLs.

---

## P2-1. Self-service unlink, delete and export

**Problem.** There is no way for a member to unlink a provider, delete their account, or get their data out. Everything routes through an admin. Retires §9 item 7. Retires §9 item 8 if the logout change is included. Retires the "no self-service tooling" caveat in §10.

**The handlers already exist** — `auth.UnlinkDiscord`, `auth.UnlinkYouTube` and `auth.DeleteUser` are written and tested by admin use. This is a wiring job, not new logic.

**Routes** in [`server.go`](../internal/web/server.go) beside `GET /account`:

```
POST /account/unlink/discord
POST /account/unlink/youtube
POST /account/delete
GET  /account/export
```

Each handler starts from `UserFrom(r)` and acts on `u.ID` only, so there is no object-level authorization to get wrong — a member can only ever reach their own record.

**Self-delete gets through the guard for free.** `DeleteUser(db, id, actorID)` refuses when `id == actorID`, which exists to stop an admin nuking themselves from the admin list. Calling `auth.DeleteUser(s.db, u.ID, "")` passes an empty actor, clears the guard, and needs no change to `DeleteUser`. After it returns, clear the session cookie and redirect home — the rows are already gone.

**One guard to add:** refuse to unlink the last remaining provider, or the account becomes unreachable with data still in it. Check the *other* provider is present first, and if it is not, tell them to use delete instead.

**Confirmation.** `onsubmit="return confirm(...)"` on the delete form.

```
ponytail: JS confirm only. Without JS the POST goes straight through.
Upgrade to a /account/delete confirm page if that ever bites.
```

**Export** (Art. 20, retires part of §10): one handler, `encoding/json` over the account row, the skater profile and the access request, with `Content-Disposition: attachment`. Everything it needs is already exposed by existing store helpers.

**UI.** [`account.html`](../web/templates/pages/account.html) currently lists connection status and a logout link. Add the unlink buttons next to each provider, and a clearly separated danger area for export and delete.

**Check.** A throwaway account: export returns its data, unlink clears the provider columns, delete removes the row and logs the browser out.

---

## P2-2. Finish `DeleteUser`

**Problem.** [`DeleteUser`](../internal/auth/store.go) lines 168-190 clears `access_requests`, `sessions` and `users`, and stops there. Retires §9 items 4, 5 and 6.

Three leftovers and one ordering trap:

**a. The skater profile survives.** Nothing deletes `skater_profiles`. `ListTeam` filters on `p.user_id IS NOT NULL AND p.user_id != ''`, which tests whether the column is *populated*, not whether that user still exists. A deleted person's `real_name`, `bio` and `location` stay on `/team`. Add:

```go
if _, err := tx.Exec(`DELETE FROM skater_profiles WHERE user_id=?`, id); err != nil {
    return err
}
```

**b. Cached clips survive.** No `DELETE FROM youtube_videos` exists anywhere. Read the channel before dropping the user row, since that is the only link:

```go
var channel string
_ = tx.QueryRow(`SELECT IFNULL(youtube_channel_id,'') FROM users WHERE id=?`, id).Scan(&channel)
// ... after the users delete:
if channel != "" {
    if _, err := tx.Exec(`DELETE FROM youtube_videos WHERE channel_id=?`, channel); err != nil {
        return err
    }
}
```

**c. Articles block deletion entirely.** `ErrHasArticles` aborts the whole operation for any author, so an erasure request from someone who has written a post cannot be completed at all.

Do **not** reach for the obvious fix here. Every article query inner-joins the author:

```go
// internal/article/article.go lines 58-67, and the same JOIN in ListAll,
// ListByAuthor and the single-article get.
const cols = `a.id, a.slug, ..., a.author_id, u.display_name, ...`

func ListPublished(db *sql.DB) ([]Article, error) {
	return query(db, `SELECT `+cols+` FROM articles a JOIN users u ON u.id = a.author_id WHERE a.status = 'published' ORDER BY a.published_at DESC`)
}
```

Point `author_id` at a row that does not exist and **the article silently disappears from every list and detail page**. Nulling it is not an option either: the column is `NOT NULL`.

Recommended: a **tombstone user**. Seed one reserved row, then reassign on delete.

```sql
-- 008_tombstone_user.sql
INSERT INTO users (id, username, display_name, role)
SELECT 'deleted-user', 'deleted', 'Former member', 'member'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = 'deleted-user');
```

```go
if _, err := tx.Exec(`UPDATE articles SET author_id='deleted-user' WHERE author_id=?`, id); err != nil {
    return err
}
```

Articles stay published, the byline reads "Former member", the inner join still resolves, and no query or template changes. `ErrHasArticles` and its callers can go. Exclude `deleted-user` from `/admin/users` so nobody tries to merge or delete it.

**d. Wrap the whole thing in a transaction.** `DeleteUser` currently fires four separate `Exec` calls with no transaction, so a failure partway through leaves a half-deleted account — the worst possible outcome for an erasure request. `MergeUsers` already uses `db.Begin()` in the same file; copy that shape. This is what makes the snippets above use `tx.Exec`.

**The ordering trap.** `maybeEnsureProfile` runs from `injectUser` on *every request*, and `EnsureForUser` recreates a missing profile. Deleting a profile while its owner is still signed in and still holds the Discord role brings back a fresh empty one on their next page load. Deleting the account first ends their sessions, which closes the window — so inside `DeleteUser`, sessions must go before profiles. A transaction makes the whole thing atomic and the concern disappears, which is another reason for (d).

**Check.** Worth a real test file, `internal/auth/store_test.go` — this is the one path where a bug means we told someone their data was erased when it was not. Seed a user with a profile, a session, an access request, a cached video and an article; delete; assert the profile, session, request and video rows are gone and the article survives with `author_id = 'deleted-user'`.

---

## P3-1. Self-host the three web fonts

**Problem.** [`base.html`](../web/templates/layouts/base.html) lines 13-16 pull Pill Gothic 600mg, Monster Chiller and Turbo Jungle from `fonts.cdnfonts.com` on every page, handing that CDN every visitor's IP before anyone has consented to anything. Removes a bullet from policy §6.

**Fix.** Download the three `woff2` files into `web/static/fonts/`, add `@font-face` blocks to [`app.css`](../web/static/css/app.css), and delete the four `<link>` tags. The three `font-family` declarations at lines 204, 225 and 236 keep working unchanged. Also drops three blocking third-party requests from the critical path.

**Check licensing first.** These are free-to-use faces on a CDN, which does not automatically mean they are licensed for redistribution from our own origin. Confirm before committing the files, and if any one of them is not redistributable, keep that single face remote and self-host the rest.

## P3-2. Delete `SESSION_SECRET`

Dead config. Set at [`config.go`](../internal/config/config.go) lines 14 and 37, read by nothing. Sessions are opaque random tokens looked up by SHA-256 hash; there is nothing to sign. Remove the field, the `.env.example` line, and the three README mentions — including line 102, which tells the operator to set a non-default value in production for no effect whatsoever.

## P3-3. `r.RemoteAddr` behind Caddy

The README's reverse proxy config forwards `X-Real-IP` and `X-Forwarded-For`, and the app reads neither, so the IP we record is `127.0.0.1`. Worth knowing if anyone is ever tempted to "fix" the IP column: don't. **P1-2 deletes the collection instead, and this evaporates with it.** Listed only so the observation is not rediscovered as a bug.

---

## Suggested commits

Small and independently revertable. Each one that closes a §9 item edits the policy in the same commit.

1. `privacy: consent gate before gtag.js` — P0-1, drops §9 item 1, rewrites §7 analytics row.
2. `privacy: purge expired sessions at boot` — P1-1, drops §9 item 2.
3. `privacy: stop storing session ip and user-agent` — P1-2, drops §9 item 3, edits §4.2 and §8. **Trap 2 applies.**
4. `auth: complete DeleteUser, wrap in tx, tombstone authors` — P2-2 plus its test, drops §9 items 4, 5 and 6. **Traps 1 and 3 apply.**
5. `web: self-service unlink, delete and export` — P2-1, drops §9 items 7 and 8, rewrites §10.
6. `chore: self-host fonts, drop dead SESSION_SECRET` — P3.

Commit 4 is the one to slow down on. It is the only fix here where a silent bug produces a false statement to a data subject, and it carries two of the three traps.

After commit 5, §9 should be empty. Delete the section, drop the §2 and §11 references to it, and update the closing note.
