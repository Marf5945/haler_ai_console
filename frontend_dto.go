package main

import "encoding/json"

// frontendDTO is the Wails boundary DTO adapter.
//
// Several internal packages correctly use time.Time for persistence and
// domain logic, but Wails' binding generator tries to expand every exported
// return struct and emits noisy "Not found: time.Time" warnings. Returning a
// JSON-shaped DTO at the App boundary keeps internal types intact while giving
// the frontend plain objects whose time fields are already RFC3339 strings.
func frontendDTO(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return value
	}
	return out
}
