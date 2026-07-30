# WoW retail combat-log format V22

Format reference for Phase 0A.1 of the Yeetcraft companion. Product capability
status is tracked separately in
[COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md); the implementation
sequence is in [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md).

This document records external documentation and project conclusions. It does
not claim that a real Mythic+ log has been inspected.

## Source claims and project conclusion

The selected current reference, the WowCoach.gg machine-readable specification,
states:

- `format_version: 22`;
- `verified_against_patch: 12.0+`;
- `last_updated: 2026-05-08`.

Its prose page additionally says V22 covers The War Within 11.x. That statement
is retained as a source claim only. The project's narrower conclusion is:

> The companion targets WoW retail patch 12.0+ and prepares for combat-log
> format V22 according to the selected current reference.

The project has not independently established V22 behavior on 11.x or verified
the format against a real 12.0+ Mythic+ log.

## Line envelope and unresolved timestamp conflict

The selected 12.0+ reference documents this abstract line envelope:

```text
M/D/YYYY HH:MM:SS.fff±Z  EVENT_TYPE,field1,field2,...
```

It specifies a locale-dependent date order, a millisecond fragment, a timezone
offset, and two spaces (or a tab on some clients) between timestamp and CSV.

A maintained V22 parser issue includes an observed 2025 V22 line beginning:

```text
3/29/2025 19:57:40.9471 SPELL_HEAL,...
```

That example has four fractional digits, no explicit signed timezone suffix,
and one visible separator space. Older Warcraft Wiki file examples omit both
year and timezone.

These sources conflict materially. Phase 0A.1 therefore does **not** designate
any timestamped synthetic file as valid parser success-test input. Timestamped
fixtures use the observed V22 envelope only as a provisional shape and are
marked `shape-incomplete` in the
[fixture README](../testdata/logs/synthetic/README.md). Phase 0A.2 must resolve
the envelope from a real retail 12.0+ log.

## CSV rules

The selected specification documents:

- comma-separated fields;
- double quotes around quoted strings;
- embedded commas inside quoted strings;
- the literal `nil` for many absent values.

A conforming parser must use CSV-aware tokenization rather than splitting
blindly on commas.

## Version header

The selected reference gives this exact CSV payload:

```text
COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1
```

| Position | Key/value | Meaning |
| -------- | --------- | ------- |
| 0–1 | `COMBAT_LOG_VERSION,22` | Format version |
| 2–3 | `ADVANCED_LOG_ENABLED,1` | Advanced logging configured on |
| 4–5 | `BUILD_VERSION,12.0.0` | Game patch/build label |
| 6–7 | `PROJECT_ID,1` | Retail project |

The specification says this header may appear mid-log after the logger restarts
and should be treated as a hard state boundary. It also warns that the advanced
marker is not sufficient for fragments: event shape must confirm whether an
advanced block is present.

The current reference displays the header payload without a timestamp while
also saying every line has a timestamp. That internal ambiguity is unresolved;
[`version-header.txt`](../testdata/logs/synthetic/version-header.txt) preserves
the exact documented payload but is not asserted to be a complete raw log line.

## Common source/target header

Source/target events use these exact zero-based CSV offsets:

| Offset | Field | Type |
| ------ | ----- | ---- |
| 0 | `event` | string |
| 1 | `source_guid` | string |
| 2 | `source_name` | quoted string |
| 3 | `source_flags` | hexadecimal uint32 |
| 4 | `source_raid_flags` | hexadecimal uint32 |
| 5 | `dest_guid` | string |
| 6 | `dest_name` | quoted string |
| 7 | `dest_flags` | hexadecimal uint32 |
| 8 | `dest_raid_flags` | hexadecimal uint32 |

Spell and range events then use `spell_id`, `spell_name`, and `spell_school` at
offsets 9–11. Swing events do not have this prefix.

## Advanced logging block

The selected V22 specification defines a 19-field block:

