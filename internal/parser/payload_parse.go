package parser

import (
	"math"
	"strconv"
)

const noUnitGUID = "0000000000000000"

func parseTypedPayload(eventType string, fields []string, common *CommonHeader) TypedResult {
	switch eventType {
	case "SPELL_DAMAGE":
		return parseSpellDamage(fields, common, false)
	case "RANGE_DAMAGE":
		return parseSpellDamage(fields, common, true)
	case "SWING_DAMAGE":
		return parseSwingDamage(fields, common)
	case "ENVIRONMENTAL_DAMAGE":
		return parseEnvironmentalDamage(fields, common)
	case "ENCOUNTER_START":
		return parseEncounterStart(fields)
	case "ENCOUNTER_END":
		return parseEncounterEnd(fields)
	case "CHALLENGE_MODE_END":
		return parseChallengeModeEnd(fields)
	default:
		return TypedResult{Status: TypedStatusNotApplicable}
	}
}

func parsedTyped(payload TypedPayload, diagnostics ValidationDiagnostics) TypedResult {
	return TypedResult{
		Status:      TypedStatusParsed,
		Payload:     payload,
		Diagnostics: diagnostics,
	}
}

func invalidTyped(err *TypedPayloadError) TypedResult {
	return TypedResult{Status: TypedStatusInvalid, Error: err}
}

func fieldCountError(eventType string, expected, expectedMax, actual int) *TypedPayloadError {
	return &TypedPayloadError{
		Kind:        TypedErrorFieldCount,
		EventType:   eventType,
		Expected:    expected,
		ExpectedMax: expectedMax,
		Actual:      actual,
	}
}

func valueError(kind TypedErrorKind, eventType, fieldName string, fieldIndex int) *TypedPayloadError {
	return &TypedPayloadError{
		Kind:       kind,
		EventType:  eventType,
		FieldName:  fieldName,
		FieldIndex: fieldIndex,
	}
}

func requireString(fields []string, eventType string, index int, name string) (string, *TypedPayloadError) {
	if fields[index] == "" {
		return "", valueError(TypedErrorEmptyRequired, eventType, name, index)
	}
	return fields[index], nil
}

func parseIntField(fields []string, eventType string, index int, name string) (int64, *TypedPayloadError) {
	if fields[index] == "" {
		return 0, valueError(TypedErrorEmptyRequired, eventType, name, index)
	}
	value, err := strconv.ParseInt(fields[index], 10, 64)
	if err != nil {
		return 0, valueError(TypedErrorInteger, eventType, name, index)
	}
	return value, nil
}

func parseHexField(fields []string, eventType string, index int, name string) (uint32, *TypedPayloadError) {
	if fields[index] == "" {
		return 0, valueError(TypedErrorEmptyRequired, eventType, name, index)
	}
	value, ok := parseHexUint32(fields[index])
	if !ok {
		return 0, valueError(TypedErrorHex, eventType, name, index)
	}
	return value, nil
}

func parseFloatField(fields []string, eventType string, index int, name string) (float64, *TypedPayloadError) {
	if fields[index] == "" {
		return 0, valueError(TypedErrorEmptyRequired, eventType, name, index)
	}
	value, err := strconv.ParseFloat(fields[index], 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, valueError(TypedErrorFloat, eventType, name, index)
	}
	return value, nil
}

func parseBoolOrNil(fields []string, eventType string, index int, name string) (BoolOrNil, *TypedPayloadError) {
	switch fields[index] {
	case "nil":
		return BoolNil, nil
	case "1":
		return BoolTrue, nil
	case "":
		return BoolNil, valueError(TypedErrorEmptyRequired, eventType, name, index)
	default:
		return BoolNil, valueError(TypedErrorBoolean, eventType, name, index)
	}
}

func parseBooleanInt(fields []string, eventType string, index int, name string) (bool, *TypedPayloadError) {
	switch fields[index] {
	case "0":
		return false, nil
	case "1":
		return true, nil
	case "":
		return false, valueError(TypedErrorEmptyRequired, eventType, name, index)
	default:
		return false, valueError(TypedErrorBoolean, eventType, name, index)
	}
}

func validateCommonFields(fields []string, eventType string) *TypedPayloadError {
	for _, field := range []struct {
		index int
		name  string
	}{
		{1, "source_guid"},
		{2, "source_name"},
		{5, "dest_guid"},
		{6, "dest_name"},
	} {
		if _, err := requireString(fields, eventType, field.index, field.name); err != nil {
			return err
		}
	}
	for _, field := range []struct {
		index int
		name  string
	}{
		{3, "source_flags"},
		{4, "source_raid_flags"},
		{7, "dest_flags"},
		{8, "dest_raid_flags"},
	} {
		if _, err := parseHexField(fields, eventType, field.index, field.name); err != nil {
			return err
		}
	}
	return nil
}

