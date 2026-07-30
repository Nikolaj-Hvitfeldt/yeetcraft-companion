# Combat-log capabilities

Phase 0 capability matrix and evidence log for the Yeetcraft companion.

| | |
| --- | --- |
| **Phase** | 0A.1 and 0B.1 complete — Phase 0A.2 real-log validation pending |
| **Last updated** | 2026-07-30 |
| **Fixtures** | Original synthetic corpus ([provenance](../testdata/logs/synthetic/README.md)) |

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
- Distinguish ordinary deaths from yeet-like deaths only when evidence supports it; otherwise queue review.
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

| Status | Meaning |
| ------ | ------- |
| **Not investigated** | No project evidence recorded |
| **Documented** | External documentation records the field or behavior; no project implementation has tested it |
| **Synthetic fixture prepared** | An original fixture exists; no parser or other implementation has passed it |
| **Synthetically tested** | A parser or other implementation passes the fixture |
| **Partially verified** | Real-log evidence exists, but gaps or edge cases remain |
| **Verified with real log** | Reproducible on representative real retail Mythic+ logs for the target build |
| **Not reliably available** | Investigated; log-only capture on one client cannot supply this consistently |

**Confidence** (when status is not *Not investigated*): `none` · `low` · `medium` · `high` — how trustworthy the signal is for **automated** companion behavior.

Phase 0A.1 may assign at most **Synthetic fixture prepared**. No capability is
verified with a real log in this phase.

---

## Capability matrix

### Identity and roster

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Player identity | Map a death to one of four configured tracked characters | GUID + configured mapping; optional name/realm cross-check | Not investigated | — | One logger may not see all party members; alts and name changes | Any death scenario with full party | `needs_review`; never guess from similar names |
| Character name and realm | Display and disambiguate who died | `UNIT_DIED`, `SPELL_*`, or aura lines with name–realm | Not investigated | — | Special characters, connected realms, rename | Death during trash or boss | Store log name; prompt review if unmapped |
| Player GUID | Stable key for idempotency and mapping | `Player-XXXX-…` tokens in combat-log fields | Documented | none | Presence and visibility for every party member require a real log | Any death scenario | Cannot auto-upload until GUID mapped |
| Class and specialization | Context for death review and future UI | `SPELL_CAST_SUCCESS`, aura, or roster-related events | Not investigated | — | Spec may require inference; not required for MVP counts | Boss or trash death | Omit or show "unknown" in review |
| Role | Tank/healer/DPS context for review | Same as class; may not appear explicitly in log | Not investigated | — | Role is not in Yeetcraft DB today; log may omit | Any party death | Omit; use Yeetcraft frontend character data later if synced |
| Group membership | Know which units belong to the M+ group | Party roster events (unverified hypothesis), unit flags, or repeated unit tokens | Not investigated | — | `INSTANCE_ENCOUNTER_*` is not documented in the selected V22 reference; logger may only see subset of group | Completed M+ with four configured tracked characters | Restrict to configured GUID/name list; ignore outsiders |

### Instance and run

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Dungeon or instance identity | Map run to Yeetcraft dungeon | `ZONE_CHANGE`, `MAP_CHANGE`, `CHALLENGE_MODE_*`, instance name/ID | Not investigated | — | Name vs Yeetcraft canonical list may differ | Completed M+ run | Manual dungeon selection in review; prefer game ID if present |
| Dungeon difficulty | Confirm Mythic+ vs other content | Difficulty flags in instance/challenge events | Not investigated | — | Wrong difficulty → ignore run | Completed M+ run | Exclude run from auto-upload |
| Keystone level | Store key level on run | `CHALLENGE_MODE_START` or related metadata | Not investigated | — | May be missing outside M+ | Completed M+ run | Optional field; manual entry in review |
| Run start | Open a bounded run record | Challenge start, instance entry, or state-machine entry signals | Not investigated | — | Ambiguous start vs key insert | Completed or abandoned M+ | Mark run `candidate` until confirmed |
| Run completion | Close run as successful | `CHALLENGE_MODE_END`, completion chat, or timer events | Not investigated | — | Signal may arrive late or in next file | Completed M+ run | Timeout-based `abandoned` if no completion |
| Run abandonment | Close run without inventing success | Timeout, zone leave, disconnect, or interrupted session | Not investigated | — | Hard to distinguish pause vs quit | Abandoned M+; disconnect mid-run | Keep events; mark run `abandoned`; review before upload |

