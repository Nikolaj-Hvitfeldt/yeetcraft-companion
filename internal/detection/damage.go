package detection

import (
	"strings"

	"github.com/Nikolaj-Hvitfeldt/yeetcraft-companion/internal/parser"
)

const (
	defaultDamageCapacity = 32
	defaultLookbackLines  = 500
)

type damageBuffer struct {
	capacity int
	entries  []DamageHit
}

func newDamageBuffer(capacity int) *damageBuffer {
	if capacity <= 0 {
		capacity = defaultDamageCapacity
	}
	return &damageBuffer{capacity: capacity}
}

func (b *damageBuffer) add(hit DamageHit) {
	if b.capacity == 0 {
		return
	}
	if len(b.entries) == b.capacity {
		copy(b.entries, b.entries[1:])
		b.entries[len(b.entries)-1] = hit
		return
	}
	b.entries = append(b.entries, hit)
}

func (b *damageBuffer) snapshotBefore(lineNumber int) []DamageHit {
	if len(b.entries) == 0 {
		return nil
	}
	out := make([]DamageHit, 0, len(b.entries))
	for i := len(b.entries) - 1; i >= 0; i-- {
		if b.entries[i].LineNumber >= lineNumber {
			continue
		}
		if lineNumber-b.entries[i].LineNumber > defaultLookbackLines {
			break
		}
		out = append(out, b.entries[i])
	}
	return out
}

func extractDamageHit(event parser.Event) (DamageHit, bool) {
	if event.Common == nil {
		return DamageHit{}, false
	}
	switch event.EventType {
	case "SPELL_DAMAGE", "SPELL_PERIODIC_DAMAGE", "RANGE_DAMAGE":
		return hitFromSpellDamage(event)
	case "SWING_DAMAGE":
		return hitFromSwingDamage(event)
	case "ENVIRONMENTAL_DAMAGE":
		return hitFromEnvironmentalDamage(event)
	default:
		return DamageHit{}, false
	}
}

func hitFromSpellDamage(event parser.Event) (DamageHit, bool) {
	if event.Typed.Status != parser.TypedStatusParsed {
		return DamageHit{}, false
	}
	var spell parser.SpellPrefix
	var damage parser.DamageSuffix
	switch payload := event.Typed.Payload.(type) {
	case parser.SpellDamagePayload:
		spell = payload.Spell
		damage = payload.Damage
	case parser.RangeDamagePayload:
		spell = payload.Spell
		damage = payload.Damage
	default:
		return DamageHit{}, false
	}
	return DamageHit{
		LineNumber: event.LineNumber,
		Timestamp:  event.Envelope.Raw,
		EventType:  event.EventType,
		SourceGUID: event.Common.SourceGUID,
		SourceName: event.Common.SourceName,
		SpellID:    spell.ID,
		SpellName:  spell.Name,
		Amount:     damageAmount(damage),
		Overkill:   damage.Overkill,
	}, true
}

func hitFromSwingDamage(event parser.Event) (DamageHit, bool) {
	if event.Typed.Status != parser.TypedStatusParsed {
		return DamageHit{}, false
	}
	payload, ok := event.Typed.Payload.(parser.SwingDamagePayload)
	if !ok {
		return DamageHit{}, false
	}
	return DamageHit{
		LineNumber: event.LineNumber,
		Timestamp:  event.Envelope.Raw,
		EventType:  event.EventType,
		SourceGUID: event.Common.SourceGUID,
		SourceName: event.Common.SourceName,
		Amount:     damageAmount(payload.Damage),
		Overkill:   payload.Damage.Overkill,
	}, true
}

func hitFromEnvironmentalDamage(event parser.Event) (DamageHit, bool) {
	if event.Typed.Status != parser.TypedStatusParsed {
		return DamageHit{}, false
	}
	payload, ok := event.Typed.Payload.(parser.EnvironmentalDamagePayload)
	if !ok {
		return DamageHit{}, false
	}
	return DamageHit{
		LineNumber:        event.LineNumber,
		Timestamp:         event.Envelope.Raw,
		EventType:         event.EventType,
		SourceGUID:        event.Common.SourceGUID,
		SourceName:        event.Common.SourceName,
		Amount:            damageAmount(payload.Damage),
		Overkill:          payload.Damage.Overkill,
		EnvironmentalType: payload.EnvironmentalType.Value,
	}, true
}

func damageAmount(damage parser.DamageSuffix) int64 {
	if damage.RawAmount > 0 {
		return damage.RawAmount
	}
	return damage.BaseAmount
}

func rankCauses(hits []DamageHit) []CauseCandidate {
	if len(hits) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(hits))
	out := make([]CauseCandidate, 0, 3)
	for _, hit := range hits {
		key := causeKey(hit)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, CauseCandidate{
			Rank:       len(out) + 1,
			Hit:        hit,
			Confidence: causeConfidence(hit, len(out) == 0),
		})
		if len(out) == 3 {
			break
		}
	}
	return out
}

func causeKey(hit DamageHit) string {
	if hit.SpellID != 0 || hit.SpellName != "" {
		return hit.SourceGUID + "|" + hit.EventType + "|" + hit.SpellName
	}
	return hit.SourceGUID + "|" + hit.EventType + "|" + hit.EnvironmentalType
}

func causeConfidence(hit DamageHit, primary bool) Confidence {
	if !primary {
		return ConfidenceMedium
	}
	if hit.Overkill > 0 {
		return ConfidenceHigh
	}
	if hit.Amount > 0 {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

func isPlayerGUID(guid string) bool {
	return strings.HasPrefix(guid, "Player-")
}
