# Combat-log capabilities

> **Status:** Research placeholder. No parser exists yet. **Do not treat statements below as verified facts** unless backed by fixtures in `testdata/logs/` or official documentation.

This document will record what World of Warcraft retail combat logs can and cannot support for Yeetcraft companion features (deaths, yeets, dungeon identification, party roster, Mythic+ boundaries).

## Purpose

Yeetcraft tracks **deaths** and **yeets** per player per dungeon per season. The companion must derive that information from local combat-log files written by the WoW client.

## Known format (high level)

**Assumption:** Retail writes UTF-8 text files named like `WoWCombatLog-YYYY-MM-DD_HHMMSS.txt` under a `Logs` directory in the WoW installation (typically `_retail_/Logs` on Windows).

**Requires proof in Phase 0:**

- [ ] Exact default path on Windows for this user base.
- [ ] Log rotation behavior when starting a new log file.
- [ ] Whether Advanced Combat Logging must be enabled for required events.

## Event categories to investigate

| Category | Desired signal | Status |
| -------- | -------------- | ------ |
| Player death | Who died, when, in which encounter | **Unknown** — needs log samples |
| Yeet | Operational definition needed (e.g. avoidable death, specific ability?) | **Open question** |
| Instance / dungeon | Zone or map change, challenge mode start | **Unknown** |
| Party roster | Group member names and classes | **Unknown** |
| Mythic+ start/end | Timer, key level, completion/failure | **Unknown** |
| Pull / combat boundaries | For attributing deaths to trash vs boss | **Unknown** |

## WoW combat-log line format

**Assumption:** Lines are timestamped, comma-separated fields documented in community references and Blizzard’s combat log event list.

**TODO — Phase 0**

- [ ] Capture sample lines for `COMBAT_LOG_VERSION`, `ZONE_CHANGE`, `ENCOUNTER_START`, `ENCOUNTER_END`, `UNIT_DIED`, `SPELL_AURA_APPLIED`, etc.
- [ ] Note build-specific differences if any.
- [ ] Document which events are sufficient vs necessary for Yeetcraft stats.

## Limitations (anticipated — unverified)

Possible gaps to validate with real logs:

- Combat log may not include all group members until they generate events.
- Cross-realm or name normalization (special characters, realm suffixes).
- Distinguishing intentional deaths vs disconnects.
- Defining “yeet” consistently from log data alone without addon assistance.

## Addon (deferred)

A WoW addon might emit custom events or mark yeets explicitly. **No addon work is planned in the foundation bootstrap.** If log-only parsing proves insufficient, revisit in a separate task.

## Test fixtures

Place anonymized snippets under `testdata/logs/`. Each fixture should include:

- A short README comment at the top of the file (or a sibling `.md`) describing scenario and WoW build.
- Redacted player names if needed.
- Expected parser outputs once the parser exists (golden files).

**Current fixtures:** none (`.gitkeep` only).

## References to consult (external)

- Blizzard / WoW wiki combat log event documentation (verify against current retail build).
- Community parsers for inspiration only — do not copy GPL code without license review.

## Open questions

1. What is the authoritative operational definition of a **yeet** for Yeetcraft (game event vs heuristic)?
2. Can one M+ run span multiple combat-log files?
3. Is key level available in combat log, or only dungeon name?
4. How should the companion handle logs from multiple WoW installations or accounts?

Record answers here as Phase 0 progresses.
