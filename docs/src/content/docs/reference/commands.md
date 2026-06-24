---
title: Command reference
description: Every knit command, its read/mutation class, and key output fields.
owner: rnwolfe
lastReviewed: 2026-06-23
---

Noun-verb, service-namespaced. **Reads** need no gate; **mutations** require `--allow-mutations`.
The canonical, always-current contract is `knit schema` (machine-readable) and `knit agent`
(the embedded usage doc).

## Global flags

| Flag | Purpose |
|---|---|
| `--format json\|plain\|tsv`, `--json` | Output format. Reads use the `{schemaVersion, data, nextCursor?}` envelope. |
| `--select a,b.c` | Dot-path field projection on `data`. |
| `--limit N` | Bound list size (default 50). |
| `--concise` | Drop null/empty fields (fewer tokens). |
| `--allow-mutations` | Permit state-changing operations (off by default). |
| `--dry-run` | Print the plan; change nothing. |
| `--no-input` | Never prompt; fail with exit 13 instead. |
| `--wrap-untrusted` / `--no-wrap-untrusted` | Force prompt-injection fencing on/off (default-on for agents). |

## Reads

| Command | Description | Key fields |
|---|---|---|
| `profile get [me\|<id>]` | Profile of self or a user | `id, username, name, biography, profilePictureUrl` |
| `post list [--since --until --cursor]` | Your posts | post fields + `nextCursor` |
| `post get <id>` | One post | post fields |
| `reply list <id>` | Top-level replies | reply fields |
| `reply tree <id>` | Full conversation | reply fields |
| `search posts <kw> [--mode keyword\|tag] [--media-type …]` | Keyword/topic search | post fields + `scope` |
| `mentions list` | Public posts mentioning you | post fields |
| `insights post <id>` | Per-post metrics | `views, likes, replies, reposts, quotes, shares` |
| `insights account [--metrics …]` | Account metrics | + `followersCount, followerDemographics` |

**post fields:** `id, username, text, mediaType, mediaUrl, permalink, timestamp, isQuotePost,
replyAudience, linkAttachmentUrl, quotedPostId`.
**reply** adds `hideStatus, repliedToId, rootPostId, hasReplies`.

## Mutations (gated)

| Command | Description |
|---|---|
| `post create [--text --image --video --link --reply-to --quote --topic --apply]` | Publish (reviewed-artifact). |
| `post repost <id>` | Repost a post. |
| `post delete <id>` | Delete a post (idempotent). |
| `reply create <id> [--text --image]` | Reply to a post. |
| `reply hide <id>` / `reply unhide <id>` | Hide/unhide a reply (idempotent). |

## Auth & diagnostics

| Command | Description |
|---|---|
| `auth login [--token-stdin]` | Browser paste-callback, or headless token on stdin. |
| `auth status` | Test token; redact; show expiry + quota. |
| `auth refresh` | Extend the 60-day token. |
| `auth logout` | Remove local credentials. |
| `doctor [--for-agent]` | Config / credentials / connectivity / token checks. |
| `schema` | Machine-readable command tree + exit codes + live safety state. |
| `agent` | Print the embedded SKILL.md (`KNIT_HELP=agent` does the same). |

See also the [exit codes](/reference/exit-codes/).
