# Privacy Policy - Sesh Hub

**Website:** https://hub.seshsofa.nl

**Last updated:** 18 September 2026

## 1. Who we are

Sesh Hub ("we", "us" or "our") operates the website and web application available at https://hub.seshsofa.nl (the "Service").

We are based in the Netherlands. For the purposes of the General Data Protection Regulation (GDPR) and the Dutch Implementation Act (Uitvoeringswet AVG), we are the controller of the personal data described in this policy.

**Contact.** Privacy questions and data subject requests: `tewmten@gmail.com`

**Postal address.** Louwesweg 1, 1066EA Amsterdam, The Netherlands.

## 2. What this policy is

Sesh Hub is a small community site for the Sesh Sofa crew. You sign in with Discord or YouTube, and we keep an account record for you in a database on our server.

We want to be exact about that, because an earlier version of this policy claimed we stored nothing at all. That was wrong. We do store personal data: your Discord ID, your display name, your avatar URL, your YouTube channel if you link one, and a login session.

This policy lists every category of personal data we hold, why we hold it, when it is created, who can read it, when it changes, and when it is deleted.

## 3. How you sign in

Authentication is entirely via OAuth. We never see or store a password.

**Discord.** We request the scopes `identify` and `guilds.members.read`. `identify` returns your Discord user ID, username, global display name and avatar hash. `guilds.members.read` lets us read your role list in the Sesh Sofa Discord server, which is how we decide whether you are a member, a skater, a host or an admin. We do **not** request the `email` scope, so Discord does not give us your email address.

**YouTube (Google).** We request the OAuth scope `https://www.googleapis.com/auth/youtube.readonly`, with offline access so we receive a refresh token. This returns your channel ID, channel title and channel thumbnail, and lets us list the videos on your own channel. We do not request Gmail, Drive, Google Calendar, or your Google account email.

Both flows use PKCE (S256) and a `state` parameter to prevent request forgery.

Your Discord **role IDs are not stored**. We read them at the moment you sign in, convert them into a single role label (`admin`, `skater`, `member` or `pending`) plus a `host` yes/no flag, and store only that result.

### 3.1 YouTube API Services and Google user data

