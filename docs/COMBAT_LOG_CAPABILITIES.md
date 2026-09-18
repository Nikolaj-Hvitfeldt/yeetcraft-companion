# Combat-log capabilities

Phase 0 capability matrix and evidence log for the Yeetcraft companion.

|                  |                                                                                |
| ---------------- | ------------------------------------------------------------------------------ |
| **Phase**        | Phase 0 accepted for MVP progression; residual evidence collection continues   |
| **Last updated** | 2026-09-18                                                                     |
| **Fixtures**     | Original synthetic corpus ([provenance](../testdata/logs/synthetic/README.md)) |

Yeetcraft tracks **deaths** and **yeets** per player per dungeon per season. The companion must derive that information from local combat-log files written by the WoW client on **one** logging PC. This document records what the log can support — not what the product wishes were true.

**Related docs:** [V22 format reference](./COMBAT_LOG_FORMAT_V22.md) ·
[IMPLEMENTATION_PLAN.md §8.2](./IMPLEMENTATION_PLAN.md#82-phase-0--combat-log-evidence-spike-bounded-poc) ·
[synthetic fixture provenance](../testdata/logs/synthetic/README.md) ·
[YEETCRAFT_INTEGRATION.md](./YEETCRAFT_INTEGRATION.md)

---

## Desired product capabilities (reference)

These are **product goals**, not verified log capabilities:

- Detect configured party members' deaths during Mythic+ runs.
- Attribute each death to a dungeon run and (where possible) boss encounter vs trash.
- Infer likely cause from recent combat-log context with stated confidence.
- Default detected deaths to ordinary deaths and allow explicit user
  reclassification to `yeet` or `ignored`; automatic yeet logic may suggest
  but does not decide.
- Bound runs (start, completion, abandonment) without inventing boundaries.
- Resume incremental reading across restarts, rotation, and truncation.

Nothing below is verified until Phase 0 evidence is recorded.

---

## Status vocabulary

Use exactly one status per capability row. A prepared fixture is not a passing
test. Update to **Synthetically tested** only after Phase 0B implementation
passes that fixture.

Limited Phase 0B may follow Phase 0A.1 before Phase 0A.2. A technical sub-
capability may reach **Synthetically tested** only when an implementation passes
an exact source-backed fixture. Shape-incomplete death scenarios must not be
promoted to passing success fixtures. Phase 0A.2 later validates and adjusts
Phase 0B rather than blocking all parser work.

| Status                         | Meaning                                                                                       |
| ------------------------------ | --------------------------------------------------------------------------------------------- |
| **Not investigated**           | No project evidence recorded                                                                  |
| **Documented**                 | External documentation records the field or behavior; no project implementation has tested it |
| **Synthetic fixture prepared** | An original fixture exists; no parser or other implementation has passed it                   |
| **Synthetically tested**       | A parser or other implementation passes the fixture                                           |
| **Partially verified**         | Real-log evidence exists, but gaps or edge cases remain                                       |
| **Verified with real log**     | Reproducible on representative real retail Mythic+ logs for the target build                  |
| **Not reliably available**     | Investigated; log-only capture on one client cannot supply this consistently                  |

**Confidence** (when status is not _Not investigated_): `none` · `low` · `medium` · `high` — how trustworthy the signal is for **automated** companion behavior.

Phase 0A.1 may assign at most **Synthetic fixture prepared**. No capability is
verified with a real log in this phase.

---

## Capability matrix

### Identity and roster

| Capability               | Desired outcome                                      | Expected event / evidence source                           | Status             | Confidence | Known limitations                                                                                | Required fixture / scenario             | Fallback behavior                                           |
| ------------------------ | ---------------------------------------------------- | ---------------------------------------------------------- | ------------------ | ---------- | ------------------------------------------------------------------------------------------------ | --------------------------------------- | ----------------------------------------------------------- |
| Player identity          | Map a death to one of four configured tracked people | GUID + configured mapping; optional name/realm cross-check | Partially verified | high       | One tracked person used two character GUIDs; mappings must remain explicit                       | Five completed runs across two sessions | `needs_review`; never guess from similar names              |
| Character name and realm | Display and disambiguate who died                    | `UNIT_DIED`, `SPELL_*`, or aura lines with name–realm      | Partially verified | high       | Special characters, connected realms, rename                                                     | Five completed runs across two sessions | Store log name; prompt review if unmapped                   |
| Player GUID              | Stable key for idempotency and mapping               | `Player-XXXX-…` tokens in combat-log fields                | Partially verified | high       | Representative evidence is still limited to the fixed friend group                               | Five completed runs across two sessions | Cannot auto-upload until GUID mapped                        |
| Class and specialization | Context for death review and future UI               | Class-specific `SPELL_CAST_SUCCESS` and aura lines         | Partially verified | medium     | Inferred from distinctive spells, not an authoritative roster field; not required for MVP counts | Two completed runs                      | Omit or show "unknown" in review                            |
| Role                     | Tank/healer/DPS context for review                   | Same as class; may not appear explicitly in log            | Not investigated   | —          | Role is not in Yeetcraft DB today; log may omit                                                  | Any party death                         | Omit; use Yeetcraft frontend character data later if synced |
| Group membership         | Know which units belong to the M+ group              | Configured GUIDs, unit flags, and repeated unit tokens     | Partially verified | high       | No authoritative roster event was established; explicit configured GUIDs remain the authority    | Five completed runs across two sessions | Restrict to configured GUID list; ignore outsiders          |

### Instance and run

| Capability                   | Desired outcome                     | Expected event / evidence source                                  | Status             | Confidence | Known limitations                                                                          | Required fixture / scenario      | Fallback behavior                                             |
| ---------------------------- | ----------------------------------- | ----------------------------------------------------------------- | ------------------ | ---------- | ------------------------------------------------------------------------------------------ | -------------------------------- | ------------------------------------------------------------- |
| Dungeon or instance identity | Map run to Yeetcraft dungeon        | `ZONE_CHANGE`, `MAP_CHANGE`, `CHALLENGE_MODE_*`, instance name/ID | Partially verified | high       | Name vs Yeetcraft canonical list may differ                                                      | Five completed M+ runs           | Manual dungeon selection in review; prefer game ID if present |
| Dungeon difficulty           | Confirm Mythic+ vs other content    | Difficulty flags in encounter/challenge events                    | Partially verified | high       | Wrong difficulty → ignore run                                                                    | Five completed M+ runs           | Exclude run from auto-upload                                  |
| Keystone level               | Store key level on run              | `CHALLENGE_MODE_START`                                            | Partially verified | high       | May be missing outside M+                                                                        | Five completed M+ runs           | Optional field; manual entry in review                        |
| Run start                    | Open a bounded run record           | `CHALLENGE_MODE_START`                                            | Partially verified | high       | Zeroed `CHALLENGE_MODE_END` records appeared immediately before starts and must be ignored       | Five completed M+ runs           | Mark run `candidate` until confirmed                          |
| Run completion               | Close run as completed              | `CHALLENGE_MODE_END`                                              | Partially verified | high       | `success=1` appeared on an overtime completion; treat as completion, not proof the key was timed | Five completed M+ runs           | Timeout-based `abandoned` if no completion                    |
| Run abandonment              | Close run without inventing success | Timeout, zone leave, disconnect, or interrupted session           | Not investigated   | —          | Hard to distinguish pause vs quit                                                          | Abandoned M+; disconnect mid-run | Keep events; mark run `abandoned`; review before upload       |

### Encounters and combat context

| Capability                | Desired outcome                          | Expected event / evidence source               | Status             | Confidence | Known limitations                                 | Required fixture / scenario        | Fallback behavior                                       |
| ------------------------- | ---------------------------------------- | ---------------------------------------------- | ------------------ | ---------- | ------------------------------------------------- | ---------------------------------- | ------------------------------------------------------- |
| Boss encounter start      | Set active boss for death attribution    | `ENCOUNTER_START`, journal encounter ID        | Partially verified | high       | Evidence includes completed and failed pulls, but only the fixed-group sessions | Nineteen real encounter windows | `encounter_id` null → dungeon-level only                |
| Boss encounter completion | Clear active boss                        | `ENCOUNTER_END`                                | Partially verified | high       | Stale encounter state must be cleared immediately                              | Nineteen real encounter windows | Clear active encounter; do not retain the previous boss |
| Trash combat              | Attribute deaths outside boss encounters | Absence of an active encounter + combat events | Partially verified | high       | Multi-pack trash may blur                                                     | Thirty-one reviewed trash deaths | Classify as trash death with null encounter             |

### Death and cause

| Capability                     | Desired outcome                      | Expected event / evidence source                                                    | Status             | Confidence | Known limitations                                                                                    | Required fixture / scenario                                                     | Fallback behavior                                        |
| ------------------------------ | ------------------------------------ | ----------------------------------------------------------------------------------- | ------------------ | ---------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Player death                   | Detect who died and when             | `UNIT_DIED` for configured GUIDs                                                    | Partially verified | high       | Forty-three reviewed deaths across five runs support the MVP; broader populations/builds remain open | Five completed runs across two sessions                                        | Pause upload if later evidence shows systematic misses   |
| Final damage source            | Identify last relevant damage dealer | Recent `SPELL_DAMAGE` / `SPELL_PERIODIC_DAMAGE` / `SWING_DAMAGE` before `UNIT_DIED` | Partially verified | high       | A five-player wipe included three deaths without a lethal-overkill hit; retain medium-confidence alternatives | Forty-three reviewed deaths                                             | Rank causes; accept unknown cause without losing death   |
| Spell or ability causing death | Human-readable ability name and ID   | Spell fields on damage events                                                       | Partially verified | high       | Second session produced 28 high-confidence and three medium-confidence primaries                     | Forty-three reviewed deaths                                                    | Store best rank; medium/low confidence remains reviewable |
| Boss attribution               | Link death to active boss encounter  | Active `ENCOUNTER_START` + death timestamp                                          | Partially verified | high       | Killing source may be an add; encounter and cause remain separate                                    | Twelve boss-context and thirty-one trash deaths                                | Separate boss encounter from `death_causes` source       |
| Environmental damage           | Detect environmental damage          | `ENVIRONMENTAL_DAMAGE`, fall, drowning, etc.                                        | Partially verified | medium     | Twelve real `Falling` events parsed, all nonlethal; lethal and void semantics remain unverified       | Twelve nonlethal falls; lethal scenario still open                             | Count a real death as `death`; user may reclassify       |
| Knockback or displacement      | Suggest yeet-like review evidence    | Knockback auras, `SPELL_AURA_APPLIED`, position-less inference                      | Not investigated   | —          | Rarely explicit; high false-positive risk                                                            | Knockback followed by death                                                     | Never auto-classify; user may change `death` to `yeet`   |
| Falling into the void          | Suggest void/edge review evidence    | Fall damage, environmental, no recent enemy hit                                     | Not investigated   | —          | Indistinguishable from some environmental deaths                                                     | Environmental or falling death                                                  | Default `death`; user may reclassify to `yeet`           |

### File mechanics

| Capability                             | Desired outcome                                           | Expected event / evidence source                                                        | Status               | Confidence | Known limitations                                                      | Required fixture / scenario                                                                                                                                                                                                                                                                                                                                          | Fallback behavior                                                       |
| -------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------------------------------------- | -------------------- | ---------- | ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Combat-log file rotation or truncation | Resume without duplicate or skipped events                | File size shrink, new `WoWCombatLog-*.txt`, byte offset + file identity                 | Not investigated     | —          | Unverified assumption: retail may rotate on a new logging session      | Rotation/truncation scenario                                                                                                                                                                                                                                                                                                                                         | Unique client event IDs + offset persistence; re-read policy in Phase 2 |
| Malformed CSV handling                 | Reject malformed input without crashing                   | Unterminated quoted field                                                               | Synthetically tested | high       | Technical behavior only; real-log malformed-input frequency is unknown | [`malformed-csv.txt`](../testdata/logs/synthetic/malformed-csv.txt)                                                                                                                                                                                                                                                                                                  | Count malformed line and continue                                       |
| Partial final-line handling            | Buffer an incomplete final record                         | Truncated event prefix                                                                  | Synthetically tested | high       | WoW append behavior is not verified                                    | [`truncated-line.txt`](../testdata/logs/synthetic/truncated-line.txt)                                                                                                                                                                                                                                                                                                | Retain partial bytes until append                                       |
| Unknown event handling                 | Preserve ingestion when a new event appears               | Unknown event token                                                                     | Synthetically tested | high       | Unknown semantics remain uninterpreted                                 | [`unknown-event.txt`](../testdata/logs/synthetic/unknown-event.txt)                                                                                                                                                                                                                                                                                                  | Count unknown event and continue                                        |
| Unsupported version handling           | Fail safely on unsupported or malformed format boundaries | `COMBAT_LOG_VERSION` with fictional version, malformed structure, or non-retail project | Synthetically tested | high       | Header envelope and real version transitions remain unresolved         | [`unsupported-version.txt`](../testdata/logs/synthetic/unsupported-version.txt), [`version-v22-then-unsupported.txt`](../testdata/logs/synthetic/version-v22-then-unsupported.txt), [`version-v22-then-malformed.txt`](../testdata/logs/synthetic/version-v22-then-malformed.txt), [`version-project-id-2.txt`](../testdata/logs/synthetic/version-project-id-2.txt) | Quarantine; do not continue V22 interpretation                          |

### Parser foundation (Phase 0B.1)

| Technical capability               | Status               | Evidence                                                                                                                                                                                         | Boundary                                                                                  |
| ---------------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- |
| CSV-aware tokenization             | Synthetically tested | [`csv-quoted-fields.txt`](../testdata/logs/synthetic/csv-quoted-fields.txt) and parser tests                                                                                                     | No event semantics inferred                                                               |
| Common-header extraction           | Synthetically tested | [`common-header-invalid-flags.txt`](../testdata/logs/synthetic/common-header-invalid-flags.txt), [`parser-smoke-valid.txt`](../testdata/logs/synthetic/parser-smoke-valid.txt)                   | Explicit documented event allowlist only                                                  |
| Bounded incremental line reading   | Synthetically tested | `internal/parser/line_reader_test.go`                                                                                                                                                            | No filesystem watching, rotation, or offsets                                              |
| Version-boundary quarantine        | Synthetically tested | [`version-v22-then-unsupported.txt`](../testdata/logs/synthetic/version-v22-then-unsupported.txt), [`version-v22-then-malformed.txt`](../testdata/logs/synthetic/version-v22-then-malformed.txt) | Unsupported, malformed, and non-retail boundaries fail closed; later V22 does not recover |
| Retail project validation          | Synthetically tested | [`version-project-id-2.txt`](../testdata/logs/synthetic/version-project-id-2.txt), [`version-project-id-non-integer.txt`](../testdata/logs/synthetic/version-project-id-non-integer.txt)         | Only documented retail `PROJECT_ID,1` activates V22                                       |
| Provisional signed-offset envelope | Synthetically tested | [`timestamp-signed-offset.txt`](../testdata/logs/synthetic/timestamp-signed-offset.txt)                                                                                                          | Shape matching only; not verified against a real 12.0+ log                                |
| Malformed category reporting       | Synthetically tested | CLI tests for CSV, version-header, and common-header counts                                                                                                                                      | Diagnostics expose counts only, never record contents                                     |

### Typed payload parsing (limited Phase 0B.2)

| Technical capability                                                  | Status               | Evidence                                                                                                          | Boundary                                                                               |
| --------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `SPELL_DAMAGE` and `RANGE_DAMAGE` typed payloads                      | Synthetically tested | [`typed-damage-v22.txt`](../testdata/logs/synthetic/typed-damage-v22.txt) and parser tests                        | Exact selected-reference advanced-enabled V22 layout only                              |
| `SWING_DAMAGE` typed payload and optional off-hand field              | Synthetically tested | [`typed-damage-v22.txt`](../testdata/logs/synthetic/typed-damage-v22.txt) and parser tests                        | Main-hand omission and explicit `nil`/`1` are structural states; no damage inference   |
| `ENVIRONMENTAL_DAMAGE` typed payload                                  | Synthetically tested | [`typed-damage-v22.txt`](../testdata/logs/synthetic/typed-damage-v22.txt) and parser tests                        | Event parsing does not establish a death or yeet                                       |
| Advanced combat-log block                                             | Synthetically tested | Typed parser field-offset and numeric-boundary tests                                                              | Ownership and source expectations are non-fatal diagnostics, not real-log verification |
| `ENCOUNTER_START`, `ENCOUNTER_END`, and `CHALLENGE_MODE_END` payloads | Synthetically tested | [`typed-metadata-v22.txt`](../testdata/logs/synthetic/typed-metadata-v22.txt) and parser tests                    | Neutral metadata parsing only; visibility and run semantics unverified                 |
| Structural typed failures vs source-expectation diagnostics           | Synthetically tested | [`typed-payload-invalid-v22.txt`](../testdata/logs/synthetic/typed-payload-invalid-v22.txt), parser and CLI tests | Errors remove typed payloads; diagnostics retain them; counters expose no raw values   |
| `CHALLENGE_MODE_START` typed payload                                  | Documented, excluded | Selected reference declares an integer array but not exact raw CSV serialization                                  | Event remains recognized with `TypedStatusNotApplicable`                               |

### Documented format references

Exact line, header, common-field, advanced-block, and event layouts are kept in
[COMBAT_LOG_FORMAT_V22.md](./COMBAT_LOG_FORMAT_V22.md). The timestamp envelope
has a documented source conflict, so no timestamped fixture is a valid parser
success test in Phase 0A.1.

---

## Status summary (Phase 0 accepted)

- Parser-foundation and selected typed-parsing technical capabilities remain
  **Synthetically tested**, with Phase 0A.2 retail adjustments for decimal
  damage-school tokens and float `CHALLENGE_MODE_END` timer fields.
- **`internal/detection`** provides in-memory run/encounter context, recent
  incoming-damage buffers, ranked cause candidates, and death-candidate output.
  Exercised synthetically and against two local retail logs via
  `logprobe --deaths`.
- Player death, boss encounter boundaries, run metadata, and ranked likely cause
  are **Partially verified** across two retail 12.1.0 sessions: five completed
  keys, nineteen encounter windows, and 43 reviewed player deaths.
- GUID filtering retained 35 tracked deaths and excluded eight untracked
  fifth-player deaths. Five tracked character GUIDs map to four tracked people
  because one person used a different character between runs.
- The second session added a failed boss pull, repeated death after
  resurrection, a five-player trash wipe, an overtime completion, and twelve
  real but nonlethal falling-damage events.
- Manual review is the MVP classification authority: accepted deaths default to
  `death` and may be changed by a user to `yeet` or `ignored`.
- Phase 0 was **accepted for MVP progression on 2026-09-18**. Automatic yeet
  suggestions, abandonment, reload/restart continuity, and file mechanics
  remain explicit non-blocking evidence/Phase 2 work.

---

## Representative Phase 0 scenarios

Each scenario must produce at least one anonymized fixture slice (or a documented reason it could not). Link fixtures in the [Evidence log](#evidence-log) when created.

| #   | Scenario                               | Primary capabilities exercised                     | Fixture name (planned)                                                                                                                                       | Status                      |
| --- | -------------------------------------- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------- |
| 1   | Ordinary player death during trash     | Player death, trash combat, group membership, GUID | Local gitignored sessions + [`spell-damage-death.txt`](../testdata/logs/synthetic/spell-damage-death.txt), [`swing-damage-death.txt`](../testdata/logs/synthetic/swing-damage-death.txt) | Partially verified with real logs |
| 2   | Player death during a boss encounter   | Boss start, boss attribution, player death         | Local gitignored sessions + [`boss-context-death.txt`](../testdata/logs/synthetic/boss-context-death.txt) | Partially verified with real logs |
| 3   | Death after several damage events      | Final damage source, spell/ability, confidence     | Local gitignored sessions                                                                                                                                    | Partially verified with real logs |
| 4   | Environmental or falling death         | Environmental damage, falling/void, classification | Twelve real nonlethal falls + [`environmental-death.txt`](../testdata/logs/synthetic/environmental-death.txt)                                                | Damage verified; lethal case open |
| 5   | Knockback followed by death            | Knockback/displacement, possible yeet              | `knockback-death.txt`                                                                                                                                        | Non-blocking; not collected |
| 6   | Completed Mythic+ run                  | Run start/completion, dungeon ID, key level        | Five local gitignored runs                                                                                                                                   | Partially verified with real logs |
| 7   | Abandoned Mythic+ run                  | Run abandonment, incomplete boundaries             | `mplus-abandoned.txt`                                                                                                                                        | Phase 2 evidence backlog |
| 8   | Disconnect or reload during a run      | Run abandonment, missed events, recovery           | `disconnect-mid-run.txt`                                                                                                                                     | Phase 2 evidence backlog |
| 9   | Combat-log file rotation or truncation | File rotation/truncation, incremental read         | `log-rotate.txt`                                                                                                                                             | Phase 2 evidence backlog |
| 10  | Unknown or newly introduced event type | Parser resilience, unknown event handling          | [`unknown-event.txt`](../testdata/logs/synthetic/unknown-event.txt)                                                                                          | Synthetic fixture prepared  |

---

## Evidence rules

When recording observations (Phase 0 onward):

1. **Game version** — Record retail build/version string and collection date for every sample.
2. **Advanced Combat Logging** — Record whether it was enabled and any relevant WoW settings.
3. **Anonymized excerpts only** — Small slices in [`testdata/logs/`](../testdata/logs/); redact real names if needed.
4. **No full raw logs in git** — Complete production logs stay on the collector's PC (see [`.gitignore`](../.gitignore)).
5. **No credentials or chat** — Strip unrelated `CHAT_MSG_*` and account identifiers from fixtures.
6. **Link evidence to fixtures** — Each matrix status change cites fixture path(s) and scenario #.
7. **Regression linkage** — When parser tests exist, each fixture gets a matching test and golden output reference.

### Evidence log

| Date       | Build                                | ACL enabled                       | Scenario #                         | Fixture                                                                                                                                                                                                                                               | Observation summary                                                                                                                                                          | Matrix rows updated                                                       |
| ---------- | ------------------------------------ | --------------------------------- | ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| 2026-08-26 | 12.1.0 retail                        | Header `1`                        | Local retail M+ session            | `local-data/raw-logs/` (not committed)                                                                                                                                                                                                                | User reviewed three completed runs and 31 deaths as correct: 27 tracked, four untracked, 11 boss-context, 20 trash, 28 high-confidence and three medium-confidence causes; also observed a failed boss pull, resurrection/redeath, full trash wipe, overtime completion, and twelve nonlethal falls | Run/encounter/death/cause rows expanded; environmental damage partially verified |
| 2026-08-19 | 12.1.0 retail                        | Header `0`, advanced rows present | Local retail M+ session            | `local-data/raw-logs/` (not committed)                                                                                                                                                                                                                | User confirmed 12/12 deaths, both rosters, one boss death, and plausible primary causes; configured GUID filtering retained 8 tracked deaths and excluded 4 untracked deaths | Identity/roster, run, encounter, death, and cause rows partially verified |
| 2026-07-30 | V22 docs target 12.0+                | Not applicable                    | Synthetic preparation              | [`synthetic/`](../testdata/logs/synthetic/README.md)                                                                                                                                                                                                  | Original fixtures prepared; timestamp envelope and `UNIT_DIED` suffix unresolved; no parser or real-log validation                                                           | Format-related rows only                                                  |
| 2026-07-30 | V22 docs target 12.0+                | Not applicable                    | Phase 0B.1 technical tests         | [`synthetic/`](../testdata/logs/synthetic/README.md)                                                                                                                                                                                                  | Bounded streaming, CSV, common-header, unknown, malformed, partial-tail, and version-quarantine tests pass; no death inference                                               | File mechanics and parser foundation only                                 |
| 2026-07-30 | V22 selected reference targets 12.0+ | Not applicable                    | Limited Phase 0B.2 technical tests | [`typed-damage-v22.txt`](../testdata/logs/synthetic/typed-damage-v22.txt), [`typed-metadata-v22.txt`](../testdata/logs/synthetic/typed-metadata-v22.txt), [`typed-payload-invalid-v22.txt`](../testdata/logs/synthetic/typed-payload-invalid-v22.txt) | Seven selected layouts parse into typed payloads; primitive failures and non-fatal diagnostics are separately counted; no real log or death semantics                        | Typed parser rows only                                                    |

---

## Phase 0A.2 real-log validation checklist

No item below was completed in Phase 0A.1. Phase 0A.2 validates and adjusts
Phase 0B; it does not block all limited Phase 0B parser work.

- [x] Verify that one logger sees all party deaths in the observed session.
- [x] Verify filtering between configured tracked character GUIDs and each
      run's untracked fifth party member.
- [x] Verify dungeon and run boundaries for five completed runs.
- [x] Verify encounter boundaries for nineteen encounter windows, including a
      failed pull followed by a successful pull.
- [x] Verify ordering of recent damage before 43 reviewed deaths, including a
      wipe and repeated death after resurrection.
- [x] Compare final damage source with user-confirmed plausible causes.
- [x] Confirm real `Falling` damage visibility (twelve nonlethal events).
- [ ] Capture lethal environmental and knockback/void deaths (non-blocking
      future suggestion research).
- [ ] Exercise file append, buffering, and truncation.
- [ ] Compare real lines field-by-field with synthetic fixtures, including
      timestamp envelope, flags, advanced block, suffixes, and ordering.

---

## Residual evidence backlog after Phase 0 acceptance

Source-backed Phase 0B parser behavior is implemented and tested for:

- CSV-aware tokenization;
- version-header CSV payload;
- unsupported-version behavior;
- common-header extraction;
- exact source-backed damage event payloads;
- advanced-block extraction where the layout is exact;
- unknown and malformed input;
- partial-line buffering.

The following remain open but do not block Phase 1:

- abandoned-run closure;
- `/reload`, full restart, and multi-file run continuity;
- live append, rotation, truncation, and persisted offsets;
- lethal environmental and knockback/void evidence for automatic suggestions;
- exact unresolved V22 suffix/unknown-field semantics.

Shape-incomplete death scenarios must not be promoted to passing success
fixtures.

---

## Phase 0 success criteria

Phase 0 was accepted for MVP progression after the following go/no-go criteria
were evidenced or bounded with explicit fallback behavior:

- [x] Representative samples can be read with the bounded streaming scanner;
      live append/offset recovery is assigned to Phase 2.
- [x] Relevant event fields parse without crashing; unknown events are counted,
      not fatal.
- [x] Player deaths are reliable enough for the fixed-group MVP go decision.
- [x] Dungeon, completed-run, boss, and trash boundaries can be assessed.
- [x] Unsupported inferences have explicit fallback/review behavior.
- [x] The evidence-based **go** decision is recorded here and in the
      implementation plan.

**Phase 0 accepted: 2026-09-18.** Evidence collection continues without
blocking Phase 1.

---

## Stop conditions

Reopen the Phase 0 go decision and pause migrations/uploads if later evidence
shows:

| Condition                                   | Indicator                                                                 |
| ------------------------------------------- | ------------------------------------------------------------------------- |
| Player deaths cannot be identified reliably | Missed deaths in most representative runs for tracked party               |
| Player identity cannot be mapped safely     | GUID/name ambiguity would cause wrong Yeetcraft player assignment         |
| Dungeon context cannot be recovered         | No stable instance/game ID and no reasonable manual fallback              |
| Death causes are too ambiguous              | Majority of deaths would be `unknown` with no useful review signal        |
| Required logging behavior is impractical    | Advanced Combat Logging or setup burden unacceptable for the friend group |

If stopped: document outcome here, update [IMPLEMENTATION_PLAN.md §8.2 stop condition](./IMPLEMENTATION_PLAN.md#82-phase-0--combat-log-evidence-spike-bounded-poc), and re-evaluate multi-client logging or manual import — **not** an addon implementation in this phase.

---

## Unresolved questions

Track these after Phase 0; do not treat them as verified until evidenced or
resolved during contract/session design.

| #   | Question                                                                                                               | Blocks                         |
| --- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| 1   | Which evidence is strong enough to suggest a possible yeet? Manual user classification remains authoritative.         | Future classification automation |
| 2   | Can one M+ run span **multiple combat-log files**?                                                                     | Run ID recipe, offset strategy |
| 3   | Which signals distinguish an abandoned run from a paused or interrupted run?                                          | Phase 2 session state          |
| 4   | Does one logger remain reliable across future builds and less representative group configurations?                     | Production monitoring          |
| 5   | How should run identity continue across reload/restart/file rotation?                                                   | Phase 1 ID contract / Phase 2  |
| 6   | How should GUID-first characters map to Yeetcraft players without exposing private identity data publicly?             | Phase 1 identity contract      |

---

## External references

The source bibliography, access dates, version claims, and format conflicts are
centralized in [COMBAT_LOG_FORMAT_V22.md](./COMBAT_LOG_FORMAT_V22.md). Do not
copy unverified event semantics into this matrix or treat a documented field as
verified product behavior.
