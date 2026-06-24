# Contributing to knit

Thanks for helping make `knit` better. This is a Go CLI built to the
[agent-CLI contract](AGENTS.md) — read-only by default, machine-readable, prompt-injection-aware.

## Development setup

```bash
git clone https://github.com/rnwolfe/knit
cd knit
go build ./...            # build
go test ./...             # tests (must stay green)
go vet ./...              # vet
gofmt -l .                # must print nothing
go run ./cmd/knit --help  # try it
```

Go 1.23+. No CGo (`CGO_ENABLED=0`) — keep it that way for static cross-compiles.

## The contract is load-bearing — don't regress it

See [AGENTS.md](AGENTS.md) for the full list. The essentials:

- **stdout = data, stderr = chatter.** Never print progress/notes to stdout.
- Every **mutation** calls `rt.Guard(op)` first and supports `--dry-run`.
- Reads emit the stable `{schemaVersion, data, nextCursor?}` envelope; **output field names are
  append-only**.
- Free text from Threads is fenced as untrusted in agent mode.
- Secrets via stdin/env, **never argv**.

The **schema-snapshot golden** (`internal/cli/testdata/schema.golden.json`) is a CI gate. After
an intentional surface change, regenerate it:

```bash
KNIT_UPDATE_GOLDEN=1 go test ./internal/cli/
```

and include the golden diff in your PR so the change is reviewed.

## Commit & PR conventions

- **Conventional Commits** (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`). The
  changelog and semver bump are derived from these.
- Sign off your commits (**DCO**): `git commit -s`. No CLA.
- Each PR: green CI, a test for new behavior, and a CHANGELOG entry under `## [Unreleased]`.
- Keep changes focused; one logical change per PR.

## Filing issues
Use the issue forms (bug / feature). Bug reports need `knit version`, OS, the exact command, and
relevant `--json` output with **any token redacted**.
