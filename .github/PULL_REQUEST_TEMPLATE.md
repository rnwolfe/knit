## What & why
<!-- What does this change and why? Link any issue: Closes #NNN -->

## Checklist
- [ ] Conventional Commit title (`feat:`/`fix:`/`docs:`/…)
- [ ] `go build ./... && go vet ./... && go test ./...` pass; `gofmt -l .` clean
- [ ] Output field names unchanged or **append-only** (no renames/removals)
- [ ] If the command surface changed: regenerated the schema golden (`KNIT_UPDATE_GOLDEN=1 go test ./internal/cli/`)
- [ ] CHANGELOG updated under `## [Unreleased]`
- [ ] Signed off (`git commit -s`)
