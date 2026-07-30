# Implementation plan

> **Status:** Foundation bootstrap complete. **Phase 0 not started.**

This document tracks planned work for the yeetcraft-companion repository. Timelines and ordering may change as combat-log research and Yeetcraft API design progress.

## Goals

1. Watch the local WoW combat-log directory for new log files and appended lines.
2. Parse relevant combat-log events into structured session data.
3. Persist sessions locally for crash recovery and offline review.
4. Present a review step so the user can confirm deaths, yeets, dungeon, and players before upload.
5. Upload confirmed stats to Yeetcraft via a versioned HTTP API.

## Non-goals (for now)

- WoW addon (deferred; may supplement log parsing later).
- Direct database access to Yeetcraft PostgreSQL.
- Importing Yeetcraft Go packages or sharing a monorepo module.
- Implementing or changing the Yeetcraft API from this repository.

## Phases (draft)

### Phase 0 — Research and fixtures

**TODO — not started**

- [ ] Collect anonymized combat-log samples under `testdata/logs/` (Mythic+ runs with deaths/yeets).
- [ ] Document observed log line formats in [COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md).
- [ ] Confirm WoW combat-log path conventions on Windows (primary target).
- [ ] List open questions for session boundaries (key start/end signals).

### Phase 1 — Parser foundation

**TODO**

- [ ] Define event types and parser API in `internal/parser/`.
- [ ] Unit tests driven by `testdata/logs/` fixtures.
- [ ] No SQLite, no network, no file watcher yet.

### Phase 2 — Log watching and session assembly

**TODO**

- [ ] Implement `internal/logwatcher/` for tailing/appending combat logs.
- [ ] Implement `internal/session/` to correlate parser output into a session model.
- [ ] Configuration via `internal/config/` (log directory path, etc.).

### Phase 3 — Local storage

**TODO**

- [ ] Choose SQLite driver and schema for sessions and upload queue.
- [ ] Implement `internal/storage/`.
- [ ] Migration strategy for local DB versions.

### Phase 4 — Review flow

**TODO**

- [ ] Define review UX (CLI first; Wails desktop shell evaluated separately).
- [ ] Implement `internal/review/` against stored sessions.

### Phase 5 — Yeetcraft upload

**TODO**

- [ ] Implement `internal/uploader/` against the versioned Yeetcraft HTTP API.
- [ ] Auth via API key header (see Yeetcraft `docs/API.md` — verify before coding).
- [ ] Idempotency and retry behavior for failed uploads.

### Phase 6 — Desktop packaging (optional / parallel)

**TODO**

- [ ] Evaluate Wails or alternative shell after CLI path is stable.
- [ ] Installer and auto-update strategy (open question).

## Assumptions

- Retail WoW combat logs use the standard `WoWCombatLog-*.txt` format under the game's `Logs` directory.
- Mythic+ session detection can be inferred from combat-log events alone (unverified — needs Phase 0 proof).
- Yeetcraft will expose (or already exposes) an HTTP endpoint suitable for batch stat uploads (schema TBD in Yeetcraft repo).

## Open questions

- Minimum viable event set for death vs yeet classification in logs.
- How season and dungeon IDs map between log-derived names and Yeetcraft’s canonical lists.
- Whether upload should be automatic after review or always require explicit confirmation.
- macOS/Linux support priority relative to Windows.

## References

- Yeetcraft architecture (read-only): sibling `Yeetcraft/docs/ARCHITECTURE.md`
- Integration notes: [YEETCRAFT_INTEGRATION.md](./YEETCRAFT_INTEGRATION.md)
- Combat log research: [COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md)
