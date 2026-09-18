package main

import (
	"fmt"
	"io"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/detection"
)

func printDeathSummary(stdout io.Writer, deaths []detection.DeathCandidate) {
	fmt.Fprintf(stdout, "player_deaths: %d\n", len(deaths))
	for i, death := range deaths {
		fmt.Fprintf(stdout, "death_%d_line: %d\n", i+1, death.LineNumber)
		if death.Timestamp != "" {
			fmt.Fprintf(stdout, "death_%d_timestamp: %s\n", i+1, death.Timestamp)
		}
		fmt.Fprintf(stdout, "death_%d_victim_guid: %s\n", i+1, death.VictimGUID)
		if death.HoldReason != "" {
			fmt.Fprintf(stdout, "death_%d_hold_reason: %s\n", i+1, death.HoldReason)
		}
		if death.Run.Active {
			fmt.Fprintf(stdout, "death_%d_run_map_id: %d\n", i+1, death.Run.MapID)
			fmt.Fprintf(stdout, "death_%d_run_key_level: %d\n", i+1, death.Run.KeystoneLevel)
		} else {
			fmt.Fprintf(stdout, "death_%d_run_active: 0\n", i+1)
		}
		if death.Encounter.Active || death.Encounter.EncounterID != 0 {
			fmt.Fprintf(stdout, "death_%d_encounter_id: %d\n", i+1, death.Encounter.EncounterID)
		} else {
			fmt.Fprintf(stdout, "death_%d_encounter_active: 0\n", i+1)
		}
		fmt.Fprintf(stdout, "death_%d_cause_count: %d\n", i+1, len(death.Causes))
		for _, cause := range death.Causes {
			prefix := fmt.Sprintf("death_%d_cause_%d", i+1, cause.Rank)
			fmt.Fprintf(stdout, "%s_confidence: %s\n", prefix, cause.Confidence)
			fmt.Fprintf(stdout, "%s_source_type: %s\n", prefix, cause.Cause.SourceType)
			if cause.Cause.SpellID != 0 {
				fmt.Fprintf(stdout, "%s_spell_id: %d\n", prefix, cause.Cause.SpellID)
			}
			if cause.Cause.EnvironmentalType != "" {
				fmt.Fprintf(stdout, "%s_environmental_type: %s\n", prefix, cause.Cause.EnvironmentalType)
			}
			if cause.Cause.PlayerOrigin {
				fmt.Fprintf(stdout, "%s_player_origin: 1\n", prefix)
			}
			if cause.Cause.HasCreatureID {
				fmt.Fprintf(stdout, "%s_creature_id: %d\n", prefix, cause.Cause.CreatureID)
			}
			fmt.Fprintf(stdout, "%s_amount: %d\n", prefix, cause.Cause.Amount)
			fmt.Fprintf(stdout, "%s_overkill: %d\n", prefix, cause.Cause.Overkill)
		}
	}
}
