# AGENTS.md — knit

Agent-focused CLI for **Instagram's Threads** (official Threads API). Built to the
agent-cli-factory contract: read-only by default, in-binary mutation gate, structured errors +
exit codes, machine-readable `schema --json`, embedded `SKILL.md`, token-bounded output,
prompt-injection fencing.

> **Status: implemented.** Commands talk to the real Threads API (`internal/api`) with OAuth +
> keyring auth (`internal/auth`). `spec.md` is the source of truth. Remaining work is polish
> (`cli-publish`): docs site, VHS demo, discoverability.

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
  root.go                     #   global flags, Runtime, the Guard gate, fencing decision, error mapping
  post.go reply.go profile.go #   the noun-verb surface (post/reply/profile/search/mentions/insights)
  search.go mentions.go insights.go
  authcmd.go                  #   auth login/status/logout/refresh (OAuth + keyring)
  doctor.go                   #   real config/credentials/connectivity/token checks
  fence.go                    #   prompt-injection fencing of untrusted text (§8)
  misc.go                     #   schema / agent / version
  suggest.go                  #   "did you mean" (levenshtein)
  cli_test.go schema_golden_test.go
internal/api/                 # real Threads client: client.go (transport+error mapping),
                              #   threads.go (reads), publish.go (writes), types.go, iface.go
internal/auth/                # auth.go (KNIT_TOKEN/keyring/0600 storage), oauth.go (the flows)
internal/output/              # output contract: stdout=data, --format, --select, --limit, the read envelope
internal/errs/                # stable exit-code table + structured CLIError
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
- Commands depend on the `api.Threads` interface via `rt.API`; tests inject fakes through the
  `apiFactory` seam. Secrets via stdin/env, **never argv**.
- The **schema-snapshot golden** (`internal/cli/testdata/schema.golden.json`) is a CI gate.
  After an intentional surface change: `KNIT_UPDATE_GOLDEN=1 go test ./internal/cli/`.

## Known follow-ups
- Multi-image/video **carousel** posts: the container path supports `children`, but `post create`
  currently publishes a single media item. Extend `api.Publish` + the command to build carousels.
- `insights account --metric follower_demographics` requires a `breakdown` param — surface it
  as a flag when wiring richer demographics.
