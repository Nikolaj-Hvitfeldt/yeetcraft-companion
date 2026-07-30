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
- timestamped death-scenario fixtures use the observed V22 timestamp shape
  provisionally and are labeled `shape-incomplete`;
- all fixtures containing `UNIT_DIED` are `shape-incomplete`;
- `CHALLENGE_MODE_START` remains untyped because the selected reference does
  not define the raw CSV serialization of its comma-containing affix array;
- focused technical fixtures may be parser success tests without verifying the
  timestamp envelope or event semantics.

Phase 0B.1 passes the technical fixtures listed below for bounded line reading,
CSV tokenization, common-header extraction, unknown input, and version
quarantine. Shape-incomplete death scenarios remain unsuitable as death-
detection success fixtures. Phase 0A.2 must validate and adjust the provisional
envelope behavior.

Limited Phase 0B.2 passes exact-layout synthetic fixtures for seven selected
typed damage and metadata layouts. Structural typed failures and non-fatal
source-expectation diagnostics are tested separately. This does not promote
death, run, identity, boss, cause, or yeet capabilities.

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
| `version-header.txt` | Synthetically tested header payload; line envelope unresolved | WowCoach `spec.yaml` `version_header` | V22 and advanced-logging marker |
| `spell-damage-death.txt` | `shape-incomplete` | `SPELL_DAMAGE`: WowCoach `spec.yaml` offsets 0–41; `UNIT_DIED`: unresolved suffix; timestamp: conflicting sources | Tracked spell damage followed by death |
| `swing-damage-death.txt` | `shape-incomplete` | `SWING_DAMAGE`: WowCoach `spec.yaml` offsets 0–37; `UNIT_DIED`: unresolved suffix; timestamp: conflicting sources | Tracked swing damage followed by death |
| `environmental-death.txt` | `shape-incomplete` | `ENVIRONMENTAL_DAMAGE`: WowCoach `spec.yaml` offsets 0–38; `UNIT_DIED`: unresolved suffix; timestamp: conflicting sources | Environmental damage followed by death |
| `boss-context-death.txt` | `shape-incomplete` | `ENCOUNTER_*` and `SPELL_DAMAGE`: WowCoach `spec.yaml`; `UNIT_DIED` and timestamp unresolved | Boss-context death ordering |
| `untracked-party-death.txt` | `shape-incomplete` | `UNIT_DIED` suffix and timestamp unresolved | Untracked party member death |
| `untracked-party-damage.txt` | `shape-incomplete` | `SPELL_DAMAGE`: WowCoach `spec.yaml`; timestamp conflicting | Damage from untracked party member |
| `unknown-event.txt` | Synthetically tested intentional unknown input | Common header shape only; unknown suffix is deliberately undefined | Unknown-event resilience without recognition |
| `malformed-csv.txt` | Synthetically tested intentional invalid input | Not applicable | Unterminated quoted field |
| `truncated-line.txt` | Synthetically tested intentional partial input | Prefix of source-backed `SPELL_DAMAGE`; intentionally cut with no final newline | Partial final-line buffering |
| `unsupported-version.txt` | Synthetically tested unsupported input | Header key/value structure from WowCoach `spec.yaml`; fictional version `99` | Unsupported version handling |
| `parser-smoke-valid.txt` | Synthetically tested technical parser fixture | V22 header, documented metadata/common-header offsets, intentional unknown event | End-to-end CLI summary without death semantics |
| `csv-quoted-fields.txt` | Synthetically tested technical parser fixture | Standard CSV quoting rules | Quoted comma, literal `nil`, trailing fields |
| `common-header-invalid-flags.txt` | Synthetically tested malformed flags | V22 common-header offsets | Controlled hexadecimal flag warnings |
| `common-header-too-few-fields.txt` | Synthetically tested truncated common header | V22 common-header offsets | Controlled structural error |
| `version-malformed-kv.txt` | Synthetically tested malformed header | V22 key/value structure intentionally truncated | Controlled header-structure error |
| `version-repeated-boundary.txt` | Synthetically tested repeated header | V22 version-header structure | Supported state boundary |
| `version-v22-then-unsupported.txt` | Synthetically tested quarantine sequence | V22/V99 synthetic headers and generic post-boundary rows | Fail-closed quarantine; no recovery on later V22 |
| `version-v22-then-malformed.txt` | Synthetically tested malformed-boundary sequence | V22 header shape with invalid advanced marker | Malformed header quarantines; no recovery on later V22 |
| `version-project-id-2.txt` | Synthetically tested unsupported project | V22 header with fictional non-retail project selection | Retail-only project validation |
| `version-project-id-non-integer.txt` | Synthetically tested malformed project ID | V22 header with non-integer project value | Malformed header quarantine |
| `timestamp-signed-offset.txt` | Synthetically tested provisional reference envelope | Reference timestamp shape with signed offset and two-space separator | Raw-envelope preservation; not real-log verification |
| `typed-damage-v22.txt` | Synthetically tested technical payloads | WowCoach selected canonical `spec.yaml`: exact advanced-enabled `SPELL_DAMAGE`, `RANGE_DAMAGE`, `SWING_DAMAGE`, and `ENVIRONMENTAL_DAMAGE` offsets | Typed payloads, advanced ownership placement, nullable booleans, and optional off-hand field |
| `typed-metadata-v22.txt` | Synthetically tested technical payloads | WowCoach selected canonical `spec.yaml`: exact `ENCOUNTER_START`, `ENCOUNTER_END`, and `CHALLENGE_MODE_END` offsets | Neutral metadata parsing with documented trailing optional fields |
| `typed-payload-invalid-v22.txt` | Synthetically tested invalid and diagnostic input | Exact selected-reference widths with synthetic primitive and source-expectation deviations | Separate typed parse errors from non-fatal diagnostics; verify privacy-safe counters |

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
