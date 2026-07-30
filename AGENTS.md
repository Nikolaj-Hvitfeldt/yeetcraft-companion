# Agent instructions — yeetcraft-companion

Operational guide for AI coding assistants working in this repository.

## Repository boundaries

- **Write scope:** only files under `yeetcraft-companion/`.
- **Read-only reference:** sibling `Yeetcraft/` may be inspected for architecture, API conventions, auth, data model, and test patterns. Do **not** modify, format, rename, or delete files there unless a task explicitly grants write access.
- **No cross-repo Go imports:** this module is `github.com/Nikolaj-Hvitfeldt/yeetcraft-companion`. Never import `yeetcraft/backend` or other Yeetcraft packages.
- **No runtime coupling:** the companion must not depend on Yeetcraft source, PostgreSQL, local Yeetcraft build output, or filesystem paths into the Yeetcraft repo at runtime.
- **Git root:** run all Git and Go commands from `yeetcraft-companion/`. Do not nest a Git repository inside Yeetcraft.
- **Commits:** only create commits or push when the user explicitly asks.

## Product scope

The companion watches WoW combat logs, parses relevant events, persists state locally, lets the user review results, and uploads confirmed stats to Yeetcraft over HTTP.

**Deferred:** WoW addon integration. Do not implement addon code unless a future task requests it.

**Out of scope for companion tasks:** implementing or changing the Yeetcraft API. If a task requires API changes, stop and describe the needed Yeetcraft-side work instead of editing that repo.

## Dependency rules

- Prefer the Go standard library until a feature truly needs a third-party package.
- Do not add Wails, SQLite drivers, HTTP clients beyond stdlib needs, or filesystem watchers until the relevant phase is approved.
- Do not introduce abstractions for unimplemented features (no premature interfaces or plugin systems).
- When adding dependencies, document why in the PR or commit message and keep the module graph minimal.

## Privacy and data handling

Combat logs contain player names, guild tags, and other identifiable information.

- Never commit real combat logs, `.env` files, API keys, or SQLite databases.
- Raw logs belong under ignored paths or user-local directories, not in the repo.
- Synthetic fixtures belong under `testdata/` and must be minimal and anonymized.
- Upload code must send only data the user has reviewed and confirmed (once review exists).
- Log diagnostic output must avoid printing full combat-log lines in production paths.

## Testing expectations

- Use Go’s `testing` package.
- Run `gofmt`, `go test ./...`, and `go vet ./...` before finishing substantive Go changes.
- Place small synthetic log snippets in `testdata/logs/` for parser tests (when parser exists).
- Integration tests that hit the Yeetcraft API require explicit test credentials and should be tagged or guarded so CI does not call production.
- Do not invent verified combat-log behavior; mark unknowns as assumptions and add tests when samples are available.

## Package layout

```text
cmd/yeetcraft-companion/   CLI entrypoint (desktop shell may come later)
internal/config/           Environment and paths
internal/logwatcher/       Combat-log directory watching
internal/parser/           WoW combat-log line parsing
internal/session/          Active M+ session state
internal/storage/          Local persistence (SQLite planned)
internal/uploader/         Yeetcraft HTTP client
internal/review/           Pre-upload review flow
testdata/logs/             Synthetic fixtures
docs/                      Plans and integration notes
```

Keep handlers thin; put parsing and persistence logic in the appropriate `internal/` package.

## Yeetcraft integration (reference)

- Yeetcraft writes use `X-API-Key` or `Authorization: Bearer` with a shared secret; mutations fail closed when the server has no `API_KEY`.
- Public reads need no key. The canonical contract will live in the Yeetcraft repo (see `docs/API.md` there when implementing upload).
- Companion upload must target a **versioned** HTTP API; do not hard-code unverified endpoint shapes. See [docs/YEETCRAFT_INTEGRATION.md](docs/YEETCRAFT_INTEGRATION.md).

## Phased work

Follow [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md). **Phase 0 is not started.** Do not implement parser, SQLite, Wails, or upload until the corresponding phase task says so.

## Documentation

- Update README when user-facing behavior or build steps change.
- Keep `docs/` TODO sections honest — distinguish assumptions, open questions, and verified facts.
- Do not claim combat-log capabilities without log samples or Blizzard documentation to support them.

## Verification checklist

Before marking a task complete:

1. Confirm only `yeetcraft-companion/` was modified.
2. Run `gofmt`, `go test ./...`, `go vet ./...` from this repo root.
3. Report `git status` and note any open decisions.

## Open questions (global)

- Exact Yeetcraft upload API version and payload schema (owned by Yeetcraft repo).
- Which combat-log events reliably indicate deaths, yeets, dungeon, and party roster.
- Desktop UI technology timing (CLI-first vs early Wails shell).

Resolve these in docs or implementation tasks; do not guess in production code.
