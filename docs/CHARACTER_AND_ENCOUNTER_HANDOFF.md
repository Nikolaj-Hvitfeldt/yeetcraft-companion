# Character and encounter integration — handoff

Companion-side context for Yeetcraft's canonical character metadata work and
future **Nemesis Boss** support.

| Field | Value |
| --- | --- |
| Status | Phase 0 evidence; no upload contract or server writes |
| Owner of canonical character/encounter data | Yeetcraft repository |
| Owner of log parsing and local inference | `yeetcraft-companion` |
| Last updated | 2026-08-24 |

This document is not an API contract. The canonical companion contract remains
planned for the Yeetcraft repository at `contracts/companion/v1/` and does not
exist yet.

---

## Why this document exists

Yeetcraft is preparing to make WoW characters first-class server metadata while
the companion continues collecting Phase 0 logs. A future event pipeline will
need to connect:

```text
combat-log Player GUID
  → configured tracked character
  → Yeetcraft character
  → Yeetcraft player/profile owner
```

Boss insights require a separate connection:

```text
active ENCOUNTER_START / ENCOUNTER_END window
  → journal encounter ID
  → canonical Yeetcraft encounter
  → accepted death event
  → player-level Nemesis Boss aggregation
```

The repositories remain independent and communicate only through a future
versioned HTTP API.

---

## Verified Phase 0 evidence

From one local retail 12.1.0 session:

- 403,044 complete log lines parsed;
- two completed Mythic+ runs detected;
- seven boss encounter windows detected;
- twelve of twelve user-confirmed player deaths detected;
- eight tracked deaths retained;
- four untracked fifth-player deaths excluded;
- five tracked character GUIDs mapped to four tracked people;
- one tracked person used a Shadow Priest in one run and an Unholy Death Knight
  in another;
- one boss-context death and eleven trash deaths were user-confirmed;
- primary lethal causes were user-confirmed as plausible.

This is promising but remains **partially verified** because environmental,
knockback/void, abandonment, reload/restart, and additional-run scenarios are
still outstanding. See
[`COMBAT_LOG_CAPABILITIES.md`](./COMBAT_LOG_CAPABILITIES.md).

---

## Current companion behavior

Implemented:

- roster-neutral V22 parsing in `internal/parser`;
- exact common source/destination GUID extraction;
- in-memory configured GUID filtering in `internal/detection`;
- run context from challenge metadata;
- encounter context from `ENCOUNTER_START` / `ENCOUNTER_END`;
- ranked damage causes;
- privacy-safe `cmd/logprobe --deaths --track-guid ...`.

Not implemented:

- persistent character configuration;
- SQLite;
- file watching/resumable offsets;
- upload client;
- review UI;
- server identity lookup;
- canonical contract fixtures;
- Wails.