| Relative offset | Field |
| --------------- | ----- |
| 0 | `info_guid` |
| 1 | `owner_guid` |
| 2 | `current_hp` |
| 3 | `max_hp` |
| 4 | `attack_power` |
| 5 | `spell_power` |
| 6 | `armor` |
| 7 | `absorb` |
| 8 | `unknown_1` |
| 9 | `unknown_2` |
| 10 | `power_type` |
| 11 | `current_power` |
| 12 | `max_power` |
| 13 | `power_cost` |
| 14 | `position_x` |
| 15 | `position_y` |
| 16 | `ui_map_id` |
| 17 | `facing` |
| 18 | `item_level` |

The block occupies offsets 12–30 for spell/range events and 9–27 for swing and
environmental damage. `SPELL_DAMAGE` and `ENVIRONMENTAL_DAMAGE` describe the
target; `SWING_DAMAGE` describes the source. Consumers must match `info_guid`
instead of assuming ownership.

The V22 parser issue confirms the inserted two fields at relative offsets 8–9,
but observed `unknown_1` was nonzero despite the selected specification saying
it is always zero. The fixture values are synthetic and do not assign semantics
to either unknown field.

## Exact event layouts

Offsets below are zero-based from the event name and come from the selected
V22 `spec.yaml`.

### `SPELL_DAMAGE` and `RANGE_DAMAGE`

`RANGE_DAMAGE` inherits `SPELL_DAMAGE`.

| Offsets | Content |
| ------- | ------- |
| 0–8 | Common header |
| 9–11 | Spell prefix |
| 12–30 | Advanced block describing target |
| 31 | `base_amount` |
| 32 | `raw_amount` |
| 33 | `overkill` (`-1` means nonlethal) |
| 34 | `school` |
| 35 | `resisted` |
| 36 | `blocked` |
| 37 | `absorbed` |
| 38 | `critical` |
| 39 | `glancing` |
| 40 | `crushing` |
| 41 | `ability_hint` (`ST` or `AOE`) |

### `SWING_DAMAGE`

| Offsets | Content |
| ------- | ------- |
| 0–8 | Common header |
| 9–27 | Advanced block describing source |
| 28 | `base_amount` |
| 29 | `raw_amount` |
| 30 | `overkill` |
| 31 | `school` (physical `0x1`) |
| 32 | `resisted` |
| 33 | `blocked` |
| 34 | `absorbed` |
| 35 | `critical` |
| 36 | `glancing` |
| 37 | `crushing` |
| 38 | optional `is_off_hand` |

### `ENVIRONMENTAL_DAMAGE`

The selected specification says the source GUID is
`0000000000000000`.

| Offsets | Content |
| ------- | ------- |
| 0–8 | Common header |
| 9–27 | Advanced block describing target |
| 28 | `environmental_type` |
| 29 | `base_amount` |
| 30 | `raw_amount` |
| 31 | `overkill` |
| 32 | `school` |
| 33 | `resisted` |
| 34 | `blocked` |
| 35 | `absorbed` |
| 36 | `critical` |
| 37 | `glancing` |
| 38 | `crushing` |

Documented environmental types are `Falling`, `Lava`, `Fire`, `Slime`,
`Drowning`, and `Fatigue`.

### `UNIT_DIED`

The selected V22 specification confirms the common header, no spell prefix, and
no advanced block, but does not enumerate a suffix. Warcraft Wiki documents
`recapID` and `unconsciousOnDeath` for the related API/event family. Because
that secondary description is not an exact V22 file-layout specification,
Phase 0A.1 treats every synthetic `UNIT_DIED` line as `shape-incomplete` and
unsuitable for parser success tests.

### `SPELL_INSTAKILL`

The selected V22 specification confirms the spell prefix and describes the
event as a forced-kill signal, but does not provide an exact suffix table.
No `SPELL_INSTAKILL` success fixture is created.

### `ENCOUNTER_START`

This metadata event has no common source/target header.

| Offset | Field |
| ------ | ----- |
| 0 | `ENCOUNTER_START` |
| 1 | `encounter_id` |
| 2 | `encounter_name` |
| 3 | `difficulty_id` |
| 4 | `group_size` |
| 5 | optional `instance_id` |

Difficulty ID `8` is documented as Mythic+.

### `ENCOUNTER_END`

