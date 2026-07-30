# Combat-log capabilities

Phase 0 capability matrix and evidence log for the Yeetcraft companion.

| | |
| --- | --- |
| **Phase** | 0 — not started |
| **Last updated** | 2026-07-30 |
| **Fixtures** | None yet ([`testdata/logs/`](../testdata/logs/) contains `.gitkeep` only) |

Yeetcraft tracks **deaths** and **yeets** per player per dungeon per season. The companion must derive that information from local combat-log files written by the WoW client on **one** logging PC. This document records what the log can support — not what the product wishes were true.

**Related docs:** [IMPLEMENTATION_PLAN.md §8.2](./IMPLEMENTATION_PLAN.md#82-phase-0--combat-log-evidence-spike-bounded-poc) · [YEETCRAFT_INTEGRATION.md](./YEETCRAFT_INTEGRATION.md)

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

Use exactly one status per capability row. Update only when evidence is recorded in the [Evidence log](#evidence-log).

| Status | Meaning |
| ------ | ------- |
| **Not investigated** | No representative sample examined yet |
| **Hypothesis** | Expected from docs, community knowledge, or prior WoW experience — **not proven on project fixtures** |
| **Partially verified** | Observed in at least one anonymized sample; gaps, edge cases, or build limits remain |
| **Verified** | Reproducible on multiple representative fixtures for the target retail build |
| **Not reliably available** | Investigated; log-only capture on one client cannot supply this consistently |
| **Rejected** | Investigated; signal unsuitable for automated use (manual fallback required) |

**Confidence** (when status is not *Not investigated*): `none` · `low` · `medium` · `high` — how trustworthy the signal is for **automated** companion behavior.

---

## Capability matrix

### Identity and roster

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Player identity | Map a death to a Yeetcraft **person** (Niko, Seb, Martin, Niklas) | GUID + configured mapping; optional name/realm cross-check | Not investigated | — | One logger may not see all party members; alts and name changes | Any death scenario with full party | `needs_review`; never guess from similar names |
| Character name and realm | Display and disambiguate who died | `UNIT_DIED`, `SPELL_*`, or aura lines with name–realm | Not investigated | — | Special characters, connected realms, rename | Death during trash or boss | Store log name; prompt review if unmapped |
| Player GUID | Stable key for idempotency and mapping | `Player-XXXX-…` tokens in combat-log fields | Not investigated | — | May be absent on some lines until unit generates events | Any death scenario | Cannot auto-upload until GUID mapped |
| Class and specialization | Context for death review and future UI | `SPELL_CAST_SUCCESS`, aura, or roster-related events | Not investigated | — | Spec may require inference; not required for MVP counts | Boss or trash death | Omit or show "unknown" in review |
| Role | Tank/healer/DPS context for review | Same as class; may not appear explicitly in log | Not investigated | — | Role is not in Yeetcraft DB today; log may omit | Any party death | Omit; use Yeetcraft frontend character data later if synced |
| Group membership | Know which units belong to the M+ group | Party roster events, `INSTANCE_ENCOUNTER_*`, or repeated unit tokens | Not investigated | — | Logger may only see subset of group; outsiders in log | Completed M+ with four tracked players | Restrict to configured GUID/name list; ignore outsiders |

### Instance and run

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Dungeon or instance identity | Map run to Yeetcraft dungeon | `ZONE_CHANGE`, `MAP_CHANGE`, `CHALLENGE_MODE_*`, instance name/ID | Not investigated | — | Name vs Yeetcraft canonical list may differ | Completed M+ run | Manual dungeon selection in review; prefer game ID if present |
| Dungeon difficulty | Confirm Mythic+ vs other content | Difficulty flags in instance/challenge events | Not investigated | — | Wrong difficulty → ignore run | Completed M+ run | Exclude run from auto-upload |
| Keystone level | Store key level on run | `CHALLENGE_MODE_START` or related metadata | Not investigated | — | May be missing outside M+ | Completed M+ run | Optional field; manual entry in review |
| Run start | Open a bounded run record | Challenge start, instance entry, or state-machine entry signals | Not investigated | — | Ambiguous start vs key insert | Completed or abandoned M+ | Mark run `candidate` until confirmed |
| Run completion | Close run as successful | `CHALLENGE_MODE_COMPLETE`, completion chat, or timer events | Not investigated | — | Signal may arrive late or in next file | Completed M+ run | Timeout-based `abandoned` if no completion |
| Run abandonment | Close run without inventing success | Timeout, zone leave, disconnect, or interrupted session | Not investigated | — | Hard to distinguish pause vs quit | Abandoned M+; disconnect mid-run | Keep events; mark run `abandoned`; review before upload |

### Encounters and combat context

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Boss encounter start | Set active boss for death attribution | `ENCOUNTER_START`, journal encounter ID | Not investigated | — | Trash vs boss boundary fuzzy | Boss death scenario | `encounter_id` null → dungeon-level only |
| Boss encounter completion | Clear active boss | `ENCOUNTER_END` | Not investigated | — | End may not fire on wipe/leave | Completed boss pull | Retain last known encounter until zone change |
| Trash combat | Attribute deaths outside boss encounters | Absence of active encounter + combat events | Not investigated | — | Multi-pack trash may blur | Trash death scenario | Classify as trash death with null encounter |

### Death and cause

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Player death | Detect who died and when | `UNIT_DIED` (and related) for tracked units | Not investigated | — | Missed if unit not visible to logger | All death scenarios | **Stop condition** if unreliable — see below |
| Final damage source | Identify last relevant damage dealer | Recent `SPELL_DAMAGE` / `SWING_*` / `ENVIRONMENTAL_*` before `UNIT_DIED` | Not investigated | — | DoT, absorb, delay mechanics | Death after several damage events | Rank causes; accept `unknown` |
| Spell or ability causing death | Human-readable ability name and ID | Spell fields on damage events | Not investigated | — | Same as final damage ambiguity | Death after several damage events | Store best rank; low confidence → review |
| Boss attribution | Link death to active boss encounter | Active `ENCOUNTER_START` + death timestamp | Not investigated | — | Add death may be attributed to boss incorrectly | Boss death scenario | Separate boss encounter from `death_causes` source |
| Environmental damage | Detect environmental lethal damage | `ENVIRONMENTAL_DAMAGE`, fall, drowning, etc. | Not investigated | — | May overlap with yeet heuristics | Environmental or falling death | Category `unknown` or `possible_yeet`; review |
| Knockback or displacement | Evidence for yeet-like deaths | Knockback auras, `SPELL_AURA_APPLIED`, position-less inference | Not investigated | — | Rarely explicit; high false-positive risk | Knockback followed by death | `possible_yeet` + review; never auto-yeet without policy |
| Falling into the void | Evidence for void/edge deaths | Fall damage, environmental, no recent enemy hit | Not investigated | — | Indistinguishable from some environmental deaths | Environmental or falling death | Conservative `death` or `possible_yeet`; review |

### File mechanics

| Capability | Desired outcome | Expected event / evidence source | Status | Confidence | Known limitations | Required fixture / scenario | Fallback behavior |
| ---------- | --------------- | -------------------------------- | ------ | ---------- | ----------------- | --------------------------- | ----------------- |
| Combat-log file rotation or truncation | Resume without duplicate or skipped events | File size shrink, new `WoWCombatLog-*.txt`, byte offset + file identity | Not investigated | — | **Hypothesis:** retail rotates on new file session | Rotation/truncation scenario | Unique client event IDs + offset persistence; re-read policy in Phase 2 |

### Hypotheses not in matrix above (log format)

| Topic | Statement | Status |
| ----- | --------- | ------ |
| Default log path (Windows retail) | `_retail_/Logs/WoWCombatLog-*.txt` | Hypothesis |
| Line format | Timestamped comma-separated combat-log events | Hypothesis |
| Advanced Combat Logging | Required for sufficient detail | Not investigated |

---

## Status distribution (current)

| Status | Count | Capabilities |
| ------ | ----- | ------------ |
| Not investigated | **23** | All matrix rows above |
| Hypothesis | **0** | — (format hypotheses tracked separately) |
| Partially verified | **0** | — |
| Verified | **0** | — |
| Not reliably available | **0** | — |
| Rejected | **0** | — |

*Update this table as Phase 0 progresses.*

---

## Representative Phase 0 scenarios

Each scenario must produce at least one anonymized fixture slice (or a documented reason it could not). Link fixtures in the [Evidence log](#evidence-log) when created.

| # | Scenario | Primary capabilities exercised | Fixture name (planned) | Status |
| - | -------- | ------------------------------ | ---------------------- | ------ |
| 1 | Ordinary player death during trash | Player death, trash combat, group membership, GUID | `trash-death.txt` | Not collected |
| 2 | Player death during a boss encounter | Boss start, boss attribution, player death | `boss-death.txt` | Not collected |
| 3 | Death after several damage events | Final damage source, spell/ability, confidence | `multi-hit-death.txt` | Not collected |
| 4 | Environmental or falling death | Environmental damage, falling/void, classification | `environmental-death.txt` | Not collected |
| 5 | Knockback followed by death | Knockback/displacement, possible yeet | `knockback-death.txt` | Not collected |
| 6 | Completed Mythic+ run | Run start/completion, dungeon ID, key level | `mplus-complete.txt` | Not collected |
| 7 | Abandoned Mythic+ run | Run abandonment, incomplete boundaries | `mplus-abandoned.txt` | Not collected |
| 8 | Disconnect or reload during a run | Run abandonment, missed events, recovery | `disconnect-mid-run.txt` | Not collected |
| 9 | Combat-log file rotation or truncation | File rotation/truncation, incremental read | `log-rotate.txt` | Not collected |
| 10 | Unknown or newly introduced event type | Parser resilience, unknown event handling | `unknown-event.txt` | Not collected |

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

*No entries yet. Phase 0 has not started.*

| Date | Build | ACL enabled | Scenario # | Fixture | Observation summary | Matrix rows updated |
| ---- | ----- | ----------- | ---------- | ------- | ------------------- | ------------------- |
| — | — | — | — | — | — | — |

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

## External references (consult only)

- Blizzard / WoW wiki combat log event list — verify against **current retail build**.
- Community parsers — inspiration only; respect licenses.

Do not copy unverified event semantics into this matrix without fixture proof.