func parseSpellPrefix(fields []string, eventType string) (SpellPrefix, *TypedPayloadError) {
	var prefix SpellPrefix
	var err *TypedPayloadError
	if prefix.ID, err = parseIntField(fields, eventType, 9, "spell_id"); err != nil {
		return SpellPrefix{}, err
	}
	if prefix.Name, err = requireString(fields, eventType, 10, "spell_name"); err != nil {
		return SpellPrefix{}, err
	}
	if prefix.School, err = parseHexField(fields, eventType, 11, "spell_school"); err != nil {
		return SpellPrefix{}, err
	}
	return prefix, nil
}

func parseAdvanced(fields []string, eventType string, start int) (AdvancedCombatLog, *TypedPayloadError) {
	var advanced AdvancedCombatLog
	var err *TypedPayloadError
	if advanced.InfoGUID, err = requireString(fields, eventType, start, "advanced_info_guid"); err != nil {
		return AdvancedCombatLog{}, err
	}
	if advanced.OwnerGUID, err = requireString(fields, eventType, start+1, "advanced_owner_guid"); err != nil {
		return AdvancedCombatLog{}, err
	}

	intFields := []struct {
		offset int
		name   string
		target *int64
	}{
		{2, "advanced_current_hp", &advanced.CurrentHP},
		{3, "advanced_max_hp", &advanced.MaxHP},
		{4, "advanced_attack_power", &advanced.AttackPower},
		{5, "advanced_spell_power", &advanced.SpellPower},
		{6, "advanced_armor", &advanced.Armor},
		{7, "advanced_absorb", &advanced.Absorb},
		{8, "advanced_unknown_1", &advanced.Unknown1},
		{9, "advanced_unknown_2", &advanced.Unknown2},
		{10, "advanced_power_type", &advanced.PowerType},
		{11, "advanced_current_power", &advanced.CurrentPower},
		{12, "advanced_max_power", &advanced.MaxPower},
		{13, "advanced_power_cost", &advanced.PowerCost},
		{16, "advanced_ui_map_id", &advanced.UIMapID},
		{18, "advanced_item_level", &advanced.ItemLevel},
	}
	for _, field := range intFields {
		*field.target, err = parseIntField(fields, eventType, start+field.offset, field.name)
		if err != nil {
			return AdvancedCombatLog{}, err
		}
	}
	if advanced.PositionX, err = parseFloatField(fields, eventType, start+14, "advanced_position_x"); err != nil {
		return AdvancedCombatLog{}, err
	}
	if advanced.PositionY, err = parseFloatField(fields, eventType, start+15, "advanced_position_y"); err != nil {
		return AdvancedCombatLog{}, err
	}
	if advanced.Facing, err = parseFloatField(fields, eventType, start+17, "advanced_facing"); err != nil {
		return AdvancedCombatLog{}, err
	}
	return advanced, nil
}

func parseDamageSuffix(fields []string, eventType string, start int) (DamageSuffix, *TypedPayloadError) {
	var damage DamageSuffix
	var err *TypedPayloadError
	intFields := []struct {
		offset int
		name   string
		target *int64
	}{
		{0, "base_amount", &damage.BaseAmount},
		{1, "raw_amount", &damage.RawAmount},
		{2, "overkill", &damage.Overkill},
		{4, "resisted", &damage.Resisted},
		{5, "blocked", &damage.Blocked},
		{6, "absorbed", &damage.Absorbed},
	}
	for _, field := range intFields {
		*field.target, err = parseIntField(fields, eventType, start+field.offset, field.name)
		if err != nil {
			return DamageSuffix{}, err
		}
	}
	if damage.School, err = parseHexField(fields, eventType, start+3, "school"); err != nil {
		return DamageSuffix{}, err
	}
	if damage.Critical, err = parseBoolOrNil(fields, eventType, start+7, "critical"); err != nil {
		return DamageSuffix{}, err
	}
	if damage.Glancing, err = parseBoolOrNil(fields, eventType, start+8, "glancing"); err != nil {
		return DamageSuffix{}, err
	}
	if damage.Crushing, err = parseBoolOrNil(fields, eventType, start+9, "crushing"); err != nil {
		return DamageSuffix{}, err
	}
	return damage, nil
}

func advancedDiagnostics(advanced AdvancedCombatLog) ValidationDiagnostics {
	if advanced.Unknown1 != 0 || advanced.Unknown2 != 0 {
		return DiagnosticAdvancedUnknownFieldNonZero
	}
	return 0
}