Sesh Hub **uses YouTube API Services**. Information obtained through those services is also subject to [Google's Privacy Policy](https://www.google.com/policies/privacy).

**Access.** When you sign in or link YouTube, we access Google user data (YouTube API Data) limited to: your YouTube channel ID, channel title, channel thumbnail, the list and public metadata of videos on that channel (title, description, publication date, thumbnail URL, duration, view count, like count, comment count, tags), and a refresh token so we can repeat that read while you remain linked.

**Use.** We use that data only to (1) create or link your Sesh Hub account, (2) show your channel on your skater profile, and (3) cache up to 50 of your recent public uploads so `/videos` and profile pages can load without calling YouTube on every view. An API key (not your OAuth token) may refresh the public title, channel name, view count, like count and comment count on those already-cached clips about once an hour. We do not fetch comment text. We do not use Google user data for any other purpose.

**Store.** Channel fields and the refresh token live in the `users` table on our Amsterdam SQLite database. Cached clip metadata lives in `youtube_videos`. The refresh token is stored in plain text. See sections 4.1 and 4.6.

**Share.** We do not sell, rent, or transfer Google user data to third parties, advertising platforms, data brokers, or information resellers. Public clip metadata is displayed on this Service; that display is the user-facing feature. Our server calls Google's YouTube Data API with our server's IP address, not yours. Optional Google Analytics (section 6) is a separate cookie, loaded only after you Allow it, and **does not receive** YouTube OAuth tokens, refresh tokens, or channel IDs from our application.

**Limited Use.** We comply with Google's Limited Use requirements for this data, including data aggregated, anonymized, or derived from it:

- We use it only to provide or improve user-facing features that are prominent in the Service (sign-in, skater and friends profiles, the videos gallery).
- We do not transfer it except to operate those features, for security (investigating abuse), to comply with law, or as part of a sale of the Service after your explicit prior consent.
- We do not use it for targeted, personalized, retargeted, or interest-based advertising, to determine credit-worthiness, or to train generalized (non-personalized) AI or ML models.
- Humans do not read Authorized Data except when you ask us to (for example account support or deletion), for security, or to comply with law.

**Revoke and delete.** Unlink YouTube or delete your account at `/account`. Independently, revoke Sesh Hub's access at [Google's security settings](https://security.google.com/settings/security/permissions). Revoking invalidates the refresh token immediately. Unlink or account deletion clears the token, channel fields, and cached clips. Privacy questions or complaints: the contact address in section 1.

## 4. What we store, and its full lifecycle

Our database is a SQLite file on the server that runs the Service.

### 4.1 Your account (`users` table)

**What:** an internal account ID, your username, display name, avatar URL, role label (`admin`, `skater`, `friend`, `pending`, and any leftover `member` until migrated), host flag, Discord ID, Discord username, YouTube channel ID, YouTube channel title, your **YouTube OAuth refresh token**, the time of your last YouTube sync, an optional **YouTube Feed Filter** (JSON of field/operator/value rules and optional upload-type flags you set so only matching uploads from your linked YouTube channel appear on `/videos` and your public roster page), and creation and last-updated timestamps.

**Why:** to recognise you across visits, to show your name and avatar in the interface, to decide what you are allowed to see and edit, to fetch your clips if you have linked YouTube, and to apply your YouTube Feed Filter to the public gallery.

**Created:** the first time you sign in with Discord or YouTube. The YouTube Feed Filter is created when you save one on `/dashboard/profile`.

**Read:** on every request you make while signed in, to resolve your session to an account. Your display name is also shown publicly as the author of any article you write. The YouTube Feed Filter is read to decide which of your cached clips appear on `/videos` and your public roster page.

**Updated:** on **every** subsequent sign-in. We re-copy your current username, display name, avatar and roles from the provider, so changing your name or avatar on Discord changes it here the next time you log in. Linking or unlinking a provider also updates this record. Saving or clearing the YouTube Feed Filter on `/dashboard/profile` updates this record.

**Deleted:** by you at `/account`, or by an administrator. The YouTube Feed Filter is deleted with the account. Articles you wrote stay published with the byline "Former member". See section 4.5.

The refresh token deserves a specific mention: it is a long-lived credential that lets us request read-only access to your YouTube channel without you signing in again. It is stored in the database in plain text. It is cleared when a YouTube account is unlinked or the account is deleted. You can also revoke it yourself at any time from [Google's security settings](https://security.google.com/settings/security/permissions), which invalidates it immediately regardless of what we hold.

### 4.2 Your login session (`sessions` table)

**What:** a SHA-256 hash of your session token, your account ID, an expiry time and a creation time.

The session token itself is a 256-bit random value. The plain value exists only in the cookie in your browser; we store only its hash, so the contents of our database cannot be used to impersonate you.

We do **not** store your IP address or User-Agent.

**Why:** to keep you signed in between requests.

**Created:** each time you sign in. Valid for 30 days.

**Read:** on every request, to look up your account.

**Updated:** never. Sessions are created and deleted, not modified.

**Deleted:** when you click log out, which deletes **all** of your sessions (every browser and device). Expired session rows are also purged when the server starts. All of your sessions are deleted if your account is deleted.

### 4.3 Your skater profile (`skater_profiles` table)

**What:** a public slug, your skater name, your real name, a biography, your stance, a roster status, avatar and banner URLs, optional **corner-radius values**, a **border style name** (none, a solid default or custom colour, or a named animation), and for a custom colour an optional **hex colour**, optional **border width** (pixels) and **border blur** (pixels, default 0), your location, sponsors, social links, signature tricks, a featured video ID, and **former slugs** (only while nobody else is using that URL). If you upload a site photo, we also store a re-encoded JPEG file on the same Amsterdam server (`data/avatars/{id}.jpg`, shown at `/media/avatars/{id}.jpg`). The original upload is not kept. We strip metadata by re-encoding. Maximum upload size is 15 MB; we store a square JPEG at most 1024 pixels on a side. You may also upload a public gallery of up to 10 photos; those files are stored as described in section 4.10.

**Why:** to show the FS Team roster at `/team` and the Friends roster at `/friends`, and your own profile page at `/team/{slug}` or `/friends/{slug}`. Former slugs 302 to your current page so old links keep working, until that slug is claimed again.

**Created:** automatically when you hold the skater or admin role in the Sesh Sofa Discord and sign in; when you hold the Discord friends role; when an administrator approves your access request (Friends); or when a former `member` account is migrated to Friends. Your skater name is initially taken from Discord (or your YouTube channel title if Discord is not connected), and is kept in step with it on subsequent visits. Your public slug starts as a slugified form of that name (migrated friends may start with a unique id slug until you edit it). The check that creates the profile runs on every request you make, so as long as your account exists and still holds a roster role, a profile will exist.

**Read: this profile is public.** Anyone on the internet, signed in or not, can see it. Note in particular that the name shown publicly is your **display name (stored as real name) if you have filled it in**, and falls back to your skater name only if you have not. Your **slug** is the public URL `/team/{slug}` (FS Team) or `/friends/{slug}` (Friends). Former slugs are public too: visiting them redirects to your current page. Your **location**, **biography**, **profile photo** (site upload or Discord/YouTube fallback, plus the frame shape and any border style), and **gallery photos** are also shown publicly when set. Please do not put anything in these fields that you would not want a stranger to read.

**Updated:** by you, at `/dashboard/profile`. That form edits your display name, slug, biography, stance, location, featured video, YouTube Feed Filter, photo frame (corner roundness, border style, optional custom hex colour, border width, and border blur), and an optional site-only profile photo. Gallery photos are added and deleted at `/dashboard/gallery`. Changing your slug records the previous one as a redirect and frees it for anyone to claim later. Logging in still refreshes your Discord skater name and the provider avatar URL on your account record; it does not overwrite a slug you chose, and it does not overwrite a site photo you uploaded. Clearing the site photo falls back to Discord or YouTube. An administrator can separately change your roster status.

**Deleted:** when your account is deleted, or by an administrator. Former-slug redirects are deleted with the profile. The site photo file and any gallery photo files are deleted with the profile. If someone else takes an old slug, that redirect row is deleted so their page wins.

### 4.4 Your access request (`access_requests` table)

**What:** your account ID, a status of pending, approved or rejected, the account ID of the administrator who reviewed it, and timestamps.

**Why:** if you sign in with YouTube, or with Discord while outside the Sesh Sofa server, or while in the server without the hub-admin, skater, or friends Discord role, you land in an approval queue rather than getting a public roster slot.

**Created:** when you ask for access. **Read:** by administrators reviewing the queue, and by you to see your own status. **Updated:** when an administrator approves or rejects it; approval also changes your role to friend and creates a public Friends profile. Approval cannot make you FS Team (that requires the Discord skater role). **Deleted:** together with your account.

### 4.5 Articles you write (`articles` table)

Articles record the account ID of their author, and your display name is published alongside the article. Article text is written by you and may contain whatever you put in it.

If your account is deleted, authorship is reassigned to a reserved "Former member" record so the article stays on `/news`. The byline no longer shows your name.

### 4.6 Cached clips from your YouTube channel (`youtube_videos` table)

**What:** for up to 50 recent uploads on a linked channel, the video ID, your channel ID, the public channel title, title, description, publication date, thumbnail URL, duration, YouTube live-broadcast status (`none`, `live`, or `upcoming`), view count, like count, comment count, tags, and an `is_hidden` flag you can set so a clip stays in the cache but does not appear on `/videos` or your public roster page. We do not store comment text. Live-broadcast status is used only to classify upload type (video / short / live / premiere) for the YouTube Feed Filter.

**Why:** so the `/videos` gallery and public roster profiles load from our database instead of calling the YouTube API on every page view. Counts of zero are stored but not shown. The hide flag lets you pull a single cached upload off those public lists without changing your YouTube Feed Filter.

**Created and updated:** in the background when a signed-in team skater or friend visits the Service and the cache for their channel is more than an hour old; and, if an API key is configured, about once an hour for every clip already in the table (public statistics only). You can hide or unhide a clip from **Edit YouTube feed** on your roster page; that only updates `is_hidden`.

**Read:** publicly, on `/videos` and on FS Team / Friends profile pages, except clips you have hidden. You can still see those hidden clips yourself while editing your YouTube feed.

**Deleted:** when your account is deleted, or when an administrator deletes the account that owned the channel.

### 4.7 Episode archive (`episodes` table)

The archive of past Spot Challenge episodes at `/episodes` includes a winner name for each episode. These are community handles and channel names, some of them seeded from the show's history, not account records. They are public.

### 4.8 Server logs

The application writes error-level logs to standard output. These can include an account ID when something fails for a specific user, for example a failed YouTube sync. The application does **not** keep an HTTP access log, and does not write request paths, IP addresses or User-Agent strings to its logs.

The Service sits behind a reverse proxy, which may keep its own request logs at the infrastructure level, including IP addresses, according to its default configuration.

There is also an unused `sync_logs` table in our database schema. Nothing ever writes to it; we mention it only for completeness.

### 4.9 What we do not do at all

We do not send email. There is no newsletter and no notification system. The only files you can upload are an optional square profile photo on `/dashboard/profile` (JPEG or PNG, 15 MB) and up to 10 gallery photos on `/dashboard/gallery` (JPEG or PNG, 15 MB each), stored as described in sections 4.3 and 4.10. We do not sell, rent or trade personal data. We do not profile you, build advertising audiences, or use your data for anything beyond operating the site.

### 4.10 Your gallery photos (`gallery_photos` table)

**What:** up to 10 photo records per public roster profile: an internal file id, the profile id, a sort position, and a created timestamp. Each photo is a re-encoded JPEG on the same Amsterdam server (`data/gallery/{id}.jpg`, shown at `/media/gallery/{id}.jpg`). The original upload is not kept. We strip metadata by re-encoding. Maximum upload size is 15 MB per photo; we keep the original aspect ratio and store a JPEG whose long edge is at most 1600 pixels.

**Why:** so you can show a short public slideshow on your roster page, and so `/photos` can list everyone’s gallery.

**Created:** when you add a JPEG or PNG on `/dashboard/gallery`, while you still have fewer than 10 photos.

**Read: these photos are public.** Anyone on the internet can see them on your roster page and on `/photos`.

**Updated:** when you drag to reorder photos on `/dashboard/gallery`. We only change the sort position, not the image file. You add or delete photos separately.

**Deleted:** when you delete a photo on `/dashboard/gallery`, when your account is deleted, or when an administrator deletes the profile. The database row and the JPEG file both go.

## 5. Cookies

| Cookie | Purpose | Lifetime | Notes |
|--------|---------|----------|-------|
| `seshhub_session` | Keeps you signed in | 30 days | Strictly necessary. `HttpOnly`, `SameSite=Lax`, and `Secure` in production. Contains a random token, no personal data. |
| `seshhub_oauth` | Holds the OAuth `state` and PKCE verifier during sign-in | 10 minutes | Strictly necessary. Deleted as soon as sign-in completes. |
| `seshhub_consent` | Remembers whether you allowed Google Analytics | 180 days | `HttpOnly`, `SameSite=Lax`, and `Secure` in production. Values `yes` or `no`. |
| Google Analytics cookies | Audience statistics | Set by Google, typically up to 2 years | **Not** strictly necessary. Only set after you click Allow. See section 6. |

Disabling the two strictly necessary cookies will stop you from being able to sign in. You can control all cookies through your browser settings. You can change the analytics choice by clearing `seshhub_consent` and answering the banner again.

## 6. Third parties, and who receives your IP address

Any time your browser loads something from another company's server, that company receives your IP address, because that is how the internet works. The following are loaded by pages on this Service:

- **Google Analytics 4**, from `googletagmanager.com`, only after you allow it. Without that choice, the tag is not sent to your browser at all.
- **Web fonts** from `fonts.cdnfonts.com`, on every page. These faces are commercially licensed, so we load them from the CDN rather than copying the files onto our server.
- **YouTube video thumbnails** from `img.youtube.com` and `i.ytimg.com`, on the videos gallery, episode archive and profile pages.
- **YouTube video players** from `youtube-nocookie.com`, on skater profile pages that have clips. We deliberately use YouTube's privacy-enhanced domain, which does not set tracking cookies until you press play.
- **Discord avatar images** from `cdn.discordapp.com`, wherever a Discord (or YouTube) avatar is displayed and you have not uploaded a site photo. These are loaded with `referrerpolicy="no-referrer"`, so Discord is not told which page you were on. Site photos are served from this Service at `/media/avatars/…`. Gallery photos are served from this Service at `/media/gallery/…`.

Our server also talks to these services directly. In those cases your IP address is not sent; ours is.

- **Discord's API**, during sign-in, to read your profile and your roles in the Sesh Sofa server.
- **Google's YouTube Data API**, during sign-in and during clip sync, to read your channel and its public videos; and, if configured, during an hourly stats poll of video IDs already in our database (title, channel name, view/like/comment counts only).
- **Subotto** (`subotto.seshsofa.nl`), to fetch current episode information for the homepage. This is an anonymous request containing no information about you.

Each of these companies handles your data under its own privacy policy. You can revoke our access to your Discord account from Discord's settings. You can revoke YouTube / Google access from [Google's security settings](https://security.google.com/settings/security/permissions). Google's handling of data is described in [Google's Privacy Policy](https://www.google.com/policies/privacy).

## 7. Purposes and legal basis

| What we do | Why | Legal basis |
|------------|-----|-------------|
| Create and maintain your account record | You cannot use a members' site without an account | Art. 6(1)(b) - performance of a contract, being the service you signed up for |
| Keep you signed in via a session cookie | Otherwise you would log in on every page | Art. 6(1)(b), and Art. 6(1)(f) legitimate interest in account security |
| Read your Discord roles to decide your permissions | To keep members' areas restricted to members | Art. 6(1)(f) - legitimate interest in access control |
| Publish your skater profile, gallery photos, and article authorship | This is the purpose of the roster, `/photos`, and the news section, and you control the content | Art. 6(1)(f), with your role in publishing it |
| Cache your YouTube clips | To show the gallery without hammering the YouTube API | Art. 6(1)(f) - legitimate interest in a functioning site |
| Google Analytics | Audience statistics | Art. 6(1)(a) - consent, via the banner. Declining (or ignoring the banner) means the tag never loads. |

## 8. Retention

- **Account record:** kept until you delete it at `/account`, or an administrator deletes it.
- **Session record:** valid for 30 days. Logging out deletes every session for your account. Expired rows are purged when the server starts.
- **Skater profile:** kept until the account is deleted, or an administrator removes the profile. A site photo file and gallery photo files are kept for the same time and deleted with the profile.
- **Cached YouTube clips:** kept until the owning account is deleted.
- **Access requests:** kept until the account is deleted.
- **Articles and episode entries:** kept as part of the site's published archive. Deleted authors are shown as "Former member".

## 9. Your rights under the GDPR

You have the right of access (Art. 15), rectification (Art. 16), erasure (Art. 17), restriction (Art. 18), data portability (Art. 20), and objection (Art. 21).

**How to exercise these rights.** Most of them are on `/account` while you are signed in: unlink a provider, download a JSON export of your account, profile (including photo URL and frame settings), gallery photo URLs, former slugs, YouTube Feed Filter and access request, or delete the account. Logging out ends every session. You can also email the contact address in section 1. We will respond within one month, as Art. 12(3) requires.

- **Access or portability:** use **Download my data** on `/account`, or ask us by email. The export includes the public URL of a site photo if you uploaded one, and the public URLs of gallery photos, not the image bytes.
- **Erasure:** use **Delete my account** on `/account`. That removes your account record (including the YouTube Feed Filter), sessions, access request, skater profile, former-slug redirects, cached clips, any site profile photo file, and any gallery photo files. Articles stay published as "Former member".
- **Rectification:** most profile fields update themselves from Discord or YouTube on your next sign-in. Your display name, slug, biography, location, featured clip, YouTube Feed Filter, site photo and photo frame (including border style, width, and blur) are yours to edit at `/dashboard/profile`. Gallery photos are yours to add and delete at `/dashboard/gallery`.
- **Objection or restriction:** tell us what you object to and we will stop it or explain why we believe we may continue.

Some of the underlying personal data also lives with Discord and Google, and we cannot reach into their systems. Revoking our OAuth access from their settings pages is immediate and does not require us.

You have the right to lodge a complaint with the Dutch Data Protection Authority, the Autoriteit Persoonsgegevens: https://autoriteitpersoonsgegevens.nl

## 10. Security

What is actually true about how we protect this data:

- Session tokens are stored as SHA-256 hashes, so a copy of our database does not let an attacker log in as you.
- Session tokens are 256 bits of cryptographic randomness.
- The session cookie is `HttpOnly`, so scripts in the page cannot read it, and `SameSite=Lax`.
- Traffic is served over HTTPS in production, and the session cookie is marked `Secure` there.
- Both OAuth flows use PKCE and a `state` parameter.
- We never receive or store a password.
- We request the narrowest OAuth scopes the Service can work with, and specifically do not request your email address from Discord.

What we are not going to claim: that a breach is impossible. The previous version of this policy said our architecture "eliminated the possibility of a data breach". It does not. We hold a database with personal data in it, including YouTube refresh tokens in plain text, on a single server. We keep the amount of data small and the attack surface narrow, and that is a meaningful reduction in risk, but it is not immunity.

## 11. Age restriction - 18 years and over

The Service is intended for people aged 18 or over. If you are under 18, please do not use it.

We do not verify anyone's age, and we have no practical way to do so. If we become aware that an account belongs to someone under 18, we will delete it and its associated data. A parent or guardian who believes their child has used the Service should contact us at the address in section 1.

## 12. Where your data is processed

The Service runs on a single virtual private server in Amsterdam, the Netherlands. The database file, any site profile photos, and any gallery photos sit on that server.

Some of the third parties in section 6 are based in the United States, including Google and Discord. When your browser loads their resources, or when our server calls their APIs, data reaches them there. Those transfers rely on the safeguards those companies provide, such as the EU-US Data Privacy Framework and standard contractual clauses. We have no separate transfer mechanism of our own to offer beyond choosing not to send them more than is necessary.

## 13. Changes to this Privacy Policy

We will update this policy when what we do changes, and change the "Last updated" date at the top. If we make a change that materially affects you, we will try to say so on the site rather than quietly editing this page.

## 14. Contact

Privacy questions and data subject requests go to the address in section 1.

---

*This policy describes Sesh Hub as it is actually built. If you find something in here that does not match how the Service behaves, please tell us - that is a bug in the policy and we want to correct it.*