### Encounters and combat context

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Boss encounter start | Set active boss for death attribution | `ENCOUNTER_START`, journal encounter ID | Documented | none | Exact field layout is documented; real M+ visibility and semantics are not | [`boss-context-death.txt`](../testdata/logs/synthetic/boss-context-death.txt) | `encounter_id` null → dungeon-level only |
| Boss encounter completion | Clear active boss | `ENCOUNTER_END` | Documented | none | Exact field layout is documented; real M+ visibility and semantics are not | [`boss-context-death.txt`](../testdata/logs/synthetic/boss-context-death.txt) | Retain last known encounter until zone change |
| Trash combat | Attribute deaths outside boss encounters | Absence of active encounter + combat events | Not investigated | — | Multi-pack trash may blur | Trash death scenario | Classify as trash death with null encounter |

### Death and cause

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Player death | Detect who died and when | `UNIT_DIED` (and related) for tracked units | Documented | none | Exact V22 file suffix and one-logger visibility remain unresolved; death fixtures are shape-incomplete | All death scenarios | **Stop condition** if unreliable — see below |
| Final damage source | Identify last relevant damage dealer | Recent `SPELL_DAMAGE` / `SWING_*` / `ENVIRONMENTAL_*` before `UNIT_DIED` | Documented | none | Exact damage layouts are documented; ordering and semantic cause require a real log | Death after several damage events | Rank causes; accept `unknown` |
| Spell or ability causing death | Human-readable ability name and ID | Spell fields on damage events | Documented | none | Field availability is documented; attribution remains untested | Death after several damage events | Store best rank; low confidence → review |
| Boss attribution | Link death to active boss encounter | Active `ENCOUNTER_START` + death timestamp | Not investigated | — | Add death may be attributed to boss incorrectly | Boss death scenario | Separate boss encounter from `death_causes` source |
| Environmental damage | Detect environmental lethal damage | `ENVIRONMENTAL_DAMAGE`, fall, drowning, etc. | Documented | none | Exact event payload is documented; timestamp envelope and real semantics remain unresolved | [`environmental-death.txt`](../testdata/logs/synthetic/environmental-death.txt) | Category `unknown` or `possible_yeet`; review |
| Knockback or displacement | Evidence for yeet-like deaths | Knockback auras, `SPELL_AURA_APPLIED`, position-less inference | Not investigated | — | Rarely explicit; high false-positive risk | Knockback followed by death | `possible_yeet` + review; never auto-yeet without policy |
| Falling into the void | Evidence for void/edge deaths | Fall damage, environmental, no recent enemy hit | Not investigated | — | Indistinguishable from some environmental deaths | Environmental or falling death | Conservative `death` or `possible_yeet`; review |

### File mechanics

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Combat-log file rotation or truncation | Resume without duplicate or skipped events | File size shrink, new `WoWCombatLog-*.txt`, byte offset + file identity | Not investigated | — | Unverified assumption: retail may rotate on a new logging session | Rotation/truncation scenario | Unique client event IDs + offset persistence; re-read policy in Phase 2 |
| Malformed CSV handling | Reject malformed input without crashing | Unterminated quoted field | Synthetically tested | high | Technical behavior only; real-log malformed-input frequency is unknown | [`malformed-csv.txt`](../testdata/logs/synthetic/malformed-csv.txt) | Count malformed line and continue |
| Partial final-line handling | Buffer an incomplete final record | Truncated event prefix | Synthetically tested | high | WoW append behavior is not verified | [`truncated-line.txt`](../testdata/logs/synthetic/truncated-line.txt) | Retain partial bytes until append |
| Unknown event handling | Preserve ingestion when a new event appears | Unknown event token | Synthetically tested | high | Unknown semantics remain uninterpreted | [`unknown-event.txt`](../testdata/logs/synthetic/unknown-event.txt) | Count unknown event and continue |
| Unsupported version handling | Fail safely on unsupported or malformed format boundaries | `COMBAT_LOG_VERSION` with fictional version, malformed structure, or non-retail project | Synthetically tested | high | Header envelope and real version transitions remain unresolved | [`unsupported-version.txt`](../testdata/logs/synthetic/unsupported-version.txt), [`version-v22-then-unsupported.txt`](../testdata/logs/synthetic/version-v22-then-unsupported.txt), [`version-v22-then-malformed.txt`](../testdata/logs/synthetic/version-v22-then-malformed.txt), [`version-project-id-2.txt`](../testdata/logs/synthetic/version-project-id-2.txt) | Quarantine; do not continue V22 interpretation |

### Parser foundation (Phase 0B.1)

