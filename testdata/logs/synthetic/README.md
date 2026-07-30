# Synthetic combat-log fixtures

Every file in this directory is original synthetic data. Names, realms, GUIDs,
timestamps, spell IDs, encounter IDs, map IDs, and numeric values are
fictional. No real player data or copied third-party log lines are included.

The field shapes are based on the sources recorded in the
[V22 format reference](../../../docs/COMBAT_LOG_FORMAT_V22.md). Product
capability status is tracked in the
[capability matrix](../../../docs/COMBAT_LOG_CAPABILITIES.md), and phase
boundaries are defined in the
[implementation plan](../../../docs/IMPLEMENTATION_PLAN.md).

## Important limitation

Current V22 references conflict on the exact timestamp envelope, and the
selected V22 specification does not enumerate the `UNIT_DIED` suffix. Therefore:

- `version-header.txt` preserves the exact documented header CSV payload but is
  not asserted to be a complete raw log line;
- timestamped scenario fixtures use the observed V22 timestamp shape
  provisionally and are labeled `shape-incomplete`;
- all fixtures containing `UNIT_DIED` are `shape-incomplete`;
- no file in this directory is yet suitable as a complete parser success test.

Phase 0B may promote individual fixtures only after an implementation passes
exact source-backed fixtures. Shape-incomplete death scenarios must not become
passing success fixtures. Limited Phase 0B may proceed before Phase 0A.2 for
technical parser behavior; Phase 0A.2 later validates and adjusts those
assumptions.

## Fictional roster

- `TrackedAlpha-SyntheticRealm` through
  `TrackedDelta-SyntheticRealm`: four configured tracked characters.
- `UntrackedEcho-SyntheticRealm`: an untracked party member.

The untracked party member exists only to exercise local roster context. Their
death must not become a Yeetcraft statistic, and their raw identity must never
be uploaded.

## Inventory and provenance

| Fixture | Classification | Exact layout source | Purpose |
| ------- | -------------- | ------------------- | ------- |
| `version-header.txt` | Header payload complete; line envelope unresolved | WowCoach `spec.yaml` `version_header` | V22 and advanced-logging marker |
| `spell-damage-death.txt` | `shape-incomplete` | `SPELL_DAMAGE`: WowCoach `spec.yaml` offsets 0–41; `UNIT_DIED`: unresolved suffix; timestamp: conflicting sources | Tracked spell damage followed by death |
| `swing-damage-death.txt` | `shape-incomplete` | `SWING_DAMAGE`: WowCoach `spec.yaml` offsets 0–37; `UNIT_DIED`: unresolved suffix; timestamp: conflicting sources | Tracked swing damage followed by death |
| `environmental-death.txt` | `shape-incomplete` | `ENVIRONMENTAL_DAMAGE`: WowCoach `spec.yaml` offsets 0–38; `UNIT_DIED`: unresolved suffix; timestamp: conflicting sources | Environmental damage followed by death |
| `boss-context-death.txt` | `shape-incomplete` | `ENCOUNTER_*` and `SPELL_DAMAGE`: WowCoach `spec.yaml`; `UNIT_DIED` and timestamp unresolved | Boss-context death ordering |
| `untracked-party-death.txt` | `shape-incomplete` | `UNIT_DIED` suffix and timestamp unresolved | Untracked party member death |
| `untracked-party-damage.txt` | `shape-incomplete` | `SPELL_DAMAGE`: WowCoach `spec.yaml`; timestamp conflicting | Damage from untracked party member |
| `unknown-event.txt` | Intentional unknown input | Common header shape only; unknown suffix is deliberately undefined | Unknown-event resilience |
| `malformed-csv.txt` | Intentional invalid input | Not applicable | Unterminated quoted field |
| `truncated-line.txt` | Intentional partial input | Prefix of source-backed `SPELL_DAMAGE`; intentionally cut | Partial final-line buffering |
| `unsupported-version.txt` | Intentional unsupported input | Header key/value structure from WowCoach `spec.yaml`; fictional version `99` | Unsupported version handling |

Primary source URLs:

- <https://wowcoach.gg/docs/combat-log/spec.yaml>
- <https://wowcoach.gg/docs/combat-log/line-format>
- <https://github.com/Toreole/BasicCombatlogParser/issues/35>
- <https://warcraft.wiki.gg/wiki/COMBAT_LOG_EVENT>

All were accessed 2026-07-30. See the format reference for source titles,
documented versions, exact offsets, and conflicts.

## What these fixtures do not prove

They do not prove:

- that one logger sees every party member's deaths or preceding damage;
- that challenge, encounter, or dungeon boundaries appear reliably;
- that damage ordering identifies the semantic cause of death;
- that an environmental death is a yeet;
- file append, buffering, rotation, or truncation behavior in the WoW client;
- tracked-roster identity mapping against real GUIDs.

Those questions remain for Phase 0A.2 using an anonymized real retail 12.0+
Mythic+ log.
