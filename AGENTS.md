# AGENTS.md — knit

Agent-focused CLI for **Instagram's Threads** (official Threads API). Built to the
agent-cli-factory contract: read-only by default, in-binary mutation gate, structured errors +
exit codes, machine-readable `schema --json`, embedded `SKILL.md`, token-bounded output,
prompt-injection fencing.

> **Status: scaffold.** The command surface, contract plumbing, and tests are real and green,
> but the target logic is a local placeholder. `cli-implement` wires the real Threads API +
> auth (see "Wiring the real target" below). `spec.md` is the source of truth.

## Build / test / run

```bash
go build ./...            # build
go vet ./...              # vet
go test ./...             # contract tests (must stay green — incl. schema-snapshot gate)
go run ./cmd/knit --help  # example-led help
go run ./cmd/knit schema  # machine-readable command tree + exit codes + live safety state
```

## Layout

```
cmd/knit/main.go              # os.Exit(cli.Run(...)) only — no logic
internal/cli/                 # kong grammar + per-noun commands
  root.go                     #   global flags, Runtime, the Guard mutation gate, error mapping
  post.go reply.go profile.go #   the noun-verb surface (post/reply/profile/search/mentions/insights)
  search.go mentions.go insights.go
  misc.go                     #   auth / doctor / schema / agent / version
  suggest.go                  #   "did you mean" (levenshtein)
  cli_test.go                 #   contract tests
internal/output/              # output contract: stdout=data, --format, --select, --limit, the read envelope
internal/errs/                # stable exit-code table + structured CLIError
internal/store/               # PLACEHOLDER target (local JSON) — REPLACE in cli-implement
internal/skill/SKILL.md       # embedded agent contract (printed by `knit agent`)
internal/version/             # ldflags version + ReadBuildInfo fallback
```

## Conventions (do not regress)

- `main()` only calls `cli.Run(args, stdin, stdout, stderr) int` — everything testable in-process.
- **stdout = data, stderr = chatter.** Never print progress/notes to stdout.
- JSON is 2-space indented with `SetEscapeHTML(false)` (permalinks/media URLs must survive).
- Every **mutation** calls `rt.Guard(op)` FIRST, supports `--dry-run`, and (for publish) the
  reviewed-artifact pattern (`--dry-run` → plan+hash, `--apply <hash>`).
- Reads emit the stable envelope via `rt.Out.EmitEnvelope(data, nextCursor)` —
  `{schemaVersion, data, nextCursor?}`. Output field names are **append-only**.
- Exit codes come from `internal/errs` — never collapse to bare `1`.

## Wiring the real target (cli-implement)

1. Replace `internal/store/` with `internal/api/` (direct HTTP to `graph.threads.net`) and add
   `internal/auth/` (OS keyring via `99designs/keyring`, `0600` fallback).
2. Wire auth per `spec.md` §Auth: `--token-stdin` (primary, headless) + paste-the-callback-URL
   (human onboarding); long-lived 60-day token with `auth refresh`. Secrets via stdin/env,
   **never argv**. `KNIT_CLIENT_ID` / `KNIT_CLIENT_SECRET` for the Threads app credentials.
3. Repoint each command's `Run` at the API client; fill the full output schema from `spec.md`.
4. Fence untrusted text (post/reply/search/mentions text + profile bio) in agent mode (contract §8).
5. Resolve the open items in `spec.md`: repost endpoint, hide-reply mechanism, delete endpoint.
