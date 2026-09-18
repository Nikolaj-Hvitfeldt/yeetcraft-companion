//go:build contractsibling

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type batchRequest struct {
	Events []batchEvent `json:"events"`
}

type batchEvent struct {
	ClientEventID string   `json:"clientEventId"`
	CharacterGUID string   `json:"characterGuid"`
	DeathInstant  string   `json:"deathInstant"`
	Ordinal       int      `json:"ordinal"`
	Run           batchRun `json:"run"`
}

type batchRun struct {
	ClientRunID               string `json:"clientRunId"`
	ChallengeModeStartInstant string `json:"challengeModeStartInstant"`
	ChallengeMapID            int    `json:"challengeMapId"`
	KeystoneLevel             int    `json:"keystoneLevel"`
}

func TestSiblingPositiveExamplesMatchHardcodedVectors(t *testing.T) {
	root := filepath.Join("..", "..", "..", "Yeetcraft", "contracts", "companion", "v1", "examples", "request")
	files := []string{
		"minimal-trash-death.json",
		"boss-context-death.json",
		"repeated-death-after-resurrection.json",
		"mixed-duplicate-needs-review.json",
	}

	for _, name := range files {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err != nil {
			t.Skipf("sibling checkout not present: %v", err)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		var payload batchRequest
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}

		for i, event := range payload.Events {
			runID, err := ClientRunID(RunIDParams{
				ChallengeModeStartInstant: event.Run.ChallengeModeStartInstant,
				ChallengeMapID:            event.Run.ChallengeMapID,
				KeystoneLevel:             event.Run.KeystoneLevel,
			})
			if err != nil {
				t.Fatalf("%s event %d ClientRunID: %v", name, i, err)
			}
			if runID != event.Run.ClientRunID {
				t.Fatalf("%s event %d clientRunId: got %q want %q", name, i, runID, event.Run.ClientRunID)
			}

			eventID, err := ClientEventID(EventIDParams{
				ClientRunID:   event.Run.ClientRunID,
				CharacterGUID: event.CharacterGUID,
				DeathInstant:  event.DeathInstant,
				Ordinal:       event.Ordinal,
			})
			if err != nil {
				t.Fatalf("%s event %d ClientEventID: %v", name, i, err)
			}
			if eventID != event.ClientEventID {
				t.Fatalf("%s event %d clientEventId: got %q want %q", name, i, eventID, event.ClientEventID)
			}
		}
	}
}
