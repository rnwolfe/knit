---
title: Quickstart
description: Install knit and reach first success in under a minute.
owner: rnwolfe
lastReviewed: 2026-06-23
---

## Install

```bash
go install github.com/rnwolfe/knit/cmd/knit@latest   # best for agents
brew install rnwolfe/tap/knit                        # best for humans
```

Or download a signed binary from [Releases](https://github.com/rnwolfe/knit/releases)
(linux · macOS · windows, amd64 · arm64).

## Authenticate

You need a Threads access token. If you don't have one yet, follow
[Get a Threads token](/tutorials/get-a-token/) first.

```bash
printf '%s' "$THREADS_TOKEN" | knit auth login --token-stdin   # headless; token never on argv
knit auth status --json                                        # account · expiry · publish quota
```

## First reads

```bash
knit profile get me --json
knit post list --json | jq '.data[] | {id, text}'
knit search posts "ai agents" --json | jq -r '.scope'   # "public" or "self"
```

Every read returns a stable envelope:

```json
{ "schemaVersion": 1, "data": [ /* ... */ ], "nextCursor": "…" }
```

`--limit N` bounds the list; `--select id,text` projects fields; pass `nextCursor` back via
`--cursor` to page.

## Publish (gated)

Publishing is blocked unless you opt in. Preview first, then apply:

```bash
knit post create --text "hello from knit" --dry-run        # plan + hash, nothing posted
knit post create --text "hello from knit" --allow-mutations
```

## Verify your setup

```bash
knit doctor --for-agent --json
```

Next: the [command reference](/reference/commands/) or [why knit is agent-safe](/explanation/agent-safety/).
