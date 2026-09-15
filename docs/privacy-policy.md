# Privacy Policy - Sesh Hub

**Website:** https://hub.seshsofa.nl

**Last updated:** 15 September 2026

## 1. Who we are

Sesh Hub ("we", "us" or "our") operates the website and web application available at https://hub.seshsofa.nl (the "Service").

We are based in the Netherlands. For the purposes of the General Data Protection Regulation (GDPR) and the Dutch Implementation Act (Uitvoeringswet AVG), we are the controller of the personal data described in this policy.

**Contact.** Privacy questions and data subject requests: `privacy@seshsofa.nl`

**[TODO before publishing: confirm this address works, or replace it. GDPR Art. 13 requires a working contact route, and pointing people at "community channels" is not enough.]**

## 2. What this policy is

Sesh Hub is a small community site for the Sesh Sofa crew. You sign in with Discord or YouTube, and we keep an account record for you in a database on our server.

We want to be exact about that, because an earlier version of this policy claimed we stored nothing at all. That was wrong. We do store personal data: your Discord ID, your display name, your avatar URL, your YouTube channel if you link one, and a login session that records the IP address you signed in from.

This policy lists every category of personal data we hold, why we hold it, when it is created, who can read it, when it changes, and when it is deleted. Section 9 lists the places where our current implementation falls short of what we would like to promise. We would rather tell you than pretend.

## 3. How you sign in

Authentication is entirely via OAuth. We never see or store a password.

**Discord.** We request the scopes `identify` and `guilds.members.read`. `identify` returns your Discord user ID, username, global display name and avatar hash. `guilds.members.read` lets us read your role list in the Sesh Sofa Discord server, which is how we decide whether you are a member, a skater, a host or an admin. We do **not** request the `email` scope, so Discord does not give us your email address.

**YouTube (Google).** We request the scope `youtube.readonly`, with offline access so we receive a refresh token. This returns your channel ID, channel title and channel thumbnail, and lets us list the videos on your own channel.

Both flows use PKCE (S256) and a `state` parameter to prevent request forgery.

Your Discord **role IDs are not stored**. We read them at the moment you sign in, convert them into a single role label (`admin`, `skater`, `member` or `pending`) plus a `host` yes/no flag, and store only that result.

## 4. What we store, and its full lifecycle

Our database is a SQLite file on the server that runs the Service.

### 4.1 Your account (`users` table)

**What:** an internal account ID, your username, display name, avatar URL, role label, host flag, Discord ID, Discord username, YouTube channel ID, YouTube channel title, your **YouTube OAuth refresh token**, the time of your last YouTube sync, and creation and last-updated timestamps.

**Why:** to recognise you across visits, to show your name and avatar in the interface, to decide what you are allowed to see and edit, and to fetch your clips if you have linked YouTube.

**Created:** the first time you sign in with Discord or YouTube.

**Read:** on every request you make while signed in, to resolve your session to an account. Your display name is also shown publicly as the author of any article you write.

**Updated:** on **every** subsequent sign-in. We re-copy your current username, display name, avatar and roles from the provider, so changing your name or avatar on Discord changes it here the next time you log in. Linking or unlinking a provider also updates this record.

**Deleted:** by an administrator, on request or at their discretion. There is no button for you to do this yourself. See sections 9 and 10.

The refresh token deserves a specific mention: it is a long-lived credential that lets us request read-only access to your YouTube channel without you signing in again. It is stored in the database in plain text. It is cleared when a YouTube account is unlinked or the account is deleted. You can also revoke it yourself at any time from your Google account's security settings, which invalidates it immediately regardless of what we hold.

### 4.2 Your login session (`sessions` table)

**What:** a SHA-256 hash of your session token, your account ID, **the IP address you signed in from**, **your browser's User-Agent string**, an expiry time and a creation time.

The session token itself is a 256-bit random value. The plain value exists only in the cookie in your browser; we store only its hash, so the contents of our database cannot be used to impersonate you.

**Why:** to keep you signed in between requests.

**Created:** each time you sign in. Valid for 30 days.

**Read:** on every request, to look up your account.

**Updated:** never. Sessions are created and deleted, not modified.

**Deleted:** when you click log out, which deletes **only the session you are currently using**. If you are signed in on another browser or device, that session stays valid until it expires. All of your sessions are deleted if your account is deleted.

