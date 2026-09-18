# Character and encounter integration — handoff

Companion-side context for Yeetcraft's canonical character metadata work and
future **Nemesis Boss** support.

| Field | Value |
| --- | --- |
| Status | Phase 0 accepted; Phase 1 contract review is current |
| Owner of canonical character/encounter data | Yeetcraft repository |
| Owner of log parsing and local inference | `yeetcraft-companion` |
| Last updated | 2026-09-18 |

This document is not an API contract. The canonical companion contract lives in
the Yeetcraft repository at `../yeetcraft/contracts/companion/v1/` (WP1 Markdown
reviewed, not implemented). Companion producer mapping:
[`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md).

---

## Why this document exists

Yeetcraft is preparing to make WoW characters first-class server metadata while
the companion collects additional non-blocking log evidence. A future event pipeline will
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

## Phase 0 evidence accepted for MVP progression

Across two reviewed local retail 12.1.0 sessions:

- 1,266,271 complete log lines parsed;
- five completed Mythic+ runs detected;
- nineteen boss encounter windows detected;
- 43 player deaths reviewed as correct;
- 35 tracked deaths retained;
- eight untracked fifth-player deaths excluded;
- five tracked character GUIDs mapped to four tracked people;
- one tracked person used a Shadow Priest in one run and an Unholy Death Knight
  in another;
- twelve boss-context deaths and 31 trash deaths;
- a failed boss pull, full-party trash wipe, and repeated death after
  resurrection were handled;
- the second session produced 28 high-confidence and three medium-confidence
  primary causes;
- twelve real `Falling` events parsed successfully, all nonlethal.

Phase 0 was accepted on 2026-09-18 because no stop condition was triggered for
the fixed-group MVP. Environmental automation, abandonment, reload/restart, and
file mechanics remain explicit non-blocking evidence or Phase 2 work. See
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

**WP1 sequencing blocker:** v1 ingest is blocked until Yeetcraft implements
nullable unique `characters.guid` per
[`../yeetcraft/docs/CHARACTERS_AND_BOSS_NEMESIS.md`](../yeetcraft/docs/CHARACTERS_AND_BOSS_NEMESIS.md).
This is recorded in the canonical contract; it is **not implemented** yet.

---

## Manual classification authority

The MVP does not depend on automatic yeet inference:

1. Every accepted detected death starts as `death`.
2. A user may change it to `yeet`, return it to `death`, or mark it `ignored`.
3. An accepted event contributes to exactly one aggregate category.
4. Reclassification moves one count atomically and never increases total
   mistakes.
5. Cause confidence and future yeet heuristics may suggest review, but must not
   overwrite a user's confirmed choice.

Phase 1 must encode these transitions in the canonical contract and reconcile
them with existing manual aggregate edits.

---

## Fields the future contract will need

The eventual canonical Yeetcraft-owned contract should resolve or represent:

### Event identity

- schema version (WP2);
- client installation ID (diagnostics only; excluded from ID hashes — WP1);
- deterministic batch ID (`batchId` UUID in body — WP1);
- deterministic run ID (`clientRunId` recipe — WP1);
- deterministic death event ID (`clientEventId` recipe — WP1);
- occurrence timestamp (canonical RFC 3339 UTC — **Phase 2 gap**; see WP1 review).

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
- detector suggestion, if any;
- user-confirmed classification and correction state.

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

## Safe work while collecting the residual evidence backlog

Allowed companion-only tasks, when explicitly approved:

- detection tests for wipes, simultaneous deaths, empty cause buffers, and
  stale encounter prevention;
- pure in-memory session state tests for completed/abandoned/interrupted runs;
- privacy-safe run-level `logprobe` summaries;
- documentation of GUID-first configuration and review behavior;
- non-canonical JSON examples clearly labeled as contract drafts;
- Phase 0 expected-vs-detected test reports.

Non-blocking real-evidence gaps:

- lethal environmental and knockback/void evidence for future suggestions;
- exact abandonment/restart behavior;
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

- [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md)
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

## Suggested prompt for the current Phase 1 contract session

```text
Phase 0 is accepted. Plan only; do not implement migrations or uploads.

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
- manual death/yeet/ignored classification and correction transitions;
- manual aggregate reconciliation.

Keep canonical contract files in Yeetcraft only. Do not commit real GUIDs,
credentials, raw logs, or personal data. Produce separate file maps and
validation plans for each repository.
```

