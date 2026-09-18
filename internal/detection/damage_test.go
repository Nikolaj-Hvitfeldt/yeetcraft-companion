package detection

import "testing"

func TestSourceTypeFromEventType(t *testing.T) {
	tests := []struct {
		eventType string
		want      SourceType
	}{
		{"SPELL_DAMAGE", SourceTypeSpell},
		{"SPELL_PERIODIC_DAMAGE", SourceTypeSpell},
		{"RANGE_DAMAGE", SourceTypeRange},
		{"SWING_DAMAGE", SourceTypeMelee},
		{"ENVIRONMENTAL_DAMAGE", SourceTypeEnvironmental},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got, ok := sourceTypeFromEventType(tt.eventType)
			if !ok {
				t.Fatalf("sourceTypeFromEventType(%q) = ok false", tt.eventType)
			}
			if got != tt.want {
				t.Fatalf("sourceTypeFromEventType(%q) = %q, want %q", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestExtractCreatureID(t *testing.T) {
	tests := []struct {
		name   string
		guid   string
		wantID int64
		wantOK bool
	}{
		{
			name:   "full creature guid",
			guid:   "Creature-0-9999-9999-99999-900001-0000000001",
			wantID: 900001,
			wantOK: true,
		},
		{
			name:   "player guid",
			guid:   "Player-9999-00000001",
			wantOK: false,
		},
		{
			name:   "short creature guid",
			guid:   "Creature-Synthetic",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractCreatureID(tt.guid)
			if ok != tt.wantOK {
				t.Fatalf("extractCreatureID(%q) ok = %v, want %v", tt.guid, ok, tt.wantOK)
			}
			if got != tt.wantID {
				t.Fatalf("extractCreatureID(%q) = %d, want %d", tt.guid, got, tt.wantID)
			}
		})
	}
}

func TestStorableCauseRedactsPlayerOrigin(t *testing.T) {
	hit := DamageHit{
		EventType:  "SPELL_DAMAGE",
		SourceGUID: "Player-9999-00000005",
		SourceName: "UntrackedEcho-SyntheticRealm",
		SpellID:    900012,
		SpellName:  "Synthetic Friendly Fire",
		Amount:     60000,
		Overkill:   300,
	}

	cause := storableCauseFromHit(hit)
	if !cause.PlayerOrigin {
		t.Fatal("expected player-origin marker")
	}
	if cause.HasCreatureID || cause.CreatureID != 0 {
		t.Fatalf("creature id must not be set for player origin: %+v", cause)
	}
	if cause.SpellID != 900012 {
		t.Fatalf("spell id = %d, want 900012", cause.SpellID)
	}
	if cause.SourceType != SourceTypeSpell {
		t.Fatalf("source type = %q, want spell", cause.SourceType)
	}
}

func TestStorableCauseExtractsCreatureID(t *testing.T) {
	hit := DamageHit{
		EventType:  "SPELL_DAMAGE",
		SourceGUID: "Creature-0-9999-9999-99999-900013-0000000001",
		SourceName: "Synthetic Mob",
		SpellID:    900013,
		Amount:     90000,
	}

	cause := storableCauseFromHit(hit)
	if cause.PlayerOrigin {
		t.Fatal("unexpected player-origin marker")
	}
	if !cause.HasCreatureID || cause.CreatureID != 900013 {
		t.Fatalf("creature id = %+v, want 900013", cause)
	}
}

func TestStorableCauseEnvironmentalType(t *testing.T) {
	hit := DamageHit{
		EventType:         "ENVIRONMENTAL_DAMAGE",
		SourceGUID:        "0000000000000000",
		EnvironmentalType: "Falling",
		Amount:            40000,
	}

	cause := storableCauseFromHit(hit)
	if cause.SourceType != SourceTypeEnvironmental {
		t.Fatalf("source type = %q, want environmental", cause.SourceType)
	}
	if cause.EnvironmentalType != "Falling" {
		t.Fatalf("environmental type = %q, want Falling", cause.EnvironmentalType)
	}
}

func TestRankCausesContiguousRanks(t *testing.T) {
	hits := []DamageHit{
		{EventType: "SPELL_DAMAGE", SourceGUID: "Creature-0-9999-9999-99999-900001-0000000001", SpellID: 900001, Amount: 100},
		{EventType: "SWING_DAMAGE", SourceGUID: "Creature-0-9999-9999-99999-900002-0000000001", Amount: 90},
		{EventType: "RANGE_DAMAGE", SourceGUID: "Creature-0-9999-9999-99999-900003-0000000001", SpellID: 900003, Amount: 80},
		{EventType: "ENVIRONMENTAL_DAMAGE", SourceGUID: "0000000000000000", EnvironmentalType: "Lava", Amount: 70},
	}

	causes := rankCauses(hits)
	if len(causes) != 3 {
		t.Fatalf("causes = %d, want 3", len(causes))
	}
	for i, cause := range causes {
		wantRank := i + 1
		if cause.Rank != wantRank {
			t.Fatalf("cause[%d].Rank = %d, want %d", i, cause.Rank, wantRank)
		}
	}
}

func TestRankCausesNeverStoresSpellName(t *testing.T) {
	hits := []DamageHit{
		{
			EventType:  "SPELL_DAMAGE",
			SourceGUID: "Creature-0-9999-9999-99999-900001-0000000001",
			SpellID:    900001,
			SpellName:  "Synthetic Secret Name",
			Amount:     100,
		},
	}

	causes := rankCauses(hits)
	if len(causes) != 1 {
		t.Fatalf("causes = %d, want 1", len(causes))
	}
	if causes[0].Cause.SpellID != 900001 {
		t.Fatalf("spell id = %d, want 900001", causes[0].Cause.SpellID)
	}
}