The IP address and User-Agent are the most privacy-sensitive things we retain, and we should be straight with you about them: nothing in the Service ever reads them. They are written when you log in and never used. See section 9.

### 4.3 Your skater profile (`skater_profiles` table)

**What:** a public slug, your skater name, your real name, a biography, your stance, a roster status, avatar and banner URLs, your location, sponsors, social links, signature tricks, and a featured video ID.

**Why:** to show the team roster at `/team` and your own profile page at `/team/{slug}`.

**Created:** automatically, the first time you sign in holding the skater or admin role in the Sesh Sofa Discord. Your skater name is initially taken from your Discord username, and is kept in step with it on subsequent visits. The check that creates it runs on every request you make, so as long as your account exists and still holds the role, a profile will exist.

**Read: this profile is public.** Anyone on the internet, signed in or not, can see it. Note in particular that the name shown publicly is your **real name if you have filled it in**, and falls back to your skater name only if you have not. Your **location** and **biography** are also shown publicly when set. Please do not put anything in these fields that you would not want a stranger to read.

**Updated:** by you, at `/dashboard/profile`. That form currently edits your real name, biography, stance, location and featured video. An administrator can separately change your roster status.

**Deleted:** by an administrator. See section 9 - this is not currently removed automatically when an account is deleted.

### 4.4 Your access request (`access_requests` table)

**What:** your account ID, a status of pending, approved or rejected, the account ID of the administrator who reviewed it, and timestamps.

**Why:** if you sign in but are not a member of the Sesh Sofa Discord server, you land in an approval queue rather than getting access.

**Created:** when you ask for access. **Read:** by administrators reviewing the queue, and by you to see your own status. **Updated:** when an administrator approves or rejects it; approval also changes your role to member. **Deleted:** together with your account.

### 4.5 Articles you write (`articles` table)

Articles record the account ID of their author, and your display name is published alongside the article. Article text is written by you and may contain whatever you put in it.

Because articles are attributed, **authorship currently blocks account deletion outright** - see section 9.

### 4.6 Cached clips from your YouTube channel (`youtube_videos` table)

**What:** for up to 50 recent uploads on a linked channel, the video ID, your channel ID, title, description, publication date, thumbnail URL, duration, view count, like count and tags.

**Why:** so the `/videos` gallery and skater profiles load from our database instead of calling the YouTube API on every page view.

**Created and updated:** in the background when a signed-in skater visits the Service and the cache for their channel is more than an hour old.

**Read:** publicly, on `/videos` and on skater profile pages.

**Deleted:** never, by anything. See section 9.

### 4.7 Episode archive (`episodes` table)

The archive of past Spot Challenge episodes at `/episodes` includes a winner name for each episode. These are community handles and channel names, some of them seeded from the show's history, not account records. They are public.

### 4.8 Server logs

The application writes error-level logs to standard output. These can include an account ID when something fails for a specific user, for example a failed YouTube sync. The application does **not** keep an HTTP access log, and does not write request paths, IP addresses or User-Agent strings to its logs.

The Service sits behind a reverse proxy, which may keep its own request logs at the infrastructure level, including IP addresses, according to its default configuration.

There is also an unused `sync_logs` table in our database schema. Nothing ever writes to it; we mention it only for completeness.

### 4.9 What we do not do at all

We do not send email. There is no newsletter and no notification system. There is no file upload feature, so you cannot upload photos or documents to us. We do not sell, rent or trade personal data. We do not profile you, build advertising audiences, or use your data for anything beyond operating the site.

## 5. Cookies

| Cookie | Purpose | Lifetime | Notes |
|--------|---------|----------|-------|
| `seshhub_session` | Keeps you signed in | 30 days | Strictly necessary. `HttpOnly`, `SameSite=Lax`, and `Secure` in production. Contains a random token, no personal data. |
| `seshhub_oauth` | Holds the OAuth `state` and PKCE verifier during sign-in | 10 minutes | Strictly necessary. Deleted as soon as sign-in completes. |
| Google Analytics cookies | Audience statistics | Set by Google, typically up to 2 years | **Not** strictly necessary. See sections 6 and 9. |

Disabling the two strictly necessary cookies will stop you from being able to sign in. You can control all cookies through your browser settings.

## 6. Third parties, and who receives your IP address

Any time your browser loads something from another company's server, that company receives your IP address, because that is how the internet works. The following are loaded by pages on this Service:

