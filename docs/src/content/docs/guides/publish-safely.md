---
title: Publish safely (the gate & dry-run)
description: How knit's mutation gate, dry-run plan+hash, and reviewed-artifact apply work.
owner: rnwolfe
lastReviewed: 2026-06-23
---

Publishing to Threads is **public and irreversible**, so `knit` makes it deliberate.

## The gate

Every mutation (`post create`, `post repost`, `post delete`, `reply create`, `reply hide`,
`reply unhide`) is blocked unless you pass `--allow-mutations`:

```bash
$ knit post create --text "gm" --json
{ "code": "MUTATION_BLOCKED", "remediation": "re-run with --allow-mutations (add --dry-run)" }
# exit 12
```

This is enforced **in the binary**, so an agent shelling out directly can't bypass it — and it's
a structured code (exit 12), not an interactive prompt that would deadlock a headless run.

## Preview with `--dry-run`

```bash
knit post create --text "hello" --allow-mutations --dry-run
```

emits the exact plan plus a content **hash**, and publishes nothing:

```json
{ "dryRun": true, "hash": "ce601243a2b60382", "plan": { "action": "post.create", "text": "hello", … } }
```

## Reviewed-artifact apply

For a stricter workflow, publish *only* the exact plan you reviewed:

```bash
knit post create --text "hello" --apply ce601243a2b60382 --allow-mutations
```

If the plan changed since you took the hash, `knit` refuses with `PLAN_MISMATCH` (exit 2) — this
closes the time-of-check/time-of-use gap a blind `--yes` would open.

## Idempotent verbs

`reply hide`/`unhide` and `post delete` are idempotent (contract §9): re-running, or deleting an
already-gone post, is a soft success — so an agent's retries don't hard-fail.

## Watch your quota

The Threads API caps publishing at **250 posts / 24h**. `knit auth status` and `post create
--dry-run` surface `publishQuotaRemaining` so an agent backs off before a `429` (exit 7).
