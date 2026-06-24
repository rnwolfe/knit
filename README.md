<div align="center">

# knit

**`bird` is for X. `knit` is for [Threads](https://www.threads.net) — and it's actually agent-safe.**

An agent-friendly CLI for Instagram's Threads, built on the official API. Read-only by default,
posting gated *in the binary*, prompt-injection-fenced, machine-readable.

[![CI](https://github.com/rnwolfe/knit/actions/workflows/ci.yml/badge.svg)](https://github.com/rnwolfe/knit/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/rnwolfe/knit?sort=semver)](https://github.com/rnwolfe/knit/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/rnwolfe/knit.svg)](https://pkg.go.dev/github.com/rnwolfe/knit)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

<a href="https://knitcli.sh"><b>knitcli.sh</b></a> · <a href="https://rnwolfe.github.io/knit"><b>Docs</b></a>

![knit demo](demo/knit.gif)

</div>

## Why knit

Plenty of Threads CLIs exist. None is built for an autonomous agent to drive safely. `knit` is:

- 🛑 **Read-only by default.** Every mutation is blocked unless you pass `--allow-mutations` —
  enforced *in the binary*, returning a structured `MUTATION_BLOCKED` (exit 12), not an
  interactive "type y" prompt that deadlocks a headless agent.
- 🧵 **Reviewed-artifact publishing.** `post create --dry-run` emits the exact plan + a hash;
  `--apply <hash>` publishes only that plan — no blind `--yes` on an irreversible public post.
- 🤖 **Machine-readable.** `knit schema` dumps the command tree, exit codes, and live safety
  state; `knit agent` (or `KNIT_HELP=agent`) prints the embedded usage contract — no repo, no
  network.
- ✂️ **Token-bounded.** Reads return `{schemaVersion, data, nextCursor?}`; `--limit`/`--select`
  keep responses inside an agent's context window.
- 🔒 **Prompt-injection-aware.** It's a public feed — post/reply/search/mention text and bios are
  fenced as untrusted data, never instructions.
- ✅ **Official API.** Low breakage risk and ToS-compliant — unusual for the agent-CLI crop.

## Install

```bash
go install github.com/rnwolfe/knit/cmd/knit@latest   # best for agents (one line, pinnable)
brew install rnwolfe/tap/knit                        # best for humans
```

Or grab a signed binary from [Releases](https://github.com/rnwolfe/knit/releases) (linux ·
macOS · windows, amd64 · arm64).

## Quickstart

```bash
# 1. Authenticate (headless) — token never touches argv:
printf '%s' "$THREADS_TOKEN" | knit auth login --token-stdin
#    ...or browser: `knit auth login` then paste the redirected URL. (How to get a token: docs.)

knit auth status --json          # account, token expiry, publish quota
knit post list --json            # your recent posts (stable envelope)
knit search posts "ai agents"    # ⚠️ scope:self until advanced access is granted

# Publishing is gated — preview, then apply:
knit post create --text "hello from knit" --dry-run        # plan + hash, nothing posted
knit post create --text "hello from knit" --allow-mutations
```

`knit --help` is example-led; `knit schema` is the machine-readable contract.

## Authenticate

`knit` wraps a Threads access token (full read **+ publish** to your account — treat it like a
password).

- **Headless / agent**: `KNIT_TOKEN` env, or `… | knit auth login --token-stdin`.
- **Browser**: set `KNIT_CLIENT_ID`/`KNIT_CLIENT_SECRET` (your Threads app), run `knit auth
  login`, open the printed URL, approve, paste the redirected URL back. No local server or cert.
- `knit auth status` tests the token and redacts it; `knit auth refresh` extends the 60-day
  token; `knit auth logout` clears **local** creds (revoke upstream in Threads settings — see
  [SECURITY.md](SECURITY.md)).
- Token storage: OS keyring → `0600` file fallback (perms-warned). Secrets **never** via argv.

Getting a token (Meta app + tester setup) is walked through in the [docs](https://rnwolfe.github.io/knit).

## Cookbook

```bash
knit post list --json | jq '.data[] | {id, text}'
knit post list --select id,permalink --format tsv
knit search posts ai --json | jq -r '.scope'          # "public" or "self"
knit insights account --json | jq '.data.followersCount'
knit reply hide <reply-id> --allow-mutations          # idempotent
knit post delete <media-id> --allow-mutations          # idempotent (gone == ok)
knit doctor --for-agent --json                         # config / token / connectivity
```

## Exit codes

`0` ok · `2` usage · `3` empty · `4` auth required · `5` not found · `6` permission
(scope/advanced-access) · `7` rate limited (250/24h publish, 2200/24h search) · `12` mutation
blocked · `13` input required. Full table: `knit schema`.

## For agents

`knit` ships an embedded usage contract: `knit agent` (or `KNIT_HELP=agent knit`) prints it;
`knit schema` is the structured command tree + exit codes + live safety state. Driving it from a
coding agent? See [AGENTS.md](AGENTS.md).

## Contributing & security

PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). For vulnerabilities, **do not open a public
issue**: see [SECURITY.md](SECURITY.md) (private reporting + the secret-handling threat model).

## License

[MIT](LICENSE) © Ryan Wolfe
