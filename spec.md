# spec.md — knit

> The build spec for an agent-focused CLI wrapping **Instagram's Threads** (the official
> Threads API). Written by `cli-plan`; consumed by `cli-scaffold`, `cli-implement`, and
> `cli-publish`. Keep it current — it is the single source of truth.
>
> Spiritual sibling to the `bird` (X/Twitter) CLI, but built to the agent-CLI contract — and
> deliberately fixing bird's headline anti-pattern: **posting safety lives in the binary, not
> in a skill wrapper an agent can bypass.**

## Target
- **Service**: Instagram's **Threads**, via the **official Threads API** (`https://graph.threads.net/v1.0`).
  Graph-style REST over plain HTTPS. Base host is distinct from Instagram's Graph API.
- **Surface**: **Official API** (not reverse-engineered) — this is a rare lower-risk target
  for this factory. Endpoints used:
  - Profile: `GET /me`, `GET /{user-id}` → id, username, name, biography, profile picture.
  - Own posts: `GET /{user-id}/threads` (list), `GET /{media-id}` (single, field-selected).
  - Publish (two-step): `POST /{user-id}/threads` (create container) → `POST /{user-id}/threads_publish`
    (`creation_id=…`). Supports text, image, video, carousel, `reply_to_id`, `quote_post_id`,
    link attachment, topic tag, location.
  - Replies: `GET /{media-id}/replies` (top-level), `GET /{media-id}/conversation` (full tree),
    reply via the publish flow with `reply_to_id`, hide/unhide via `POST /{reply-id}/manage_reply`.
  - Insights: `GET /{media-id}/insights`, `GET /{user-id}/threads_insights` → views, likes,
    replies, reposts, quotes, shares; account followers + demographics.
  - Keyword/topic search: `GET /keyword_search?q=…` (`search_mode=TAG` for topics).
  - Mentions: `GET /me/mentions` → public posts mentioning the authed user.
