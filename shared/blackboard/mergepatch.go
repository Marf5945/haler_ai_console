package blackboard

import (
	"encoding/json"
	"fmt"
)

// immutableFields may never be altered by an amendment (spec §4).
var immutableFields = map[string]bool{
	"id": true, "v": true, "actor": true, "created_at": true, "sig": true,
	// kind and target_id route authority. Keeping them immutable prevents an
	// otherwise-valid amendment from turning data into a privileged decision
	// or silently retargeting a previously adjudicated modifier.
	"kind": true, "target_id": true,
}

// ApplyMergePatch applies an RFC 7386 JSON Merge Patch to target and
// returns the patched document. Immutable envelope fields present in the
// patch are stripped before application (top level only, per spec §4).
func ApplyMergePatch(target, patch json.RawMessage) (json.RawMessage, error) {
	var patchVal interface{}
	if err := json.Unmarshal(patch, &patchVal); err != nil {
		return nil, fmt.Errorf("invalid patch: %w", err)
	}
	patchObj, ok := patchVal.(map[string]interface{})
	if !ok {
		// RFC 7386: a non-object patch replaces the target wholesale.
		// For events that would destroy the envelope, so reject it.
		return nil, fmt.Errorf("patch must be a JSON object for event amendment")
	}
	for f := range immutableFields {
		delete(patchObj, f)
	}

	var targetVal interface{}
	if err := json.Unmarshal(target, &targetVal); err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}
	targetObj, ok := targetVal.(map[string]interface{})
	if !ok {
		targetObj = map[string]interface{}{}
	}

	merged := mergeObjects(targetObj, patchObj)
	out, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// mergeObjects implements RFC 7386 semantics: null deletes, objects recurse,
// everything else replaces.
func mergeObjects(target, patch map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(target)+len(patch))
	for k, v := range target {
		out[k] = v
	}
	for k, pv := range patch {
		if pv == nil {
			delete(out, k)
			continue
		}
		pObj, pIsObj := pv.(map[string]interface{})
		tObj, tIsObj := out[k].(map[string]interface{})
		if pIsObj && tIsObj {
			out[k] = mergeObjects(tObj, pObj)
			continue
		}
		if pIsObj {
			// Merge into empty object so nested nulls are honored.
			out[k] = mergeObjects(map[string]interface{}{}, pObj)
			continue
		}
		out[k] = pv
	}
	return out
}
