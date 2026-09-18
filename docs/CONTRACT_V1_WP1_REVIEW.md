# Companion v1 contract — WP1 producer review

| Field | Value |
| ----- | ----- |
| Status | Phase 1 review notes — not a contract |
| Canonical spec | `../yeetcraft/contracts/companion/v1/` (Yeetcraft-owned) |
| Work package | WP1 + WP2 field mapping + WP3 acknowledgement review |
| Last updated | 2026-09-18 |

This document maps **WP1 normative decisions** from the Yeetcraft canonical
contract to **current companion code** (`internal/parser`, `internal/detection`,
`cmd/logprobe`). It is evidence for Phase 1 review, not a second source of
truth.

WP1 normative decisions are frozen. **WP2** field names below map to the canonical
request schema at
[`../yeetcraft/contracts/companion/v1/schema/ingest-batch-request.schema.json`](../yeetcraft/contracts/companion/v1/schema/ingest-batch-request.schema.json).
WP3 acknowledgement semantics are frozen in canonical
[`CONTRACT.md`](../yeetcraft/contracts/companion/v1/CONTRACT.md#acknowledgement-semantics).

---

## Review method

| Source | Role |
| ------ | ---- |
| [`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../yeetcraft/contracts/companion/v1/CONTRACT.md) | Locked WP1 semantics |
| `internal/parser/` | Neutral V22 parsing, provisional timestamp envelope |
| `internal/detection/` | In-memory run/encounter context, damage buffers, death candidates |
| `cmd/logprobe/` | CLI integration of parser + detection (`--deaths`, `--track-guid`) |

Verified absences: `internal/storage/` (SQLite stub), `internal/uploader/`
(stub), `internal/session/` (stub), persistent config, file watcher offsets.

---

## WP1 cross-cutting gaps

| Contract requirement | Current companion state | Gap severity |
| -------------------- | ----------------------- | ------------ |
| Canonical RFC 3339 UTC instants for ID hashing | `DeathCandidate.Timestamp` and damage hits store `parser.Envelope.Raw` (timezone-less or offset-bearing string); `TryParseEnvelopeTimestamp` exists but is not used by detection | **Blocker for ID recipes** |
| Persisted WoW-log timezone for timezone-less stamps | No configuration or persistence | **Blocker** |
| Persisted `clientRunId` / `clientEventId` / ordinals | No SQLite; IDs not computed | **Phase 2** |
| Full vs partial scan ID stability | Single-pass in-memory scan only; no offset resume | **Phase 2** |
| Fail-closed when tracked config absent | `NewTracker()` with empty GUID list sets `allPlayers=true` and records every player death | **Phase 2 behavior change** |
| Client-side untracked filter before upload | Filtering exists only via `--track-guid` CLI flags; no persisted roster | **Phase 2** |
| Cause privacy (no untracked/player GUIDs in causes) | `DamageHit` retains `SourceGUID`, `SourceName`; `logprobe` prints `source_guid` | **Phase 2 redaction** |
| Deaths outside active Mythic+ run | `observeDeath` does not require `Run.Active`; deaths with `run_active: 0` observed in `logprobe` output | **Phase 2 gating / review** |
| `installationId` diagnostics | Not generated or persisted | **Phase 2** (excluded from ID recipes per contract) |
| Per-run `seasonId` hint | Not produced; no server season list client | **Phase 2** |
| Server GUID → character resolution | No HTTP client | **Phase 4** (blocked on Yeetcraft `characters.guid`) |

### Sequencing blocker (Yeetcraft)

Companion v1 ingest cannot be implemented end-to-end until Yeetcraft adds
nullable unique `characters.guid` per
[`../yeetcraft/docs/CHARACTERS_AND_BOSS_NEMESIS.md`](../yeetcraft/docs/CHARACTERS_AND_BOSS_NEMESIS.md).
**Not implemented in either repository.**

---

## Field mapping — envelope and batch (WP2 wire names)

| Contract field | WP2 type | Current producer | Status |
| -------------- | -------- | ---------------- | ------ |
| Endpoint | `POST /api/companion/v1/deaths/batch` | None | Not implemented |
| Auth | `COMPANION_API_KEY`, fail-closed 503 | None | Not implemented |
| `schemaVersion` | integer `1` | None | Phase 4 uploader |
| `batchId` | UUID | None | Phase 4 uploader |
| `installationId` | optional UUID | None | Phase 2 gap |
| `events` | death event array | `Tracker.deaths` (in-memory only) | Phase 2 serialization |
| Batch replay / `batch_conflict` | server fingerprint | None | Phase 4 |
| Per-event `duplicate` / `event_id_conflict` | `clientEventId` boundary | None | Phase 2 IDs + Phase 4 |

---

## Field mapping — deterministic IDs

| Contract field / input | Producer today | Gap |
| ---------------------- | -------------- | --- |
| `challengeModeStartInstant` | `runTracker` does not record start timestamp; only map ID, key level, dungeon name from `CHALLENGE_MODE_START` fields | **No canonical instant** — must capture envelope time at start event (Phase 2) |
| `challengeMapId` | `RunContext.MapID` from `CHALLENGE_MODE_START` field index 2 (`internal/detection/run.go`) | Available in memory during scan |
| `keystoneLevel` | `RunContext.KeystoneLevel` from `CHALLENGE_MODE_START` field index 4 | Available in memory during scan |
| `clientRunId` hash | Not computed | Phase 2: implement NFC + length-prefixed SHA-256 per contract |
| `victimGuid` | `DeathCandidate.VictimGUID` from `UNIT_DIED` `DestGUID` when `isPlayerGUID` | Available |
| `deathInstant` | `DeathCandidate.Timestamp` = raw envelope string, not RFC 3339 UTC | **Normalization gap** |
| `ordinal` | Not computed; no collision handling for same-instant deaths | Phase 2: order `UNIT_DIED` per victim at canonical instant |
| `clientEventId` hash | Not computed | Phase 2 |
| Season excluded from `clientRunId` | N/A locally | Contract locked; companion must not add season to hash |
| `installationId` excluded from IDs | N/A | Contract locked |

### Timestamp and timezone evidence

- `parser.SplitEnvelope` extracts a provisional timestamp string; shapes are
  documented as **not verified against real 12.0+ logs**
  (`internal/parser/envelope.go`).
- Retail sessions used raw envelope strings in detection output; canonical UTC
  normalization is an open Phase 2 task tied to persisted timezone config.
- DST ambiguity and invalid stamps must route to **local review**, not silent
  hash acceptance (contract WP1).

### Restart and partial-scan behavior

| Scenario | Current behavior | Contract expectation | Gap |
| -------- | ---------------- | -------------------- | --- |
| Process restart | Full file rescan from beginning; no persisted offsets | Reuse persisted IDs or hold events when prefix incomplete | Phase 2 (`logwatcher`, `storage`) |
| Partial tail / incomplete line | `parser.ScanReader` reports `incomplete_trailing`; logprobe exits code 3 | Do not assign new ordinals without complete prefix | Parser signals exist; no persistence |
| Full rescan with complete log | In-memory replay reproduces death order | Reconstruct ordinals from full `UNIT_DIED` sequence | Ordinal logic not implemented |
| Mid-file resume | Not implemented | Reuse stored IDs for already-seen events | Phase 2 |

---

## Field mapping — GUID-first identity

| Contract concept | Current producer | Status |
| ---------------- | ---------------- | ------ |
| `characterGuid` on wire (tracked only) | `DeathCandidate.VictimGUID` when `tracks()` returns true | Producer exists; upload path missing |
| Client-side tracked filter | `Tracker.tracked` map from `NewTracker(trackGUIDs...)`; CLI `--track-guid` | Prototype only; **empty list = track all players** (inverse of production fail-closed) |
| Omit untracked GUIDs from wire | Deaths for non-listed GUIDs skipped when filter configured | Works in logprobe; not persisted config |
| Omit names/realms from wire | Parser extracts names in `CommonHeader`; detection does not attach them to `DeathCandidate` | Aligned for death object; cause hits still carry `SourceName` |
| Fail closed without tracked config | Not implemented — empty GUID list tracks everyone | **Phase 2 required change** |
| Server `resolved` / `unknown` | No server | Yeetcraft Phase 3 |
| Server `ambiguous` | No server | Reserved; depends on unique `characters.guid` |
| Yeetcraft `characters.guid` prerequisite | No companion server lookup | **Cross-repo blocker** |

---

## Field mapping — season and dungeon resolution

| Contract concept | Current producer | Status |
| ---------------- | ---------------- | ------ |
| Challenge `MapID` | `RunContext.MapID` | In-memory evidence |
| Keystone level | `RunContext.KeystoneLevel` | In-memory evidence |
| Dungeon name (evidence only) | `RunContext.DungeonName` from `CHALLENGE_MODE_START` field 1 | Evidence only; not canonical server ID |
| Run active flag | `RunContext.Active`; cleared on successful `CHALLENGE_MODE_END` | Partial; abandonment not detected |
| Run start/end canonical instants | Not captured | Phase 2 |
| Completion state | `observeEnd` sets `active=false` on `Success` only | No explicit completed/abandoned enum |
| Per-run `seasonId` hint | Not produced | Phase 2 UI/config |
| Server timestamp season resolution | N/A | Yeetcraft Phase 3 (`starts_at`/`ends_at` not in schema today) |
| `dungeons.challenge_map_id` lookup | N/A | Yeetcraft Phase 3 (column absent today) |
| Unmapped map → `needs_review` | N/A | Server Phase 3 |
| `is_current` as ingest authority | N/A | Contract forbids; companion must not send current season as authority |

### Deaths outside active runs

`Tracker.observeDeath` does not check `Run.Active`. Deaths during trash between
keys, before `CHALLENGE_MODE_START`, or after completion retain
`death_N_run_active: 0` in logprobe output while still emitting
`victim_guid` and causes.

Contract implication (WP2): deaths without a verifiable Mythic+ start context
must not appear in upload batches (`run_context_incomplete`). Phase 2 must
gate `observeDeath` output before serialization.

---

## Field mapping — death event (WP2 wire names)

| Contract field | WP2 type | Current producer | Status |
| -------------- | -------- | ---------------- | ------ |
| `clientEventId` | `sha256:` digest | Not computed | Phase 2 |
| `characterGuid` | player GUID | `DeathCandidate.VictimGUID` when tracked | Producer exists |
| `deathInstant` | canonical instant | `DeathCandidate.Timestamp` (raw envelope) | **Normalization gap** |
| `ordinal` | integer ≥ 0 | Not computed | Phase 2 |
| `category` | optional `"death"` | Not set | Omit on upload (server default) |
| `run` | run object | See run table | Partial |
| `encounter` | object or `null` | See encounter table | Partial |
| `causes` | ranked-cause array | `DeathCandidate.Causes` | Needs redaction + mapping |

## Field mapping — run object (`run`)

| Contract field | WP2 type | Current producer | Status |
| -------------- | -------- | ---------------- | ------ |
| `clientRunId` | `sha256:` digest | Not computed | Phase 2 |
| `challengeModeStartInstant` | canonical instant | Not captured at `CHALLENGE_MODE_START` | **Blocker** |
| `challengeMapId` | integer | `RunContext.MapID` | In-memory |
| `keystoneLevel` | integer | `RunContext.KeystoneLevel` | In-memory |
| `seasonId` | optional UUID hint | Not produced | Phase 2 UI/config |

## Field mapping — encounter (`encounter`)

| Contract field | WP2 type | Current producer | Status |
| -------------- | -------- | ---------------- | ------ |
| `encounter` | `null` or object | `EncounterContext.Active` at death | Trash → `null` |
| `encounter.encounterId` | integer ≥ 1 | `EncounterContext.EncounterID` | In-memory when active |
| Encounter name on wire | **omitted** | `EncounterContext.EncounterName` | Must not upload |
| Stale encounter clearing | N/A | `ENCOUNTER_END` clears matching ID | Implemented |

---

## Field mapping — ranked cause (`causes[]`)

| Contract field | WP2 type | Current producer | Status |
| -------------- | -------- | ---------------- | ------ |
| `rank` | 1–3 | `CauseCandidate.Rank` | Aligned |
| `sourceType` | `spell` \| `range` \| `melee` \| `environmental` | `DamageHit.EventType` | Phase 2 mapping |
| `spellId` | conditional integer | `DamageHit.SpellID` | Available for spell/range |
| `creatureId` | optional integer | Not extracted (GUID in `SourceGUID`) | Phase 2 template ID extraction |
| `environmentalType` | conditional string | `DamageHit.EnvironmentalType` | Available for environmental |
| `amount` | integer ≥ 0 | `DamageHit.Amount` | Available |
| `overkill` | integer ≥ 0 | `DamageHit.Overkill` | Available |
| `confidence` | high/medium/low | `causeConfidence` | Aligned; not classification authority |
| Spell name on wire | **omitted** | `DamageHit.SpellName` | Must not upload |
| Source GUID/name on wire | **forbidden** | `DamageHit.SourceGUID`, `SourceName` | **Phase 2 redaction** |
| Omit raw log lines | N/A | Parser only | Aligned |

---

## Field mapping — classification (ingest vs website)

| Contract concept | Current producer | Status |
| ---------------- | ---------------- | ------ |
| Ingest category | Contract: server defaults to `death`; ingest omits or constrains category | No upload; detection does not assign category |
| `yeet` / `ignored` | Website post-ingest correction (WP4) | Not companion wire v1 |
| User manual classification | `internal/review/` stub | Phase 5 |
| Detector yeet suggestion | Not implemented | Out of MVP scope |

---

## Code references (evidence)

Tracked filter and death accumulation:

```16:34:c:\Users\nhvit\Repositories\yeetcraft-companion\internal\detection\tracker.go
// NewTracker returns a tracker. When trackGUIDs is empty, every player GUID
// death is recorded; otherwise only listed GUIDs match.
func NewTracker(trackGUIDs ...string) *Tracker {
	// ...
	if len(trackGUIDs) == 0 {
		t.allPlayers = true
		return t
	}
```

Raw timestamp on death (not canonical instant):

```88:95:c:\Users\nhvit\Repositories\yeetcraft-companion\internal\detection\tracker.go
	t.deaths = append(t.deaths, DeathCandidate{
		LineNumber: event.LineNumber,
		Timestamp:  event.Envelope.Raw,
		VictimGUID: victim,
		Run:        t.run.context(),
		Encounter:  t.encounter.context(),
		Causes:     rankCauses(hits),
	})
```

Run context without start instant:

```30:46:c:\Users\nhvit\Repositories\yeetcraft-companion\internal\detection\run.go
func (r *runTracker) observeStart(fields []string) {
	// ...
	r.active = true
	r.mapID = mapID
	r.dungeonName = fields[1]
	r.keystoneLevel = keyLevel
}
```

---

## WP1 acceptance checklist

- [x] Every WP1 contract section mapped to a producer or explicit gap
- [x] Raw/timezone-less timestamps documented
- [x] Absent installation persistence documented
- [x] Restart/partial-scan behavior documented
- [x] Deaths outside active runs documented
- [x] Fail-closed tracked configuration gap documented (prototype tracks all)
- [x] Cause redaction gap documented
- [x] `characters.guid` sequencing blocker recorded (not implemented)

## WP2 acceptance checklist

- [x] Every WP2 request field mapped to a producer or explicit gap
- [x] `sourceType` enum mapped from `DamageHit.EventType`
- [x] Omitted wire fields documented (`encounterName`, spell names, source GUID/name)
- [x] Run gating rule aligned with `run_context_incomplete`
- [x] No companion-side schema copy; canonical checksum fixtures deferred to Phase 2/3

---

## WP3 — credential boundary and retry/acknowledgement gaps

Canonical WP3 spec:
[`../yeetcraft/contracts/companion/v1/CONTRACT.md`](../yeetcraft/contracts/companion/v1/CONTRACT.md)
(acknowledgement, limits, auth, error taxonomy).

| Contract requirement | Current companion state | Gap |
| -------------------- | ----------------------- | --- |
| `COMPANION_API_KEY` separate from browser `API_KEY` | No HTTP client | Phase 4 uploader |
| Fail-closed **503** `companion_api_unconfigured` | No server probe | Phase 4 must surface operator-actionable message without logging credentials |
| `X-API-Key` / `Authorization: Bearer` only | None | Phase 4 |
| No `?token=` on API routes | N/A locally | Phase 4 must not add query-token auth |
| Persist `batchId` + await HTTP **200** ordered results | No uploader or SQLite ack state | Phase 4 + Phase 2 storage |
| Retry **5xx** / **429** with same `batchId` + body | No retry policy | Phase 4 |
| Treat **409** `batch_conflict` / `event_id_conflict` as non-retryable | None | Phase 4 must halt and surface for review |
| Treat **422** / **401** / **413** / **415** as non-retryable config/payload fixes | None | Phase 4 |
| Map per-event `accepted` / `duplicate` / `needs_review` / `rejected` | None | Phase 4 parses [`ingest-batch-response.schema.json`](../yeetcraft/contracts/companion/v1/schema/ingest-batch-response.schema.json) |
| `installationId` diagnostics only (not auth) | Not generated | Phase 2; must not be sent as a credential |
| Route-level rate limit handling (**429**) | None | Phase 4 backoff |
| Never log credentials or full payloads | `logprobe` prints cause GUIDs today | Phase 2 redaction + Phase 4 transport logging policy |

### WP3 acceptance checklist

- [x] WP3 auth credential boundary recorded (separate key; no `API_KEY` reuse)
- [x] Retry vs non-retryable HTTP outcomes mapped to current gaps
- [x] Batch idempotency replay behavior documented as Phase 4 responsibility
- [x] Per-event outcome parsing deferred to Phase 4 (no companion schema copy)
- [x] No contradiction with frozen WP1/WP2 ID or payload shapes

## Next step

**WP4** — Yeetcraft correction ADRs and revision-protected adjustment ledger
(server context only; not companion wire schema).
