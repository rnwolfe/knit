# Changelog

All notable changes to this project are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-25

### Added
- Initial release: agent-friendly CLI for Instagram's Threads (official Threads API).
- Read commands: `profile get`, `post list/get`, `reply list/tree`, `search posts`,
  `mentions list`, `insights post/account`.
- Mutations (gated by `--allow-mutations`): `post create/repost/delete`, `reply create/hide/unhide`.
- Auth: `auth login` (`--token-stdin` + paste-the-callback-URL), `status`, `logout`, `refresh`.
- Agent-CLI contract: read-only by default, `--json`/`--format`/`--select`/`--limit`,
  stable `{schemaVersion,data,nextCursor}` envelope, structured errors + exit codes,
  `schema --json`, embedded `agent`/`KNIT_HELP=agent`, prompt-injection fencing, `doctor`.

[Unreleased]: https://github.com/rnwolfe/knit/commits/main