func parseAbilityHint(raw string) AbilityHint {
	kind := AbilityHintUnknown
	switch raw {
	case "ST":
		kind = AbilityHintSingleTarget
	case "AOE":
		kind = AbilityHintAreaOfEffect
	}
	return AbilityHint{Value: raw, Kind: kind}
}

func parseEnvironmentalType(raw string) EnvironmentalType {
	kind := EnvironmentalTypeUnknown
	switch raw {
	case "Falling":
		kind = EnvironmentalTypeFalling
	case "Lava":
		kind = EnvironmentalTypeLava
	case "Fire":
		kind = EnvironmentalTypeFire
	case "Slime":
		kind = EnvironmentalTypeSlime
	case "Drowning":
		kind = EnvironmentalTypeDrowning
	case "Fatigue":
		kind = EnvironmentalTypeFatigue
	}
	return EnvironmentalType{Value: raw, Kind: kind}
}

func parseSpellDamage(fields []string, common *CommonHeader, isRange bool) TypedResult {
	eventType := fields[0]
	if len(fields) != 42 {
		return invalidTyped(fieldCountError(eventType, 42, 42, len(fields)))
	}
	if err := validateCommonFields(fields, eventType); err != nil {
		return invalidTyped(err)
	}
	spell, err := parseSpellPrefix(fields, eventType)
	if err != nil {
		return invalidTyped(err)
	}
	advanced, err := parseAdvanced(fields, eventType, 12)
	if err != nil {
		return invalidTyped(err)
	}
	damage, err := parseDamageSuffix(fields, eventType, 31)
	if err != nil {
		return invalidTyped(err)
	}
	rawHint, err := requireString(fields, eventType, 41, "ability_hint")
	if err != nil {
		return invalidTyped(err)
	}
	hint := parseAbilityHint(rawHint)
	diagnostics := advancedDiagnostics(advanced)
	if common != nil && advanced.InfoGUID != common.DestGUID {
		diagnostics |= DiagnosticAdvancedInfoGUIDMismatch
	}
	if hint.Kind == AbilityHintUnknown {
		diagnostics |= DiagnosticAbilityHintUnknown
	}
	if isRange {
		return parsedTyped(RangeDamagePayload{Spell: spell, Target: advanced, Damage: damage, AbilityHint: hint}, diagnostics)
	}
	return parsedTyped(SpellDamagePayload{Spell: spell, Target: advanced, Damage: damage, AbilityHint: hint}, diagnostics)
}

func parseSwingDamage(fields []string, common *CommonHeader) TypedResult {
	eventType := fields[0]
	if len(fields) != 38 && len(fields) != 39 {
		return invalidTyped(fieldCountError(eventType, 38, 39, len(fields)))
	}
	if err := validateCommonFields(fields, eventType); err != nil {
		return invalidTyped(err)
	}
	advanced, err := parseAdvanced(fields, eventType, 9)
	if err != nil {
		return invalidTyped(err)
	}
	damage, err := parseDamageSuffix(fields, eventType, 28)
	if err != nil {
		return invalidTyped(err)
	}
	offHand := OptionalBoolOmitted
	if len(fields) == 39 {
		value, boolErr := parseBoolOrNil(fields, eventType, 38, "is_off_hand")
		if boolErr != nil {
			return invalidTyped(boolErr)
		}
		if value == BoolTrue {
			offHand = OptionalBoolTrue
		} else {
			offHand = OptionalBoolNil
		}
	}
	diagnostics := advancedDiagnostics(advanced)
	if common != nil && advanced.InfoGUID != common.SourceGUID {
		diagnostics |= DiagnosticAdvancedInfoGUIDMismatch
	}
	if damage.School != 0x1 {
		diagnostics |= DiagnosticSwingSchoolUnexpected
	}
	return parsedTyped(SwingDamagePayload{Source: advanced, Damage: damage, IsOffHand: offHand}, diagnostics)
}

func parseEnvironmentalDamage(fields []string, common *CommonHeader) TypedResult {
	eventType := fields[0]
	if len(fields) != 39 {
		return invalidTyped(fieldCountError(eventType, 39, 39, len(fields)))
	}
	if err := validateCommonFields(fields, eventType); err != nil {
		return invalidTyped(err)
	}
	advanced, err := parseAdvanced(fields, eventType, 9)
	if err != nil {
		return invalidTyped(err)
	}
	rawType, err := requireString(fields, eventType, 28, "environmental_type")
	if err != nil {
		return invalidTyped(err)
	}
	environmentalType := parseEnvironmentalType(rawType)
	damage, err := parseDamageSuffix(fields, eventType, 29)
	if err != nil {
		return invalidTyped(err)
	}
	diagnostics := advancedDiagnostics(advanced)
	if common != nil {
		if advanced.InfoGUID != common.DestGUID {
			diagnostics |= DiagnosticAdvancedInfoGUIDMismatch
		}
		if common.SourceGUID != noUnitGUID {
			diagnostics |= DiagnosticEnvironmentalSourceNotZero
		}
	}
	if environmentalType.Kind == EnvironmentalTypeUnknown {
		diagnostics |= DiagnosticEnvironmentalTypeUnknown
	}
	return parsedTyped(EnvironmentalDamagePayload{
		Target:            advanced,
		EnvironmentalType: environmentalType,
		Damage:            damage,
	}, diagnostics)
}