- **Google Analytics 4**, from `googletagmanager.com`, on every page. This is analytics: it sets cookies and reports your visit to Google. Please read section 9 about this one.
- **Web fonts** from `fonts.cdnfonts.com`, on every page.
- **YouTube video thumbnails** from `img.youtube.com` and `i.ytimg.com`, on the videos gallery, episode archive and profile pages.
- **YouTube video players** from `youtube-nocookie.com`, on skater profile pages that have clips. We deliberately use YouTube's privacy-enhanced domain, which does not set tracking cookies until you press play.
- **Discord avatar images** from `cdn.discordapp.com`, wherever an avatar is displayed. These are loaded with `referrerpolicy="no-referrer"`, so Discord is not told which page you were on.

Our server also talks to these services directly. In those cases your IP address is not sent; ours is.

- **Discord's API**, during sign-in, to read your profile and your roles in the Sesh Sofa server.
- **Google's YouTube Data API**, during sign-in and during clip sync, to read your channel and its public videos.
- **Subotto** (`subotto.seshsofa.nl`), to fetch current episode information for the homepage. This is an anonymous request containing no information about you.

Each of these companies handles your data under its own privacy policy. You can revoke our access to your Discord or YouTube account at any time from that platform's own settings.

## 7. Purposes and legal basis

| What we do | Why | Legal basis |
|------------|-----|-------------|
| Create and maintain your account record | You cannot use a members' site without an account | Art. 6(1)(b) - performance of a contract, being the service you signed up for |
| Keep you signed in via a session cookie | Otherwise you would log in on every page | Art. 6(1)(b), and Art. 6(1)(f) legitimate interest in account security |
| Read your Discord roles to decide your permissions | To keep members' areas restricted to members | Art. 6(1)(f) - legitimate interest in access control |
| Publish your skater profile and article authorship | This is the purpose of the roster and the news section, and you control the content | Art. 6(1)(f), with your role in publishing it |
| Cache your YouTube clips | To show the gallery without hammering the YouTube API | Art. 6(1)(f) - legitimate interest in a functioning site |
| Google Analytics | Audience statistics | Requires your consent under Art. 6(1)(a) and the ePrivacy rules. **We do not currently ask for it.** See section 9. |

## 8. Retention

- **Account record:** kept until the account is deleted by an administrator. There is no automatic expiry, so if you stop using the Service your account stays until someone removes it.
- **Session record:** the session stops working 30 days after you sign in. The database row, including the IP address and User-Agent, is **not automatically deleted** at that point. See section 9.
- **Skater profile:** kept until deleted by an administrator.
- **Cached YouTube clips:** kept indefinitely.
- **Access requests:** kept until the account is deleted.
- **Articles and episode entries:** kept as part of the site's published archive.

## 9. Known limitations of our current implementation

This section exists because we would rather disclose these than write a policy that promises more than our code delivers. Each of these is on our list to fix.

1. **Google Analytics runs without asking you.** Analytics cookies require consent, and we currently load Google Analytics on every page without showing a consent banner or offering a way to decline. That is a shortcoming on our side, not a legal position we are defending. If you want to prevent it today, a browser tracker blocker or Google's own opt-out browser add-on will do so. A consent gate is the highest priority item on our privacy backlog.

2. **Expired session rows are not purged.** An expired session cannot be used to sign in - we check the expiry on every lookup - but the row itself, with its IP address and User-Agent, stays in the database after it stops working.

3. **We store an IP address and User-Agent that we never use.** No part of the Service reads either field. They are written at login and then ignored. We intend to stop collecting them rather than find a use for them.

4. **Deleting an account does not remove the public skater profile.** Account deletion removes your account record, your sessions and your access request, but not your skater profile. Since the roster is filtered by whether a profile has an account ID rather than whether that account still exists, a deleted person's profile - including real name, biography and location - would remain visible on `/team` unless the profile is also deleted separately. Until this is fixed, we will delete both when you ask us to; please do ask for both explicitly so nothing is missed.

   There is a related ordering trap we have to get right, and you are entitled to know about it: because the profile-creation check runs on every request, deleting only the profile while you are still signed in and still hold the Discord role causes a fresh, empty profile to be created again on your next page load. Your real name, biography and location would not come back, but a public roster entry would. A proper erasure therefore means deleting the account first, which ends your sessions, and the profile second.

