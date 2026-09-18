# Companion v1 contract — WP1 producer review

| Field | Value |
| ----- | ----- |
| Status | Phase 1 review notes — not a contract |
| Canonical spec | `../yeetcraft/contracts/companion/v1/` (Yeetcraft-owned) |
| Work package | WP1 only |
| Last updated | 2026-09-18 |

This document maps **WP1 normative decisions** from the Yeetcraft canonical
contract to **current companion code** (`internal/parser`, `internal/detection`,
`cmd/logprobe`). It is evidence for Phase 1 review, not a second source of
truth.

Payload field names, request envelopes, and acknowledgement codes are
**deferred to WP2/WP3**. Where a WP1 decision implies a future field, the
mapping names the conceptual contract input and marks payload shape as deferred.

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

## Field mapping — envelope and batch (WP1 semantics)

| Contract concept | WP1 decision | Current producer | Status |
| ---------------- | ------------ | ---------------- | ------ |
| Endpoint | `POST /api/companion/v1/deaths/batch` | None | Not implemented |
| Auth | `COMPANION_API_KEY`, fail-closed 503 | None | Not implemented |
| `batchId` | UUID in JSON body; no `Idempotency-Key` header | None | Deferred to WP2 payload + Phase 4 uploader |
| `schemaVersion` | Must agree with path `v1` | None | Deferred to WP2 |
| `installationId` | Diagnostics only; excluded from ID hashes | None | Phase 2 gap |
| Batch replay / `batch_conflict` | Server-side fingerprint | None | Phase 4 |
| Per-event `duplicate` / `event_id_conflict` | `clientEventId` boundary | None | Phase 2 ID computation + Phase 4 |

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

Contract implication: Phase 2 must either attach deaths only to bounded runs,
hold out-of-run deaths for review, or omit them from upload batches — **payload
rules deferred to WP2**.

---

## Field mapping — encounter context

| Contract concept (WP2 payload) | Current producer | Status |
| ------------------------------ | ---------------- | ------ |
| Journal encounter ID | `EncounterContext.EncounterID` from typed `ENCOUNTER_START` | In-memory |
| Encounter name (evidence) | `EncounterContext.EncounterName` | In-memory; redaction rules apply on upload |
| Active encounter at death | `EncounterContext.Active` | Set true between START/END |
| Trash death (`encounter` null) | `Encounter.Active == false` at death | Observed for trash deaths |
| Stale encounter clearing | `ENCOUNTER_END` clears matching ID | Implemented in `encounterTracker` |

---

## Field mapping — cause evidence and privacy

| Contract concept | Current producer | Status |
| ---------------- | ---------------- | ------ |
| Ranked causes (max 3) | `rankCauses` stops at 3 unique keys | Aligned with planned limit |
| Contiguous ranks starting at 1 | `len(out)+1` in loop | Aligned |
| Spell ID / name | `DamageHit.SpellID`, `SpellName` from typed damage payloads | Available |
| Amount / overkill | `DamageHit.Amount`, `Overkill` | Available |
| Confidence | `causeConfidence` → high/medium/low | Detector evidence only; not classification authority |
| Source kind / creature IDs | Partial — `EnvironmentalType`; creature GUID in `SourceGUID` | Needs normalized `sourceType` + game ID (WP2) |
| Redact player/pet-owner GUIDs in causes | **Not redacted** — `SourceGUID` preserved | **Phase 2 gap** |
| Redact player names in causes | **Not redacted** — `SourceName` preserved | **Phase 2 gap** |
| Omit raw log lines | Parser only; no upload | Aligned |

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

---

## Next step

**WP2** — define run, encounter, death, ranked-cause payload fields and
cross-field invariants, then create `ingest-batch-request.schema.json` and
synthetic request examples in Yeetcraft `contracts/companion/v1/schema/` and
`examples/`.