func parseEncounterStart(fields []string) TypedResult {
	eventType := fields[0]
	if len(fields) != 5 && len(fields) != 6 {
		return invalidTyped(fieldCountError(eventType, 5, 6, len(fields)))
	}
	var payload EncounterStartPayload
	var err *TypedPayloadError
	if payload.EncounterID, err = parseIntField(fields, eventType, 1, "encounter_id"); err != nil {
		return invalidTyped(err)
	}
	if payload.EncounterName, err = requireString(fields, eventType, 2, "encounter_name"); err != nil {
		return invalidTyped(err)
	}
	if payload.DifficultyID, err = parseIntField(fields, eventType, 3, "difficulty_id"); err != nil {
		return invalidTyped(err)
	}
	if payload.GroupSize, err = parseIntField(fields, eventType, 4, "group_size"); err != nil {
		return invalidTyped(err)
	}
	if len(fields) == 6 {
		instanceID, parseErr := parseIntField(fields, eventType, 5, "instance_id")
		if parseErr != nil {
			return invalidTyped(parseErr)
		}
		payload.InstanceID = &instanceID
	}
	return parsedTyped(payload, 0)
}

func parseEncounterEnd(fields []string) TypedResult {
	eventType := fields[0]
	if len(fields) != 6 && len(fields) != 7 {
		return invalidTyped(fieldCountError(eventType, 6, 7, len(fields)))
	}
	var payload EncounterEndPayload
	var err *TypedPayloadError
	if payload.EncounterID, err = parseIntField(fields, eventType, 1, "encounter_id"); err != nil {
		return invalidTyped(err)
	}
	if payload.EncounterName, err = requireString(fields, eventType, 2, "encounter_name"); err != nil {
		return invalidTyped(err)
	}
	if payload.DifficultyID, err = parseIntField(fields, eventType, 3, "difficulty_id"); err != nil {
		return invalidTyped(err)
	}
	if payload.GroupSize, err = parseIntField(fields, eventType, 4, "group_size"); err != nil {
		return invalidTyped(err)
	}
	if payload.Success, err = parseBooleanInt(fields, eventType, 5, "success"); err != nil {
		return invalidTyped(err)
	}
	if len(fields) == 7 {
		duration, parseErr := parseIntField(fields, eventType, 6, "duration_ms")
		if parseErr != nil {
			return invalidTyped(parseErr)
		}
		payload.DurationMS = &duration
	}
	return parsedTyped(payload, 0)
}

func parseChallengeModeEnd(fields []string) TypedResult {
	eventType := fields[0]
	if len(fields) < 4 || len(fields) > 7 {
		return invalidTyped(fieldCountError(eventType, 4, 7, len(fields)))
	}
	var payload ChallengeModeEndPayload
	var err *TypedPayloadError
	if payload.MapID, err = parseIntField(fields, eventType, 1, "map_id"); err != nil {
		return invalidTyped(err)
	}
	if payload.Success, err = parseBooleanInt(fields, eventType, 2, "success"); err != nil {
		return invalidTyped(err)
	}
	if payload.KeystoneLevel, err = parseIntField(fields, eventType, 3, "keystone_level"); err != nil {
		return invalidTyped(err)
	}
	if len(fields) >= 5 {
		value, parseErr := parseIntField(fields, eventType, 4, "total_time_ms")
		if parseErr != nil {
			return invalidTyped(parseErr)
		}
		payload.TotalTimeMS = &value
	}
	if len(fields) >= 6 {
		value, parseErr := parseFloatField(fields, eventType, 5, "on_time_seconds")
		if parseErr != nil {
			return invalidTyped(parseErr)
		}
		payload.OnTimeSeconds = &value
	}
	if len(fields) == 7 {
		value, parseErr := parseIntField(fields, eventType, 6, "timer_limit_seconds")
		if parseErr != nil {
			return invalidTyped(parseErr)
		}
		payload.TimerLimitSeconds = &value
	}
	return parsedTyped(payload, 0)
}