| Technical capability | Status | Evidence | Boundary |
| -------------------- | ------ | -------- | -------- |
| CSV-aware tokenization | Synthetically tested | [`csv-quoted-fields.txt`](../testdata/logs/synthetic/csv-quoted-fields.txt) and parser tests | No event semantics inferred |
| Common-header extraction | Synthetically tested | [`common-header-invalid-flags.txt`](../testdata/logs/synthetic/common-header-invalid-flags.txt), [`parser-smoke-valid.txt`](../testdata/logs/synthetic/parser-smoke-valid.txt) | Explicit documented event allowlist only |
| Bounded incremental line reading | Synthetically tested | `internal/parser/line_reader_test.go` | No filesystem watching, rotation, or offsets |
| Version-boundary quarantine | Synthetically tested | [`version-v22-then-unsupported.txt`](../testdata/logs/synthetic/version-v22-then-unsupported.txt), [`version-v22-then-malformed.txt`](../testdata/logs/synthetic/version-v22-then-malformed.txt) | Unsupported, malformed, and non-retail boundaries fail closed; later V22 does not recover |
| Retail project validation | Synthetically tested | [`version-project-id-2.txt`](../testdata/logs/synthetic/version-project-id-2.txt), [`version-project-id-non-integer.txt`](../testdata/logs/synthetic/version-project-id-non-integer.txt) | Only documented retail `PROJECT_ID,1` activates V22 |
| Provisional signed-offset envelope | Synthetically tested | [`timestamp-signed-offset.txt`](../testdata/logs/synthetic/timestamp-signed-offset.txt) | Shape matching only; not verified against a real 12.0+ log |
| Malformed category reporting | Synthetically tested | CLI tests for CSV, version-header, and common-header counts | Diagnostics expose counts only, never record contents |

### Documented format references

Exact line, header, common-field, advanced-block, and event layouts are kept in
[COMBAT_LOG_FORMAT_V22.md](./COMBAT_LOG_FORMAT_V22.md). The timestamp envelope
has a documented source conflict, so no timestamped fixture is a valid parser
success test in Phase 0A.1.

---

## Status summary (Phase 0B.1)

- Parser-foundation technical capabilities listed above are **Synthetically tested**.
- Shape-incomplete death scenarios remain no higher than **Documented** or
  **Synthetic fixture prepared**; no death capability was promoted.
- No real Mythic+ log was examined, so nothing is **Partially verified** or
  **Verified with real log**.
- Exact counts should be generated when the matrix is next revised rather than
  maintained manually.

---

## Representative Phase 0 scenarios

