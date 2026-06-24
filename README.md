# knit

**An agent-friendly CLI for [Instagram's Threads](https://www.threads.net).** Read-only by
default, safe for autonomous agents to drive, and built on the official Threads API.

> Think of it as `bird` for Threads — but with the safety baked into the binary, not a wrapper
> an agent can shell past.

> ⚠️ **Scaffold stage.** The command surface and the agent-CLI contract plumbing are real and
> tested; the target logic is currently a local placeholder. Real Threads API + auth wiring
> lands next. See `spec.md` and `AGENTS.md`.

## Why knit

Several Threads CLIs already exist — none is built for agents. `knit` is the only one that is:

- **Read-only by default.** Publishing, replying, hiding, and reposting are blocked unless you
  pass `--allow-mutations` — enforced *in the binary*, returning a structured `MUTATION_BLOCKED`
  (exit 12) an agent can act on. No interactive "type y" prompt that deadlocks a headless agent.
- **Reviewed-artifact publishing.** `post create --dry-run` emits the exact plan plus a hash;
  `--apply <hash>` publishes only that plan — closing the gap a blind `--yes` opens on an
  irreversible, public post.
- **Machine-readable.** `knit schema` dumps the command tree, exit codes, and live safety state;
  `knit agent` prints the embedded usage contract — no repo, no network.
- **Token-bounded + stable output.** Every read returns `{schemaVersion, data, nextCursor?}`;
  `--limit`/`--select` keep responses inside an agent's context window.
- **Prompt-injection aware.** It's a public feed: text from posts, replies, search, mentions,
  and bios is treated as untrusted data, not instructions.
- **Safe secrets.** Tokens live in the OS keyring (never argv); official API → low breakage risk.

## Install

```bash
go install github.com/rnwolfe/knit/cmd/knit@latest   # best for agents
brew install rnwolfe/tap/knit                        # best for humans
```

## Quickstart

```bash
knit auth login --token-stdin < token.txt   # headless; or `knit auth login` to paste a callback URL
knit auth status --json                      # scopes, token expiry, advanced-access state

knit post list --json                        # your recent posts
knit post get <media-id> --json
knit search posts "coffee" --json            # ⚠️ scope:self unless advanced access granted
knit insights account --json

knit post create --text "hello threads" --dry-run     # plan + hash, changes nothing
knit post create --text "hello threads" --allow-mutations
```

`knit --help` is example-led; `knit schema` is the machine-readable contract.

## Exit codes

`0` ok · `2` usage · `3` empty · `4` auth required · `5` not found · `6` permission
(scope/advanced-access) · `7` rate limited (250/24h publish, 500/7d search) · `12`
mutation blocked · `13` input required. Full table: `knit schema`.

## License

MIT — see [LICENSE](LICENSE).