| Offset | Field |
| ------ | ----- |
| 0 | `ENCOUNTER_END` |
| 1 | `encounter_id` |
| 2 | `encounter_name` |
| 3 | `difficulty_id` |
| 4 | `group_size` |
| 5 | `success` |
| 6 | optional `duration_ms` |

### `CHALLENGE_MODE_START`

| Offset | Field |
| ------ | ----- |
| 0 | `CHALLENGE_MODE_START` |
| 1 | `dungeon_name` |
| 2 | `map_id` |
| 3 | `challenge_mode_id` |
| 4 | `keystone_level` |
| 5 | `affixes` integer array |

### `CHALLENGE_MODE_END`

| Offset | Field |
| ------ | ----- |
| 0 | `CHALLENGE_MODE_END` |
| 1 | `map_id` |
| 2 | `success` |
| 3 | `keystone_level` |
| 4 | optional `total_time_ms` |
| 5 | optional `on_time_seconds` |
| 6 | optional `timer_limit_seconds` |

These layouts document available fields, not their visibility or reliability in
a real Mythic+ run.

## Phase 0B usage boundaries

Limited Phase 0B may use the exact layouts above for source-backed parser work
without waiting for Phase 0A.2. Suitable now:

- version-header CSV payload;
- common header;
- `SPELL_DAMAGE`, `SWING_DAMAGE`, and `ENVIRONMENTAL_DAMAGE` suffix tables;
- advanced block where offsets are exact in this document;
- metadata events with complete suffix tables (`ENCOUNTER_*`, `CHALLENGE_MODE_*`).

Blocked or shape-incomplete until Phase 0A.2:

- raw timestamp envelope;
- `UNIT_DIED` suffix;
- `SPELL_INSTAKILL` suffix;
- death-scenario fixtures that depend on unresolved layouts.

Phase 0A.2 validates and adjusts Phase 0B assumptions; it does not block all
limited parser work.

## Known conflicts and limitations

1. **Timestamp envelope:** current references conflict as described above.
2. **Version-header envelope:** the selected page shows an unprefixed header
   while describing all lines as timestamped.
3. **`UNIT_DIED`:** exact V22 file suffix is absent from the selected spec.
4. **`SPELL_INSTAKILL`:** exact suffix is absent from the selected spec.
5. **Advanced unknown fields:** one observed V22 sample conflicts with the
   selected spec's “always zero” note.
6. **Real behavior:** visibility, ordering, and semantics in Mythic+ require
   Phase 0A.2 evidence.

## External sources

All sources were accessed 2026-07-30.

| Source title | URL | Documented format/game version | Use |
| ------------ | --- | ------------------------------ | --- |
| WoW Combat Log Reference | <https://wowcoach.gg/docs/combat-log> | V22; retail patch 12.0+ | Current overview |
| WoW Combat Log Format Specification | <https://wowcoach.gg/docs/combat-log/spec.yaml> | V22; verified against 12.0+; updated 2026-05-08 | Primary field offsets |
| Line Format & Common Header | <https://wowcoach.gg/docs/combat-log/line-format> | V22; source also claims 11.x and 12.0+ | Envelope, header, common fields |
| Advanced Combat Logging | <https://wowcoach.gg/docs/combat-log/advanced-logging> | Current V22 reference | Advanced block |
| Metadata & Other Event Suffixes | <https://wowcoach.gg/docs/combat-log/metadata-events> | Current V22 reference | Encounter and challenge fields |
| COMBAT_LOG_EVENT | <https://warcraft.wiki.gg/wiki/COMBAT_LOG_EVENT> | Multi-version API/file reference; examples are legacy | Secondary `UNIT_DIED` and header cross-check |
| Support Combatlog Version 22 | <https://github.com/Toreole/BasicCombatlogParser/issues/35> | Observed V22 line from 2025 | Timestamp and advanced-block conflict |

## Related documentation

- [Capability status and real-log checklist](./COMBAT_LOG_CAPABILITIES.md)
- [Implementation plan](./IMPLEMENTATION_PLAN.md)
- [Synthetic fixture provenance](../testdata/logs/synthetic/README.md)
