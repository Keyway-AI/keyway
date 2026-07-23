package diff

import (
	"time"

	"github.com/architsharma/keyway/internal/model"
	"github.com/google/uuid"
)

// Compute diffs two contract versions and returns the classified change events
// (PRD §9). Consumers are matched across versions by StableID; a consumer
// present in only one version yields a consumer_added / consumer_removed event.
//
// Field-level changes are decomposed into atomic operations and run through
// Classify (classify.go). Each added/removed element of a set-valued field
// (audiences, issuers, algorithms, required_claims) produces exactly one event —
// so adding a single audience yields exactly one `widened` event (AC-7).
func Compute(from, to model.ContractVersion) []model.ChangeEvent {
	now := time.Now().UTC()
	fromByID := indexByStableID(from.Consumers)
	toByID := indexByStableID(to.Consumers)

	var events []model.ChangeEvent
	emit := func(consumerID, field string, op FieldOp, oldV, newV, classifyVal any, c model.Consumer) {
		cl := Classify(field, op, classifyVal, confidenceFor(c, field))
		events = append(events, model.ChangeEvent{
			ID:          uuid.NewString(),
			FromVersion: from.ID,
			ToVersion:   to.ID,
			ConsumerID:  consumerID,
			Field:       field,
			OldValue:    oldV,
			NewValue:    newV,
			Class:       cl.Class,
			Severity:    cl.Severity,
			Confidence:  confidenceFor(c, field),
			Evidence:    evidenceFor(c, field),
			DetectedAt:  now,
		})
	}

	// Consumers removed (present in `from`, absent in `to`).
	for id, c := range fromByID {
		if _, ok := toByID[id]; !ok {
			emit(id, FieldConsumer, OpConsumerDel, c.Name, nil, nil, c)
		}
	}

	// Consumers added or changed.
	for id, cTo := range toByID {
		cFrom, existed := fromByID[id]
		if !existed {
			emit(id, FieldConsumer, OpConsumerAdd, nil, cTo.Name, nil, cTo)
			continue
		}
		diffExpectations(cFrom.Expects, cTo.Expects, cTo, emit)
		diffJWKS(cFrom.JWKSBehavior, cTo.JWKSBehavior, cTo, emit)
	}

	return events
}

type emitFn func(consumerID, field string, op FieldOp, oldV, newV, classifyVal any, c model.Consumer)

func diffExpectations(from, to model.Expectations, c model.Consumer, emit emitFn) {
	diffSet(FieldAudiences, from.Audiences, to.Audiences, c, emit)
	diffSet(FieldIssuers, from.Issuers, to.Issuers, c, emit)
	diffSet(FieldAlgorithms, from.Algorithms, to.Algorithms, c, emit)
	diffSet(FieldRequiredClaims, from.RequiredClaims, to.RequiredClaims, c, emit)

	if from.ClockSkewSec != to.ClockSkewSec {
		op := OpIncrease
		if to.ClockSkewSec < from.ClockSkewSec {
			op = OpDecrease
		}
		emit(c.StableID, FieldClockSkew, op, from.ClockSkewSec, to.ClockSkewSec, nil, c)
	}
}

func diffJWKS(from, to model.JWKSBehavior, c model.Consumer, emit emitFn) {
	// refreshes_on_unknown_kid — only compare when both sides are known, so
	// merely *learning* the value (nil -> bool) does not page.
	if from.RefreshesOnUnknownKID != nil && to.RefreshesOnUnknownKID != nil &&
		*from.RefreshesOnUnknownKID != *to.RefreshesOnUnknownKID {
		op := OpFalseToTrue
		if *from.RefreshesOnUnknownKID && !*to.RefreshesOnUnknownKID {
			op = OpTrueToFalse
		}
		emit(c.StableID, FieldRefreshUnknown, op, *from.RefreshesOnUnknownKID, *to.RefreshesOnUnknownKID, nil, c)
	}

	// cache_ttl_sec — only when both known and different.
	if from.CacheTTLSec != nil && to.CacheTTLSec != nil && *from.CacheTTLSec != *to.CacheTTLSec {
		op := OpIncrease
		if *to.CacheTTLSec < *from.CacheTTLSec {
			op = OpDecrease
		}
		emit(c.StableID, FieldCacheTTL, op, *from.CacheTTLSec, *to.CacheTTLSec, nil, c)
	}
}

// diffSet emits one event per element added or removed between two string sets.
func diffSet(field string, from, to []string, c model.Consumer, emit emitFn) {
	fromSet := toStringSet(from)
	toSet := toStringSet(to)
	for _, v := range to {
		if _, ok := fromSet[v]; !ok {
			emit(c.StableID, field, OpAdd, nil, v, v, c)
		}
	}
	for _, v := range from {
		if _, ok := toSet[v]; !ok {
			emit(c.StableID, field, OpRemove, v, nil, v, c)
		}
	}
}

func indexByStableID(cs []model.Consumer) map[string]model.Consumer {
	m := make(map[string]model.Consumer, len(cs))
	for _, c := range cs {
		m[c.StableID] = c
	}
	return m
}

func toStringSet(in []string) map[string]struct{} {
	m := make(map[string]struct{}, len(in))
	for _, v := range in {
		m[v] = struct{}{}
	}
	return m
}

// confidenceFor returns the confidence for a field, preferring a field-specific
// entry, then an "overall" entry, then 1.0.
func confidenceFor(c model.Consumer, field string) float64 {
	if c.Confidence != nil {
		if v, ok := c.Confidence[field]; ok {
			return v
		}
		if v, ok := c.Confidence["overall"]; ok {
			return v
		}
	}
	return 1.0
}

// evidenceFor collects provenance locators recorded for a field.
func evidenceFor(c model.Consumer, field string) []string {
	recs, ok := c.Provenance[field]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		if r.Locator != "" {
			out = append(out, r.Locator)
		} else if r.Source != "" {
			out = append(out, r.Source)
		}
	}
	return out
}
