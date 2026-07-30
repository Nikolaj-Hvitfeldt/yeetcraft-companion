# Agent instructions — yeetcraft-companion

Operational guide for AI coding assistants working in this repository.

## Repository boundaries

- **This repository** owns the standalone Yeetcraft companion application (`github.com/Nikolaj-Hvitfeldt/yeetcraft-companion`).
- **Write scope:** only files under `yeetcraft-companion/`.
- **Read-only reference:** sibling `./yeetcraft` is a separate repository. Inspect it for architecture, API conventions, data model, authentication, and tests. Do **not** modify, format, rename, or delete files there unless a task explicitly grants write access.
- **No cross-repo Go imports.** Never import `yeetcraft/backend` or other Yeetcraft packages.
- **No runtime coupling.** The companion must not depend on Yeetcraft source, PostgreSQL, local Yeetcraft build output, or filesystem paths into the Yeetcraft repo at runtime.
- **Runtime integration** uses a **versioned HTTP API** only.
- **Cross-repository work:** if changes are needed in both repositories, describe an explicit cross-repository task with separate changes in each repo. Do not edit `./yeetcraft` from a companion-only task.
- **Git and Go commands** run from `yeetcraft-companion/`. Do not nest a Git repository inside Yeetcraft.
- **Commits and pushes** only when the user explicitly requests them.

## Contract ownership

- The **canonical companion API contract** is owned by the Yeetcraft repository.
- **Planned contract location (not implemented yet):** `./yeetcraft/contracts/companion/v1/`
  - Verified absent at bootstrap; do not describe this path as existing until it does.
- This repository may later contain **generated code or test fixtures** derived from that contract. Those copies must **not** become an alternative source of truth.
- Contract changes must be **coordinated explicitly** across both repositories.
- Until the companion contract exists, treat Yeetcraft `docs/API.md` as reference for the **existing** web API only — not as the companion upload spec.

## Development behavior

- Work only within the **phase and scope** requested by the user.
- **Inspect** relevant existing files before making changes.
- **Preserve** unrelated user changes; avoid unrelated refactors.
- Use **`rg`** for repository searches where available.
- Keep packages **focused**; put parsing, persistence, and upload logic in the appropriate `internal/` package.
- Prefer the Go **standard library** until a feature truly needs a third-party dependency.
- Do **not** add dependencies without a concrete need.
- Do **not** create speculative abstractions for unimplemented features.
- Do not add Wails, SQLite drivers, filesystem watchers, or non-stdlib HTTP clients until the relevant phase task approves them.

## Architecture constraints

- Core parsing and domain logic must remain **testable without Wails or a GUI**.
- UI code must **not** own parsing, persistence, or upload logic.
- **SQLite** holds local companion state; it is not a replica of Yeetcraft's PostgreSQL database.
- Upload behavior must eventually be **retryable and idempotent**.
- Combat-log lines and API payloads are **untrusted input**.
- **Unknown combat-log events** must not crash ingestion; skip or record safely.
- **WoW addon** support is deferred until after the companion MVP works. Do not implement addon code unless a future task requests it.
- **Out of scope for companion-only tasks:** implementing or changing the Yeetcraft API. Stop and describe required Yeetcraft-side work instead.

## Privacy and repository hygiene

Combat logs contain player names and other identifiable information.

- Never commit credentials, access tokens, personal data, raw user combat logs, `.env` files, SQLite databases, diagnostics, or build output.
- Test fixtures under `testdata/` must be **small, synthetic, and anonymized**. Do not copy complete production logs into `testdata/`.
- Never include secrets in documentation, examples, test output, or error messages.
- Upload code must send only data the user has **reviewed and confirmed** (once review exists).
- Avoid printing full combat-log lines in production diagnostic paths.

## Testing expectations

- Use Go's `testing` package.
- Use **table-driven tests** for parser behavior where appropriate.
- Add **regression tests** for fixed bugs where practical.
- Test **failure paths and malformed input**, not only successful cases.
- Place synthetic log snippets in `testdata/logs/` for parser tests (when parser exists).
- Guard integration tests that hit the Yeetcraft API so CI does not call production without explicit credentials.
- Do not invent verified combat-log behavior; mark unknowns as assumptions until fixtures prove them.
- Before finishing substantive Go changes, run on changed files where applicable:
  - `gofmt`
  - `go test ./...`
  - `go vet ./...`
  - any additional repository checks relevant to the changed files
- **Report checks that could not be run.**

## Package layout

```text
cmd/yeetcraft-companion/   CLI entrypoint (desktop shell may come later)
internal/config/           Environment and paths
internal/logwatcher/       Combat-log directory watching
internal/parser/           WoW combat-log line parsing (placeholder until Phase 0+)
internal/detection/        Death-candidate detection, recent-damage tracking, later classification support (planned; not implemented)
internal/session/          Active M+ session state
internal/storage/          Local persistence (SQLite planned)
internal/uploader/         Yeetcraft HTTP client
internal/review/           Pre-upload review flow
testdata/logs/             Synthetic fixtures
docs/                      Plans and integration notes
```

## Documentation

- Update README or `docs/` when an architectural decision, API contract, schema, setup process, or implementation phase changes.
- Keep TODO sections honest — distinguish **assumptions**, **open questions**, and **verified facts**.
- Do not describe assumptions as verified behavior.
- Record unresolved decisions explicitly instead of silently guessing.
- See [docs/YEETCRAFT_INTEGRATION.md](docs/YEETCRAFT_INTEGRATION.md) for companion-side integration notes.

## Phased work

Follow [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md). **Phase 0 is not started.**

- Do not implement **SQLite**, **Wails**, **upload**, or the **addon** until the corresponding phase task approves them.
- Do not implement **parser** or **detection** by default.
- A **bounded streaming parser and detection probe** are in scope when an explicitly approved **Phase 0** task requests them.

## Definition of done

Before marking a task complete:

1. Code **compiles**; relevant tests **pass**.
2. Expected **failure states** are handled.
3. Documentation reflects **material decisions**.
4. No secrets, databases, raw logs, diagnostics, or build output are included in changes.
5. Confirm only `yeetcraft-companion/` was modified (unless an explicit cross-repo task says otherwise).
6. Run and report validation (`gofmt`, `go test ./...`, `go vet ./...`, plus any other relevant checks).
7. Final response lists **changed files**, **validation performed**, and remaining **risks or open questions**.

## Open questions

Resolve in docs or scoped tasks; do not guess in production code:

- Companion upload API schema (awaiting `./yeetcraft/contracts/companion/v1/`).
- Combat-log events for deaths, yeets, dungeon, and party roster.
- Desktop shell timing (CLI-first vs early Wails).
