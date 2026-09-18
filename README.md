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

**Phase 0 was accepted for MVP progression on 2026-09-18. Phase 1 contract
review merged. Phase 2 headless capture foundation is implemented on
`feat/phase-2/headless-capture` and ready for human review / PR to companion
`dev`.** The repository contains a bounded streaming V22 parser, source-backed
typed parsing for selected damage and metadata events, fail-closed tracked-character
configuration, death detection with contract ID recipes, local SQLite persistence,
combat-log tailing with restart/truncation/rotation handling, the headless
`cmd/yeetcraft-companion` capture command, and the privacy-safe `cmd/logprobe`
diagnostic CLI.

Two reviewed local retail sessions cover five completed runs and 43 deaths
(35 tracked, eight untracked), including boss/trash attribution, a failed pull,
a full-party wipe, repeated death after resurrection, and high/medium cause
confidence. Detected deaths default to ordinary deaths; the **Yeetcraft
website** reclassifies them as `yeet` or `ignored` (companion local review is
not classification authority). Automatic yeet detection is
not an MVP gate.

**Implemented (Phase 2):**

- Fail-closed tracked-character config (`YEETCRAFT_TRACKED_GUIDS`, `WOW_LOG_TIMEZONE`)
- Local SQLite storage with persist-once offsets
- File watching / resume (`internal/logwatcher`)
- Headless capture command (`cmd/yeetcraft-companion --log-file <path> [--once]`)

**Not implemented yet:**

- Upload to Yeetcraft (`internal/uploader`, Phase 4)
- Review UI (`internal/review`, Phase 5)
- Wails desktop shell (Phase 6)
- WoW addon integration (deferred)

Typed parsing is synthetically tested and partially validated against local
retail 12.1.0 logs kept under `local-data/` (gitignored). Use `logprobe --file
<path> --deaths [--track-guid <Player-GUID>]` to inspect death candidates
without uploading raw logs.

See [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) for the phased roadmap and Phase 2 acceptance evidence.

## Prerequisites

- Go 1.22 or later

## Build and run

From the repository root:

```bash
go build -o bin/yeetcraft-companion ./cmd/yeetcraft-companion
export YEETCRAFT_TRACKED_GUIDS=Player-0001-00000001
export WOW_LOG_TIMEZONE=America/New_York
./bin/yeetcraft-companion --log-file /path/to/WoWCombatLog.txt
```

For a single poll (tests and debugging):

```bash
./bin/yeetcraft-companion --once --log-file /path/to/WoWCombatLog.txt --db-path ./data/companion.db
```

Without `--log-file`, the command prints usage and exits non-zero. Production
capture requires tracked GUID configuration and fails closed when it is missing.

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
- [Character and encounter handoff](docs/CHARACTER_AND_ENCOUNTER_HANDOFF.md)
- [Combat-log capabilities](docs/COMBAT_LOG_CAPABILITIES.md)
- [Agent instructions](AGENTS.md) — for AI coding assistants working in this repo

## License

MIT — see [LICENSE](LICENSE). The GitHub repository is public; Yeetcraft uses the same license family.