Do not add those deferred systems merely because the Yeetcraft website gains a
character table. Follow the approved phase scope in
[`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md).

---

## Identity rules for both repositories

1. A Yeetcraft **player** is a person/profile owner.
2. A Yeetcraft **character** is one WoW character owned by a player.
3. One player may own multiple character GUIDs.
4. Combat-log GUID is the primary runtime identity signal.
5. Name/realm may cross-check a mapping but must not silently override GUID.
6. An unknown or ambiguous GUID must enter review; never guess ownership.
7. The untracked fifth player must not be auto-created or uploaded.
8. Raw player logs, real GUIDs, names, and realms remain local/private unless a
   later approved contract explicitly requires minimal structured fields.
9. Do not commit real GUID mappings to either repository.

The website may introduce nullable unique character GUIDs now, but companion
configuration and private server population remain separate later tasks.

---

## Fields the future contract will need

The eventual canonical Yeetcraft-owned contract should resolve or represent:

### Event identity

- schema version;
- client installation ID;
- deterministic batch ID;
- deterministic run ID;
- deterministic death event ID;
- occurrence timestamp.

### Character identity

- combat-log player GUID;
- character name and realm only if approved as necessary;
- server-resolved character ID;
- server-resolved player ID;
- explicit outcome for unknown/ambiguous/untracked GUIDs.

### Run and dungeon context

- challenge map ID;
- key level;
- start/end timestamps and completion state;
- optional dungeon name as evidence, not the canonical identity.

### Encounter context

- journal encounter ID;
- encounter name as evidence;
- active encounter at death time;
- nullable encounter for trash deaths.

### Cause evidence

- source GUID/type;
- spell ID/name;
- amount/overkill where retained;
- rank;
- confidence;
- review/classification status.

These are contract design inputs, not permission to implement uploads during
Phase 0.

---

## Nemesis Boss semantics

Recommended server-side meaning:

> The boss encounter during which a player has accumulated the most accepted
> deaths across all owned characters.

Companion implications:

- report the active encounter separately from the final damage source;
- a boss add may deliver the final hit while the death still belongs to the
  active boss encounter;
- report no encounter for trash deaths;
- do not compute the canonical nemesis locally;
- do not treat every environmental or displacement death as a yeet;
- preserve enough structured evidence for later server review.

The Yeetcraft server owns encounter catalogs, accepted event state, aggregation,
and tie rules.

---

## Safe work while waiting for more logs

Allowed companion-only tasks, when explicitly approved:

- detection tests for wipes, simultaneous deaths, empty cause buffers, and
  stale encounter prevention;
- pure in-memory session state tests for completed/abandoned/interrupted runs;
- privacy-safe run-level `logprobe` summaries;
- documentation of GUID-first configuration and review behavior;
- non-canonical JSON examples clearly labeled as contract drafts;
- Phase 0 expected-vs-detected test reports.

Still blocked by real evidence:

- reliable environmental/knockback/void classification;
- exact abandonment/restart behavior;
- production go/no-go across representative runs;
- final `UNIT_DIED` suffix semantics.

Still deferred by phase:

- SQLite and offsets;
- filesystem watcher;
- uploader and retry queue;
- Wails;
- server writes.

---

## Cross-repository boundary for the character task

The next website character implementation belongs in the Yeetcraft repository.
It may read this repository's docs as evidence, but it must not:

- import companion packages;
- read local companion files at runtime;
- copy private GUID mappings into committed seeds;
- implement the companion uploader;
- claim Nemesis Boss works before event ingest exists.

The companion repository should not be modified during a Yeetcraft-only
character metadata task unless the user explicitly adds companion scope.

---

## Next-session references

### In this repository

- [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md)
- [`COMBAT_LOG_CAPABILITIES.md`](./COMBAT_LOG_CAPABILITIES.md)
- [`YEETCRAFT_INTEGRATION.md`](./YEETCRAFT_INTEGRATION.md)
- `internal/parser/`
- `internal/detection/`
- `cmd/logprobe/`

### In the sibling Yeetcraft repository

- `docs/CHARACTERS_AND_BOSS_NEMESIS.md` — primary implementation brief
- `docs/ARCHITECTURE.md`
- `docs/API.md`
- `docs/TESTING.md`
- `frontend/src/data/player-characters.ts`
- `backend/db/schema.sql`

---

## Suggested prompt for a later cross-repository contract session

```text
Plan only; do not implement migrations or uploads.

Read Yeetcraft/docs/CHARACTERS_AND_BOSS_NEMESIS.md and
yeetcraft-companion/docs/CHARACTER_AND_ENCOUNTER_HANDOFF.md, plus both
repositories' architecture/integration docs.

Draft the canonical companion v1 contract ownership and review plan for:
- GUID-first character resolution;
- tracked/untracked outcomes;
- run and dungeon identity;
- nullable encounter attribution;
- ranked cause evidence;
- idempotent batch/event IDs;
- manual aggregate reconciliation.

Keep canonical contract files in Yeetcraft only. Do not commit real GUIDs,
credentials, raw logs, or personal data. Produce separate file maps and
validation plans for each repository.
```

