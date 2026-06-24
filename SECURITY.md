# Security Policy

`knit` handles a **Threads access token that grants full read _and publish_ access** to your
account. We take its handling seriously and document the threat model explicitly below.

## Supported versions

| Version | Supported |
|---------|-----------|
| latest `0.x` | ✅ |
| older `0.x` | ❌ (upgrade to latest) |

`knit` is pre-1.0; security fixes land on the latest minor only.

## Reporting a vulnerability

**Do not open a public issue for security reports.**

- Use **GitHub Private Vulnerability Reporting**: the repo's **Security → Report a vulnerability** tab.
- Target response: **within 48 hours**; coordinated-disclosure window up to 90 days.
- Good-faith research is welcome (safe harbor): no rate-limit/DoS testing against Meta, no
  accessing accounts that aren't yours, and use throwaway test apps/accounts.
- A useful report includes a reachable PoC, affected version (`knit version`), and OS.

## Secret-handling threat model

What `knit` protects, where it can leak, and the mitigations in place.

### What the secret is
A Threads OAuth **access token** (short-lived 1h or long-lived 60d) and, during `auth login`,
the **client secret** of your Threads app. Both are account-sensitive.

### Storage
- **Primary**: OS keyring — macOS Keychain, Linux Secret Service, Windows Credential Manager
  (via `99designs/keyring`).
- **Fallback**: a `0600` file at `$XDG_DATA_HOME/knit/credentials.json`. `knit` **warns on
  stderr if the file's permissions are looser than `0600`**.
- The token is **never written to logs** or stdout; `auth status` **redacts** it by default
  (`abcd…wxyz`).

### Leakage vectors and mitigations
| Vector | Mitigation |
|--------|------------|
| **argv** (`ps`, `/proc`, shell history) | Secrets are **never** accepted as flags. Tokens come from **stdin** (`--token-stdin`) or env; client id/secret from env only. |
| **logs / stdout** | stdout is data-only; the token is redacted in `auth status`; errors never echo the token. |
| **env in CI logs** | `KNIT_TOKEN`/`KNIT_CLIENT_SECRET` are read from the environment but never printed. Use masked CI secrets. |
| **prompt injection** (a public post telling your agent to exfiltrate the token) | Free text from Threads (posts/replies/search/mentions/bios) is **fenced as untrusted by default** in agent mode (`--wrap-untrusted`). |
| **disk at rest** | keyring first; `0600` file fallback with a perms warning. |

### Revocation (separate from logout)
`knit auth logout` removes **local** credentials only — it does **not** revoke the token
upstream (the Threads API has no token-revocation endpoint). To fully revoke, remove the app's
access from **Threads → Settings → Account → Website permissions / Apps**, and rotate the app
secret in the Meta App Dashboard.

### If a token leaks
1. Revoke the app's access in Threads settings (above).
2. Rotate the **Threads app secret** in the Meta App Dashboard.
3. `knit auth logout` locally, then `knit auth login` to mint a fresh token.

## Scopes requested (least privilege)
`knit auth login` requests only: `threads_basic`, `threads_content_publish`,
`threads_read_replies`, `threads_manage_replies`, `threads_manage_insights`. It does **not**
request `threads_keyword_search`/`threads_manage_mentions`/`threads_delete` unless you need
them — narrow the scope set in `internal/auth/oauth.go` (`DefaultScopes`) for read-only use.
