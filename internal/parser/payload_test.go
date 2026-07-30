package parser

import (
	"bytes"
	"strings"
	"testing"
)

func syntheticCommon(eventType, sourceGUID, destGUID string) []string {
	return []string{
		eventType,
		sourceGUID,
		"Synthetic Source",
		"0xa48",
		"0x0",
		destGUID,
		"Synthetic Target",
		"0x512",
		"0x0",
	}
}

func syntheticAdvanced(infoGUID string) []string {
	return []string{
		infoGUID,
		noUnitGUID,
		"100",
		"200",
		"10",
		"20",
		"30",
		"0",
		"0",
		"0",
		"0",
		"50",
		"100",
		"0",
		"1.25",
		"2.5",
		"999",
		"3.14",
		"600",
	}
}

func syntheticDamage(school string) []string {
	return []string{"100", "120", "-1", school, "0", "0", "0", "1", "nil", "nil"}
}

func syntheticSpellDamage(eventType string) []string {
	fields := syntheticCommon(eventType, "Creature-Source", "Player-Target")
	fields = append(fields, "12345", "Synthetic Spell", "0x4")
	fields = append(fields, syntheticAdvanced("Player-Target")...)
	fields = append(fields, syntheticDamage("0x4")...)
	return append(fields, "ST")
}

func syntheticSwingDamage() []string {
	fields := syntheticCommon("SWING_DAMAGE", "Creature-Source", "Player-Target")
	fields = append(fields, syntheticAdvanced("Creature-Source")...)
	return append(fields, syntheticDamage("0x1")...)
}

func syntheticEnvironmentalDamage() []string {
	fields := syntheticCommon("ENVIRONMENTAL_DAMAGE", noUnitGUID, "Player-Target")
	fields = append(fields, syntheticAdvanced("Player-Target")...)
	fields = append(fields, "Falling")
	return append(fields, syntheticDamage("0x1")...)
}

func parseSyntheticFields(fields []string) Event {
	return ParseLine(1, strings.Join(fields, ","), supportedState())
}

