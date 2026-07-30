# Yeetcraft Companion

Desktop companion for capturing World of Warcraft combat-log data, reviewing session results locally, and uploading stats to [Yeetcraft](https://github.com/Nikolaj-Hvitfeldt/Yeetcraft).

Yeetcraft is a season-aware Hall of Shame for tracking Mythic+ deaths and yeets. The companion automates the path from in-game combat logs to Yeetcraft’s leaderboard — without requiring manual spreadsheet work mid-session.

## Relationship to Yeetcraft

| Repository | Role |
| ---------- | ---- |
| [Yeetcraft](https://github.com/Nikolaj-Hvitfeldt/Yeetcraft) | Web app + Go API + PostgreSQL (canonical product and API contract owner) |
| **yeetcraft-companion** (this repo) | Standalone desktop companion; independent Go module and release |

These are **separate products**:

- This repository does **not** import Yeetcraft Go packages or depend on Yeetcraft source, PostgreSQL, or build artifacts at runtime.
- During development, the sibling `Yeetcraft` checkout may be opened in the same Cursor workspace **for reference only** (architecture, auth patterns, data model). It is read-only unless a task explicitly grants write access.
- Runtime communication will use a **versioned HTTP API** defined and owned by the Yeetcraft repository. No API implementation or contract generation is part of the companion bootstrap.

## Current status

**Phase 0 evidence work is in progress.** The repository contains a bounded
streaming V22 parser, source-backed typed parsing for selected damage and
metadata events, fail-closed version/project quarantine, synthetic fixtures,
and the privacy-safe `cmd/logprobe` diagnostic CLI.

The following are **not** implemented yet:

- File watching
- Local SQLite storage
- Death, cause, identity, or run detection
- Upload to Yeetcraft
- Review UI (including a future Wails-based desktop shell)
- WoW addon integration (deferred)

Typed parsing is synthetically tested against the project's selected canonical
V22 reference. It is not verification against a real retail log and does not
establish that one logger can observe deaths or causes reliably.

See [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) for the planned phases.

## Prerequisites

- Go 1.22 or later

## Build and run

From the repository root:

```bash
go build -o bin/yeetcraft-companion ./cmd/yeetcraft-companion
./bin/yeetcraft-companion
```

Expected output:

```text
yeetcraft-companion 0.0.0-dev
```

## Test and lint

```bash
gofmt -w .
go test ./...
go vet ./...
```

## Configuration

Copy `.env.example` to `.env` and adjust placeholders when features that read configuration are implemented. Never commit real API keys or combat logs.

## Documentation

- [Implementation plan](docs/IMPLEMENTATION_PLAN.md)
- [Yeetcraft integration](docs/YEETCRAFT_INTEGRATION.md)
- [Combat-log capabilities](docs/COMBAT_LOG_CAPABILITIES.md)
- [Agent instructions](AGENTS.md) — for AI coding assistants working in this repo

## License

MIT — see [LICENSE](LICENSE). The GitHub repository is public; Yeetcraft uses the same license family.
