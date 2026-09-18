# Companion Phase 2 implementation file map

> **Status: Phase 2 headless capture foundation implemented on `feat/phase-2/headless-capture` (WP 2.1–2.7). Ready for human review / PR to companion `dev`. Upload, review UI, and Wails remain deferred.**

| Field | Value |
| ----- | ----- |
| Work package | WP5 map; implemented as WP 2.1–2.7 on one branch |
| Owner | yeetcraft-companion |
| Phase mapped here | **Phase 2** (headless capture foundation) |
| Canonical contract | [`../yeetcraft/contracts/companion/v1/`](../../yeetcraft/contracts/companion/v1/README.md) (Yeetcraft-owned) |
| Yeetcraft Phase 3 map | [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md) |

This document is a **file map**, not a new protocol and not a second contract.

**Do not copy** Yeetcraft `contracts/companion/v1/` schemas, examples, or `CONTRACT.md` into this repository as a source of truth. Phase 2 tests may later hold **mechanically verified derived fixtures** under `testdata/` with a recorded canonical checksum. Checksum/drift harness: **deferred to Validate**.

WP1 producer gaps remain in [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md).

---

## Phase 2 acceptance evidence (WP 2.7)

Recorded in [`IMPLEMENTATION_PLAN.md` §8.4](./IMPLEMENTATION_PLAN.md#84-phase-2--headless-companion-foundation).

| Check | Location |
| ----- | -------- |
| Missing tracked config exits non-zero; writes nothing | `cmd/yeetcraft-companion/main_test.go` |
| Synthetic log → expected event count and contract IDs | `cmd/yeetcraft-companion/main_test.go` |
| Restart mid-log → same IDs, no duplicates | `cmd/yeetcraft-companion/main_test.go` |
| Truncation / rotation persist-once | `cmd/yeetcraft-companion/main_test.go`, `internal/logwatcher/watcher_test.go` |
| Offset + events in one transaction; crash-before-ack | `internal/storage/storage_test.go` |
| Fail-closed production config | `internal/config/config_test.go`, `cmd/yeetcraft-companion/main_test.go` |
| Run state machine (abandon on close/timeout, never invent completion) | `internal/session/session_test.go` |
| CI on Linux + Windows | `.github/workflows/go.yml` |

---

## Locked production behavior (Phase 2)

Production capture **fails closed** when tracked-character configuration is absent or corrupt.

| Path | Production rule | Status |
| ---- | --------------- | ------ |
| `internal/config` | Missing, empty, or corrupt tracked GUID list → do not start capture | **Implemented** |
| `cmd/yeetcraft-companion` | Fail closed; never fall back to track-all | **Implemented** |
| `internal/detection` | Production callers must not use empty-list `allPlayers` | **Implemented** (`NewTracker` errors; `NewDiagnosticTracker` only for logprobe) |
| `cmd/logprobe` | Diagnostic CLI only. Track-all remains an **explicit diagnostic** mode | **Implemented** (`--track-all-diagnostic`) |

Untracked names, realms, and GUIDs must not be prepared for later upload. Cause evidence is redacted locally before persistence.

---

## Sequencing

| Dependency | Owner | Notes |
| ---------- | ----- | ----- |
| Frozen v1 wire contract | Yeetcraft | Read [`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../../yeetcraft/contracts/companion/v1/CONTRACT.md). Do not fork it. |
| Nullable unique `characters.guid` | Yeetcraft character slice | **End-to-end ingest** blocker. Phase 2 persists local GUIDs and IDs without calling the server. |
| `dungeons.challenge_map_id`, `seasons.starts_at`/`ends_at` | Yeetcraft Phase 3 schema | Server resolution. Companion still sends challenge map ID and optional season hint. |
| HTTP uploader | **Phase 4** (`internal/uploader`) | Not Phase 2 — stub only |
| Review UI | **Phase 5** (`internal/review`) | Not Phase 2 — stub only. Local hold-for-review **state** lives in `internal/storage` / `internal/session`. |
| Wails / addon | Phase 6 / 8 | Not Phase 2. Phase 2 is **CLI-first / headless**. |

Phase 2 exit is local: restart/truncation/rotation tests pass; events persist exactly once; IDs follow the canonical recipes; production tracking fails closed.

---

## Phase 2 file map

**Status:** `existing` = present and used in Phase 2 capture; `stub` = package comment only (deferred phase); `extend` = Phase 0 code extended in Phase 2.

### Parser, detection, and logprobe

| Path | Status | Role in Phase 2 |
| ---- | ------ | --------------- |
| `internal/parser/*` | existing — extend | Streaming V22 parser; canonical instants (`ResolveCanonicalInstant`); resumable `ScanReaderFrom`; `state_json.go` for SQLite resume |
| `internal/detection/*` | existing — extend | Fail-closed tracking, run gating, ordinals, cause redaction, `RestoreRun` for restart |
| `cmd/logprobe/*` | existing — extend | Diagnostic CLI; explicit `--track-all-diagnostic`; redacted death output |

### Phase 2 capture packages (implemented)

| Path | Status | Role in Phase 2 |
| ---- | ------ | --------------- |
| `internal/config/config.go` | **implemented** | Tracked GUIDs, WoW-log timezone, installation diagnostics ID; fail closed |
| `internal/logwatcher/*` | **implemented** | Offsets, identity, truncation, rotation, restart resume |
| `internal/storage/*` | **implemented** | SQLite open/migrate, runs/events/causes/files/settings, atomic commit |
| `internal/session/ids.go` | **implemented** | Contract `clientRunId` / `clientEventId` recipes |
| `internal/session/session.go` | **implemented** | Run state machine: idle, candidate, active, completing, completed, abandoned |
| `cmd/yeetcraft-companion/main.go` | **implemented** | Headless wiring: config → storage → logwatcher → parser → detection → session → storage |
| `cmd/yeetcraft-companion/capture.go` | **implemented** | Capture loop, `--once` test mode, graceful abandon on interrupt |
| `cmd/yeetcraft-companion/main_test.go` | **implemented** | End-to-end fail-closed, IDs, restart, truncation, rotation |

### Explicitly **not** Phase 2 (stubs remain)

| Path | Status | Phase | Role |
| ---- | ------ | ----- | ---- |
| `internal/uploader/doc.go` | stub — do not implement | **4** | Versioned HTTP client, `batchId`, retries |
| `internal/review/doc.go` | stub — do not implement | **5** | Operator review UI/flow |

`.env.example` documents tracked GUIDs, timezone, and optional paths. **Do not** implement upload against `YEETCRAFT_API_KEY` in Phase 2.

---

## Validate package

Recorded in Phase 1 Validate. Canonical checksum strategy:
[`CONTRACT_V1_DERIVED_FIXTURES.md`](./CONTRACT_V1_DERIVED_FIXTURES.md).

Relative-link and no-schema-fork checks: [`scripts/verify-canonical-checksums.ps1`](../scripts/verify-canonical-checksums.ps1).

**Deferred to Phase 2/3 CI:** `go test` materializing derived JSON; requiring a Yeetcraft checkout in companion-only CI.

---

## Related documentation

| Document | Role |
| -------- | ---- |
| [`../yeetcraft/contracts/companion/v1/README.md`](../../yeetcraft/contracts/companion/v1/README.md) | Canonical contract ownership |
| [`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../../yeetcraft/contracts/companion/v1/CONTRACT.md) | Normative recipes and payloads |
| [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md) | Yeetcraft Phase 3 file map |
| [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md) | Producer review (Phase 2 gaps closed where noted in §8.4) |
| [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md) | Phased roadmap |