5. **Cached YouTube clips are not deleted with an account.** Nothing in the Service deletes rows from the clip cache, so cached titles, descriptions and your channel ID would survive account deletion unless removed by hand.

6. **If you have written an article, deletion is currently blocked.** Our deletion routine refuses to run for any account that has authored an article, to avoid orphaning published content. In practice that means an erasure request from an author needs their articles reassigned or removed first, which we will handle manually.

7. **There is no self-service anything.** You cannot delete your own account, unlink Discord or YouTube, or export your data from within the Service. All of these require asking an administrator. Logging out is the only self-service data action available.

8. **Logging out only ends one session.** Other devices stay signed in until their 30 days elapse.

## 10. Your rights under the GDPR

You have the right of access (Art. 15), rectification (Art. 16), erasure (Art. 17), restriction (Art. 18), data portability (Art. 20), and objection (Art. 21).

Unlike the previous version of this policy, we are not going to tell you there is nothing to access or delete. There is a real account record with your name, your Discord ID and your session history in it.

**How to exercise these rights.** Email the contact address in section 1. Because we have no self-service tooling, an administrator will action your request by hand. We will respond within one month, as Art. 12(3) requires.

- **Access or portability:** we will export the contents of your account record, your skater profile and your access request.
- **Erasure:** we will delete your account record, your sessions, your access request and your skater profile. Please read section 9 items 4 to 6, which describe what needs to be done manually and why articles complicate it. We will tell you specifically what was removed.
- **Rectification:** note that most of your profile fields update themselves from Discord or YouTube on your next sign-in, so correcting your name there is usually faster than asking us. Your biography and location are yours to edit at `/dashboard/profile`.
- **Objection or restriction:** tell us what you object to and we will stop it or explain why we believe we may continue.

Some of the underlying personal data also lives with Discord and Google, and we cannot reach into their systems. Revoking our OAuth access from their settings pages is immediate and does not require us.

You have the right to lodge a complaint with the Dutch Data Protection Authority, the Autoriteit Persoonsgegevens: https://autoriteitpersoonsgegevens.nl

## 11. Security

What is actually true about how we protect this data:

- Session tokens are stored as SHA-256 hashes, so a copy of our database does not let an attacker log in as you.
- Session tokens are 256 bits of cryptographic randomness.
- The session cookie is `HttpOnly`, so scripts in the page cannot read it, and `SameSite=Lax`.
- Traffic is served over HTTPS in production, and the session cookie is marked `Secure` there.
- Both OAuth flows use PKCE and a `state` parameter.
- We never receive or store a password.
- We request the narrowest OAuth scopes the Service can work with, and specifically do not request your email address from Discord.

What we are not going to claim: that a breach is impossible. The previous version of this policy said our architecture "eliminated the possibility of a data breach". It does not. We hold a database with personal data in it, including YouTube refresh tokens in plain text, on a single server. We keep the amount of data small and the attack surface narrow, and that is a meaningful reduction in risk, but it is not immunity.

## 12. Age restriction - 18 years and over

The Service is intended for people aged 18 or over. If you are under 18, please do not use it.

We do not verify anyone's age, and we have no practical way to do so. If we become aware that an account belongs to someone under 18, we will delete it and its associated data. A parent or guardian who believes their child has used the Service should contact us at the address in section 1.

## 13. Where your data is processed

The Service runs on a single virtual private server, and the database file sits on that server.

**[TODO before publishing: name the hosting provider and the country the server is in. The transfer position below depends on it.]**

Some of the third parties in section 6 are based in the United States, including Google and Discord. When your browser loads their resources, or when our server calls their APIs, data reaches them there. Those transfers rely on the safeguards those companies provide, such as the EU-US Data Privacy Framework and standard contractual clauses. We have no separate transfer mechanism of our own to offer beyond choosing not to send them more than is necessary.

## 14. Changes to this Privacy Policy

We will update this policy when what we do changes, and change the "Last updated" date at the top. If we make a change that materially affects you, we will try to say so on the site rather than quietly editing this page.

## 15. Contact

Privacy questions and data subject requests go to the address in section 1.

---

*This policy describes Sesh Hub as it is actually built, including the parts we are not proud of. Section 9 is the list of things we know are wrong and intend to fix. If you find something in here that does not match how the Service behaves, please tell us - that is a bug in the policy and we want to correct it.*