Each scenario must produce at least one anonymized fixture slice (or a documented reason it could not). Link fixtures in the [Evidence log](#evidence-log) when created.

| # | Scenario | Primary capabilities exercised | Fixture name (planned) | Status |
| - | -------- | ------------------------------ | ---------------------- | ------ |
| 1 | Ordinary player death during trash | Player death, trash combat, group membership, GUID | [`spell-damage-death.txt`](../testdata/logs/synthetic/spell-damage-death.txt), [`swing-damage-death.txt`](../testdata/logs/synthetic/swing-damage-death.txt) | Synthetic, shape-incomplete |
| 2 | Player death during a boss encounter | Boss start, boss attribution, player death | [`boss-context-death.txt`](../testdata/logs/synthetic/boss-context-death.txt) | Synthetic, shape-incomplete |
| 3 | Death after several damage events | Final damage source, spell/ability, confidence | — | Not prepared |
| 4 | Environmental or falling death | Environmental damage, falling/void, classification | [`environmental-death.txt`](../testdata/logs/synthetic/environmental-death.txt) | Synthetic, shape-incomplete |
| 5 | Knockback followed by death | Knockback/displacement, possible yeet | `knockback-death.txt` | Not collected |
| 6 | Completed Mythic+ run | Run start/completion, dungeon ID, key level | `mplus-complete.txt` | Not collected |
| 7 | Abandoned Mythic+ run | Run abandonment, incomplete boundaries | `mplus-abandoned.txt` | Not collected |
| 8 | Disconnect or reload during a run | Run abandonment, missed events, recovery | `disconnect-mid-run.txt` | Not collected |
| 9 | Combat-log file rotation or truncation | File rotation/truncation, incremental read | `log-rotate.txt` | Not collected |
| 10 | Unknown or newly introduced event type | Parser resilience, unknown event handling | [`unknown-event.txt`](../testdata/logs/synthetic/unknown-event.txt) | Synthetic fixture prepared |

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

| Date | Build | ACL enabled | Scenario # | Fixture | Observation summary | Matrix rows updated |
| ---- | ----- | ----------- | ---------- | ------- | ------------------- | ------------------- |
| 2026-07-30 | V22 docs target 12.0+ | Not applicable | Synthetic preparation | [`synthetic/`](../testdata/logs/synthetic/README.md) | Original fixtures prepared; timestamp envelope and `UNIT_DIED` suffix unresolved; no parser or real-log validation | Format-related rows only |
| 2026-07-30 | V22 docs target 12.0+ | Not applicable | Phase 0B.1 technical tests | [`synthetic/`](../testdata/logs/synthetic/README.md) | Bounded streaming, CSV, common-header, unknown, malformed, partial-tail, and version-quarantine tests pass; no death inference | File mechanics and parser foundation only |

---

## Phase 0A.2 real-log validation checklist

No item below was completed in Phase 0A.1. Phase 0A.2 validates and adjusts
Phase 0B; it does not block all limited Phase 0B parser work.

- [ ] Verify that one logger sees all party deaths.
- [ ] Verify filtering between four configured tracked characters and an
  untracked party member.
- [ ] Verify dungeon and run boundaries.
- [ ] Verify encounter boundaries.
- [ ] Verify ordering of recent damage before death.
- [ ] Compare final damage source with the semantic cause of death.
- [ ] Capture environmental and knockback/void deaths.
- [ ] Exercise file append, buffering, and truncation.
- [ ] Compare real lines field-by-field with synthetic fixtures, including
  timestamp envelope, flags, advanced block, suffixes, and ordering.

---

## Limited Phase 0B scope (may follow 0A.1)

Because a real Mythic+ log is not currently available, limited Phase 0B may
implement and test only source-backed technical behavior:

- CSV-aware tokenization;
- version-header CSV payload;
- unsupported-version behavior;
- common-header extraction;
- exact source-backed damage event payloads;
- advanced-block extraction where the layout is exact;
- unknown and malformed input;
- partial-line buffering.

The following remain blocked pending Phase 0A.2:

- final raw timestamp-envelope compatibility;
- exact V22 `UNIT_DIED` layout;
- real party visibility;
- real event ordering;
- production-quality death detection;
- run and encounter reliability;
- death-cause accuracy;
- yeet classification.

Shape-incomplete death scenarios must not be promoted to passing success
fixtures.

---

## Phase 0 success criteria

Phase 0 is complete only when **all** of the following are evidenced in this document and fixtures:

- [ ] Representative samples can be **read incrementally** (append/tail simulation documented).
- [ ] Relevant event fields can be **parsed without crashing**; unknown events counted, not fatal.
- [ ] **Player deaths** can be detected reliably enough for a go/no-go decision (configured party, representative runs).
- [ ] **Dungeon, run, and encounter boundaries** can be assessed (even if some remain manual).
- [ ] **Unsupported inferences** are listed explicitly with fallback behavior.
- [ ] The project can make an **evidence-based go/no-go** decision documented in this file and the implementation plan.

**Phase 0 is not complete.**

---

## Stop conditions

Pause companion implementation **before backend or upload work** if Phase 0 shows:

| Condition | Indicator |
| --------- | --------- |
| Player deaths cannot be identified reliably | Missed deaths in most representative runs for tracked party |
| Player identity cannot be mapped safely | GUID/name ambiguity would cause wrong Yeetcraft player assignment |
| Dungeon context cannot be recovered | No stable instance/game ID and no reasonable manual fallback |
| Death causes are too ambiguous | Majority of deaths would be `unknown` with no useful review signal |
| Required logging behavior is impractical | Advanced Combat Logging or setup burden unacceptable for the friend group |

If stopped: document outcome here, update [IMPLEMENTATION_PLAN.md §8.2 stop condition](./IMPLEMENTATION_PLAN.md#82-phase-0--combat-log-evidence-spike-bounded-poc), and re-evaluate multi-client logging or manual import — **not** an addon implementation in this phase.

---

## Unresolved questions

Answer during Phase 0; do not treat as verified until evidenced.

| # | Question | Blocks |
| - | -------- | ------ |
| 1 | Operational **yeet** definition for Yeetcraft (game event vs heuristic) | Classification automation |
| 2 | Can one M+ run span **multiple combat-log files**? | Run ID recipe, offset strategy |
| 3 | Is **key level** present in combat log for Midnight M+? | Run metadata |
| 4 | Does one logger see **all four** party deaths consistently? | MVP single-PC assumption |
| 5 | Minimum event set for **run boundaries** on Midnight retail? | Session/detection design |
| 6 | Name/slug alignment with Yeetcraft `players` (see [YEETCRAFT_INTEGRATION.md](./YEETCRAFT_INTEGRATION.md)) vs log names | Identity mapping |

---

## External references

The source bibliography, access dates, version claims, and format conflicts are
centralized in [COMBAT_LOG_FORMAT_V22.md](./COMBAT_LOG_FORMAT_V22.md). Do not
copy unverified event semantics into this matrix or treat a documented field as
verified product behavior.
