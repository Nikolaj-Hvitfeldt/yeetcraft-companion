# Yeetcraft Companion — Technical Implementation Plan

| Field | Value |
| ----- | ----- |
| **Status** | Active |
| **Current milestone** | Phase 1 — repository integration and canonical v1 contract review |
| **Canonical repository** | [yeetcraft-companion](https://github.com/Nikolaj-Hvitfeldt/yeetcraft-companion) |
| **Related repository** | [yeetcraft](https://github.com/Nikolaj-Hvitfeldt/Yeetcraft) (website, backend, PostgreSQL, API, canonical companion contract) |
| **Last updated** | 2026-09-18 |

Automatic capture, local processing, and reliable upload of Mythic+ death events.

| Field | Value |
| ----- | ----- |
| Primary focus | Windows companion application and Yeetcraft integration |
| Existing system | React/Vite PWA, Go API, PostgreSQL/Supabase, Vercel + Render |
| Initial users | Four tracked people with explicit character mappings (fixed friend-group MVP) |
| Addon | Optional later extension; **not** a dependency for the companion MVP |
| Document status | Phase 0 accepted; Phase 1 contract decisions are the next gate |

**Current direction:** The headless Phase 0 proof of concept established a
fixed-group MVP go decision. Complete the cross-repository Phase 1 contract
review before changing either database or building upload/UI systems. One
logger remains the MVP capture model; future uploads must be idempotent and
survive offline use and Render cold starts.

---

## Executive summary

The companion app becomes a local bridge between World of Warcraft and Yeetcraft. It follows the player's combat-log file, converts relevant lines into structured runs and death events, stores those events locally, and uploads them to the existing Go backend. The website continues to display the current aggregated statistics, while the new event layer creates an audit trail and enables richer features later.

The project should not begin with a polished desktop shell. Its primary uncertainty is **data quality**: which events are visible to one logging client, how reliably a Mythic+ run can be bounded, how a death cause can be inferred, and when a “yeet” can be distinguished from a normal death. **Phase 0** therefore produces evidence from real logs and a written capability matrix. Every later phase depends on that result.

The safest migration is **additive**. Existing public GET routes, manual editing, token handling, offline frontend behavior, and player/dungeon statistics remain functional. A separate versioned ingest endpoint accepts immutable events. The server inserts them idempotently and updates the existing aggregate table within the same transaction. The companion and Yeetcraft remain **separate Git repositories** and communicate only through this versioned HTTP contract.

---

## Decision record

### 2026-09-18 — Accept Phase 0 and begin Phase 1

- Two reviewed retail sessions covering five completed Mythic+ runs produced 43
  player death candidates. The detector retained 35 configured tracked deaths
  and excluded eight untracked fifth-player deaths.
- The evidence includes boss and trash deaths, a failed boss pull, a full-party
  trash wipe, a repeated death after resurrection in one encounter, an overtime
  completion, repeated version headers, and high/medium cause confidence.
- No Phase 0 stop condition was triggered. One logging client provided useful
  identity, run, encounter, death, and cause evidence for the fixed-group MVP.
- Phase 0 is therefore **accepted for MVP progression**. This is a go decision,
  not a claim that every combat-log behavior is verified.
- Abandonment, reload/restart continuity, file rotation/truncation, lethal
  environmental deaths, and knockback/void evidence remain an ongoing,
  non-blocking research backlog. Reliability mechanics belong primarily to
  Phase 2.

### 2026-09-18 — Manual death/yeet classification is the MVP authority

- Every accepted `UNIT_DIED` candidate starts as an ordinary `death` (server
  default on ingest).
- The **Yeetcraft website** may reclassify it to `yeet`, return it to `death`,
  or mark it `ignored` ([Yeetcraft ADR 001](../../yeetcraft/docs/adr/001-post-ingest-classification-and-corrections.md)).
  Companion local review is not classification authority.
- Exactly one accepted event contributes to exactly one aggregate category;
  `death → yeet` moves one count atomically and never increases total mistakes.
- Cause confidence and future automatic yeet logic may produce review
  suggestions, but must not override a confirmed website classification.
- Perfect automatic yeet detection is not a Phase 0 or MVP gate.

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
- Detected deaths default to ordinary deaths. The Yeetcraft website reclassifies
  them as `yeet` or `ignored` ([ADR 001](../../yeetcraft/docs/adr/001-post-ingest-classification-and-corrections.md)).

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
│   └── companion/v1/          # canonical draft contract in Yeetcraft (API not implemented)
└── docs/
```

**Companion** (`./yeetcraft-companion` — this repository):

```text
yeetcraft-companion/
├── AGENTS.md
├── docs/
│   ├── IMPLEMENTATION_PLAN.md
│   ├── PHASE_2_FILE_MAP.md    # Phase 2 file map; capture foundation implemented on feat/phase-2/headless-capture
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

- Place the canonical v1 request/response schema and examples in the **Yeetcraft repository** at `../yeetcraft/contracts/companion/v1/` (draft Markdown + JSON Schema + examples; **API not implemented**).
- During development, companion contract tests may read `../yeetcraft/contracts/companion/v1/` from the sibling checkout.
- The compiled companion supports an explicit schema version and must also be testable **without** the sibling repository by using its own **derived** request fixtures.
- **Do not maintain two independently edited canonical schemas.** Generated code or fixtures in this repository are derived copies, not source of truth.
- Phase 2/3 paths: [`PHASE_2_FILE_MAP.md`](./PHASE_2_FILE_MAP.md) and Yeetcraft [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md).

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

**Exact PostgreSQL table and column names are Phase 3 DDL.** Semantics below
must follow the frozen ingest contract and ADRs; this table is not a second
wire spec and must not reintroduce names/realms, append-only PATCH deltas, or
per-installation auth.

| Concept | Purpose | Phase 1-aligned fields |
| ------- | ------- | ---------------------- |
| `players` | Real people/profile owners | Existing `id`, `display_name`, `avatar_url` |
| `characters` | WoW characters owned by a player | Character-slice blocker: nullable **unique** `guid` ([`CHARACTERS_AND_BOSS_NEMESIS.md`](../../yeetcraft/docs/CHARACTERS_AND_BOSS_NEMESIS.md)) |
| `seasons` | Season boundary | Existing `name`, `expansion`, `is_current` (**UI default only; never ingest authority**). Phase 3: non-overlapping `starts_at` / `ends_at` |
| `dungeons` | Canonical dungeon | Existing `name`, `short_name`. Phase 3: nullable unique `challenge_map_id` (not a generic `game_instance_id`) |
| `season_dungeons` | Seasonal pool | Existing `season_id`, `dungeon_id`, `display_order` |
| `encounters` | Boss belonging to a dungeon | Journal `encounterId` on the wire; server catalog is later Yeetcraft work |
| `runs` | One Mythic+ attempt | `client_run_id` UNIQUE from the locked recipe; start instant, map ID, keystone level. Companion `seasonId` is a **validated hint** |
| `death_events` | One observable tracked-player death | `client_event_id` UNIQUE; server default `category = death`; event revision for website corrections. `encounter_id` nullable (trash = null) |
| `death_causes` | Ranked cause evidence | `rank`, `source_type`, `spell_id`, `creature_id`, `amount`, `overkill`, `confidence`. **No** player names, realms, untracked GUIDs, or spell display names |
| Adjustment ledger | ADR 002 | Immutable **legacy baseline** + **one replaceable** manual adjustment per player × season × dungeon × category. Not append-only `deaths_delta` / `yeets_delta` |
| `ingest_batches` | Idempotency diagnostics | `batch_id` UNIQUE + request-body fingerprint. Not an authorization principal |
| Per-installation clients | **Deferred** past the fixed-group MVP | `installationId` is spoofable diagnostics only; v1 uses one `COMPANION_API_KEY` |

*Not implemented in either repository yet. Do not copy this table into the companion as a schema editor.*

### 4.3 Why boss and cause are separate

A player's “boss nemesis” is the boss encounter active when the player died. The literal source of the final damage can instead be the boss, an add, a summoned object, a damage-over-time effect, or the environment. Store both `encounter_id` on `death_events` and ranked source evidence in `death_causes`.

| Question | Data used |
| -------- | --------- |
| Which boss kills this player most often? | `death_events.character_id`/player owner + `encounter_id` |
| Which ability causes most deaths? | `death_causes.spell_id`, using rank 1 or confirmed cause |
| Is the dungeon dangerous outside bosses? | `death_events` where `encounter_id` is null |
| Did the boss or an add deliver the final hit? | `death_causes.source_type` and `creature_id` |
| Which character dies most? | `death_events.character_id` |
| Which person dies most across alts? | `characters.player_id` |

### 4.4 Category and review model

| Classification | Meaning | Aggregate effect | Authority |
| -------------- | ------- | ---------------- | --------- |
| `death` | Accepted ordinary death; ingest default | +1 death | Server on accept; website may correct |
| `yeet` | Website-confirmed yeet | +1 yeet | Yeetcraft website (ADR 001) |
| `ignored` | Duplicate, false positive, or excluded | None | Yeetcraft website (ADR 001) |

`possible_yeet` is a future detector **suggestion**, not an aggregate category.
An unknown or ambiguous cause does not make the death disappear; it remains a
normal death until the website changes its classification.

**Accounting rule:** Reclassification moves one count atomically between
deaths and yeets and must never increase total mistakes. A confirmed website
classification takes precedence over later detector reprocessing. Companion
review UI does not replace website authority.

### 4.5 Stable IDs

Do not base identity only on player name, log line number, or a random UUID
generated each time a file is reread. A restart, file copy, or retry would
create duplicates.

**WP1 locked recipe** (canonical spec:
`../yeetcraft/contracts/companion/v1/CONTRACT.md`):

```text
clientRunId = SHA-256(
  domainTag=yeetcraft-run-v1,
  challengeModeStartInstant,   # RFC 3339 UTC; season deliberately excluded
  challengeMapId,
  keystoneLevel
)

clientEventId = SHA-256(
  domainTag=yeetcraft-death-v1,
  clientRunId,
  victimGuid,
  deathInstant,                # RFC 3339 UTC
  ordinal                      # zero-based among same-instant UNIT_DIED for victim
)
```

`installationId`, log-file identity, party GUIDs, and season are **excluded**
from ID hashes. Hash inputs use UTF-8 NFC strings with 32-bit big-endian
length prefixes per field. Output format: `sha256:` + 64 lowercase hex digits.

Companion producer gaps for Phase 2 are recorded in
[`docs/CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md). Persist
generated IDs in SQLite once Phase 2 storage exists. Server UNIQUE constraints
are the final deduplication boundary.

---

## 5. Versioned ingest API

The **canonical** companion v1 ingest contract lives in Yeetcraft. This section
is a pointer plus companion retry notes. It is **not** a second wire spec.
Do not copy request JSON here.

| Artifact | Location |
| -------- | -------- |
| Normative spec | [`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../../yeetcraft/contracts/companion/v1/CONTRACT.md) |
| Request / response / error schemas | [`../yeetcraft/contracts/companion/v1/schema/`](../../yeetcraft/contracts/companion/v1/schema/) |
| Synthetic examples | [`../yeetcraft/contracts/companion/v1/examples/`](../../yeetcraft/contracts/companion/v1/examples/) |
| Phase 3 file map | [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md) |

Status: **Draft — reviewed, not implemented.** The route does not exist until
Yeetcraft Phase 3. Companion upload is Phase 4.

### 5.1 Endpoint (frozen)

```http
POST /api/companion/v1/deaths/batch
X-API-Key: <COMPANION_API_KEY>
Content-Type: application/json
```

- `batchId` is a **UUID in the JSON body**; there is no `Idempotency-Key`
  header in v1.
- `COMPANION_API_KEY` is a **separate** server env var and middleware instance
  from browser `API_KEY`; missing or empty key fails closed with **503**
  `companion_api_unconfigured`.
- Do **not** reuse `PATCH /api/stats/batch` for event ingest.

### 5.2 Request shape (frozen)

Use `CONTRACT.md` request payloads and the request schema. Summary only:

- Envelope: `schemaVersion: 1`, `batchId`, optional diagnostic `installationId`,
  `events` (1–500).
- Each event: `clientEventId`, tracked `characterGuid` only (no name/realm),
  canonical `deathInstant`, `ordinal`, complete `run`, `encounter` or `null`,
  optional ranked `causes`.
- Ingest-only: omit `category` or send `"death"`. The server assigns default
  `death`. `yeet` / `ignored` are website corrections ([Yeetcraft ADR 001](../../yeetcraft/docs/adr/001-post-ingest-classification-and-corrections.md)).
- Cause evidence must not include player names, realms, or untracked GUIDs.

### 5.3 Acknowledgement and retry (frozen; uploader is Phase 4)

Envelope/schema failures reject the **whole** request (no per-event body).
A processable batch returns **HTTP 200** with exactly one ordered result per
input event (`accepted`, `duplicate`, `needs_review`, `rejected`). There is
**no** HTTP 207. Unknown GUID / unmapped map / season contradiction are
**200** `needs_review` (quarantine), not 409. **409** is only `batch_conflict`
or `event_id_conflict`. Database failure rolls back the batch, returns **5xx**,
and stores no acknowledgement. Duplicate identical fingerprints must not reset
a website correction.

| Response | Companion action |
| -------- | ---------------- |
| 200 with `accepted` / `duplicate` / `needs_review` / `rejected` | Treat as acknowledged; persist ordered results locally |
| 200 replay (same `batchId` + body) | Idempotent; replace local pending state |
| 5xx with `retryable: true` (including `ingest_temporarily_unavailable`) | Retry same `batchId` and body with backoff |
| 409 `batch_conflict` / `event_id_conflict` | Non-retryable; halt and surface for review |
| 401, 413, 415, 422 | Fix request or configuration; do not blind-retry |
| 429 | Honor `Retry-After` or backoff |
| 503 `companion_api_unconfigured` | Operator must configure the server; retry later |

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
files(id, path, file_identity, generation, byte_offset, partial_line,
      parser_state_json, updated_at)
runs(id, client_run_id UNIQUE, challenge_mode_start_instant, challenge_map_id,
     keystone_level, status, metadata_json, started_at, ended_at)
events(id, client_event_id UNIQUE, run_id, character_guid, death_instant,
       ordinal, hold_reason, review_status, category, confidence,
       payload_json, created_at)
event_causes(id, event_id, rank, source_type, spell_id, creature_id,
             environmental_type, amount, overkill, player_origin, confidence)
settings(key PRIMARY KEY, value, updated_at)
schema_migrations(version PRIMARY KEY, applied_at)
```

Phase 2 implements `files`, `runs`, `events`, `event_causes`, `settings`, and
`schema_migrations`. The `uploads` table is deferred to **Phase 4** (upload
state machine and retry bookkeeping); local capture persistence does not depend
on it.

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
- **WP1 locked:** GUID-first mapping to configured tracked identities.
  Unmapped GUIDs are `unknown` / `needs_review`. Never auto-create a public
  player. End-to-end ingest is blocked on Yeetcraft `characters.guid`.

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
| **Phase 0** — Combat-log evidence spike | Prove visibility, parsing, death cause, and run signals | **Companion** | **Accepted 2026-09-18:** representative deaths accounted for; residual uncertainties documented; **no server writes** |
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

**Status: accepted for MVP progression on 2026-09-18. Real-log evidence
collection continues as a non-blocking backlog.**

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

Phase 0B was intentionally split so source-backed parser work could begin
before retail evidence was available. Phase 0A.2 subsequently validated and
adjusted those assumptions against two local, gitignored retail sessions.

**Phase 0B.1 (complete, 2026-07-30):** Implemented the bounded streaming
line reader, provisional envelope separation, CSV-aware tokenization, V22
header and fail-closed quarantine state (including malformed boundaries and
non-retail projects), explicit common-header event recognition, unknown event
preservation, provisional signed-offset envelopes, and privacy-safe
`cmd/logprobe`. Detection, typed damage payloads, tracked roster options, and
death/run inference remain deferred.

**Limited Phase 0B.2 (complete, 2026-07-30):** Added source-backed typed
payload parsing for exact advanced-enabled `SPELL_DAMAGE`, `RANGE_DAMAGE`,
`SWING_DAMAGE`, and `ENVIRONMENTAL_DAMAGE` layouts plus neutral
`ENCOUNTER_START`, `ENCOUNTER_END`, and `CHALLENGE_MODE_END` metadata.
Structural and primitive failures retain generic event recognition but provide
no typed payload; bounded source-expectation diagnostics retain parsed
payloads. `CHALLENGE_MODE_START` remains recognized but untyped because the
selected reference does not define exact raw CSV serialization for its affix
array. No death, run, identity, boss, or cause inference was added.

**Phase 0A.2 parser adjustments and detection prototype (complete, 2026-08-19):**
Retail log validation adjusted typed parsing for decimal damage-school suffix
tokens, float `CHALLENGE_MODE_END` timer fields, and `SPELL_PERIODIC_DAMAGE`
layout parity with `SPELL_DAMAGE`. Added `internal/detection` for in-memory
Mythic+ run context, boss encounter windows, recent incoming-damage buffers,
ranked cause candidates, and player death candidates. Extended `cmd/logprobe`
with privacy-safe `--deaths` and repeatable `--track-guid` filters.

**Phase 0 acceptance evidence (2026-09-18):** A second reviewed retail session
added three completed runs and 31 deaths (27 tracked, four untracked), including
11 boss-context deaths, a failed boss pull, a five-player trash wipe, repeated
death after resurrection, and 28 high-confidence plus three medium-confidence
primary causes. Twelve real `Falling` damage events parsed successfully but
were nonlethal. Across both sessions, the accepted evidence covers five runs
and 43 deaths (35 tracked, eight untracked).

**Limited Phase 0B implemented and tested:**

- CSV-aware tokenization;
- version-header CSV payload parsing;
- unsupported-version handling;
- common-header extraction;
- exact source-backed damage event payloads (`SPELL_DAMAGE`, `RANGE_DAMAGE`,
  `SWING_DAMAGE`, `ENVIRONMENTAL_DAMAGE`);
- advanced-block extraction where the layout is exact in the selected V22
  reference;
- neutral typed metadata payloads where raw CSV layout is exact
  (`ENCOUNTER_START`, `ENCOUNTER_END`, `CHALLENGE_MODE_END`);
- unknown and malformed input handling;
- partial-line buffering.

A technical sub-capability reaches **Synthetically tested** in limited Phase
0B only when an implementation passes an exact source-backed fixture. Shape-
incomplete death scenarios must not be promoted to passing success fixtures.

**Residual evidence backlog (non-blocking after Phase 0 acceptance):**

- abandoned-run closure signals;
- `/reload`, full-client restart, and multi-file run continuity;
- file append, rotation, truncation, and resumable offset behavior;
- lethal environmental and knockback/void deaths for future suggestions;
- unresolved semantics of selected V22 unknown fields and the exact
  `UNIT_DIED` suffix;
- retail 12.0+ **timestamp envelope** confirmation (timezone-less vs
  offset-bearing; DST). The v1 contract already fails closed: missing
  persisted WoW-log timezone, DST ambiguity, invalid stamp, or missing run
  start → local review, no hash, no upload. Confirming the exact envelope is
  a scoped evidence task / Phase 2 parser work. Do not invent a new wire
  timestamp format.

Phase 0B design for advanced-block semantics remains open where the selected
reference and observed V22 samples conflict. Do not infer suffix positions from
approximate field counts.

Deliverables:

- [x] CLI probe (e.g. `cmd/logprobe`) accepting a combat-log path and optional tracked GUID filters
- [x] Streaming parser reporting recognized, unknown, and malformed event counts
- [x] Recent-damage buffers and death candidate output in `internal/detection/`
- [x] Anonymized fixture slices under [testdata/logs/](../testdata/logs/)
- [x] Capability matrix in [docs/COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md)
- [x] Privacy-safe per-run expected-vs-detected review recorded in the
  capability matrix; raw logs remain local and gitignored

Phase 0A.1 deliverables:

- [x] Current V22 format research, sources, and conflicts documented
- [x] Original synthetic fixtures prepared and provenance recorded
- [x] Capability vocabulary separates prepared fixtures from passing tests
- Timestamp envelope and exact `UNIT_DIED` V22 suffix: **moved** to the
  residual evidence backlog (not a Phase 1 gate)

Tasks:

- [x] Collect multiple representative completed Mythic+ runs with advanced
  combat logging, ordinary deaths, boss/trash deaths, and a wipe
- [x] Verify observed deaths and preceding damage for all configured tracked
  characters represented in the sessions
- [x] Document available instance, difficulty, key-level, encounter, party, and
  completion markers
- [x] Measure observed ambiguity and retain medium-confidence causes for review
- [x] Make the Phase 0 go/no-go decision and classify remaining scenarios as a
  non-blocking Phase 2/evidence backlog

**Acceptance criteria:**

- Representative tracked party deaths in reviewed runs are accounted for; the
  fixed-group single-logger assumption is accepted for MVP progression.
- Uncertainties recorded in `COMBAT_LOG_CAPABILITIES.md`; no verified capabilities claimed without fixtures.
- **No server writes** and no Yeetcraft schema changes.

**Stop condition result:** Not triggered. If later evidence shows systematic
missed tracked deaths or unsafe identity mapping, pause upload work and reopen
the decision before production use.

### 8.3 Phase 1 — Repository integration and contracts

**Status: current milestone**

**Owner: explicit cross-repository task with separate changes, validation, and
commits per repository**

Deliverables:

- [x] WP1: canonical Markdown skeleton at `../yeetcraft/contracts/companion/v1/`
  (`README.md`, `CONTRACT.md`) — reviewed, not implemented
- [x] WP1: companion producer review at
  [`docs/CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md)
- [x] WP2: run, encounter, death, ranked cause payloads; request schema and
  synthetic examples (canonical in Yeetcraft)
- [x] WP3: acknowledgement semantics, limits, validation errors, response schemas
- [x] Define GUID-first character resolution and explicit
  tracked/untracked/unknown outcomes (WP1 locked; WP2 wire fields)
- [x] Define deterministic run/event IDs, batch idempotency (WP1 locked), retry-safe
  acknowledgement (WP3)
- [x] Define `death ↔ yeet` and `ignored` correction transitions so total
  mistakes remain invariant (Yeetcraft ADR 001)
- [x] Decide companion-specific authentication and credential boundaries
- [x] Define reconciliation with existing manual aggregate statistics (Yeetcraft ADR 002)
- [x] Record approved decisions in this plan or focused ADRs if the decision
  history becomes too large
- [x] Produce an exact, repository-separated Phase 2/3 file change map
  ([`docs/PHASE_2_FILE_MAP.md`](./PHASE_2_FILE_MAP.md); Yeetcraft
  [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md))
- [x] Validate schemas/examples, error fixtures, relative links, and canonical checksum recording
  ([`docs/CONTRACT_V1_DERIVED_FIXTURES.md`](./CONTRACT_V1_DERIVED_FIXTURES.md))
- [x] Review the contract against existing `PATCH /api/stats/batch`, auth,
  offline frontend behavior, and guarded test-database requirements
  (compatible with sequencing: ingest is a new route; `COMPANION_API_KEY` is
  a second middleware; frontend `expectedRevision` before companion writes;
  testdb stays schema.sql-only — see Yeetcraft `IMPLEMENTATION_MAP.md`)

**Acceptance criteria:** The canonical v1 contract is reviewed and versioned in
Yeetcraft; both repositories agree on identity, idempotency, classification,
privacy, authentication, reconciliation, and error semantics; no migration or
upload implementation is required to complete Phase 1.

**Stop condition:** Do not begin Phase 3 migrations until contract is reviewed and versioned.

### 8.4 Phase 2 — Headless companion foundation

**Status: implemented on `feat/phase-2/headless-capture`; ready for human review / PR to companion `dev`.**

Deliverables shipped: headless `cmd/yeetcraft-companion`, `internal/config`,
`internal/logwatcher`, `internal/storage`, `internal/session` (IDs + run state
machine), and wiring from fail-closed config through parser, detection, session
IDs, and SQLite. No HTTP upload, review UI, or Wails shell.

**Acceptance evidence (WP 2.7, 2026-09-18):**

| Scenario | Evidence |
| -------- | -------- |
| Fail-closed config | `cmd/yeetcraft-companion/main_test.go` `TestMissingTrackedConfigExitsNonZeroAndWritesNothing` — exit code 2, no SQLite file created |
| Synthetic log IDs | `TestSyntheticLogProducesExpectedEventCountAndIDs` — one persisted event with contract `clientRunId` / `clientEventId` |
| Restart mid-log | `TestRestartMidLogSameIDsNoDuplicates` — resume from committed offset, identical IDs, no duplicate rows |
| Truncation | `TestTruncationPersistOnce` — file shrink keeps one event; watcher generation reset covered in `internal/logwatcher/watcher_test.go` |
| Rotation | `TestRotationPersistOnce` plus `TestWatcherRotateToNewFilename` / `TestWatcherReplaceSamePathStartsNewGeneration` |
| Persist-once / crash-before-ack | `internal/storage/storage_test.go` `TestCrashBeforeAckRollbackAndReplay`, `TestOffsetDoesNotAdvanceWhenTransactionAborts` |
| Process exit abandons run | `internal/session/session_test.go` `TestManagerCloseNeverCompletesActiveRun`; continuous mode calls `session.Close()` on SIGTERM/interrupt |
| CI matrix | `.github/workflows/go.yml` runs `gofmt`, `go test ./...`, and `go vet ./...` on `ubuntu-latest` and `windows-latest` |

**Not Phase 2 (still deferred):** `internal/uploader`, `internal/review`,
`COMPANION_API_KEY`, HTTP ingest, Wails, addon.

**Acceptance criteria:** Restart, truncation, and rotation tests pass; events persist exactly once locally.

### 8.5 Phase 3 — Backend event model and ingest

**Owner: Yeetcraft repository**

**Status: not started.** File map only:
[`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md).

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
| Yeets are not directly encoded | Wrong death/yeet split | High | Default to `death`; detector may suggest review, but explicit user reclassification is authoritative |
| File rotation/truncation | Reprocessing or skipped lines | Medium | File generation identity, offsets, partial-line persistence, unique event IDs |
| Render sleeps | Delayed upload | High | SQLite outbox, background retries, acknowledgement-based deletion |
| Response lost after commit | Duplicate request | Medium | Server UNIQUE constraints and duplicate acknowledgement |
| Schema drift | Old clients rejected | Medium | Versioned endpoint, explicit compatibility window and migration messages |
| Player mapping fails | Wrong or rejected ownership | Medium | GUID-first mapping; `needs_review`; never guess between players |
| Credential leaks | Unauthorized writes | Low/medium | Separate revocable key, credential manager, redacted logs |
| Manual and automatic updates conflict | Incorrect aggregates | Medium | ADR 002: derived totals + legacy baseline + replaceable adjustment; `409 stale_revision` |

---

## 11. Manual edits, historic data, and aggregate consistency

**Locked in [Yeetcraft ADR 002](../../yeetcraft/docs/adr/002-revision-protected-adjustment-ledger.md).**
Do not treat the older option table as an open protocol choice.

```text
displayed_* = event_derived_* + legacy_baseline_* + manual_adjustment_*

manual_adjustment_* = entered_total − event_derived_* − legacy_baseline_*
```

- `player_dungeon_stats` becomes a **derived read model**.
- Legacy baseline is a **one-time immutable** import (`reason = legacy_import`).
  Do not fabricate runs, bosses, or death timestamps for historic data.
- Manual PATCH stays **absolute entered totals** for browser UX. The server
  **replaces** the single adjustment row per player × season × dungeon ×
  category; it never appends a delta history for PATCH.
- `expectedRevision` compare-and-set; stale offline writes return **409**
  `stale_revision` and remain reviewable. Frontend revision support lands
  **before** companion event writes are enabled.
- `ignored` events contribute to neither deaths nor yeets.

### 11.1 Compatibility strategy

1. Character slice (`characters.guid`) before ingest handlers.
2. Add season `starts_at` / `ends_at` and `dungeons.challenge_map_id` in
   `schema.sql` **and** the next numbered migration after the character slice
   (testdb stays schema.sql-only).
3. Import legacy baseline; keep public GET JSON shapes.
4. Add `expectedRevision` on PATCH and editor reads; outbox handles
   `stale_revision`.
5. Enable companion ingest (increments event-derived counts).
6. Correction route (ADR 001) after revision-protected aggregates exist.

*These steps are **Yeetcraft-side**. They are not implemented yet.*

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

Phase 1 locked the ingest wire and server-behavior ADRs. Remaining items are
packaging or later-phase operations, not competing protocol.

| Decision | Status |
| -------- | ------ |
| Aggregate authority | **Locked** — derived totals + immutable legacy baseline + one replaceable manual adjustment ([Yeetcraft ADR 002](../../yeetcraft/docs/adr/002-revision-protected-adjustment-ledger.md)) |
| Unknown / new deaths | **Locked** — server default `category = death`; website owns `yeet` / `ignored` ([Yeetcraft ADR 001](../../yeetcraft/docs/adr/001-post-ingest-classification-and-corrections.md)) |
| Companion authentication | **Locked** — `COMPANION_API_KEY`; fail-closed **503** `companion_api_unconfigured` |
| Player identity | **Locked** — GUID-first; client-side tracked filter |
| Run / event ID recipes | **Locked** — `CONTRACT.md`; season and `installationId` excluded |
| Partial batch response | **Locked** — HTTP **200** with one ordered result per event; no 207; `needs_review` is not 409 |
| Supported OS | **Locked** — Windows amd64 only for MVP |
| Packaged retention UI | Later (Phase 6+). Contract already: no raw-log upload; structured local history until ack |

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

Repository bootstrap and the bounded Phase 0 spike are complete for MVP
progression. The current work package is **Phase 1 contract review**. Avoid
database migrations, API implementation, SQLite, uploads, and Wails until the
canonical v1 contract is reviewed.

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

### 17.2 Phase 0 deliverables (accepted 2026-09-18)

| Deliverable | Contents |
| ----------- | -------- |
| `cmd/logprobe` | CLI accepting a combat-log path and optional tracked GUIDs |
| `internal/parser` | Streaming line parser with normalized events |
| `internal/detection` | Recent-damage buffers and death candidate output |
| `testdata/logs` | Small sanitized fixtures covering known cases |
| `docs/COMBAT_LOG_CAPABILITIES.md` | Observed fields, visibility, accuracy, and unresolved gaps |
| Test report | Per-run expected vs detected deaths and likely-cause accuracy |

### 17.3 Phase 1 deliverables (current milestone)

| Deliverable | Contents |
| ----------- | -------- |
| Yeetcraft `contracts/companion/v1/` | Canonical versioned schemas/examples — **reviewed, not implemented** |
| Decision record | Identity, idempotency, classification, auth, privacy, and reconciliation — locked in `CONTRACT.md` + Yeetcraft ADRs 001/002 |
| Repository file maps | [`PHASE_2_FILE_MAP.md`](./PHASE_2_FILE_MAP.md) (companion Phase 2); Yeetcraft [`../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md`](../../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md) (Phase 3). Maps only — Phase 2/3 not implemented. |
| Contract review evidence | Compatibility review against current API, frontend writes, and test guards — closed in §8.3 |

---

## Appendix A — Capability matrix template

| Data point | Observed source | Reliability | Fallback |
| ---------- | --------------- | ----------- | -------- |
| Player death | Combat log | To measure | Manual review/import |
| Lethal source/spell | Recent damage + death context | To measure | Unknown cause |
| Dungeon identity | Instance/run markers | To measure | Name/game-ID mapping |
| Key level | Run metadata | To measure | Optional/manual |
| Run start/end | Log marker/state machine | To measure | Inactivity/manual close |
| Yeet classification | Yeetcraft website (ADR 001); future heuristics may suggest | Website authority | Default `death` on ingest |
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
../yeetcraft/contracts/companion/v1/    # canonical draft contract (reviewed, not implemented as an API)
../yeetcraft/contracts/companion/v1/IMPLEMENTATION_MAP.md  # WP5 Yeetcraft Phase 3 file map
```

**Companion (this repository):**

```text
./AGENTS.md
./docs/IMPLEMENTATION_PLAN.md
./docs/PHASE_2_FILE_MAP.md
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
- [PHASE_2_FILE_MAP.md](./PHASE_2_FILE_MAP.md)
- [COMBAT_LOG_FORMAT_V22.md](./COMBAT_LOG_FORMAT_V22.md)
- [COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md)
- [Synthetic fixture provenance](../testdata/logs/synthetic/README.md)
- [AGENTS.md](../AGENTS.md)
