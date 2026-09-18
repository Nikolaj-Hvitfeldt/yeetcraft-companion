# Companion Phase 2 implementation file map

> **Status: Draft — file map only; Phase 2 is not done**

| Field | Value |
| ----- | ----- |
| Work package | WP5 |
| Owner | yeetcraft-companion |
| Phase mapped here | **Phase 2** (headless capture foundation) |
| Canonical contract | [`../yeetcraft/contracts/companion/v1/`](../yeetcraft/contracts/companion/v1/README.md) (Yeetcraft-owned) |
| Yeetcraft Phase 3 map | [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md) |

This document is a **file map**, not a new protocol and not a second contract.

**Do not copy** Yeetcraft `contracts/companion/v1/` schemas, examples, or `CONTRACT.md` into this repository as a source of truth. Phase 2 tests may later hold **mechanically verified derived fixtures** under `testdata/` with a recorded canonical checksum. Checksum/drift harness: **deferred to Validate**.

WP1 producer gaps remain in [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md).

---

## Locked production behavior (Phase 2)

Production capture **fails closed** when tracked-character configuration is absent or corrupt.

Verified today: [`internal/detection/tracker.go`](../internal/detection/tracker.go) `NewTracker()` with an empty GUID list sets `allPlayers = true` and records every player death. That prototype default is **not** acceptable for production capture (`cmd/yeetcraft-companion`).

| Path | Production rule |
| ---- | --------------- |
| `internal/config` | Missing, empty, or corrupt tracked GUID list → do not start capture |
| `cmd/yeetcraft-companion` | Fail closed; never fall back to track-all |
| `internal/detection` | Production callers must not use empty-list `allPlayers` |
| `cmd/logprobe` | Diagnostic CLI only. Track-all remains a **explicit diagnostic** mode, not the production default |

Untracked names, realms, and GUIDs must not be prepared for later upload. Cause evidence must be redacted locally (no player names/realms/untracked GUIDs).

---

## Sequencing

