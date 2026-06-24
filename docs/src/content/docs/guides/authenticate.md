---
title: Authenticate & manage tokens
description: How knit stores credentials and how to log in, refresh, and revoke across environments.
owner: rnwolfe
lastReviewed: 2026-06-23
---

`knit` resolves credentials in this order: **`KNIT_TOKEN` env → OS keyring → `0600` file**.

## Log in

| Environment | Command |
|---|---|
| Agent / CI (headless) | `printf '%s' "$THREADS_TOKEN" \| knit auth login --token-stdin` |
| Ephemeral (no storage) | `export KNIT_TOKEN=<token>` |
| Human (browser) | `knit auth login` then paste the redirected URL |

Secrets are **never** accepted as flags — only stdin or env. See
[Get a Threads token](/tutorials/get-a-token/) to obtain one.

## Check status

```bash
knit auth status --json
```

Tests the token live and reports the account, `expiresInDays`, the credential `source`, the
redacted token, and remaining `publishQuotaRemaining`. Exits non-zero if auth is broken.

## Refresh

Long-lived tokens last 60 days. Extend without re-login (safe on a schedule):

```bash
knit auth refresh
```

The token must be ≥24h old and unexpired; otherwise re-run `knit auth login`.

## Where secrets live

- **OS keyring** first (macOS Keychain / Linux Secret Service / Windows Credential Manager).
- **`0600` file** fallback at `$XDG_DATA_HOME/knit/credentials.json`; `knit` warns on stderr if
  its permissions are looser than `0600`.

## Log out vs revoke

```bash
knit auth logout   # removes LOCAL credentials only
```

This does **not** revoke the token upstream — the Threads API has no revocation endpoint. To
fully revoke, remove the app under **Threads → Settings → Account → Website permissions** and
rotate the app secret in the Meta dashboard. See the
[security policy](https://github.com/rnwolfe/knit/blob/main/SECURITY.md).