func TestTypedPayloadsParseEveryIncludedLayout(t *testing.T) {
	tests := []struct {
		name       string
		fields     []string
		assertType func(t *testing.T, payload TypedPayload)
	}{
		{
			"spell damage",
			syntheticSpellDamage("SPELL_DAMAGE"),
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value, ok := payload.(SpellDamagePayload)
				if !ok || value.Spell.ID != 12345 || value.AbilityHint.Kind != AbilityHintSingleTarget {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
		{
			"range damage",
			syntheticSpellDamage("RANGE_DAMAGE"),
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				if _, ok := payload.(RangeDamagePayload); !ok {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
		{
			"swing damage",
			syntheticSwingDamage(),
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value, ok := payload.(SwingDamagePayload)
				if !ok || value.IsOffHand != OptionalBoolOmitted {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
		{
			"environmental damage",
			syntheticEnvironmentalDamage(),
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value, ok := payload.(EnvironmentalDamagePayload)
				if !ok || value.EnvironmentalType.Kind != EnvironmentalTypeFalling {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
		{
			"encounter start",
			[]string{"ENCOUNTER_START", "9001", "Synthetic Encounter", "8", "5", "777"},
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value, ok := payload.(EncounterStartPayload)
				if !ok || value.InstanceID == nil || *value.InstanceID != 777 {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
		{
			"encounter end",
			[]string{"ENCOUNTER_END", "9001", "Synthetic Encounter", "8", "5", "0", "123456"},
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value, ok := payload.(EncounterEndPayload)
				if !ok || value.Success || value.DurationMS == nil {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
		{
			"challenge mode end",
			[]string{"CHALLENGE_MODE_END", "501", "1", "12", "1800000", "-12.5", "1800"},
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value, ok := payload.(ChallengeModeEndPayload)
				if !ok || !value.Success || value.OnTimeSeconds == nil || *value.OnTimeSeconds != -12.5 {
					t.Fatalf("payload = %#v", payload)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := parseSyntheticFields(tt.fields)
			if event.Typed.Status != TypedStatusParsed || event.Typed.Error != nil || event.Typed.Payload == nil {
				t.Fatalf("typed = %#v", event.Typed)
			}
			tt.assertType(t, event.Typed.Payload)
		})
	}
}

func TestTypedOptionalFieldsAndBooleanGrammar(t *testing.T) {
	t.Run("swing omitted nil true", func(t *testing.T) {
		tests := []struct {
			token string
			want  OptionalBoolOrNil
		}{
			{"", OptionalBoolOmitted},
			{"nil", OptionalBoolNil},
			{"1", OptionalBoolTrue},
		}
		for _, tt := range tests {
			fields := syntheticSwingDamage()
			if tt.token != "" {
				fields = append(fields, tt.token)
			}
			event := parseSyntheticFields(fields)
			payload, ok := event.Typed.Payload.(SwingDamagePayload)
			if event.Typed.Status != TypedStatusParsed || !ok || payload.IsOffHand != tt.want {
				t.Fatalf("token %q typed=%#v", tt.token, event.Typed)
			}
		}
	})

	t.Run("metadata optional omission", func(t *testing.T) {
		for _, fields := range [][]string{
			{"ENCOUNTER_START", "9001", "Synthetic Encounter", "8", "5"},
			{"ENCOUNTER_END", "9001", "Synthetic Encounter", "8", "5", "1"},
			{"CHALLENGE_MODE_END", "501", "0", "12"},
			{"CHALLENGE_MODE_END", "501", "0", "12", "1800000"},
			{"CHALLENGE_MODE_END", "501", "0", "12", "1800000", "12.5"},
		} {
			event := parseSyntheticFields(fields)
			if event.Typed.Status != TypedStatusParsed {
				t.Fatalf("fields=%#v typed=%#v", fields, event.Typed)
			}
		}
	})

	t.Run("bool or nil rejects false and empty", func(t *testing.T) {
		for _, token := range []string{"0", "", "true"} {
			fields := syntheticSpellDamage("SPELL_DAMAGE")
			fields[38] = token
			event := parseSyntheticFields(fields)
			if event.Typed.Status != TypedStatusInvalid || event.Typed.Payload != nil {
				t.Fatalf("token %q typed=%#v", token, event.Typed)
			}
		}
	})

	t.Run("present optional metadata cannot be empty or nil", func(t *testing.T) {
		for _, token := range []string{"", "nil"} {
			event := parseSyntheticFields([]string{"ENCOUNTER_START", "9001", "Synthetic Encounter", "8", "5", token})
			if event.Typed.Status != TypedStatusInvalid || event.Typed.Payload != nil {
				t.Fatalf("token %q typed=%#v", token, event.Typed)
			}
		}
	})

	t.Run("boolean int rejects non-binary token", func(t *testing.T) {
		event := parseSyntheticFields([]string{"ENCOUNTER_END", "9001", "Synthetic Encounter", "8", "5", "2"})
		if event.Typed.Status != TypedStatusInvalid || event.Typed.Payload != nil ||
			event.Typed.Error == nil || event.Typed.Error.Kind != TypedErrorBoolean {
			t.Fatalf("typed = %#v", event.Typed)
		}
	})
}

func TestTypedPrimitiveFailures(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func([]string) []string
		wantKind TypedErrorKind
	}{
		{
			"too few fields",
			func(fields []string) []string { return fields[:41] },
			TypedErrorFieldCount,
		},
		{
			"extra field",
			func(fields []string) []string { return append(fields, "extra") },
			TypedErrorFieldCount,
		},
		{
			"empty required",
			func(fields []string) []string { fields[10] = ""; return fields },
			TypedErrorEmptyRequired,
		},
		{
			"malformed integer",
			func(fields []string) []string { fields[31] = "not-an-integer"; return fields },
			TypedErrorInteger,
		},
		{
			"integer overflow",
			func(fields []string) []string { fields[31] = "9223372036854775808"; return fields },
			TypedErrorInteger,
		},
		{
			"malformed hex",
			func(fields []string) []string { fields[34] = "not-hex"; return fields },
			TypedErrorHex,
		},
		{
			"decimal is not hexadecimal",
			func(fields []string) []string { fields[34] = "4"; return fields },
			TypedErrorHex,
		},
		{
			"octal-looking value is not hexadecimal",
			func(fields []string) []string { fields[34] = "010"; return fields },
			TypedErrorHex,
		},
		{
			"malformed common flag",
			func(fields []string) []string { fields[3] = "not-hex"; return fields },
			TypedErrorHex,
		},
		{
			"hex overflow",
			func(fields []string) []string { fields[34] = "0x100000000"; return fields },
			TypedErrorHex,
		},
		{
			"malformed float",
			func(fields []string) []string { fields[26] = "not-a-float"; return fields },
			TypedErrorFloat,
		},
		{
			"nan",
			func(fields []string) []string { fields[26] = "NaN"; return fields },
			TypedErrorFloat,
		},
		{
			"infinity",
			func(fields []string) []string { fields[26] = "+Inf"; return fields },
			TypedErrorFloat,
		},
		{
			"malformed boolean",
			func(fields []string) []string { fields[38] = "false"; return fields },
			TypedErrorBoolean,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.mutate(syntheticSpellDamage("SPELL_DAMAGE"))
			event := parseSyntheticFields(fields)
			if event.Kind != KindCommonHeader || event.Typed.Status != TypedStatusInvalid ||
				event.Typed.Payload != nil || event.Typed.Error == nil || event.Typed.Error.Kind != tt.wantKind {
				t.Fatalf("event = %#v", event)
			}
		})
	}
}

func TestTypedFieldCountErrorsReportAllAcceptedWidths(t *testing.T) {
	tests := []struct {
		fields       []string
		wantExpected int
		wantMax      int
	}{
		{append(syntheticSwingDamage(), "nil", "extra"), 38, 39},
		{[]string{"ENCOUNTER_START", "1", "Name", "8", "5", "7", "extra"}, 5, 6},
		{[]string{"ENCOUNTER_END", "1", "Name", "8", "5", "1", "100", "extra"}, 6, 7},
		{[]string{"CHALLENGE_MODE_END", "1", "1"}, 4, 7},
	}
	for _, tt := range tests {
		event := parseSyntheticFields(tt.fields)
		if event.Typed.Status != TypedStatusInvalid || event.Typed.Error == nil ||
			event.Typed.Error.Kind != TypedErrorFieldCount ||
			event.Typed.Error.Expected != tt.wantExpected ||
			event.Typed.Error.ExpectedMax != tt.wantMax {
			t.Fatalf("fields=%#v typed=%#v", tt.fields, event.Typed)
		}
	}
}

func TestTypedIntegerBoundaries(t *testing.T) {
	fields := syntheticSpellDamage("SPELL_DAMAGE")
	fields[31] = "9223372036854775807"
	fields[33] = "-9223372036854775808"
	fields[34] = "0xffffffff"
	fields[26] = "1.7976931348623157e+308"
	fields[27] = "-1.7976931348623157e+308"
	event := parseSyntheticFields(fields)
	payload, ok := event.Typed.Payload.(SpellDamagePayload)
	if event.Typed.Status != TypedStatusParsed || !ok {
		t.Fatalf("typed = %#v", event.Typed)
	}
	if payload.Damage.BaseAmount != 9223372036854775807 ||
		payload.Damage.Overkill != -9223372036854775808 ||
		payload.Damage.School != 0xffffffff ||
		payload.Target.PositionX < 1e308 ||
		payload.Target.PositionY > -1e308 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSourceExpectationDiagnosticsRetainPayload(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		flag   ValidationDiagnostics
		assert func(t *testing.T, payload TypedPayload)
	}{
		{
			"ownership mismatch",
			func() []string {
				fields := syntheticSpellDamage("SPELL_DAMAGE")
				fields[12] = "Player-Other"
				return fields
			}(),
			DiagnosticAdvancedInfoGUIDMismatch,
			nil,
		},
		{
			"unknown ability hint",
			func() []string {
				fields := syntheticSpellDamage("SPELL_DAMAGE")
				fields[41] = "FUTURE_HINT"
				return fields
			}(),
			DiagnosticAbilityHintUnknown,
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value := payload.(SpellDamagePayload)
				if value.AbilityHint.Kind != AbilityHintUnknown || value.AbilityHint.Value != "FUTURE_HINT" {
					t.Fatalf("hint = %#v", value.AbilityHint)
				}
			},
		},
		{
			"unknown environmental type",
			func() []string {
				fields := syntheticEnvironmentalDamage()
				fields[28] = "FutureHazard"
				return fields
			}(),
			DiagnosticEnvironmentalTypeUnknown,
			func(t *testing.T, payload TypedPayload) {
				t.Helper()
				value := payload.(EnvironmentalDamagePayload)
				if value.EnvironmentalType.Kind != EnvironmentalTypeUnknown || value.EnvironmentalType.Value != "FutureHazard" {
					t.Fatalf("environmental type = %#v", value.EnvironmentalType)
				}
			},
		},
		{
			"unexpected swing school",
			func() []string {
				fields := syntheticSwingDamage()
				fields[31] = "0x2"
				return fields
			}(),
			DiagnosticSwingSchoolUnexpected,
			nil,
		},
		{
			"environmental source not zero",
			func() []string {
				fields := syntheticEnvironmentalDamage()
				fields[1] = "Creature-Unexpected"
				return fields
			}(),
			DiagnosticEnvironmentalSourceNotZero,
			nil,
		},
		{
			"advanced unknown nonzero",
			func() []string {
				fields := syntheticSpellDamage("SPELL_DAMAGE")
				fields[20] = "7"
				return fields
			}(),
			DiagnosticAdvancedUnknownFieldNonZero,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := parseSyntheticFields(tt.fields)
			if event.Typed.Status != TypedStatusParsed || event.Typed.Payload == nil ||
				event.Typed.Error != nil || !event.Typed.Diagnostics.Has(tt.flag) {
				t.Fatalf("typed = %#v", event.Typed)
			}
			if tt.assert != nil {
				tt.assert(t, event.Typed.Payload)
			}
		})
	}
}

func TestMultipleDiagnosticsRetainParsedPayload(t *testing.T) {
	fields := syntheticEnvironmentalDamage()
	fields[1] = "Creature-Unexpected"
	fields[9] = "Player-Other"
	fields[17] = "7"
	fields[28] = "FutureHazard"

	event := parseSyntheticFields(fields)
	want := DiagnosticEnvironmentalSourceNotZero |
		DiagnosticAdvancedInfoGUIDMismatch |
		DiagnosticAdvancedUnknownFieldNonZero |
		DiagnosticEnvironmentalTypeUnknown
	if event.Typed.Status != TypedStatusParsed || event.Typed.Payload == nil ||
		event.Typed.Error != nil || event.Typed.Diagnostics != want {
		t.Fatalf("typed = %#v, want diagnostics %#x", event.Typed, want)
	}
}

func TestTypedErrorAndDiagnosticsArePrivacySafe(t *testing.T) {
	fields := syntheticSpellDamage("SPELL_DAMAGE")
	fields[1] = "Player-Private-GUID"
	fields[2] = "Private Player-Realm"
	fields[31] = "private-malformed-value"
	event := parseSyntheticFields(fields)
	message := event.Typed.Error.Error()
	for _, sensitive := range []string{
		"Player-Private-GUID",
		"Private Player-Realm",
		"private-malformed-value",
		strings.Join(fields, ","),
	} {
		if strings.Contains(message, sensitive) {
			t.Fatalf("typed error exposed %q: %q", sensitive, message)
		}
	}

	fields = syntheticSpellDamage("SPELL_DAMAGE")
	fields[12] = "Player-Private-GUID"
	event = parseSyntheticFields(fields)
	if event.Typed.Status != TypedStatusParsed || event.Typed.Diagnostics == 0 {
		t.Fatalf("typed = %#v", event.Typed)
	}
}

func TestRecognizedEventsWithoutTypedParserRemainApplicableOnlyGenerically(t *testing.T) {
	for _, fields := range [][]string{
		{"CHALLENGE_MODE_START", "Synthetic Dungeon", "501", "600", "12", "[1]"},
		syntheticCommon("UNIT_DIED", noUnitGUID, "Player-Target"),
	} {
		event := parseSyntheticFields(fields)
		if event.Typed.Status != TypedStatusNotApplicable || event.Typed.Payload != nil || event.Typed.Error != nil {
			t.Fatalf("event = %#v", event)
		}
	}
}

func TestScanSummarySeparatesTypedErrorsAndDiagnostics(t *testing.T) {
	valid := strings.Join(syntheticSpellDamage("SPELL_DAMAGE"), ",")
	invalidFields := syntheticSpellDamage("SPELL_DAMAGE")
	invalidFields[31] = "bad"
	diagnosticFields := syntheticSpellDamage("RANGE_DAMAGE")
	diagnosticFields[41] = "FUTURE_HINT"
	input := strings.Join([]string{
		"COMBAT_LOG_VERSION,22,ADVANCED_LOG_ENABLED,1,BUILD_VERSION,12.0.0,PROJECT_ID,1",
		valid,
		strings.Join(invalidFields, ","),
		strings.Join(diagnosticFields, ","),
	}, "\n") + "\n"

	summary, err := ScanReader(bytes.NewBufferString(input), DefaultMaxLineSize, &ParserState{}, func(Event) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TypedParsed != 2 || summary.TypedInvalid != 1 || summary.TypedErrors.Integer != 1 ||
		summary.Diagnostics.Total != 1 || summary.Diagnostics.AbilityHintUnknown != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}
