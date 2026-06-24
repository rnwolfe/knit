---
title: Exit codes
description: knit's stable, semantic exit codes for scripting and agents.
owner: rnwolfe
lastReviewed: 2026-06-23
---

Exit codes are a stable contract — distinct per failure class, never collapsed to a bare `1`.
The authoritative table is always `knit schema | jq .exit_codes`.

| Code | Name | Meaning & typical fix |
|---|---|---|
| `0` | ok | Success. |
| `1` | generic_error | Unclassified failure. |
| `2` | usage | Bad flags/args, or `PLAN_MISMATCH` on `--apply`. |
| `3` | empty_results | Query succeeded, nothing matched. |
| `4` | auth_required | No/expired token → `knit auth login` / `knit auth refresh`. |
| `5` | not_found | The id doesn't exist or isn't visible to you. |
| `6` | permission | A scope or **advanced access** is missing (App Review). |
| `7` | rate_limited | HTTP 429 — 250 posts/24h, 2200 searches/24h, or a burst cap. |
| `8` | retryable | Transient upstream/network error; retry. |
| `10` | config_error | Missing `KNIT_CLIENT_ID`/`KNIT_CLIENT_SECRET`, or doctor failed. |
| `12` | mutation_blocked | `--allow-mutations` not set. Structured `MUTATION_BLOCKED`. |
| `13` | input_required | `--no-input` hit a needed prompt. |
| `130` | cancelled | Interrupted (SIGINT). |

Errors print as structured JSON to **stderr** (stdout stays parseable):

```json
{ "error": "…", "code": "AUTH_REQUIRED", "remediation": "run `knit auth login`" }
```