| Dependency | Owner | Notes |
| ---------- | ----- | ----- |
| Frozen v1 wire contract | Yeetcraft | Read [`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../yeetcraft/contracts/companion/v1/CONTRACT.md). Do not fork it. |
| Nullable unique `characters.guid` | Yeetcraft character slice | **End-to-end ingest** blocker. Phase 2 may persist local GUIDs and IDs without calling the server. |
| `dungeons.challenge_map_id`, `seasons.starts_at`/`ends_at` | Yeetcraft Phase 3 schema | Server resolution. Companion still sends challenge map ID and optional season hint. |
| HTTP uploader | **Phase 4** (`internal/uploader`) | Not Phase 2 |
| Review UI | **Phase 5** (`internal/review`) | Not Phase 2. Local hold-for-review **state** may live in `internal/storage` / `internal/session` in Phase 2. |
| Wails / addon | Phase 6 / 8 | Not Phase 2 |

Phase 2 exit is local: restart/truncation/rotation tests pass; events persist exactly once; IDs follow the canonical recipes; production tracking fails closed.

---

## Phase 2 file map

**Status:** `existing` = verified on disk now; `existing stub` = `doc.go` only; `to create` = not present.

### Existing parser, detection, and logprobe (extend)

| Path | Status | Role in Phase 2 |
| ---- | ------ | --------------- |
| `internal/parser/doc.go` | existing | Package docs. Parser stays roster-neutral. |
| `internal/parser/envelope.go` | existing — extend | Feed canonical RFC 3339 UTC instants; timezone-less stamps need persisted WoW-log timezone from `internal/config`. DST/invalid stamps → local hold, not hash. |
| `internal/parser/envelope_test.go` | existing — extend | Canonical-instant cases; synthetic only. |
| `internal/parser/parse.go`, `scan.go`, `line_reader.go`, `event.go`, `payload.go`, `payload_parse.go`, `csv.go`, `hex.go`, `version.go`, `errors.go` | existing | Keep streaming V22 behavior; no upload, no SQLite. |
| `internal/parser/*_test.go` | existing — extend | Malformed/unknown/partial-line regressions stay required. |
| `internal/detection/doc.go` | existing | In-memory death candidates. |
| `internal/detection/tracker.go` | existing — **behavior change** | Production must not treat empty GUID list as track-all. Capture `CHALLENGE_MODE_START` instant for `clientRunId`. Gate deaths outside an active run per contract (`run_context_incomplete` / local hold). |
| `internal/detection/run.go` | existing — extend | Persistable run start instant, map ID, keystone level. |
| `internal/detection/death.go` | existing — extend | Victim GUID, death instant, ordinal inputs; no `category` other than default death semantics. |
| `internal/detection/damage.go` | existing — extend | Ranked causes; redact player-origin identity before any payload is stored for later upload. |
| `internal/detection/tracker_test.go` | existing — extend | Fail-closed tracking, ordinals, run gating, redaction. |
| `cmd/logprobe/main.go` | existing | Diagnostic entry. |
| `cmd/logprobe/run.go` | existing — extend | Do not make track-all the undocumented default for “production-like” capture. |
| `cmd/logprobe/deaths.go` | existing — extend | Stop printing unredacted `source_guid` / player identity in diagnostic paths that could become production logs. |
| `cmd/logprobe/run_test.go` | existing — extend | Privacy and filter regressions. |
| `testdata/logs/synthetic/` | existing — extend | Synthetic ID/ordinal/fail-closed/redaction slices. No real logs. |
| `testdata/logs/synthetic/README.md` | existing — extend | Provenance; keep anonymized. |

### Existing stubs to implement in Phase 2

These packages exist as `doc.go` only. Phase 2 fills them. Adding a SQLite driver and filesystem watcher is **Phase 2-approved** when implementing these packages (not WP5).

| Path | Status | Role in Phase 2 |
| ---- | ------ | --------------- |
| `internal/config/doc.go` | existing stub | Package comment. |
| `internal/config/config.go` | to create | Log path, persisted WoW-log timezone, tracked GUID list, installation diagnostics ID (not an ID-hash input, not a credential). **Fail closed** on absent/corrupt tracked configuration. |
| `internal/config/config_test.go` | to create | Empty/corrupt/missing tracked list; timezone required for timezone-less stamps. |
| `internal/logwatcher/doc.go` | existing stub | Package comment. |
| `internal/logwatcher/watcher.go` | to create | Directory/file identity, append, rotation, truncation, restart resume. Offsets are authoritative; watch notifications are a hint. |
| `internal/logwatcher/offset.go` | to create | Committed byte offset + incomplete trailing line, advanced only after persist. |
| `internal/logwatcher/watcher_test.go` | to create | Append, restart, truncate, rotate (synthetic files). |
| `internal/storage/doc.go` | existing stub | Package comment. Local SQLite is **not** a replica of Yeetcraft PostgreSQL. |
| `internal/storage/sqlite.go` | to create | Open/migrate local DB. Schema direction is in [`IMPLEMENTATION_PLAN.md` §6.3](./IMPLEMENTATION_PLAN.md#63-local-sqlite-schema) (planned; implement in this package). |
| `internal/storage/runs.go` | to create | Persist runs with `clientRunId` atomically with ID inputs. |
| `internal/storage/events.go` | to create | Persist events with `clientEventId` and ordinal; reuse persisted IDs on partial scan; never invent ordinals without a complete prefix. |
| `internal/storage/storage_test.go` | to create | Uniqueness, crash-before-ack, full rescan vs partial resume. |
| `internal/session/doc.go` | existing stub | Package comment. |
| `internal/session/session.go` | to create | Run state machine (idle/candidate/active/completing/completed/abandoned). |
| `internal/session/ids.go` | to create | Canonical `clientRunId` / `clientEventId` recipes from [`CONTRACT.md`](../yeetcraft/contracts/companion/v1/CONTRACT.md) (UTF-8 NFC, length-prefixed SHA-256). Season and `installationId` excluded. |
| `internal/session/ids_test.go` | to create | Hash vectors against frozen recipes; synthetic inputs only. Optionally compare derived fixtures — checksum strategy **deferred to Validate**. |
| `cmd/yeetcraft-companion/main.go` | existing stub — extend | Headless capture wiring: config (fail closed) → watcher → parser → detection → session IDs → storage. No HTTP upload. |

`.env.example` already mentions log dir, SQLite path, and a placeholder API key. Phase 2 may add tracked-GUID and timezone keys there; **do not** implement upload against `YEETCRAFT_API_KEY` in Phase 2. Never commit real keys or GUIDs.

### Existing stubs explicitly **not** Phase 2

| Path | Status | Phase | Role |
| ---- | ------ | ----- | ---- |
| `internal/uploader/doc.go` | existing stub — do not implement | **4** | Versioned HTTP client, `batchId`, retries, ack of ordered results. Reads canonical contract; must not become a schema editor. |
| `internal/review/doc.go` | existing stub — do not implement | **5** | Operator review UI/flow. Website remains classification authority ([Yeetcraft ADR 001](../yeetcraft/docs/adr/001-post-ingest-classification-and-corrections.md)). |

### Docs (Phase 2 may update; WP5 only maps them)

| Path | Status | Role |
| ---- | ------ | ---- |
| `docs/IMPLEMENTATION_PLAN.md` | existing | Roadmap. WP5 checkbox records **this map**, not Phase 2 completion. |
| `docs/CONTRACT_V1_WP1_REVIEW.md` | existing | Frozen producer review. |
| `docs/YEETCRAFT_INTEGRATION.md` | existing | Integration boundaries. |
| `docs/COMBAT_LOG_CAPABILITIES.md` | existing | Evidence; do not promote unverified capabilities. |
| `README.md` | existing | Status; Phase 2 later marks watcher/SQLite as implemented only after they exist. |
| `testdata/contract/v1/` | to create (optional, Phase 2 tests) | Derived copies of Yeetcraft examples only, with recorded checksum. Never independently edited. |

---

## Validate package (not started in WP5)

Machine-validation harness: **deferred to Validate**.

Validate should cover:

- [`PHASE_2_FILE_MAP.md`](./PHASE_2_FILE_MAP.md) (this file)
- [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md) (WP5 checkbox and map links only)
- [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md) (pointer to this map)
- [`YEETCRAFT_INTEGRATION.md`](./YEETCRAFT_INTEGRATION.md) (pointer)
- Relative links to `../yeetcraft/contracts/companion/v1/` (when the sibling checkout is present)
- Confirmation this repo does **not** contain a copied canonical `schema/` tree

Do not require `go test` / `go vet` for WP5 unless Go files are edited.

---

## Related documentation

| Document | Role |
| -------- | ---- |
| [`../yeetcraft/contracts/companion/v1/README.md`](../yeetcraft/contracts/companion/v1/README.md) | Canonical contract ownership |
| [`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../yeetcraft/contracts/companion/v1/CONTRACT.md) | Normative recipes and payloads |
| [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md) | Yeetcraft Phase 3 file map |
| [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md) | Current producers vs gaps |
| [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md) | Phased roadmap |
