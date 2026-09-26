package record_repo

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/record"
)

type PreviewDecision struct {
	Index         int            `json:"index"`
	Decision      string         `json:"decision"`
	CanonicalKey  string         `json:"canonical_key,omitempty"`
	Values        map[string]any `json:"values,omitempty"`
	ChangedFields []string       `json:"changed_fields,omitempty"`
	Reason        string         `json:"reason,omitempty"`
}

// PreviewBatch simulates SaveBatch in input order. It never mutates records,
// observations, revisions or plugin queues. Concurrent writes may change the
// eventual decision at run time.
func PreviewBatch(datasetID int64, schema record.Schema, candidates []map[string]any) ([]PreviewDecision, error) {
	if len(candidates) == 0 || len(candidates) > 1000 {
		return nil, fmt.Errorf("require 1–1000 candidates")
	}
	if db.Instance() == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	state := map[string]map[string]any{}
	seen := map[string]bool{}
	out := make([]PreviewDecision, 0, len(candidates))
	for i, values := range candidates {
		prepared, err := record.Prepare(schema, values)
		if err != nil {
			out = append(out, PreviewDecision{Index: i, Decision: "invalid", Reason: err.Error()})
			continue
		}
		key := prepared.Key
		if !seen[key] {
			rows, err := db.Instance().QueryString("SELECT normalized::text AS normalized FROM records WHERE dataset_id = ? AND canonical_key = ?", datasetID, key)
			if err != nil {
				return nil, err
			}
			if len(rows) > 0 {
				var previous map[string]any
				if err := json.Unmarshal([]byte(rows[0]["normalized"]), &previous); err != nil {
					return nil, err
				}
				state[key] = previous
			}
			seen[key] = true
		}
		decision := PreviewDecision{Index: i, CanonicalKey: key, Values: prepared.Values}
		if previous, ok := state[key]; ok {
			merged, changed := record.Merge(previous, prepared.Values, schema.EmptyValuePolicy)
			decision.ChangedFields = changed
			if len(changed) == 0 {
				decision.Decision = "unchanged"
			} else {
				decision.Decision = "updated"
			}
			state[key] = merged
		} else {
			decision.Decision = "created"
			state[key] = prepared.Values
			for field := range prepared.Values {
				decision.ChangedFields = append(decision.ChangedFields, field)
			}
			sort.Strings(decision.ChangedFields)
		}
		out = append(out, decision)
	}
	return out, nil
}

// PreviewIdempotency reports a conflict for a supplied write key without
// inserting an observation. A preview without a write key cannot conflict.
func PreviewIdempotency(datasetID int64, key string, values map[string]any, schema record.Schema) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return false, nil
	}
	if len(key) > 256 {
		return false, fmt.Errorf("idempotency key is too long")
	}
	prepared, err := record.Prepare(schema, values)
	if err != nil {
		return false, err
	}
	rows, err := db.Instance().QueryString("SELECT o.normalized_fields::text AS normalized_fields,r.canonical_key FROM record_observations o JOIN records r ON r.id=o.record_id WHERE o.dataset_id=? AND o.idempotency_key=?", datasetID, key)
	if err != nil || len(rows) == 0 {
		return false, err
	}
	var previous map[string]any
	if err := json.Unmarshal([]byte(rows[0]["normalized_fields"]), &previous); err != nil {
		return false, err
	}
	return rows[0]["canonical_key"] != prepared.Key || record.Hash(previous) != record.Hash(prepared.Values), nil
}