- **Prior art** (the niche exists; none is agent-engineered to this contract):
  - [`saadiq/threads-cli`](https://github.com/saadiq/threads-cli) (TS/Bun) — closest intent:
    publish + fetch-as-JSON, separate Threads app id/secret. **Mine for OAuth mechanics.** Gaps:
    no mutation gate, no `schema --json`, token stored **plaintext** (0600 file, not keyring),
    no injection fencing. **Confirms Meta requires an HTTPS OAuth callback (uses mkcert).**
  - [`ptrlrd/threads-cli`](https://github.com/ptrlrd/threads-cli) (Python/typer/rich) — posts,
    replies, scheduling; human-first TUI, no agent affordances.
  - [`pukpuklouis/threads-cli`](https://github.com/pukpuklouis/threads-cli) (Rust) — drafts/publish/view.
  - [`The-Amoghavarsha/Threads-CLI`](https://github.com/The-Amoghavarsha/Threads-CLI) — profile-info reader.
  - [`fbsamples/threads_api`](https://github.com/fbsamples/threads_api) — Meta's **official sample
    app**; the reference implementation for the OAuth dance (not a tool).
  - **Differentiation**: same gap as the rest of the crop — `knit` is the only one read-only by
    default, mutation-gated in the binary, keyring-stored, injection-fenced, `schema --json`-able.
- **Rate limits / pagination**:
  - **Publishing: 250 posts / 24h** rolling per profile; replies counted in a similar daily
    cap. Per-minute / per-hour bursts → **HTTP 429**.
  - **Keyword search: 500 queries / rolling 7 days.**
  - Pagination is Graph-style cursor (`paging.cursors.after` / `paging.next`). Surface as the
    contract's opaque `nextCursor`.
- **ToS / risk — state loudly**:
  - ✅ Official, ToS-compliant API → **low breakage risk** (unlike most of this factory's
    crop, which rides unofficial APIs). This is a selling point.
  - ⚠️ **Public read is gated by App Review.** `keyword_search` and `mentions` only return
    *public* posts after the app obtains **advanced access** to `threads_keyword_search` /
    mentions (App Review + possible business/access verification). **Without approval, search
    is silently scoped to the authed user's own posts.** `knit` must detect this and say so —
    never let an agent believe an empty/own-only result is the whole public corpus.
  - ⚠️ **Token is high-sensitivity**: a long-lived token grants full read **and publish**
    access to the user's Threads account. Treat like a password (keyring, never argv).
  - ⚠️ **60-day token expiry** with mandatory refresh discipline (see Auth).

## Language & framework
- **Language**: **Go** (the factory default).
- **Rationale (SDK gravity > distribution > performance)**: No official SDK exists and the
  community libs (JS, Ruby, Python) are all thin wrappers over a plain HTTPS graph API, so **no
  SDK forces a language** → the default wins. Go gives the best single-binary + cold-start
  story for an agent hot-loop, and kong's typed grammar makes `schema --json` a reflection
  walk. (Note: bird is TS, but bird's TS pull was cookie/web-auth scraping — `knit` uses the
  official token API, so that pull doesn't apply here.)
- **Framework**: **kong** (`alecthomas/kong`) — command tree as tagged structs; single global
  mutation gate is one root field; schema is a reflection walk.
- **SDK/library used**: **direct HTTP** (`net/http` + small typed client in `internal/api`).
  No third-party Threads SDK — avoids an unmaintained-wrapper dependency on a moving API.
- **Blueprint**: `references/research/blueprint-go.md`
- **Language-specific gotchas to honor**: `var version="dev"` literal (go#64246) + ReadBuildInfo
  fallback; `CGO_ENABLED=0` static build; `SetEscapeHTML(false)` (permalinks/media URLs must
  survive); never colorize/progress to stdout; GoReleaser v2 `homebrew_casks` + Gatekeeper
  `xattr` cask hook.

## Auth
- **Model**: **OAuth 2.0 authorization-code** (confidential client — `client_id` +
  `client_secret`). Browser redirect to `https://threads.net/oauth/authorize`; code exchanged
  at `https://graph.threads.net/oauth/access_token`. **No PKCE, no device flow.** Short-lived
  token (1h) → exchange for **long-lived token (60d)** → **refresh** unexpired long-lived
  tokens. `redirect_uri` must exactly match an app-configured URI.
- **Auth-pattern lineage** (bottomed from on-system cluster tools): **mmoney's storage**
  (`auth login/logout/status`, keyring + 0600 file fallback) **+ saadiq/threads-cli's token
  lifecycle** (short 1h → auto-exchange long 60d → auto-refresh on expiry) **− bird's mistake**
  (bird passes secrets as **argv flags** `--auth-token`/`--ct0` → leaks to `ps`/`/proc`/history;
  `knit` is stdin/env/keyring only). **`--token-stdin` is the primary path because Meta requires
  an HTTPS OAuth callback** — confirmed by saadiq's mkcert dependency — so plain `http://localhost`
  loopback is *not* accepted.
- **Agent-completable path** (contract §7 — no browser-only step as the *sole* path). Two
  paths, both ending in a keyring-stored, non-interactively-refreshable long-lived token:
  1. **`knit auth login --token-stdin`** *(primary — agent & default)* — accept a pre-obtained
     long-lived (or short-lived) token on **stdin**, validate it, exchange/extend to
     long-lived, store in keyring. Fully non-interactive; an agent never needs a browser. The
     user provisions the token once from the Meta dashboard / a one-time exchange. **No HTTPS
     callback needed → this sidesteps the mkcert problem entirely.**
  2. **`knit auth login`** *(human onboarding — paste-the-callback-URL / manual OOB flow)* —
     **no server, no cert.** A `redirect_uri` only needs to be *registered and matched*, not
     *served*: the auth server appends `?code=…&state=…` and the browser navigates there even if
     nothing listens. So register an HTTPS placeholder (e.g. `https://localhost/knit/callback`)
     in the Threads app; `knit` prints the authorize URL (with a random `state`), the user
     approves in-browser, the redirect shows connection-refused **but the code is in the address
     bar** — the user pastes the full URL (or bare code) on **stdin**. `knit` parses `code`,
     **validates `state` (CSRF)**, exchanges with the identical `redirect_uri` → long-lived →
     keyring. This sidesteps the mkcert/HTTPS-loopback problem entirely (Meta rejects
     `http://localhost`; saadiq confirms HTTPS-only callbacks, hence its mkcert dependency).
     - **Power-user variant**: point the registered redirect at the home-lab `expose`/Caddy
       endpoint (`*.labs.rwolfe.io`) for true auto-capture instead of copy-paste.
     - **Footguns** (`cli-implement`): `code` is single-use + ~1h TTL; verify `state`; pass the
       *same* `redirect_uri` at exchange; accept full-URL *or* bare code on stdin and trim.
  - `client_id`/`client_secret` (the **Threads** app id/secret — distinct from the top-level
    Meta app's) come from `KNIT_CLIENT_ID` / `KNIT_CLIENT_SECRET` env (or a `0600` config file),
    **never argv**.
- **Secret storage**: OS keyring (`99designs/keyring`) + `0600` XDG fallback; **warn on
  insecure file perms** (contract §7).
- **Subcommands**: `auth login` (incl. `--token-stdin`) · `auth status` (tests token, redacts
  by default, reports scopes + token-expiry days remaining, **and whether advanced search
  access is granted**, non-zero on problems) · `auth logout` · `auth refresh` (extends the
  60-day token; safe to run on a cron/schedule). Plus `doctor` (`--for-agent`): env vars,
  keyring reachability, token validity/expiry, connectivity, granted scopes.

## Command surface (noun-verb, service-namespaced)
| Command | Read/Mutation | Description | Key output fields |
|---|---|---|---|
| `profile get [me\|<user-id>]` | read | Profile of self or a user id | `id, username, name, biography, profilePictureUrl` |
| `post list [--limit --since --until --cursor]` | read | List the authed user's posts | post fields (below) + `nextCursor` |
| `post get <media-id>` | read | Single post, field-selected | post fields |
| `post create` | **mutation** | Publish a post: `--text`, `--image/--video URL…`, `--carousel`, `--link`, `--reply-to <id>`, `--quote <id>`, `--topic`, `--reply-audience` | `id, permalink` (+ dry-run plan+hash) |
| `post repost <media-id>` | **mutation** | Repost a post *(verify endpoint exists in implement; flag if absent)* | `id` |
| `reply list <media-id> [--limit --cursor]` | read | Top-level replies to a post | reply fields + `nextCursor` |
| `reply tree <media-id> [--limit --cursor]` | read | Full conversation tree | reply fields (nested) |
| `reply create <media-id>` | **mutation** | Reply to a post (`--text`/media) | `id, permalink` |
| `reply hide <reply-id>` | **mutation** | Hide a reply on your post | `id, hideStatus` |
| `reply unhide <reply-id>` | **mutation** | Unhide a reply | `id, hideStatus` |
| `search posts <keyword> [--media-type text\|image\|video] [--mode keyword\|tag] [--limit --cursor]` | read | Keyword/topic search (⚠️ own-posts-only without advanced access — flagged in output) | post fields + `nextCursor` + `scope: public\|self` |
| `mentions list [--limit --cursor]` | read | Public posts mentioning you (⚠️ advanced-access gated) | post fields + `nextCursor` |
| `insights post <media-id>` | read | Per-post metrics | `views, likes, replies, reposts, quotes, shares` |
| `insights account [--metrics …] [--since --until]` | read | Account metrics + demographics | `views, likes, …, followersCount, followerDemographics` |
| `auth login \| status \| logout \| refresh` | mixed | Auth lifecycle (login/refresh are local, not target mutations) | — |
| `doctor [--for-agent]` | read | Environment/auth/connectivity diagnostics | health fields |
| `schema --json` · `agent` | read | Self-description (provided by scaffold) | — |

**Read/mutation split**: 9 read commands, 6 mutation commands. **All 6 mutations are gated by
`--allow-mutations` (default-deny)**, enforced in the binary via `GuardMutation`.

**High-stakes publish → reviewed-artifact = approval** (contract §2, beats bird): `post create`
and `reply create` under `--dry-run` emit the exact media container they *would* publish plus a
content **hash**; `knit post apply <hash>` (or `--apply <hash>`) publishes only that exact plan.
Closes the TOCTOU gap a blind `--yes` opens for an irreversible, **public** action. `auth status`
/ `post create` should also surface **remaining daily publish quota** (of the 250/24h cap) so an
agent backs off before a 429.

## Exit codes
Start from contract §4; target-specific tuning:
```
0   ok                     6  permission denied  — scope/advanced-access not granted
1   generic error             (remediation → which scope + App Review)
2   usage / parse error    7  rate limited       — 429 (publish 250/24h, search 500/7d,
3   empty results             or per-min/hour burst; remediation names which cap + retry-after)
4   auth required          8  retryable / transient (5xx, network)
   (token missing/expired  10 config error (missing KNIT_CLIENT_ID/SECRET)
    → names `knit auth …`) 12 mutation blocked  (--allow-mutations not set; MUTATION_BLOCKED)
5   not found              13 input required (--no-input hit a prompt)
                           130 cancelled (SIGINT)
```
Notable mappings: **token-expired** → `4 auth required` (remediation: `knit auth refresh` or
re-login). **Scope/advanced-access missing** → `6 permission denied` (remediation: requested
scope + App Review pointer). **Daily publish cap / search cap / burst** → `7 rate limited` with
the specific cap and any `retry-after` echoed.

## Output schema
Envelope (every read): `{ "schemaVersion": 1, "data": <object|array>, "nextCursor": <string?> }`
— `nextCursor` omitted at end-of-results (contract §6). Field names are **append-only**.

**post**
```jsonc
{
  "id": "string",
  "username": "string",
  "text": "string|null",
  "mediaType": "TEXT_POST|IMAGE|VIDEO|CAROUSEL_ALBUM|AUDIO|REPOST_FACADE",
  "mediaUrl": "string|null",
  "permalink": "string|null",
  "timestamp": "ISO-8601",
  "isQuotePost": false,
  "replyAudience": "string|null",
  "linkAttachmentUrl": "string|null",
  "quotedPostId": "string|null"
}
```
**reply** = post fields + `{ "hideStatus": "NOT_HUSHED|HIDDEN|…", "repliedToId": "string|null", "rootPostId": "string|null", "hasReplies": false }`
**profile** = `{ "id", "username", "name", "biography", "profilePictureUrl" }`
**insights (post)** = `{ "views", "likes", "replies", "reposts", "quotes", "shares" }` (ints; null if unavailable)
**insights (account)** = post-insight ints + `{ "followersCount", "followerDemographics": {…} }`
**search/mentions** add `{ "scope": "public|self" }` on the envelope so an agent knows whether
advanced access was in effect.

## Universal contract surface (provided by scaffold — confirmed no conflicts)
`--format json|plain|tsv` · `--json` · `--allow-mutations`/`--write` · `--dry-run` ·
`--yes`/`--force` · `--no-input` · `--limit` (default 50) · `--select a,b.c` ·
`--concise`/`--detailed` · `schema --json` · `agent` · `NO_COLOR`/TTY rules.
No naming collisions with the command surface above.

## Distribution
- **Targets**: GoReleaser v2 → `homebrew_casks` tap + `go install
  github.com/rnwolfe/knit/cmd/knit@latest` + signed/attested release binaries (linux/darwin/
  windows × amd64/arm64) + `curl|sh`.
- **Trial path** (human): `brew install rnwolfe/tap/knit` or a release binary.
- **Agent hot-loop path** (lowest cold start): the single static binary (`go install` or release
  download) — embedded `SKILL.md` means the agent always has the contract, no repo/network.

## Publish
- **Flag**: **full** (portfolio-bound).
- **If full**: docs site via `starlight-docs` · doc content via `harvest-docs` · release/changelog
  via the `release` skill · README + VHS demo · hygiene files (LICENSE/CONTRIBUTING/CI) ·
  discoverability. Lead the narrative on the differentiator: **a Threads CLI that's actually
  agent-safe — read-only by default, posting gated in the binary (the `bird` fix), prompt-
  injection-fenced, official-API (low breakage)**.

## Prompt-injection surface (contract §8 — default-ON fencing in agent mode)
Threads is a public social feed, so **every command that returns free text authored by other
people is attacker-controllable** and must be wrapped/fenced as untrusted by default in agent
mode: `post get`/`post list` (own, but may embed quoted/reposted third-party text),
`reply list`, `reply tree`, **`search posts`**, **`mentions list`** (the highest-risk surfaces —
arbitrary public users can craft post/reply text specifically to hijack an agent that reads
search results or mentions). Profile `biography` is also user-controlled → fence it. Insights are
numeric → no fencing needed.

---
### Open items — ALL RESOLVED during cli-implement (2026-06-23)
1. ~~`http://localhost` redirect?~~ **No** — Meta requires HTTPS callbacks. `--token-stdin`
   primary; browser `auth login` uses paste-the-callback-URL (no server/cert).
2. ~~Repost endpoint?~~ **YES** — `POST /{threads-id}/repost`. `post repost` kept.
3. ~~Hide-reply mechanism?~~ **YES** — `POST /{reply-id}/manage_reply?hide=true|false`
   (scope `threads_manage_replies`). Reply read fields include `hide_status`
   (NOT_HUSHED|UNHUSHED|HIDDEN|COVERED|BLOCKED|RESTRICTED), `has_replies`, `root_post`,
   `replied_to`, `reply_audience`. Conversation pagination = Graph `paging.cursors.after`.
4. ~~Delete endpoint?~~ **YES (added 2025)** — `DELETE /{threads-media-id}` (scope
   `threads_delete`), returns `{success, deleted_id}`, 100/24h. **`post delete` added**
   (gated, idempotent — delete-already-gone = soft success per contract §9).

### Verified API facts (for the client)
- **Token**: sent as `?access_token=` query param. Long-lived exchange `GET
  /access_token?grant_type=th_exchange_token&client_secret=&access_token=` → 60d
  (`expires_in≈5184000`). Refresh `GET /refresh_access_token?grant_type=th_refresh_token&access_token=`
  (token must be ≥24h old, <60d). Short-lived: `POST /oauth/access_token` (form) → `{access_token, user_id}`.
- **Authorize**: `https://threads.net/oauth/authorize?client_id=&redirect_uri=&scope=<comma-sep>&response_type=code&state=`;
  callback returns `?code=…#_` (strip trailing `#_`).
- **Publish**: 2-step `POST /me/threads` (media_type TEXT|IMAGE|VIDEO|CAROUSEL, text, image_url,
  video_url, children, reply_to_id, quote_post_id, link_attachment, topic_tag, reply_control,
  location_id) → creation id → `POST /me/threads_publish?creation_id=`. VIDEO: poll
  `GET /{container}?fields=status` until FINISHED. Quota check: `GET /me/threads_publishing_limit?fields=quota_usage,config`.
- **Reads**: media fields `id,media_product_type,media_type,media_url,permalink,owner,username,
  text,timestamp,shortcode,thumbnail_url,children,is_quote_post,quoted_post,reposted_post,
  link_attachment_url,topic_tag`. Profile fields `id,username,name,threads_profile_picture_url,
  threads_biography,is_verified`. Search `GET /keyword_search?q=&search_type=TOP|RECENT&search_mode=TAG&media_type=`.
  Mentions `GET /me/mentions`. Insights `GET /{media}/insights?metric=views,likes,replies,reposts,quotes,shares`,
  `GET /me/threads_insights?metric=…,followers_count,follower_demographics` (data[].name + values[].value|total_value.value).
- **Errors**: `{error:{message,type,code,error_subcode,fbtrace_id}}`. Rate limit = HTTP 429, codes
  4 (app) / 17 (user) / 32 (account) / 613 (custom); headers `X-App-Usage`, `X-Business-Use-Case-Usage`.
  Invalid/expired token = code 190.
