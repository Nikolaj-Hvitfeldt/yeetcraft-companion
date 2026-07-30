# Yeetcraft Companion — Technical Implementation Plan

| Field | Value |
| ----- | ----- |
| **Status** | Planning |
| **Current milestone** | Phase 0A.1 complete / Phase 0A.2 pending |
| **Canonical repository** | [yeetcraft-companion](https://github.com/Nikolaj-Hvitfeldt/yeetcraft-companion) |
| **Related repository** | [yeetcraft](https://github.com/Nikolaj-Hvitfeldt/Yeetcraft) (website, backend, PostgreSQL, API, canonical companion contract) |
| **Last updated** | 2026-07-30 |

Automatic capture, local processing, and reliable upload of Mythic+ death events.

| Field | Value |
| ----- | ----- |
| Primary focus | Windows companion application and Yeetcraft integration |
| Existing system | React/Vite PWA, Go API, PostgreSQL/Supabase, Vercel + Render |
| Initial users | Four configured tracked characters (fixed friend-group MVP) |
| Addon | Optional later extension; **not** a dependency for the companion MVP |
| Document status | Implementation-ready plan; technical unknowns isolated in Phase 0 |

**Recommended direction:** Build a headless Go proof of concept against real Midnight Mythic+ combat logs before changing the database or building a Wails UI. One player should be able to collect the party's visible deaths; all uploads must be idempotent and survive offline use and Render cold starts.

---

## Executive summary

The companion app becomes a local bridge between World of Warcraft and Yeetcraft. It follows the player's combat-log file, converts relevant lines into structured runs and death events, stores those events locally, and uploads them to the existing Go backend. The website continues to display the current aggregated statistics, while the new event layer creates an audit trail and enables richer features later.

The project should not begin with a polished desktop shell. Its primary uncertainty is **data quality**: which events are visible to one logging client, how reliably a Mythic+ run can be bounded, how a death cause can be inferred, and when a “yeet” can be distinguished from a normal death. **Phase 0** therefore produces evidence from real logs and a written capability matrix. Every later phase depends on that result.

The safest migration is **additive**. Existing public GET routes, manual editing, token handling, offline frontend behavior, and player/dungeon statistics remain functional. A separate versioned ingest endpoint accepts immutable events. The server inserts them idempotently and updates the existing aggregate table within the same transaction. The companion and Yeetcraft remain **separate Git repositories** and communicate only through this versioned HTTP contract.

---

## 1. Scope, goals, and non-goals

### 1.1 Goals

- Remove repeated manual entry for ordinary party deaths.
- Require the companion application on only **one** player's Windows PC.
- Record enough context to explain who died, approximately how, to what source, in which dungeon and run, and when.
- Keep raw combat logs local; upload only structured, minimal events.
- Work while the network or Render backend is unavailable and retry without duplicates.
- Preserve the current Yeetcraft UI, public read access, and manual correction workflow.
- Create an event foundation for future run history, achievements, nemesis/worst-dungeon insights, and better death explanations.

### 1.2 MVP

- Windows-only headless companion core, later wrapped in a minimal Wails UI.
- Automatic discovery or selection of `WoWCombatLog.txt`.
- Live tailing plus resumable reading after restart.
- Detection of configured Yeetcraft players' `UNIT_DIED`-style events and a bounded recent-damage context.
- Local SQLite event store and upload outbox.
- Versioned batch ingest with stable IDs and server-side duplicate protection.
- Normal deaths recorded automatically; uncertain classifications enter review.

### 1.3 Explicitly deferred

- WoW addon implementation.
- Perfect automatic yeet detection.
- macOS support.
- Automatic application updates.
- Uploading or permanently retaining raw combat logs.
- Full combat replay or a Warcraft Logs replacement.
- Replacing the current frontend, authentication model, or manual editing experience.

---

## 2. Existing Yeetcraft constraints

The **Yeetcraft repository** remains the server-side system of record. Its known architecture is a React/Vite PWA frontend, a Go backend using pgx, PostgreSQL/Supabase, Vercel hosting for the frontend, and a sleeping Render free-tier backend. Current writes are centered on `PATCH /api/stats/batch` and `X-API-Key` authentication, with Bearer accepted as an alternate. The frontend's `?token=` mechanism is a client-side unlock flow, not backend query authentication.

The new companion application lives in its own sibling repository and has **no runtime filesystem or code dependency** on Yeetcraft.

| Area | Current behavior to preserve | Companion impact |
| ---- | ---------------------------- | ---------------- |
| Frontend | Public reads, themes, persisted query cache, write outbox | No redesign required for initial ingest |
| Backend | Go handlers/repository; fail-closed write auth | Add versioned ingest route and repository transaction |
| Database | Season/player/dungeon aggregate stats | Add runs/events and mapping; retain aggregates |
| Hosting | Render may sleep | Persistent local queue, timeouts, and backoff |
| Users | Fixed friend group | Simple explicit identity mapping is acceptable |
| Tests | Go/Vitest/Playwright and guarded `_test` DB | Extend the same safety model |

**Source review conclusion:** The current repository is deliberately aggregate-first. Characters and roles are hardcoded in `frontend/src/data/player-characters.ts`, while `backend/db/schema.sql` contains only players, seasons, dungeons, season membership, and `player_dungeon_stats`. The companion project is therefore the right point to introduce an event-oriented domain model instead of extending hardcoded registries.

### 2.1 Concrete findings from the repository

| Finding | Evidence | Consequence |
| ------- | -------- | ----------- |
| Players are people, not WoW characters | `players` has `display_name`/`avatar_url`; profile data is keyed by configured tracked identity slugs in the sibling frontend registry | Keep players as owner identity; add characters |
| Characters and roles are frontend-only | `frontend/src/data/player-characters.ts` | Move canonical character identity/class to PostgreSQL |
| All statistics are mutable aggregates | `player_dungeon_stats` and `SetStatsBatch` | No run, death, or cause audit trail exists |
| Backend has one stats repository | `handler.StatsRepository` and `repository/stats.go` | Split new domain repositories/services rather than growing `stats.go` indefinitely |
| Routes are read-heavy and public | `backend/cmd/server/main.go` | Introduce v2 reads gradually; preserve v1 during migration |
| Offline browser writes already exist | frontend outbox and `docs/OFFLINE.md` | Do not reuse browser outbox for companion delivery |
| Achievements are aggregate rules | `frontend/src/utils/dungeon-achievements.ts` | Event facts can later support boss/run-specific achievements |
| Player nemesis means dungeon today | `frontend/src/utils/player-stats.ts` | Boss nemesis should be a distinct event-derived insight |

*Evidence paths refer to the sibling `./yeetcraft` repository (read-only development reference).*

---

## 3. Target architecture

**Figure 1 — Recommended dataflow.** SQLite separates local capture from unreliable network delivery.

```text
WoW client → combat-log file → Companion (parser, detection, SQLite)
                                      ↓ HTTPS (versioned ingest)
                               Yeetcraft Go API → PostgreSQL
                                      ↓ public GET
                               React PWA
```

### 3.1 Component responsibilities

| Component | Owns | Must not own |
| --------- | ---- | ------------ |
| WoW client | Writing combat-log lines | HTTP upload |
| Companion parser | Syntax parsing and normalization | Database aggregate mutations |
| Detection engine | Runs, deaths, evidence, and confidence | Irreversible classification guesses |
| SQLite | Local events, offsets, configuration, upload state | Website authority |
| Uploader | Batching, authentication, retry, and acknowledgements | Deleting unconfirmed events |
| Go ingest API | Validation, idempotency, and transactional persistence | Raw-log storage |
| PostgreSQL | Canonical accepted events and derived aggregates | Client-local file offsets |
| React frontend | Viewing and correcting accepted data | Combat-log parsing |

### 3.2 Required two-repository layout

**Yeetcraft** (`./yeetcraft`):

```text
yeetcraft/
├── backend/
├── frontend/
├── contracts/
│   └── companion/v1/          # planned; canonical schema (not implemented yet)
└── docs/
```

**Companion** (`./yeetcraft-companion` — this repository):

```text
yeetcraft-companion/
├── AGENTS.md
├── docs/
│   ├── IMPLEMENTATION_PLAN.md
│   ├── YEETCRAFT_INTEGRATION.md
│   ├── COMBAT_LOG_FORMAT_V22.md
│   └── COMBAT_LOG_CAPABILITIES.md
├── cmd/yeetcraft-companion/
├── internal/
│   ├── config/
│   ├── logwatcher/
│   ├── parser/
│   ├── session/               # run/session boundary detection
│   ├── detection/             # recent-damage and death inference (Phase 0+)
│   ├── storage/
│   ├── uploader/
│   └── review/
└── testdata/logs/
```

The repositories are **independent products** with separate Git histories and releases.

| Repository | Owns |
| ---------- | ---- |
| **yeetcraft** | PostgreSQL, API behavior, website, canonical versioned contract |
| **yeetcraft-companion** | Capture, parsing, local persistence, upload behavior, review flows, Windows packaging, future Wails UI |

They must **not** use cross-repository Go imports, shared runtime files, or direct database access.

### 3.3 Local multi-root development workspace

```text
projects/
├── yeetcraft/
├── yeetcraft-companion/
└── yeetcraft.code-workspace
```

- Open both sibling repositories in one Cursor multi-root workspace so agents can inspect both.
- Companion tasks may read `../yeetcraft`, but must **not** modify it unless the task explicitly includes Yeetcraft-side changes.
- Cross-repository features require **separate file lists, validation commands, and commits** for each repository.
- The installed companion executable communicates only with the deployed/local HTTP API; it never needs the Yeetcraft repository on the user's PC.

### 3.4 Contract ownership

- Place the canonical v1 request/response schema and examples in the **Yeetcraft repository** at `./yeetcraft/contracts/companion/v1/` (**planned — not implemented yet**).
- During development, companion contract tests may read `../yeetcraft/contracts/companion/v1/` from the sibling checkout.
- The compiled companion supports an explicit schema version and must also be testable **without** the sibling repository by using its own request fixtures.
- **Do not maintain two independently edited canonical schemas.** Generated code or fixtures in this repository are derived copies, not source of truth.

---

## 4. Core data model

### 4.1 Event principles

- Events are append-oriented facts; classification and review status may change.
- Every client-created event has a deterministic idempotency key.
- The server assigns its own primary key and retains the client event ID as a unique key.
- Evidence is stored as structured summaries, **not** raw log lines.
- Aggregates are derived or transactionally maintained from accepted classifications.
- Corrections must be auditable and must **not** create a second death.

### 4.2 Recommended domain overhaul

The target schema should treat Yeetcraft as a small run/event system. Aggregated leaderboards remain important outputs, but they should no longer be the only source data for new activity.

| Table | Purpose | Important fields |
| ----- | ------- | ---------------- |
| `players` | Real people/profile owners | `id`, `display_name`, `slug`, `avatar_url`, `active` |
| `characters` | WoW characters owned by a player | `id`, `player_id`, `guid` UNIQUE, `name`, `realm`, `region`, `class_id`, `active` |
| `character_roles` | Optional many-role presentation data | `character_id`, `role` |
| `seasons` | Existing season boundary | `id`, `name`, `expansion`, `is_current` |
| `dungeons` | Canonical dungeon | `id`, `game_instance_id`, `name`, `short_name` |
| `season_dungeons` | Existing seasonal pool | `season_id`, `dungeon_id`, `display_order` |
| `encounters` | Boss encounter belonging to a dungeon | `id`, `dungeon_id`, `journal_encounter_id`, `name`, `display_order` |
| `runs` | One Mythic+ attempt | `id`, `client_run_id` UNIQUE, `season_id`, `dungeon_id`, `key_level`, `started_at`, `ended_at`, `status` |
| `run_participants` | Characters/players present in a run | `run_id`, `character_id`, `player_id` snapshot, `role`, `joined_at` |
| `death_events` | One observable player death | `id`, `client_event_id` UNIQUE, `run_id`, `character_id`, `encounter_id` NULL, `category`, `confidence` |
| `death_causes` | Ranked likely cause evidence | `death_event_id`, `rank`, `source_type`, `source_game_id`, `source_name`, `spell_id`/`name`, `overkill` |
| `stat_adjustments` | Manual corrections and migrated historic baseline | `player_id`, `season_id`, `dungeon_id`, `deaths_delta`, `yeets_delta`, `reason` |
| `companion_clients` | Revocable installation identity | `id`, `label`, `credential_hash`, `version`, `last_seen_at`, `revoked_at` |
| `ingest_batches` | Minimal sync diagnostics | `id`, `client_id`, `batch_id` UNIQUE, counts, `received_at` |

*Schema decisions above are **target design** for Yeetcraft-side work (Phase 3+). They are not implemented in either repository yet.*

### 4.3 Why boss and cause are separate

A player's “boss nemesis” is the boss encounter active when the player died. The literal source of the final damage can instead be the boss, an add, a summoned object, a damage-over-time effect, or the environment. Store both `encounter_id` on `death_events` and ranked source evidence in `death_causes`.

| Question | Data used |
| -------- | --------- |
| Which boss kills this player most often? | `death_events.character_id`/player owner + `encounter_id` |
| Which ability causes most deaths? | `death_causes.spell_id`, using rank 1 or confirmed cause |
| Is the dungeon dangerous outside bosses? | `death_events` where `encounter_id` is null |
| Did the boss or an add deliver the final hit? | `death_causes.source_type` and `source_game_id` |
| Which character dies most? | `death_events.character_id` |
| Which person dies most across alts? | `characters.player_id` |

### 4.4 Category and review model

| Category | Meaning | Aggregate effect | Review |
| -------- | ------- | ---------------- | ------ |
| `death` | Confirmed ordinary death | +1 death | No, unless low confidence |
| `possible_yeet` | Evidence suggests displacement/environment | Temporarily +1 death or excluded by policy | Yes |
| `yeet` | Confirmed yeet | +1 yeet | No |
| `unknown` | Death is real; cause/classification unresolved | +1 death by conservative policy | Yes |
| `ignored` | Duplicate, non-party, or invalid event | None | Completed |

**Recommended accounting rule:** A real but uncertain death should count conservatively as a normal death until it is reclassified. Reclassification moves one count atomically from deaths to yeets; it must **never** increase total deaths.

### 4.5 Stable IDs

Do not base identity only on player name, log line number, or a random UUID generated each time a file is reread. A restart, file copy, or retry would create duplicates.

```text
client_run_id =
  SHA-256(client_installation_id | log_file_identity | run_start_timestamp |
          instance_identifier | sorted_party_guids)

client_event_id =
  SHA-256(client_run_id | player_guid | death_timestamp |
          normalized_event_type | sequence_disambiguator)
```

The exact run recipe is **finalized after Phase 0** reveals available metadata. Persist generated IDs in SQLite immediately. The server's UNIQUE constraints are the final deduplication boundary.

---

## 5. Versioned ingest API

*Endpoint and schema below are **planned** in this document. They are not implemented in Yeetcraft until Phase 3. Do not treat them as verified existing routes.*

### 5.1 Endpoint

```http
POST /api/ingest/v1/deaths/batch
X-API-Key: <companion credential>
Content-Type: application/json
Idempotency-Key: <batch id>
```

Using a dedicated endpoint avoids forcing event semantics into `PATCH /api/stats/batch`. Manual aggregate edits and automatic event ingestion have different validation, deduplication, and audit requirements.

### 5.2 Request shape

```json
{
  "schemaVersion": 1,
  "client": {"installationId": "uuid", "version": "0.1.0"},
  "batchId": "uuid",
  "events": [{
    "eventId": "sha256:...",
    "runId": "sha256:...",
    "occurredAt": "2026-07-30T19:43:12.123Z",
    "player": {"guid": "Player-9999-00000001", "name": "TrackedAlpha", "realm": "SyntheticRealm"},
    "dungeon": {"gameId": 0, "name": "Magisters' Terrace", "keyLevel": 12},
    "classification": {"category": "death", "confidence": 0.94},
    "cause": {
      "sourceGuid": "Creature-...",
      "sourceName": "Boss",
      "spellId": 123456,
      "spellName": "Ability",
      "overkill": 245000
    }
  }]
}
```

### 5.3 Response and retry semantics

| Outcome | HTTP | Companion action |
| ------- | ---- | ---------------- |
| All accepted or duplicate | 200 | Mark each acknowledged event uploaded |
| Mixed valid/invalid | 200/207 (design choice) | Acknowledge per event; quarantine permanent rejects |
| Malformed/version unsupported | 400/422 | Do not retry unchanged; show actionable error |
| Credential invalid | 401/403 | Pause uploader; request configuration |
| Conflict requiring mapping | 409 | Move event to `needs_review` |
| Rate limited | 429 | Respect `Retry-After` |
| Render cold start/server failure | 502/503/504 or timeout | Keep pending; exponential backoff |

The API transaction should validate identity mappings, upsert the run, insert new death events with `ON CONFLICT DO NOTHING`, recompute or adjust affected aggregate rows, and commit. If aggregate update fails, the event insert must roll back. Duplicate events return an acknowledgement rather than an error.

---

## 6. Companion application design

### 6.1 Technology choice

| Choice | Recommendation | Reason |
| ------ | -------------- | ------ |
| Core language | Go | Matches backend skills; strong file/HTTP/concurrency support; simple Windows binary |
| File watching | fsnotify plus polling fallback | Watch notifications can coalesce or be missed; offset reads remain authoritative |
| Local database | SQLite | Durable transactions, unique keys, and queryable outbox |
| Desktop shell | Wails after headless core | Native Windows packaging while reusing Go core |
| Configuration | Local app-data directory | Keeps state outside install folder and survives upgrades |
| Logging | Structured rotating local logs | Diagnosable without uploading sensitive raw combat data |

*Third-party dependencies (fsnotify, SQLite driver, Wails) are approved per phase — not part of repository bootstrap.*

### 6.2 Module boundaries

| Module | Responsibility | Principal tests |
| ------ | -------------- | ----------------- |
| `config` | WoW path, API URL, credential reference, player mappings | defaults, validation, migration |
| `logwatcher` | File identity, offsets, rotation/truncation, and new bytes | append, restart, truncate, rotate |
| `parser` | Turn a line into a normalized combat event | fixtures and malformed lines |
| `session` | Detect log session and active run boundaries | start/end/reload/crash |
| `detection` | Track recent damage; emit deaths and confidence | single death, simultaneous deaths, wipe |
| `storage` | SQLite schema, transaction, and migrations | migration, uniqueness, recovery |
| `uploader` | Batch, authentication, retry, and acknowledgement | timeouts, partial response, duplicate |
| `review` | Expose uncertain/rejected events | classification transitions |

### 6.3 Local SQLite schema

```sql
files(id, path, file_identity, byte_offset, partial_line, updated_at)
runs(id, client_run_id UNIQUE, status, metadata_json, started_at, ended_at)
events(id, client_event_id UNIQUE, run_id, player_guid, occurred_at,
       category, confidence, payload_json, review_status)
uploads(id, event_id UNIQUE, state, attempts, next_attempt_at,
        last_error_code, last_error_message, acknowledged_at)
settings(key PRIMARY KEY, value, updated_at)
schema_migrations(version PRIMARY KEY, applied_at)
```

The parser and uploader communicate through persisted state, not only in-memory channels. A crash after event creation but before upload therefore loses nothing. A crash after server acceptance but before local acknowledgement produces a duplicate request that the server safely acknowledges.

### 6.4 Tracked and untracked roster boundary

A Mythic+ group contains five players. Yeetcraft initially has four configured
tracked characters; the fifth player is normally an untracked party member.

The low-level parser remains neutral: it parses source and destination fields
without deciding whether an identity belongs to Yeetcraft. Roster filtering
happens after normalized parsing.

- Deaths for an untracked party member do not become Yeetcraft statistics.
- The companion must not create a Yeetcraft player automatically.
- Names, realms, and GUIDs for untracked party members must not be uploaded.
- An untracked member may have an anonymized local representation only when
  required for run context.
- Final GUID-to-tracked-character mapping remains an open design decision.

```text
combat-log line → neutral parser → normalized event → roster filter
                                                   ├─ tracked candidate
                                                   └─ anonymized local context
```

### 6.5 File-reading algorithm

1. Resolve and fingerprint the selected combat-log file without hashing the entire growing file.
2. Load the last committed byte offset and any incomplete trailing line.
3. Read appended bytes in bounded chunks and split only on complete line endings.
4. Parse and persist normalized events in a transaction.
5. Advance the byte offset only after the corresponding parser state is safely persisted.
6. Detect truncation when file size becomes smaller than the stored offset; create a new file generation.
7. Detect replacement/rotation using file identity and a small prefix fingerprint.
8. On restart, resume from the committed offset and rely on unique client event IDs for final protection.

### 6.6 Death-cause inference

Maintain a short bounded ring buffer of recent relevant damage per tracked player. When a death event arrives, snapshot the buffer and rank likely causes. The last damage event is useful but not automatically the semantic cause; periodic damage, absorbs, environmental effects, and delayed mechanics can complicate the result.

| Confidence | Example basis | Behavior |
| ---------- | ------------- | -------- |
| High | Clear lethal damage/overkill immediately before death | Auto-accept normal death and cause |
| Medium | Several plausible recent hits or delayed effect | Accept death; mark cause as likely |
| Low | No visible lethal source, environmental/fall gap, or incomplete context | Accept death; queue classification review |

### 6.7 Run detection

Run detection is a state machine, not a single regex. Phase 0 must determine which combat-log markers and instance metadata are present. The companion should support explicit states: **idle**, **candidate**, **active**, **completing**, **completed**, and **abandoned**. A timeout or process exit should close a run as interrupted rather than inventing a completion.

| State | Entry signal | Exit signal | Recovery |
| ----- | ------------ | ----------- | -------- |
| Idle | No candidate activity | Dungeon/run marker | None |
| Candidate | Relevant party combat in expected instance | Strong start signal or timeout | Discard/mark uncertain |
| Active | Confirmed run | Completion, zone/session change, or inactivity | Persist across restart |
| Completed | Completion marker | Upload finished | Immutable boundary |
| Abandoned | Timeout/session close without completion | Manual merge/review | Keep events |

---

## 7. Reliability and operational behavior

### 7.1 Upload state machine

| State | Meaning | Allowed transition |
| ----- | ------- | ------------------ |
| `pending` | Never attempted or retry due | `uploading` |
| `uploading` | Reserved by current worker | `uploaded`, `pending`, `failed`, `needs_review` |
| `uploaded` | Server acknowledgement stored | No automatic transition |
| `failed` | Permanent local/schema/configuration problem | `pending` after correction |
| `needs_review` | Mapping or classification requires a person | `pending` or `ignored` |
| `ignored` | Explicitly excluded | `pending` only by manual restore |

### 7.2 Render cold-start strategy

- Treat timeout, connection reset, and 5xx as temporary.
- Use small batches and a request timeout long enough to allow a cold start, but never block capture.
- Exponential backoff with jitter, capped at a reasonable interval.
- Allow manual “retry now” without resetting attempt history.
- Show pending count and last successful sync; never show “all synced” from in-memory state alone.
- Do not delete accepted local events immediately; retain a bounded history for diagnostics and review.

**Failure invariant:** A network failure may delay statistics, but it must **not** lose or duplicate a death. Capture, persistence, and delivery are separate stages.

### 7.3 Authentication

For a four-person hobby project, a dedicated companion API key is sufficient initially. It should be separate from the browser editing token so it can be revoked independently. Store it using the Windows credential manager when the Wails shell is introduced; during the headless spike, allow an environment variable or local development config excluded from Git. Never include it in logs, crash reports, URLs, or response messages.

### 7.4 Privacy and data minimization

- Raw combat logs remain on the user's PC.
- Persist only tracked party members and evidence needed for the death explanation.
- Do not upload chat, unrelated players, or full damage timelines.
- Provide a local retention policy and a clear “clear diagnostic history” action.
- Log identifiers and error codes, not credentials or complete payloads.

---

## 8. Phased implementation roadmap

| Phase | Purpose | Primary owner | Exit criterion |
| ----- | ------- | ------------- | -------------- |
| **Pre-Phase 0** — Repository bootstrap | Standalone repo and development boundaries | **Companion** | Companion repo starts/tests cleanly; Yeetcraft readable as sibling and unmodified |
| **Phase 0** — Combat-log evidence spike | Prove visibility, parsing, death cause, and run signals | **Companion** | All four party deaths in test runs accounted for; uncertainties documented; **no server writes** |
| **Phase 1** — Repository integration and contracts | Validate code paths; define v1 contracts before migrations | **Both** (separate tasks) | Contract review completed; existing API and frontend constraints confirmed |
| **Phase 2** — Headless companion foundation | Reliable tailing, offsets, parser, SQLite | **Companion** | Restart/truncation/rotation tests pass; events persist exactly once |
| **Phase 3** — Backend event model and ingest | Additive schema and transactional idempotent ingest | **Yeetcraft** | Duplicate batches do not change totals; rollback preserves consistency |
| **Phase 4** — End-to-end upload queue | Local events to backend with retries | **Companion** (+ Yeetcraft deploy) | Offline/cold-start/restart scenarios deliver once without data loss |
| **Phase 5** — Classification and review | Confidence, mappings, correction transitions | **Both** | Death→yeet changes preserve total; ambiguous events cannot silently disappear |
| **Phase 6** — Minimal Wails desktop app | Package the proven core for Windows use | **Companion** | Clean-machine test succeeds without developer tooling |
| **Phase 7** — Website enrichment | Event details without destabilizing leaderboards | **Yeetcraft** | Current routes remain stable; new screens handle empty/error/loading states |
| **Phase 8** — Optional addon discovery | Evaluate addon only for proven gaps | **Companion** (+ optional addon repo) | Clear benefit exceeds installation/maintenance cost |

### 8.1 Pre-Phase 0 — Repository bootstrap

**Status: complete** (2026-07-30)

Deliverables:

- [x] Standalone `yeetcraft-companion` Git repository beside existing Yeetcraft repository
- [x] [AGENTS.md](../AGENTS.md) with repository ownership, sibling-reference, and no-cross-repository-import rules
- [x] This plan in `docs/IMPLEMENTATION_PLAN.md` (architectural direction, not proof of combat-log capabilities)
- [x] [docs/YEETCRAFT_INTEGRATION.md](./YEETCRAFT_INTEGRATION.md) — API boundary, authentication, schema versioning
- [x] Minimal Go module, `cmd/`, and `internal/` directories without parser implementation
- [x] `.gitignore` for raw logs, SQLite, credentials, build output, and diagnostics
- [x] README with prerequisites, scope, non-goals, and build/test commands
- [x] Multi-root workspace guidance (documented; machine-specific paths not committed)
- [x] Formatting and minimal Go test proving bootstrap health

**Stop condition:** None — proceed to Phase 0 when approved.

### 8.2 Phase 0 — Combat-log evidence spike (bounded PoC)

**Status: Phase 0A.1 complete; Phase 0A.2 pending; Phase 0 is not complete**

| Sub-phase | Scope | Maximum evidence status |
| --------- | ----- | ----------------------- |
| **0A.1** | Current-format research and original synthetic fixture preparation | Synthetic fixture prepared |
| **0B (limited)** | Source-backed parser and resilience work that does not require a real log | Synthetically tested (technical sub-capabilities only) |
| **0A.2** | Real retail 12.0+ Mythic+ log validation; adjusts Phase 0B assumptions | Partially verified / verified with real log |
| **0B (full)** | Death detection, run/encounter inference, and cause accuracy after 0A.2 | Synthetically tested / partially verified |

Phase 0A.1 produces the
[V22 format reference](./COMBAT_LOG_FORMAT_V22.md) and
[synthetic fixture corpus](../testdata/logs/synthetic/README.md). Synthetic
fixtures establish test inputs, not real-world visibility or semantics.

Phase 0B is splittable. Because a real Mythic+ log is not currently available,
a **limited Phase 0B** may follow Phase 0A.1 without waiting for Phase 0A.2.
Phase 0A.2 later validates and adjusts Phase 0B; it does not block all Phase 0B
work.

**Limited Phase 0B may implement and test:**

- CSV-aware tokenization;
- version-header CSV payload parsing;
- unsupported-version handling;
- common-header extraction;
- exact source-backed damage event payloads (`SPELL_DAMAGE`, `SWING_DAMAGE`,
  `ENVIRONMENTAL_DAMAGE`);
- advanced-block extraction where the layout is exact in the selected V22
  reference;
- unknown and malformed input handling;
- partial-line buffering.

A technical sub-capability may reach **Synthetically tested** in limited Phase
0B only when an implementation passes an exact source-backed fixture. Shape-
incomplete death scenarios must not be promoted to passing success fixtures.

**Blocked until Phase 0A.2 provides a real log:**

- final raw timestamp-envelope compatibility;
- exact V22 `UNIT_DIED` layout;
- real party visibility;
- real event ordering;
- production-quality death detection;
- run and encounter reliability;
- death-cause accuracy;
- yeet classification.

Phase 0B design for advanced-block semantics remains open where the selected
reference and observed V22 samples conflict. Do not infer suffix positions from
approximate field counts.

Deliverables:

- [ ] CLI probe (e.g. `cmd/logprobe`) accepting a combat-log path and optional tracked names/GUIDs
- [ ] Streaming parser reporting recognized, unknown, and malformed event counts
- [ ] Recent-damage buffers and death candidate output in `internal/detection/`
- [ ] Anonymized fixture slices under [testdata/logs/](../testdata/logs/)
- [ ] Capability matrix in [docs/COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md)
- [ ] Test report: per-run expected vs detected deaths and likely-cause accuracy

Phase 0A.1 deliverables:

- [x] Current V22 format research, sources, and conflicts documented
- [x] Original synthetic fixtures prepared and provenance recorded
- [x] Capability vocabulary separates prepared fixtures from passing tests
- [ ] Timestamp envelope and exact `UNIT_DIED` V22 suffix confirmed from a real
  retail 12.0+ log (deferred to Phase 0A.2)

Tasks:

- [ ] Enable advanced combat logging on one client and collect multiple representative Mythic+ runs (ordinary deaths, wipe, environmental death, fall/void-like death, interrupted run)
- [ ] Verify whether the logger sees deaths and preceding damage for all four configured tracked characters when each is in range
- [ ] Document available instance, difficulty, key-level, encounter, party, and completion markers
- [ ] Measure ambiguity: missed deaths, wrong likely cause, duplicate candidates, uncertain run boundaries
- [ ] Finalize capability matrix; decide which metadata is automatic, heuristic, or manual

**Acceptance criteria:**

- All four party deaths in test runs are accounted for (or gaps are explicitly documented).
- Uncertainties recorded in `COMBAT_LOG_CAPABILITIES.md`; no verified capabilities claimed without fixtures.
- **No server writes** and no Yeetcraft schema changes.

**Stop condition:** If one logger cannot reliably observe the group's deaths, **pause before backend work**. Re-evaluate whether multiple companion clients, an addon-assisted model, or manual import is necessary.

### 8.3 Phase 1 — Repository integration and contracts

**Owner: cross-repository task (separate commits per repo)**

Deliverables:

- [ ] Architecture decision record
- [ ] JSON examples/schema at `./yeetcraft/contracts/companion/v1/` (Yeetcraft)
- [ ] Exact file change map for both repositories

**Acceptance criteria:** Contract review completed; existing `PATCH /api/stats/batch` and frontend constraints confirmed.

**Stop condition:** Do not begin Phase 3 migrations until contract is reviewed and versioned.

### 8.4 Phase 2 — Headless companion foundation

Deliverables: Go command, migrations, `logwatcher`/`parser`/`session`/`storage` packages, fixture tests.

**Acceptance criteria:** Restart, truncation, and rotation tests pass; events persist exactly once locally.

### 8.5 Phase 3 — Backend event model and ingest

**Owner: Yeetcraft repository**

Transaction boundary:

1. Authenticate and validate schema version, batch limits, and field sizes before opening the transaction.
2. Resolve client, player identity, season, and dungeon mappings.
3. Upsert the run by `client_run_id`.
4. Insert event by `client_event_id` using a unique constraint.
5. For newly inserted events only, apply the aggregate delta or queue aggregate recomputation.
6. Insert minimal evidence and batch outcome.
7. Commit; only then return per-event acknowledgement.

**Acceptance criteria:** Duplicate batches do not change totals; rollback preserves consistency.

### 8.6 Phases 4–8

See phase table above. Each phase requires its deliverables complete and exit criteria met before the next phase begins. Phase 8 (addon) remains **optional** and must not block MVP delivery.

---

## 9. Test strategy

| Layer | Scope | Required cases |
| ----- | ----- | -------------- |
| Parser unit/fixtures | Line syntax and normalization | Known event types, malformed lines, locale-independent IDs, partial lines |
| Detection unit | Recent-damage and death logic | Single death, DoT, overkill, simultaneous deaths, wipe, missing cause |
| File integration | Real filesystem behavior | Append, restart, truncate, rotate, copy, locked file |
| SQLite integration | Migrations and state transitions | Crash recovery, uniqueness, stuck uploading lease |
| Go handler | Validation/auth/response | Version, size limits, invalid identity, partial outcome |
| Repository integration | Real `_test` PostgreSQL | Atomic batch, duplicate, rollback, reclassification |
| Companion/API E2E | Real processes and test DB | Offline then online, timeout after server commit, retry |
| Wails smoke | Packaged Windows behavior | Install, path selection, tray, restart, uninstall data policy |

### 9.1 Critical adversarial scenarios

- Server commits the event, but the HTTP response is lost.
- Companion crashes after writing the event but before queue creation.
- Combat log is truncated while being read.
- Two deaths have the same visible timestamp resolution.
- A player changes realm/name or plays another character.
- Dungeon name differs from Yeetcraft naming while game ID is stable.
- Manual correction occurs before a delayed duplicate upload.
- The backend receives a supported envelope with an unsupported category.
- A low-confidence death is counted, later confirmed as a yeet, and then corrected back.

---

## 10. Risk register and planned solutions

| Risk | Impact | Probability | Planned control |
| ---- | ------ | ----------- | --------------- |
| One logger cannot see every relevant combat event | Missing deaths/cause context | Medium until tested | Phase 0 visibility matrix; mark incomplete runs; addon or multi-client only if evidence requires it |
| Run boundaries are ambiguous | Events assigned to wrong dungeon/run | Medium | State machine, persisted evidence, manual merge/review; prefer game IDs |
| Yeets are not directly encoded | Wrong death/yeet split | High | `possible_yeet` + confidence + manual confirmation; conservative counting |
| File rotation/truncation | Reprocessing or skipped lines | Medium | File generation identity, offsets, partial-line persistence, unique event IDs |
| Render sleeps | Delayed upload | High | SQLite outbox, background retries, acknowledgement-based deletion |
| Response lost after commit | Duplicate request | Medium | Server UNIQUE constraints and duplicate acknowledgement |
| Schema drift | Old clients rejected | Medium | Versioned endpoint, explicit compatibility window and migration messages |
| Player mapping fails | Wrong or rejected ownership | Medium | GUID-first mapping; `needs_review`; never guess between players |
| Credential leaks | Unauthorized writes | Low/medium | Separate revocable key, credential manager, redacted logs |
| Manual and automatic updates conflict | Incorrect aggregates | Medium | Define event vs adjustment authority; transactional recompute/delta and audit |

---

## 11. Manual edits, historic data, and aggregate consistency

The current `PATCH /api/stats/batch` edits aggregate values directly. Once event-based uploads exist, the system must define whether those values are derived facts or user-adjustable totals. Leaving both as independent authorities will eventually create drift.

| Option | Description | Assessment |
| ------ | ----------- | ---------- |
| A. Event-derived only | Aggregates always recomputed from events; manual edits become event corrections | Best for new seasons, but cannot explain historic totals |
| B. Keep aggregate table authoritative | Events increment mutable totals while PATCH can overwrite them | Simple initially, but creates two conflicting sources of truth |
| C. Events + adjustment ledger | Totals = accepted events + explicit adjustment deltas | **Recommended:** supports migration, corrections, and event truth |

**Recommended migration:** Convert every existing `player_dungeon_stats` row into one `stat_adjustments` baseline row with `reason='legacy_import'`. Do not fabricate runs, bosses, or death timestamps for historic data. For new companion-tracked activity, totals come from accepted events plus later explicit adjustments.

### 11.1 Compatibility strategy

1. Create the new tables alongside the existing schema.
2. Backfill players' hardcoded characters from `frontend/src/data/player-characters.ts` with nullable GUID/realm until real logs identify them.
3. Copy existing aggregate counts into `stat_adjustments` as a single legacy baseline per player/season/dungeon.
4. Create an aggregate query/view that sums accepted death events and adjustments.
5. Point existing repository reads at the new aggregate query while preserving their current JSON response shapes.
6. Keep `PATCH /api/stats/batch` temporarily; translate an entered absolute total into adjustment deltas.
7. Run old and new aggregate queries side by side in tests and verify exact parity.
8. After parity and companion E2E are stable, retire `player_dungeon_stats` or keep it only as a materialized/cache table with one documented writer.

*Steps 1–8 are **Yeetcraft-side** work (Phase 3+).*

---

## 12. Minimal desktop experience

The Wails UI is an operational dashboard, not a second Yeetcraft website. It should make capture state and failure recovery obvious.

- **First-run:** locate WoW installation/log file, API URL, companion credential, and tracked characters.
- **Home:** Monitoring/Paused, current file, current run, last event, pending uploads, and last successful sync.
- **Review:** uncertain classification, unknown player/dungeon, and permanent rejection.
- **Diagnostics:** app version, parser version, database path, redacted logs, and exportable diagnostic summary.
- **Tray:** status indicator, pause/resume, retry now, open Yeetcraft, and quit.

**UX rule:** Never display “Connected” as a proxy for “data is safe.” Show **capture health** and **sync health** separately.

---

## 13. Packaging and release

- Produce a deterministic Windows amd64 build from the standalone companion repository.
- Embed companion version and supported ingest schema versions.
- Package with Wails installer after the headless E2E suite passes.
- Store user data in the per-user application-data directory; upgrades must not replace SQLite.
- Publish checksums and a short release note with required backend compatibility.
- Test upgrade from the previous version with pending uploads and an active/abandoned run.
- Defer code signing and auto-update until Windows trust warnings or distribution effort justify them.

---

## 14. Addon as a later optional extension

The addon is deliberately **not** a dependency for the companion MVP. Consider it only after real use identifies metadata or usability gaps that cannot be solved reliably from the combat log alone.

### 14.1 Potential value

- Automatically enable combat logging when entering relevant content and warn when it is disabled.
- Capture authoritative in-game run lifecycle signals such as challenge start/completion when available.
- Record map/instance context, group roster, and selected positional context.
- Provide a visible “Yeetcraft logging active” indicator inside WoW.
- Support explicit confirmation of a possible yeet shortly after a run.

### 14.2 Constraints

- A WoW addon cannot make arbitrary HTTP requests to Yeetcraft.
- SavedVariables are not a reliable real-time IPC channel; disk writes typically happen on UI reload/logout/exit.
- Addon APIs and protected-combat restrictions must be respected.
- Every additional group installation increases support and version-drift cost.
- The addon should enrich metadata, **not** become the only source of death facts unless testing proves that necessary.

### 14.3 What implementation would require

- A separate capability spike for current retail WoW APIs and Midnight events.
- A versioned SavedVariables format or another allowed local handoff design.
- Companion support for discovering, parsing, and correlating addon metadata after it is flushed.
- Addon packaging, `.toc` metadata, Lua modules, migration handling, and manual installation/update documentation.
- Tests using captured SavedVariables and combat logs from the same runs.

---

## 15. Decisions to resolve

| Decision | Recommended default | When to lock |
| -------- | ------------------- | ------------ |
| Aggregate authority | Events + adjustment ledger | Before DB migration |
| Unknown death counting | Count as death until reviewed | Before ingest implementation |
| Companion authentication | Dedicated revocable API key | Before E2E upload |
| Retention | Keep structured local history; no raw upload | Before packaged release |
| Player identity | GUID-first mapping to configured tracked identities | During Phase 0/1 |
| Run ID recipe | Deterministic from available run signals | After Phase 0 |
| Partial batch response | Per-event results in HTTP 200 envelope | During contract review |
| Supported OS | Windows amd64 only for MVP | Now |

---

## 16. Definition of done for the companion MVP

- [ ] A clean Windows installation can be configured without developer tools.
- [ ] One running companion captures the configured party members' observable deaths from representative Mythic+ runs.
- [ ] A restart, log rotation, and temporary backend outage do not lose accepted events.
- [ ] The same event can be submitted repeatedly without changing totals more than once.
- [ ] Normal deaths appear in Yeetcraft without manual stat entry.
- [ ] Ambiguous causes and possible yeets are visible for review and can be corrected atomically.
- [ ] Existing public reads and manual editing remain functional.
- [ ] Raw logs and credentials are not uploaded or exposed in diagnostics.
- [ ] Unit, fixture, PostgreSQL integration, and end-to-end recovery tests pass.
- [ ] Operational status clearly separates capture health from sync health.

---

## 17. Recommended first work packages

Complete repository bootstrap first, then create **only** the Phase 0 spike. Avoid database, API, and Wails changes until the capability matrix is complete.

### 17.1 Bootstrap deliverables (complete)

| Deliverable | Contents |
| ----------- | -------- |
| `AGENTS.md` | Repository boundaries, sibling reference, task scope, and safety rules |
| `docs/IMPLEMENTATION_PLAN.md` | Markdown version of this architecture and phased plan |
| `docs/YEETCRAFT_INTEGRATION.md` | API ownership, local sibling path, contracts, and integration test model |
| `docs/COMBAT_LOG_FORMAT_V22.md` | Current V22 format fields, sources, conflicts, and open questions |
| `README.md` | Purpose, prerequisites, current phase, non-goals, and basic commands |
| `.gitignore` | Combat logs, SQLite, secrets, binaries, Wails output, and diagnostics |
| `go.mod` + skeleton | Minimal standalone Go module with no Yeetcraft code imports |

### 17.2 Phase 0 deliverables (next)

| Deliverable | Contents |
| ----------- | -------- |
| `cmd/logprobe` | CLI accepting a combat-log path and optional tracked names/GUIDs |
| `internal/parser` | Streaming line parser with normalized events |
| `internal/detection` | Recent-damage buffers and death candidate output |
| `testdata/logs` | Small sanitized fixtures covering known cases |
| `docs/COMBAT_LOG_CAPABILITIES.md` | Observed fields, visibility, accuracy, and unresolved gaps |
| Test report | Per-run expected vs detected deaths and likely-cause accuracy |

---

## Appendix A — Capability matrix template

| Data point | Observed source | Reliability | Fallback |
| ---------- | --------------- | ----------- | -------- |
| Player death | Combat log | To measure | Manual review/import |
| Lethal source/spell | Recent damage + death context | To measure | Unknown cause |
| Dungeon identity | Instance/run markers | To measure | Name/game-ID mapping |
| Key level | Run metadata | To measure | Optional/manual |
| Run start/end | Log marker/state machine | To measure | Inactivity/manual close |
| Yeet classification | Knockback/environment heuristics | Expected low/medium | Manual confirmation |
| Coordinates | Likely addon-only | Not MVP | Omit |

*Fill reliability columns during Phase 0; do not pre-fill with assumed values.*

## Appendix B — Implementation guardrails

- Do not refactor unrelated frontend or backend areas.
- Do not make the companion depend on a continuously available server.
- Do not acknowledge local uploads before the server does.
- Do not use timestamps alone as unique IDs.
- Do not silently map an unknown character to a similarly named player.
- Do not classify every environmental death as a yeet.
- Do not expose the browser write token in a desktop configuration file.
- Do not replace current aggregate tables until event behavior is proven.
- Do not implement the addon simply because it is technically possible.

## Appendix C — Cross-repository source paths

**Yeetcraft (read-only reference during development):**

```text
../yeetcraft/backend/cmd/server/main.go
../yeetcraft/backend/internal/middleware/auth.go
../yeetcraft/backend/internal/config/config.go
../yeetcraft/backend/internal/database/pool.go
../yeetcraft/backend/internal/handler/
../yeetcraft/backend/internal/repository/
../yeetcraft/backend/db/
../yeetcraft/backend/cmd/testdb/
../yeetcraft/frontend/src/api/api.ts
../yeetcraft/frontend/src/main.tsx
../yeetcraft/frontend/e2e/
../yeetcraft/frontend/package.json
../yeetcraft/contracts/companion/v1/    # planned; not present yet
```

**Companion (this repository):**

```text
./AGENTS.md
./docs/IMPLEMENTATION_PLAN.md
./docs/YEETCRAFT_INTEGRATION.md
./docs/COMBAT_LOG_CAPABILITIES.md
./cmd/
./internal/
./testdata/logs/
```

## Appendix D — Multi-repo guardrails

| Rule | Detail |
| ---- | ------ |
| Yeetcraft repository | Owns API, PostgreSQL, canonical contracts, and website behavior |
| Companion repository | Owns local capture, parsing, SQLite, uploader, review UI, and packaging |
| Imports | No cross-repository Go imports |
| Database | No companion access to PostgreSQL |
| Runtime | No dependency on a sibling checkout or shared local file |
| Planning | Reading the sibling repository is allowed; writing requires explicit cross-repository task scope |
| Changes | Every cross-repository change produces separate commits and validation results |
| Contract | Canonical contract is versioned and server-owned; companion releases declare supported versions |

---

## Related documentation

- [YEETCRAFT_INTEGRATION.md](./YEETCRAFT_INTEGRATION.md)
- [COMBAT_LOG_FORMAT_V22.md](./COMBAT_LOG_FORMAT_V22.md)
- [COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md)
- [Synthetic fixture provenance](../testdata/logs/synthetic/README.md)
- [AGENTS.md](../AGENTS.md)
